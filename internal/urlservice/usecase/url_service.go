package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"go-shortener/internal/shared/events"
	"go-shortener/internal/urlservice/domain"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.uber.org/zap"
)

const (
	// NanoID alphabet: alphanumeric (a-z, A-Z, 0-9) - 62 characters, case-sensitive
	nanoIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	nanoIDLength   = 8
	maxRetries     = 5
	maxURLLength   = 2048
)

// URLService implements the core business logic for URL shortening
type URLService struct {
	repo       URLRepository
	daprClient DaprClient // may be nil
	logger     *zap.Logger
	baseURL    string
}

// NewURLService creates a new URL service
func NewURLService(repo URLRepository, daprClient DaprClient, logger *zap.Logger, baseURL string) *URLService {
	return &URLService{
		repo:       repo,
		daprClient: daprClient,
		logger:     logger,
		baseURL:    baseURL,
	}
}

// LinkWithClicks represents a link with click count from Analytics Service
type LinkWithClicks struct {
	ShortCode   string    `json:"short_code"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	TotalClicks int64     `json:"total_clicks"`
}

// LinkListResult represents paginated list of links
type LinkListResult struct {
	Links      []LinkWithClicks `json:"links"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	TotalPages int              `json:"total_pages"`
}

// ListLinksParams represents parameters for listing links
type ListLinksParams struct {
	Page          int
	PerPage       int
	Sort          string    // "created_at" only for now
	Order         string    // "asc" or "desc"
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Search        string
}

// CreateShortURL validates, deduplicates, and creates a short URL
func (s *URLService) CreateShortURL(ctx context.Context, originalURL string) (*domain.URL, error) {
	// Validate the URL
	if err := validateURL(originalURL); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidURL, err)
	}

	// Check for duplicate (deduplication)
	existing, err := s.repo.FindByOriginalURL(ctx, originalURL)
	if err == nil {
		// URL already exists, return existing short code
		return existing, nil
	}
	if !errors.Is(err, domain.ErrURLNotFound) {
		// Unexpected error (not "not found")
		return nil, err
	}

	// Generate short code with collision retry
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check for context cancellation between retries
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Generate NanoID
		shortCode, err := gonanoid.Generate(nanoIDAlphabet, nanoIDLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}

		// Try to save
		url, err := s.repo.Save(ctx, shortCode, originalURL)
		if err != nil {
			// Check if it's a unique constraint violation (collision)
			// SQLite returns "UNIQUE constraint failed" in the error message
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				// Collision detected, retry
				continue
			}
			// Other error, return it
			return nil, err
		}

		// Success
		return url, nil
	}

	// Max retries exceeded
	return nil, domain.ErrShortCodeConflict
}

// GetByShortCode retrieves a URL by its short code
func (s *URLService) GetByShortCode(ctx context.Context, code string) (*domain.URL, error) {
	return s.repo.FindByShortCode(ctx, code)
}

// ListLinks retrieves paginated list of links with click counts
func (s *URLService) ListLinks(ctx context.Context, params ListLinksParams) (*LinkListResult, error) {
	// Validate sort field (only created_at supported)
	if params.Sort == "" {
		params.Sort = "created_at"
	}
	if params.Sort != "created_at" {
		return nil, fmt.Errorf("unsupported sort field: %s", params.Sort)
	}

	// Validate order
	if params.Order == "" {
		params.Order = "desc"
	}
	if params.Order != "asc" && params.Order != "desc" {
		return nil, fmt.Errorf("order must be asc or desc, got: %s", params.Order)
	}

	// Calculate offset
	offset := (params.Page - 1) * params.PerPage

	// Fetch URLs from repository
	urls, err := s.repo.FindAll(ctx, FindAllParams{
		CreatedAfter:  params.CreatedAfter,
		CreatedBefore: params.CreatedBefore,
		Search:        params.Search,
		SortOrder:     params.Order,
		Limit:         params.PerPage,
		Offset:        offset,
	})
	if err != nil {
		return nil, err
	}

	// Fetch total count
	total, err := s.repo.Count(ctx, CountParams{
		CreatedAfter:  params.CreatedAfter,
		CreatedBefore: params.CreatedBefore,
		Search:        params.Search,
	})
	if err != nil {
		return nil, err
	}

	// Extract short codes for click count enrichment
	shortCodes := make([]string, len(urls))
	for i, u := range urls {
		shortCodes[i] = u.ShortCode
	}

	// Get click counts from Analytics Service
	clickCounts := s.getClickCounts(ctx, shortCodes)

	// Build response with enriched data
	links := make([]LinkWithClicks, len(urls))
	for i, u := range urls {
		links[i] = LinkWithClicks{
			ShortCode:   u.ShortCode,
			ShortURL:    s.baseURL + "/" + u.ShortCode,
			OriginalURL: u.OriginalURL,
			CreatedAt:   u.CreatedAt,
			TotalClicks: clickCounts[u.ShortCode],
		}
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

	return &LinkListResult{
		Links:      links,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

// GetLinkDetail retrieves a single link with click count
func (s *URLService) GetLinkDetail(ctx context.Context, shortCode string) (*LinkWithClicks, error) {
	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Fetch click count
	clickCounts := s.getClickCounts(ctx, []string{shortCode})

	return &LinkWithClicks{
		ShortCode:   url.ShortCode,
		ShortURL:    s.baseURL + "/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
		TotalClicks: clickCounts[shortCode],
	}, nil
}

// DeleteLink deletes a link and publishes deletion event
func (s *URLService) DeleteLink(ctx context.Context, shortCode string) error {
	// Delete from repository (idempotent)
	if err := s.repo.Delete(ctx, shortCode); err != nil {
		return err
	}

	// Fire-and-forget: publish link.deleted event
	if s.daprClient != nil {
		go s.publishLinkDeletedEvent(shortCode)
	}

	return nil
}

// getClickCounts fetches click counts from Analytics Service via Dapr
func (s *URLService) getClickCounts(ctx context.Context, shortCodes []string) map[string]int64 {
	result := make(map[string]int64)

	// If no Dapr client, return zeros
	if s.daprClient == nil {
		return result
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Fetch click count for each short code
	for _, code := range shortCodes {
		// Call Analytics Service via Dapr service invocation
		resp, err := s.daprClient.InvokeMethod(timeoutCtx, "analytics-service", "analytics/"+code, "get")
		if err != nil {
			s.logger.Warn("failed to fetch click count from Analytics Service",
				zap.String("short_code", code),
				zap.Error(err),
			)
			result[code] = 0
			continue
		}

		// Parse response
		var analyticsData struct {
			TotalClicks int64 `json:"total_clicks"`
		}
		if err := json.Unmarshal(resp, &analyticsData); err != nil {
			s.logger.Warn("failed to parse Analytics Service response",
				zap.String("short_code", code),
				zap.Error(err),
			)
			result[code] = 0
			continue
		}

		result[code] = analyticsData.TotalClicks
	}

	return result
}

// publishLinkDeletedEvent publishes link deletion event to Dapr pub/sub
func (s *URLService) publishLinkDeletedEvent(shortCode string) {
	event := events.LinkDeletedEvent{
		ShortCode: shortCode,
		DeletedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("failed to marshal link deleted event",
			zap.String("short_code", shortCode),
			zap.Error(err),
		)
		return
	}

	ctx := context.Background()
	if err := s.daprClient.PublishEvent(ctx, "pubsub", "link-deleted", data); err != nil {
		s.logger.Warn("failed to publish link deleted event",
			zap.String("short_code", shortCode),
			zap.Error(err),
		)
		// Deletion already succeeded — log and continue
	}
}

// validateURL validates the URL format and constraints
func validateURL(rawURL string) error {
	// Check length
	if len(rawURL) > maxURLLength {
		return fmt.Errorf("url exceeds maximum length of %d characters", maxURLLength)
	}

	// Parse URL
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url format: %w", err)
	}

	// Check scheme (must be http or https)
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("url scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Check host is not empty
	if parsedURL.Host == "" {
		return fmt.Errorf("url must have a host")
	}

	return nil
}

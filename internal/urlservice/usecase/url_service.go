package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go-shortener/internal/urlservice/domain"

	gonanoid "github.com/matoous/go-nanoid/v2"
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
	repo URLRepository
}

// NewURLService creates a new URL service
func NewURLService(repo URLRepository) *URLService {
	return &URLService{
		repo: repo,
	}
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

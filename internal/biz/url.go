package biz

import (
	"context"
	"time"

	"go-shortener/internal/domain"

	v1 "go-shortener/api/shortener/v1"

	"github.com/go-kratos/kratos/v2/log"
)

// Application layer errors that wrap domain errors with API error types
var (
	ErrURLNotFound     = v1.ErrorUrlNotFound("url not found")
	ErrURLExpired      = v1.ErrorUrlExpired("url has expired")
	ErrInvalidURL      = v1.ErrorInvalidUrl("invalid url format")
	ErrShortCodeExists = v1.ErrorShortCodeExists("short code already exists")
	ErrInvalidCode     = v1.ErrorInvalidShortCode("invalid short code format")
)

type URL struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	ClickCount  int64
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type URLRepo interface {
	Create(ctx context.Context, url *URL) (*URL, error)
	GetByShortCode(ctx context.Context, shortCode string) (*URL, error)
	IncrementClickCount(ctx context.Context, shortCode string) error
	Delete(ctx context.Context, shortCode string) error
	List(ctx context.Context, page, pageSize int) ([]*URL, int, error)
	ExistsShortCode(ctx context.Context, shortCode string) (bool, error)
}

type URLUsecase struct {
	repo   URLRepo
	log    *log.Helper
	config *URLConfig
}

type URLConfig struct {
	BaseURL        string
	DefaultExpiry  time.Duration
	ShortCodeLen   int
	MaxCustomLen   int
	MinCustomLen   int
}

func DefaultURLConfig() *URLConfig {
	return &URLConfig{
		BaseURL:        "http://localhost:8000",
		DefaultExpiry:  0,
		ShortCodeLen:   domain.DefaultShortCodeLength,
		MaxCustomLen:   domain.MaxCustomCodeLength,
		MinCustomLen:   domain.MinCustomCodeLength,
	}
}

func NewURLUsecase(repo URLRepo, logger log.Logger) *URLUsecase {
	return &URLUsecase{
		repo:   repo,
		log:    log.NewHelper(logger),
		config: DefaultURLConfig(),
	}
}

func (uc *URLUsecase) CreateURL(ctx context.Context, originalURL string, customCode *string, expiresAt *time.Time) (*URL, error) {
	// Validate original URL using domain value object
	domainOriginalURL, err := domain.NewOriginalURL(originalURL)
	if err != nil {
		return nil, ErrInvalidURL
	}

	// Create or validate short code using domain value object
	var shortCode domain.ShortCode
	if customCode != nil && *customCode != "" {
		shortCode, err = domain.NewShortCode(*customCode)
		if err != nil {
			return nil, ErrInvalidCode
		}

		exists, err := uc.repo.ExistsShortCode(ctx, shortCode.String())
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrShortCodeExists
		}
	} else {
		shortCode, err = uc.generateUniqueCode(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Create domain entity
	domainURL := domain.NewURL(shortCode, domainOriginalURL, expiresAt)

	// Convert to DTO for persistence
	u := &URL{
		ShortCode:   domainURL.ShortCode().String(),
		OriginalURL: domainURL.OriginalURL().String(),
		ExpiresAt:   domainURL.ExpiresAt(),
	}

	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	uc.log.WithContext(ctx).Infof("Created URL: %s -> %s", shortCode.String(), originalURL)
	return created, nil
}

func (uc *URLUsecase) GetURL(ctx context.Context, shortCode string) (*URL, error) {
	u, err := uc.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrURLNotFound
	}

	if u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLExpired
	}

	return u, nil
}

func (uc *URLUsecase) RedirectURL(ctx context.Context, shortCode string) (string, error) {
	u, err := uc.GetURL(ctx, shortCode)
	if err != nil {
		return "", err
	}

	if err := uc.repo.IncrementClickCount(ctx, shortCode); err != nil {
		uc.log.WithContext(ctx).Warnf("Failed to increment click count for %s: %v", shortCode, err)
	}

	return u.OriginalURL, nil
}

func (uc *URLUsecase) GetURLStats(ctx context.Context, shortCode string) (*URL, error) {
	return uc.GetURL(ctx, shortCode)
}

func (uc *URLUsecase) DeleteURL(ctx context.Context, shortCode string) error {
	return uc.repo.Delete(ctx, shortCode)
}

func (uc *URLUsecase) ListURLs(ctx context.Context, page, pageSize int) ([]*URL, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.List(ctx, page, pageSize)
}

func (uc *URLUsecase) GetShortURL(shortCode string) string {
	return uc.config.BaseURL + "/r/" + shortCode
}

// generateUniqueCode generates a unique short code using the domain layer.
func (uc *URLUsecase) generateUniqueCode(ctx context.Context) (domain.ShortCode, error) {
	for i := 0; i < 10; i++ {
		code, err := domain.GenerateShortCode(uc.config.ShortCodeLen)
		if err != nil {
			return domain.ShortCode{}, err
		}

		exists, err := uc.repo.ExistsShortCode(ctx, code.String())
		if err != nil {
			return domain.ShortCode{}, err
		}
		if !exists {
			return code, nil
		}
	}
	return domain.ShortCode{}, ErrShortCodeExists
}

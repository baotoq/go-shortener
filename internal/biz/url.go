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

// URLUsecase handles URL business logic.
type URLUsecase struct {
	repo   domain.URLRepository
	uow    domain.UnitOfWork
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

// NewURLUsecase creates a new URLUsecase.
func NewURLUsecase(repo domain.URLRepository, uow domain.UnitOfWork, logger log.Logger) *URLUsecase {
	return &URLUsecase{
		repo:   repo,
		uow:    uow,
		log:    log.NewHelper(logger),
		config: DefaultURLConfig(),
	}
}

func (uc *URLUsecase) CreateURL(ctx context.Context, originalURL string, customCode *string, expiresAt *time.Time) (*domain.URL, error) {
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

		exists, err := uc.repo.Exists(ctx, shortCode)
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

	// Create domain entity (this raises URLCreated event)
	domainURL := domain.NewURL(shortCode, domainOriginalURL, expiresAt)

	// Execute within transaction, pass aggregate for event dispatch
	err = uc.uow.Do(ctx, func(txCtx context.Context) error {
		return uc.repo.Save(txCtx, domainURL)
	}, domainURL)
	if err != nil {
		return nil, err
	}

	return domainURL, nil
}

func (uc *URLUsecase) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	sc, err := domain.NewShortCode(shortCode)
	if err != nil {
		return nil, ErrInvalidCode
	}

	u, err := uc.repo.FindByShortCode(ctx, sc)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrURLNotFound
	}

	if u.IsExpired() {
		return nil, ErrURLExpired
	}

	return u, nil
}

// RedirectURL handles URL redirection and dispatches click events.
func (uc *URLUsecase) RedirectURL(ctx context.Context, shortCode string) (string, error) {
	return uc.RedirectURLWithContext(ctx, shortCode, "", "", "")
}

// RedirectURLWithContext handles URL redirection with additional context for analytics.
func (uc *URLUsecase) RedirectURLWithContext(ctx context.Context, shortCode, userAgent, ipAddress, referrer string) (string, error) {
	u, err := uc.GetURL(ctx, shortCode)
	if err != nil {
		return "", err
	}

	// Record click - raises URLClicked event
	// The ClickEventHandler will handle incrementing the click count
	u.RecordClick(userAgent, ipAddress, referrer)

	// Dispatch events via UoW
	err = uc.uow.Do(ctx, func(txCtx context.Context) error {
		return nil
	}, u)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("Failed to dispatch click event for %s: %v", shortCode, err)
	}

	return u.OriginalURL().String(), nil
}

func (uc *URLUsecase) GetURLStats(ctx context.Context, shortCode string) (*domain.URL, error) {
	return uc.GetURL(ctx, shortCode)
}

func (uc *URLUsecase) DeleteURL(ctx context.Context, shortCode string) error {
	sc, err := domain.NewShortCode(shortCode)
	if err != nil {
		return ErrInvalidCode
	}

	deletedAggregate := domain.NewDeletedURLAggregate(shortCode)

	// Execute within transaction, pass aggregate for event dispatch
	return uc.uow.Do(ctx, func(txCtx context.Context) error {
		return uc.repo.Delete(txCtx, sc)
	}, deletedAggregate)
}

func (uc *URLUsecase) ListURLs(ctx context.Context, page, pageSize int) ([]*domain.URL, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.FindAll(ctx, page, pageSize)
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

		exists, err := uc.repo.Exists(ctx, code)
		if err != nil {
			return domain.ShortCode{}, err
		}
		if !exists {
			return code, nil
		}
	}
	return domain.ShortCode{}, ErrShortCodeExists
}

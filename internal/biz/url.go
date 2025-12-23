package biz

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"
	"time"

	v1 "go-shortener/api/shortener/v1"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrURLNotFound     = v1.ErrorUrlNotFound("url not found")
	ErrURLExpired      = v1.ErrorUrlExpired("url has expired")
	ErrInvalidURL      = v1.ErrorInvalidUrl("invalid url format")
	ErrShortCodeExists = v1.ErrorShortCodeExists("short code already exists")
	ErrInvalidCode     = v1.ErrorInvalidShortCode("invalid short code format")
)

var shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
		ShortCodeLen:   6,
		MaxCustomLen:   20,
		MinCustomLen:   3,
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
	if err := uc.validateURL(originalURL); err != nil {
		return nil, err
	}

	var shortCode string
	var err error

	if customCode != nil && *customCode != "" {
		if err := uc.validateCustomCode(*customCode); err != nil {
			return nil, err
		}
		exists, err := uc.repo.ExistsShortCode(ctx, *customCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrShortCodeExists
		}
		shortCode = *customCode
	} else {
		shortCode, err = uc.generateUniqueCode(ctx)
		if err != nil {
			return nil, err
		}
	}

	u := &URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		ExpiresAt:   expiresAt,
	}

	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	uc.log.WithContext(ctx).Infof("Created URL: %s -> %s", shortCode, originalURL)
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

func (uc *URLUsecase) validateURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	if parsed.Host == "" {
		return ErrInvalidURL
	}

	return nil
}

func (uc *URLUsecase) validateCustomCode(code string) error {
	if len(code) < uc.config.MinCustomLen || len(code) > uc.config.MaxCustomLen {
		return ErrInvalidCode
	}

	if !shortCodeRegex.MatchString(code) {
		return ErrInvalidCode
	}

	return nil
}

func (uc *URLUsecase) generateUniqueCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateShortCode(uc.config.ShortCodeLen)
		if err != nil {
			return "", err
		}

		exists, err := uc.repo.ExistsShortCode(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", ErrShortCodeExists
}

func generateShortCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := base64.URLEncoding.EncodeToString(bytes)
	code = strings.TrimRight(code, "=")
	if len(code) > length {
		code = code[:length]
	}
	return code, nil
}

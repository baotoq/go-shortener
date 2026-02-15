package usecase

import (
	"context"
	"go-shortener/internal/urlservice/domain"
)

type URLRepository interface {
	Save(ctx context.Context, shortCode, originalURL string) (*domain.URL, error)
	FindByShortCode(ctx context.Context, code string) (*domain.URL, error)
	FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error)
}

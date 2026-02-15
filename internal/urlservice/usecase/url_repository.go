package usecase

import (
	"context"
	"go-shortener/internal/urlservice/domain"
	"time"
)

type FindAllParams struct {
	CreatedAfter  time.Time // zero value means no filter
	CreatedBefore time.Time // zero value means no filter
	Search        string    // empty means no filter
	SortOrder     string    // "asc" or "desc"
	Limit         int
	Offset        int
}

type CountParams struct {
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Search        string
}

type URLRepository interface {
	Save(ctx context.Context, shortCode, originalURL string) (*domain.URL, error)
	FindByShortCode(ctx context.Context, code string) (*domain.URL, error)
	FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error)
	FindAll(ctx context.Context, params FindAllParams) ([]domain.URL, error)
	Count(ctx context.Context, params CountParams) (int64, error)
	Delete(ctx context.Context, shortCode string) error
}

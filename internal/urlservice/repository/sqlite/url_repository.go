package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"go-shortener/internal/urlservice/domain"
	"go-shortener/internal/urlservice/repository/sqlite/sqlc"
	"go-shortener/internal/urlservice/usecase"
)

// URLRepository implements the usecase.URLRepository interface using sqlc
type URLRepository struct {
	queries *sqlc.Queries
}

// NewURLRepository creates a new SQLite-backed URL repository
func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		queries: sqlc.New(db),
	}
}

// Ensure URLRepository implements usecase.URLRepository at compile time
var _ usecase.URLRepository = (*URLRepository)(nil)

// Save creates a new URL record in the database
func (r *URLRepository) Save(ctx context.Context, shortCode, originalURL string) (*domain.URL, error) {
	url, err := r.queries.CreateURL(ctx, sqlc.CreateURLParams{
		ShortCode:   shortCode,
		OriginalUrl: originalURL,
	})
	if err != nil {
		return nil, err
	}

	return &domain.URL{
		ID:          url.ID,
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalUrl,
		CreatedAt:   url.CreatedAt,
	}, nil
}

// FindByShortCode retrieves a URL by its short code
func (r *URLRepository) FindByShortCode(ctx context.Context, code string) (*domain.URL, error) {
	url, err := r.queries.FindByShortCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, err
	}

	return &domain.URL{
		ID:          url.ID,
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalUrl,
		CreatedAt:   url.CreatedAt,
	}, nil
}

// FindByOriginalURL retrieves a URL by its original URL (for deduplication)
func (r *URLRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error) {
	url, err := r.queries.FindByOriginalURL(ctx, originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, err
	}

	return &domain.URL{
		ID:          url.ID,
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalUrl,
		CreatedAt:   url.CreatedAt,
	}, nil
}

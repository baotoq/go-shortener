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

// toNullableString converts a time.Time or string to sql.NullString
func toNullableString(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
		return v
	default:
		return value
	}
}

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

// FindAll retrieves URLs with filtering, sorting, and pagination
func (r *URLRepository) FindAll(ctx context.Context, params usecase.FindAllParams) ([]domain.URL, error) {
	// Convert Go params to sqlc nullable types
	var createdAfter, createdBefore, search interface{}

	if !params.CreatedAfter.IsZero() {
		createdAfter = params.CreatedAfter.Format("2006-01-02T15:04:05Z")
	}

	if !params.CreatedBefore.IsZero() {
		createdBefore = params.CreatedBefore.Format("2006-01-02T15:04:05Z")
	}

	if params.Search != "" {
		search = params.Search
	}

	var urls []sqlc.Url
	var err error

	// Use the appropriate query based on sort order
	if params.SortOrder == "asc" {
		urls, err = r.queries.ListURLsAsc(ctx, sqlc.ListURLsAscParams{
			CreatedAfter:  createdAfter,
			CreatedBefore: createdBefore,
			Search:        search,
			Limit:         int64(params.Limit),
			Offset:        int64(params.Offset),
		})
	} else {
		urls, err = r.queries.ListURLs(ctx, sqlc.ListURLsParams{
			CreatedAfter:  createdAfter,
			CreatedBefore: createdBefore,
			Search:        search,
			Limit:         int64(params.Limit),
			Offset:        int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	// Convert sqlc results to domain.URL
	result := make([]domain.URL, len(urls))
	for i, url := range urls {
		result[i] = domain.URL{
			ID:          url.ID,
			ShortCode:   url.ShortCode,
			OriginalURL: url.OriginalUrl,
			CreatedAt:   url.CreatedAt,
		}
	}

	return result, nil
}

// Count returns total count of URLs matching filters
func (r *URLRepository) Count(ctx context.Context, params usecase.CountParams) (int64, error) {
	var createdAfter, createdBefore, search interface{}

	if !params.CreatedAfter.IsZero() {
		createdAfter = params.CreatedAfter.Format("2006-01-02T15:04:05Z")
	}

	if !params.CreatedBefore.IsZero() {
		createdBefore = params.CreatedBefore.Format("2006-01-02T15:04:05Z")
	}

	if params.Search != "" {
		search = params.Search
	}

	return r.queries.CountURLs(ctx, sqlc.CountURLsParams{
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Search:        search,
	})
}

// Delete hard deletes a URL by short code (idempotent)
func (r *URLRepository) Delete(ctx context.Context, shortCode string) error {
	return r.queries.DeleteURL(ctx, shortCode)
}

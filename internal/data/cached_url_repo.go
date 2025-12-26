package data

import (
	"context"

	"go-shortener/internal/domain"
)

// Compile-time interface check
var _ domain.URLRepository = (*CachedURLRepository)(nil)

// CachedURLRepository wraps a URLRepository with caching capabilities.
// It implements the decorator pattern to add caching without modifying the underlying repository.
type CachedURLRepository struct {
	repo  *urlRepo
	cache URLCache
}

// NewCachedURLRepository creates a new cached repository wrapper.
func NewCachedURLRepository(repo *urlRepo, cache URLCache) domain.URLRepository {
	return &CachedURLRepository{
		repo:  repo,
		cache: cache,
	}
}

// Save persists a URL and updates the cache.
func (r *CachedURLRepository) Save(ctx context.Context, u *domain.URL) error {
	if err := r.repo.Save(ctx, u); err != nil {
		return err
	}

	// Cache after successful save
	_ = r.cache.Set(ctx, u)
	return nil
}

// FindByShortCode retrieves a URL, checking cache first.
func (r *CachedURLRepository) FindByShortCode(ctx context.Context, code domain.ShortCode) (*domain.URL, error) {
	// Try cache first
	if cached, err := r.cache.Get(ctx, code); err == nil && cached != nil {
		return cached, nil
	}

	// Cache miss, fetch from database
	u, err := r.repo.FindByShortCode(ctx, code)
	if err != nil || u == nil {
		return u, err
	}

	// Cache the result
	_ = r.cache.Set(ctx, u)
	return u, nil
}

// Delete removes a URL and invalidates the cache.
func (r *CachedURLRepository) Delete(ctx context.Context, code domain.ShortCode) error {
	if err := r.repo.Delete(ctx, code); err != nil {
		return err
	}

	_ = r.cache.Invalidate(ctx, code)
	return nil
}

// FindAll retrieves all URLs with pagination.
// This operation is not cached as it returns paginated results.
func (r *CachedURLRepository) FindAll(ctx context.Context, page, pageSize int) ([]*domain.URL, int, error) {
	return r.repo.FindAll(ctx, page, pageSize)
}

// Exists checks if a short code already exists.
// This operation is not cached to ensure accurate existence checks.
func (r *CachedURLRepository) Exists(ctx context.Context, code domain.ShortCode) (bool, error) {
	return r.repo.Exists(ctx, code)
}

// IncrementClickCount atomically increments the click count and invalidates the cache.
func (r *CachedURLRepository) IncrementClickCount(ctx context.Context, code domain.ShortCode) error {
	if err := r.repo.IncrementClickCount(ctx, code); err != nil {
		return err
	}

	_ = r.cache.Invalidate(ctx, code)
	return nil
}

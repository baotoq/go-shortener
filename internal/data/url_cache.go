package data

import (
	"context"
	"encoding/json"
	"time"

	"go-shortener/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	urlCachePrefix = "url:"
	urlCacheTTL    = 10 * time.Minute
)

// URLCache defines the interface for URL caching operations.
// Implementations should handle cache misses gracefully by returning nil, nil.
type URLCache interface {
	// Get retrieves a URL from cache by its short code.
	// Returns nil, nil if the URL is not in cache (cache miss).
	Get(ctx context.Context, shortCode domain.ShortCode) (*domain.URL, error)

	// Set stores a URL in the cache.
	Set(ctx context.Context, u *domain.URL) error

	// Invalidate removes a URL from the cache.
	Invalidate(ctx context.Context, shortCode domain.ShortCode) error
}

// Compile-time interface checks
var (
	_ URLCache = (*RedisURLCache)(nil)
	_ URLCache = (*noopURLCache)(nil)
)

// RedisURLCache implements URLCache using Redis.
type RedisURLCache struct {
	rdb *redis.Client
	log *log.Helper
}

// NewRedisURLCache creates a new Redis-based URL cache.
// Returns a no-op cache if Redis client is nil.
func NewRedisURLCache(rdb *redis.Client, logger log.Logger) URLCache {
	if rdb == nil {
		return &noopURLCache{}
	}
	return &RedisURLCache{
		rdb: rdb,
		log: log.NewHelper(logger),
	}
}

// cachedURL is the serialization format for cached URLs.
type cachedURL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	ClickCount  int64      `json:"click_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (c *RedisURLCache) cacheKey(shortCode string) string {
	return urlCachePrefix + shortCode
}

// Get retrieves a URL from Redis cache.
func (c *RedisURLCache) Get(ctx context.Context, shortCode domain.ShortCode) (*domain.URL, error) {
	data, err := c.rdb.Get(ctx, c.cacheKey(shortCode.String())).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		c.log.WithContext(ctx).Warnf("Failed to get URL from cache: %v", err)
		return nil, nil // Treat errors as cache miss
	}

	var cached cachedURL
	if err := json.Unmarshal(data, &cached); err != nil {
		c.log.WithContext(ctx).Warnf("Failed to unmarshal cached URL: %v", err)
		return nil, nil
	}

	shortCodeVO, err := domain.NewShortCode(cached.ShortCode)
	if err != nil {
		return nil, nil
	}

	originalURLVO, err := domain.NewOriginalURL(cached.OriginalURL)
	if err != nil {
		return nil, nil
	}

	return domain.ReconstructURL(
		cached.ID,
		shortCodeVO,
		originalURLVO,
		cached.ClickCount,
		cached.ExpiresAt,
		cached.CreatedAt,
		cached.UpdatedAt,
	), nil
}

// Set stores a URL in Redis cache.
func (c *RedisURLCache) Set(ctx context.Context, u *domain.URL) error {
	cached := cachedURL{
		ID:          u.ID(),
		ShortCode:   u.ShortCode().String(),
		OriginalURL: u.OriginalURL().String(),
		ClickCount:  u.ClickCount(),
		ExpiresAt:   u.ExpiresAt(),
		CreatedAt:   u.CreatedAt(),
		UpdatedAt:   u.UpdatedAt(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		c.log.WithContext(ctx).Warnf("Failed to marshal URL for cache: %v", err)
		return nil // Don't fail the operation due to cache errors
	}

	if err := c.rdb.Set(ctx, c.cacheKey(u.ShortCode().String()), data, urlCacheTTL).Err(); err != nil {
		c.log.WithContext(ctx).Warnf("Failed to cache URL: %v", err)
	}

	return nil
}

// Invalidate removes a URL from Redis cache.
func (c *RedisURLCache) Invalidate(ctx context.Context, shortCode domain.ShortCode) error {
	if err := c.rdb.Del(ctx, c.cacheKey(shortCode.String())).Err(); err != nil {
		c.log.WithContext(ctx).Warnf("Failed to invalidate URL cache: %v", err)
	}
	return nil
}

// noopURLCache is a no-op implementation when Redis is not available.
type noopURLCache struct{}

func (c *noopURLCache) Get(context.Context, domain.ShortCode) (*domain.URL, error) {
	return nil, nil
}

func (c *noopURLCache) Set(context.Context, *domain.URL) error {
	return nil
}

func (c *noopURLCache) Invalidate(context.Context, domain.ShortCode) error {
	return nil
}

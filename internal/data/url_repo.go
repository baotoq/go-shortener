package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-shortener/ent"
	"go-shortener/ent/url"
	"go-shortener/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/samber/lo"
)

// Compile-time interface check
var _ domain.URLRepository = (*urlRepo)(nil)

const (
	urlCachePrefix = "url:"
	urlCacheTTL    = 10 * time.Minute
)

// urlRepo implements domain.URLRepository interface.
type urlRepo struct {
	data *Data
	log  *log.Helper
}

// NewURLRepo creates a new URL repository.
func NewURLRepo(data *Data, logger log.Logger) domain.URLRepository {
	return &urlRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// client returns the transactional client if in a transaction, otherwise the default client.
func (r *urlRepo) client(ctx context.Context) *ent.Client {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.data.db
}

// Save persists a URL entity.
func (r *urlRepo) Save(ctx context.Context, u *domain.URL) error {
	client := r.client(ctx)

	if u.ID() == 0 {
		// Create new URL
		builder := client.URL.Create().
			SetShortCode(u.ShortCode().String()).
			SetOriginalURL(u.OriginalURL().String()).
			SetClickCount(u.ClickCount())

		if u.ExpiresAt() != nil {
			builder.SetExpiresAt(*u.ExpiresAt())
		}

		created, err := builder.Save(ctx)
		if err != nil {
			return err
		}

		u.SetID(int64(created.ID))
		r.cacheURL(ctx, u)
		return nil
	}

	// Update existing URL
	updateBuilder := client.URL.UpdateOneID(int(u.ID())).
		SetClickCount(u.ClickCount()).
		SetUpdatedAt(u.UpdatedAt())

	if u.ExpiresAt() != nil {
		updateBuilder.SetExpiresAt(*u.ExpiresAt())
	}

	_, err := updateBuilder.Save(ctx)
	if err != nil {
		return err
	}

	r.invalidateCache(ctx, u.ShortCode().String())
	return nil
}

// FindByShortCode retrieves a URL by its short code.
func (r *urlRepo) FindByShortCode(ctx context.Context, code domain.ShortCode) (*domain.URL, error) {
	if cached := r.getCachedURL(ctx, code.String()); cached != nil {
		return cached, nil
	}

	u, err := r.client(ctx).URL.Query().
		Where(url.ShortCodeEQ(code.String())).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	result := r.entToDomain(u)
	r.cacheURL(ctx, result)
	return result, nil
}

// Delete removes a URL by its short code.
func (r *urlRepo) Delete(ctx context.Context, code domain.ShortCode) error {
	_, err := r.client(ctx).URL.Delete().
		Where(url.ShortCodeEQ(code.String())).
		Exec(ctx)
	if err != nil {
		return err
	}

	r.invalidateCache(ctx, code.String())
	return nil
}

// FindAll retrieves all URLs with pagination.
func (r *urlRepo) FindAll(ctx context.Context, page, pageSize int) ([]*domain.URL, int, error) {
	client := r.client(ctx)
	offset := (page - 1) * pageSize

	urls, err := client.URL.Query().
		Order(ent.Desc(url.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := client.URL.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := lo.Map(urls, func(u *ent.URL, _ int) *domain.URL {
		return r.entToDomain(u)
	})

	return result, total, nil
}

// Exists checks if a short code already exists.
func (r *urlRepo) Exists(ctx context.Context, code domain.ShortCode) (bool, error) {
	exists, err := r.client(ctx).URL.Query().
		Where(url.ShortCodeEQ(code.String())).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// IncrementClickCount atomically increments the click count.
func (r *urlRepo) IncrementClickCount(ctx context.Context, code domain.ShortCode) error {
	_, err := r.client(ctx).URL.Update().
		Where(url.ShortCodeEQ(code.String())).
		AddClickCount(1).
		Save(ctx)
	if err != nil {
		return err
	}

	r.invalidateCache(ctx, code.String())
	return nil
}

// entToDomain converts an Ent URL entity to a domain URL.
func (r *urlRepo) entToDomain(u *ent.URL) *domain.URL {
	shortCode, _ := domain.NewShortCode(u.ShortCode)
	originalURL, _ := domain.NewOriginalURL(u.OriginalURL)

	return domain.ReconstructURL(
		int64(u.ID),
		shortCode,
		originalURL,
		u.ClickCount,
		u.ExpiresAt,
		u.CreatedAt,
		u.UpdatedAt,
	)
}

func (r *urlRepo) cacheKey(shortCode string) string {
	return fmt.Sprintf("%s%s", urlCachePrefix, shortCode)
}

type cachedURL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	ClickCount  int64      `json:"click_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (r *urlRepo) cacheURL(ctx context.Context, u *domain.URL) {
	if r.data.rdb == nil {
		return
	}

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
		r.log.WithContext(ctx).Warnf("Failed to marshal URL for cache: %v", err)
		return
	}

	if err := r.data.rdb.Set(ctx, r.cacheKey(u.ShortCode().String()), data, urlCacheTTL).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to cache URL: %v", err)
	}
}

func (r *urlRepo) getCachedURL(ctx context.Context, shortCode string) *domain.URL {
	if r.data.rdb == nil {
		return nil
	}

	data, err := r.data.rdb.Get(ctx, r.cacheKey(shortCode)).Bytes()
	if err != nil {
		return nil
	}

	var cached cachedURL
	if err := json.Unmarshal(data, &cached); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to unmarshal cached URL: %v", err)
		return nil
	}

	shortCodeVO, _ := domain.NewShortCode(cached.ShortCode)
	originalURLVO, _ := domain.NewOriginalURL(cached.OriginalURL)

	return domain.ReconstructURL(
		cached.ID,
		shortCodeVO,
		originalURLVO,
		cached.ClickCount,
		cached.ExpiresAt,
		cached.CreatedAt,
		cached.UpdatedAt,
	)
}

func (r *urlRepo) invalidateCache(ctx context.Context, shortCode string) {
	if r.data.rdb == nil {
		return
	}

	if err := r.data.rdb.Del(ctx, r.cacheKey(shortCode)).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to invalidate URL cache: %v", err)
	}
}

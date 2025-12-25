package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-shortener/ent"
	"go-shortener/ent/url"
	"go-shortener/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/samber/lo"
)

const (
	urlCachePrefix = "url:"
	urlCacheTTL    = 10 * time.Minute
)

type urlRepo struct {
	data *Data
	log  *log.Helper
}

func NewURLRepo(data *Data, logger log.Logger) biz.URLRepo {
	return &urlRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *urlRepo) Create(ctx context.Context, u *biz.URL) (*biz.URL, error) {
	builder := r.data.db.URL.Create().
		SetShortCode(u.ShortCode).
		SetOriginalURL(u.OriginalURL)

	if u.ExpiresAt != nil {
		builder.SetExpiresAt(*u.ExpiresAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	result := r.entToBiz(created)

	r.cacheURL(ctx, result)

	return result, nil
}

func (r *urlRepo) GetByShortCode(ctx context.Context, shortCode string) (*biz.URL, error) {
	if cached := r.getCachedURL(ctx, shortCode); cached != nil {
		return cached, nil
	}

	u, err := r.data.db.URL.Query().
		Where(url.ShortCodeEQ(shortCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	result := r.entToBiz(u)

	r.cacheURL(ctx, result)

	return result, nil
}

func (r *urlRepo) IncrementClickCount(ctx context.Context, shortCode string) error {
	_, err := r.data.db.URL.Update().
		Where(url.ShortCodeEQ(shortCode)).
		AddClickCount(1).
		Save(ctx)
	if err != nil {
		return err
	}

	r.invalidateCache(ctx, shortCode)

	return nil
}

func (r *urlRepo) Delete(ctx context.Context, shortCode string) error {
	_, err := r.data.db.URL.Delete().
		Where(url.ShortCodeEQ(shortCode)).
		Exec(ctx)
	if err != nil {
		return err
	}

	r.invalidateCache(ctx, shortCode)

	return nil
}

func (r *urlRepo) List(ctx context.Context, page, pageSize int) ([]*biz.URL, int, error) {
	offset := (page - 1) * pageSize

	urls, err := r.data.db.URL.Query().
		Order(ent.Desc(url.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.data.db.URL.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := lo.Map(urls, func(u *ent.URL, _ int) *biz.URL {
		return r.entToBiz(u)
	})

	return result, total, nil
}

func (r *urlRepo) ExistsShortCode(ctx context.Context, shortCode string) (bool, error) {
	exists, err := r.data.db.URL.Query().
		Where(url.ShortCodeEQ(shortCode)).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *urlRepo) entToBiz(u *ent.URL) *biz.URL {
	result := &biz.URL{
		ID:          int64(u.ID),
		ShortCode:   u.ShortCode,
		OriginalURL: u.OriginalURL,
		ClickCount:  u.ClickCount,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	if u.ExpiresAt != nil {
		result.ExpiresAt = u.ExpiresAt
	}
	return result
}

func (r *urlRepo) cacheKey(shortCode string) string {
	return fmt.Sprintf("%s%s", urlCachePrefix, shortCode)
}

func (r *urlRepo) cacheURL(ctx context.Context, u *biz.URL) {
	if r.data.rdb == nil {
		return
	}

	data, err := json.Marshal(u)
	if err != nil {
		r.log.WithContext(ctx).Warnf("Failed to marshal URL for cache: %v", err)
		return
	}

	if err := r.data.rdb.Set(ctx, r.cacheKey(u.ShortCode), data, urlCacheTTL).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to cache URL: %v", err)
	}
}

func (r *urlRepo) getCachedURL(ctx context.Context, shortCode string) *biz.URL {
	if r.data.rdb == nil {
		return nil
	}

	data, err := r.data.rdb.Get(ctx, r.cacheKey(shortCode)).Bytes()
	if err != nil {
		return nil
	}

	var u biz.URL
	if err := json.Unmarshal(data, &u); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to unmarshal cached URL: %v", err)
		return nil
	}

	return &u
}

func (r *urlRepo) invalidateCache(ctx context.Context, shortCode string) {
	if r.data.rdb == nil {
		return
	}

	if err := r.data.rdb.Del(ctx, r.cacheKey(shortCode)).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("Failed to invalidate URL cache: %v", err)
	}
}

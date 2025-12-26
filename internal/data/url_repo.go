package data

import (
	"context"

	"go-shortener/ent"
	"go-shortener/ent/url"
	"go-shortener/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/samber/lo"
)

// Compile-time interface check
var _ domain.URLRepository = (*urlRepo)(nil)

// urlRepo implements domain.URLRepository interface.
type urlRepo struct {
	data *Data
	log  *log.Helper
}

// NewURLRepo creates a new URL repository.
func NewURLRepo(data *Data, logger log.Logger) *urlRepo {
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
	return err
}

// FindByShortCode retrieves a URL by its short code.
func (r *urlRepo) FindByShortCode(ctx context.Context, code domain.ShortCode) (*domain.URL, error) {
	u, err := r.client(ctx).URL.Query().
		Where(url.ShortCodeEQ(code.String())).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.entToDomain(u), nil
}

// Delete removes a URL by its short code.
func (r *urlRepo) Delete(ctx context.Context, code domain.ShortCode) error {
	_, err := r.client(ctx).URL.Delete().
		Where(url.ShortCodeEQ(code.String())).
		Exec(ctx)
	return err
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
	return r.client(ctx).URL.Query().
		Where(url.ShortCodeEQ(code.String())).
		Exist(ctx)
}

// IncrementClickCount atomically increments the click count.
func (r *urlRepo) IncrementClickCount(ctx context.Context, code domain.ShortCode) error {
	_, err := r.client(ctx).URL.Update().
		Where(url.ShortCodeEQ(code.String())).
		AddClickCount(1).
		Save(ctx)
	return err
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

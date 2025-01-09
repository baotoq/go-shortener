package data

import (
	"context"
	"fmt"
	"go-shortener/internal/data/ent"
	"go-shortener/internal/data/ent/url"

	"go-shortener/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type urlRepo struct {
	data *Data
	log  *log.Helper
}

func NewUrlRepo(data *Data, logger log.Logger) biz.UrlRepo {
	return &urlRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *urlRepo) Save(ctx context.Context, request *ent.Url) (*ent.Url, error) {
	u, err := r.data.db.Url.Create().
		SetShortenedURL(request.ShortenedURL).
		SetOriginalURL(request.OriginalURL).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed creating: %v", err)
	}

	return u, nil
}

func (r *urlRepo) FindByShortenedUrl(ctx context.Context, shortenedURL string) (*ent.Url, error) {
	first, err := r.data.db.Url.Query().
		Where(url.ShortenedURL(shortenedURL)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return first, nil
}

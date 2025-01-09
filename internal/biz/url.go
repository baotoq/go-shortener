package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"go-shortener/api/url"
	"go-shortener/internal/data/ent"
)

type UrlRepo interface {
	Save(context.Context, *ent.Url) (*ent.Url, error)
	FindByShortenedUrl(context.Context, string) (*ent.Url, error)
}

type UrlUsecase struct {
	repo UrlRepo
	log  *log.Helper
}

func NewUrlUsecase(repo UrlRepo, logger log.Logger) *UrlUsecase {
	return &UrlUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *UrlUsecase) CreateUrl(ctx context.Context, req *url.CreateUrlRequest) (*url.CreateUrlResponse, error) {
	uc.log.WithContext(ctx).Infof("CreateUrl: %v", req.Url)

	saved, err := uc.repo.Save(ctx, &ent.Url{
		ShortenedURL: req.Url,
		OriginalURL:  req.Url,
	})

	if err != nil {
		return nil, err
	}

	return &url.CreateUrlResponse{
		Id:           int32(saved.ID),
		ShortenedUrl: saved.ShortenedURL,
	}, nil
}

func (uc *UrlUsecase) GetUrl(ctx context.Context, req *url.GetUrlRequest) (*url.GetUrlResponse, error) {
	u, err := uc.repo.FindByShortenedUrl(ctx, req.GetShortenedUrl())
	if err != nil {
		return nil, err
	}
	return &url.GetUrlResponse{
		Id:  int32(u.ID),
		Url: u.OriginalURL,
	}, nil
}

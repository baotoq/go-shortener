package service

import (
	"context"
	"time"

	v1 "go-shortener/api/shortener/v1"
	"go-shortener/internal/biz"
	"go-shortener/internal/domain"

	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ShortenerService struct {
	v1.UnimplementedShortenerServer

	uc *biz.URLUsecase
}

func NewShortenerService(uc *biz.URLUsecase) *ShortenerService {
	return &ShortenerService{uc: uc}
}

func (s *ShortenerService) CreateURL(ctx context.Context, req *v1.CreateURLRequest) (*v1.CreateURLReply, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}

	var customCode *string
	if req.CustomCode != nil {
		customCode = req.CustomCode
	}

	u, err := s.uc.CreateURL(ctx, req.OriginalUrl, customCode, expiresAt)
	if err != nil {
		return nil, err
	}

	return &v1.CreateURLReply{
		Url: s.toURLInfo(u),
	}, nil
}

func (s *ShortenerService) GetURL(ctx context.Context, req *v1.GetURLRequest) (*v1.GetURLReply, error) {
	u, err := s.uc.GetURL(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	return &v1.GetURLReply{
		Url: s.toURLInfo(u),
	}, nil
}

func (s *ShortenerService) RedirectURL(ctx context.Context, req *v1.RedirectURLRequest) (*v1.RedirectURLReply, error) {
	originalURL, err := s.uc.RedirectURL(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	return &v1.RedirectURLReply{
		OriginalUrl: originalURL,
	}, nil
}

func (s *ShortenerService) GetURLStats(ctx context.Context, req *v1.GetURLStatsRequest) (*v1.GetURLStatsReply, error) {
	u, err := s.uc.GetURLStats(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	reply := &v1.GetURLStatsReply{
		ShortCode:   u.ShortCode().String(),
		OriginalUrl: u.OriginalURL().String(),
		ClickCount:  u.ClickCount(),
		CreatedAt:   timestamppb.New(u.CreatedAt()),
	}

	if u.ExpiresAt() != nil {
		reply.ExpiresAt = timestamppb.New(*u.ExpiresAt())
	}

	return reply, nil
}

func (s *ShortenerService) DeleteURL(ctx context.Context, req *v1.DeleteURLRequest) (*v1.DeleteURLReply, error) {
	err := s.uc.DeleteURL(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	return &v1.DeleteURLReply{
		Success: true,
	}, nil
}

func (s *ShortenerService) ListURLs(ctx context.Context, req *v1.ListURLsRequest) (*v1.ListURLsReply, error) {
	urls, total, err := s.uc.ListURLs(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	urlInfos := lo.Map(urls, func(u *domain.URL, _ int) *v1.URLInfo {
		return s.toURLInfo(u)
	})

	return &v1.ListURLsReply{
		Urls:  urlInfos,
		Total: int32(total),
	}, nil
}

func (s *ShortenerService) toURLInfo(u *domain.URL) *v1.URLInfo {
	info := &v1.URLInfo{
		Id:          u.ID(),
		ShortCode:   u.ShortCode().String(),
		OriginalUrl: u.OriginalURL().String(),
		ShortUrl:    s.uc.GetShortURL(u.ShortCode().String()),
		ClickCount:  u.ClickCount(),
		CreatedAt:   timestamppb.New(u.CreatedAt()),
	}

	if u.ExpiresAt() != nil {
		info.ExpiresAt = timestamppb.New(*u.ExpiresAt())
	}

	return info
}

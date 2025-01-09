package service

import (
	"context"
	"go-shortener/internal/biz"

	pb "go-shortener/api/url"
)

type UrlService struct {
	pb.UnimplementedUrlServer
	uc *biz.UrlUsecase
}

func NewUrlService(uc *biz.UrlUsecase) *UrlService {
	return &UrlService{
		uc: uc,
	}
}

func (s *UrlService) CreateUrl(ctx context.Context, req *pb.CreateUrlRequest) (*pb.CreateUrlResponse, error) {
	return s.uc.CreateUrl(context.Background(), req)
}

func (s *UrlService) GetUrl(ctx context.Context, req *pb.GetUrlRequest) (*pb.GetUrlResponse, error) {
	return s.uc.GetUrl(context.Background(), req)
}

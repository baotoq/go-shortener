// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package shorten

import (
	"context"

	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShortenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create short URL
func NewShortenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShortenLogic {
	return &ShortenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (resp *types.ShortenResponse, err error) {
	logx.WithContext(l.ctx).Infow("shorten URL", logx.Field("original_url", req.OriginalUrl))

	// Phase 7 stub: Return mock data to prove framework works
	// Phase 8 adds: NanoID generation, database insert, duplicate check
	return &types.ShortenResponse{
		ShortCode:   "stub0001",
		ShortUrl:    l.svcCtx.Config.BaseUrl + "/stub0001",
		OriginalUrl: req.OriginalUrl,
	}, nil
}

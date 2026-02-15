// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"

	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLinkDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get link details
func NewGetLinkDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLinkDetailLogic {
	return &GetLinkDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLinkDetailLogic) GetLinkDetail(req *types.LinkDetailRequest) (resp *types.LinkDetailResponse, err error) {
	logx.WithContext(l.ctx).Infow("get link detail", logx.Field("code", req.Code))

	// Phase 7 stub: Return mock link detail
	// Phase 8 adds: Database lookup by code
	return &types.LinkDetailResponse{
		ShortCode:   req.Code,
		OriginalUrl: "https://example.com/stub",
		CreatedAt:   1700000000,
		TotalClicks: 42,
	}, nil
}

package logic

import (
	"context"

	"go-shortener/services/analytics-rpc/analytics"
	"go-shortener/services/analytics-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetClickCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetClickCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetClickCountLogic {
	return &GetClickCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetClickCountLogic) GetClickCount(in *analytics.GetClickCountRequest) (*analytics.GetClickCountResponse, error) {
	logx.WithContext(l.ctx).Infow("get click count",
		logx.Field("short_code", in.ShortCode),
	)

	// Phase 7 stub: return fake click count
	// Phase 8: Query PostgreSQL for actual click count
	return &analytics.GetClickCountResponse{
		ShortCode:   in.ShortCode,
		TotalClicks: 42,
	}, nil
}

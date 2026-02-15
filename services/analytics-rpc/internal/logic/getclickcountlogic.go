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
	// todo: add your logic here and delete this line

	return &analytics.GetClickCountResponse{}, nil
}

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
	// todo: add your logic here and delete this line

	return
}

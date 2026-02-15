// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"

	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLinksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// List all links with pagination
func NewListLinksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLinksLogic {
	return &ListLinksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLinksLogic) ListLinks(req *types.LinkListRequest) (resp *types.LinkListResponse, err error) {
	// todo: add your logic here and delete this line

	return
}

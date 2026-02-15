// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"

	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Delete link
func NewDeleteLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLinkLogic {
	return &DeleteLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteLinkLogic) DeleteLink(req *types.DeleteLinkRequest) error {
	// todo: add your logic here and delete this line

	return nil
}

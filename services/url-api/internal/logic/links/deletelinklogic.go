// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"
	"errors"

	"go-shortener/pkg/problemdetails"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

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
	logx.WithContext(l.ctx).Infow("delete link", logx.Field("code", req.Code))

	// Look up by short code to get the UUID primary key
	url, findErr := l.svcCtx.UrlModel.FindOneByShortCode(l.ctx, req.Code)
	if findErr != nil {
		if errors.Is(findErr, model.ErrNotFound) {
			return problemdetails.New(404, problemdetails.TypeNotFound, "Not Found",
				"short code '"+req.Code+"' not found")
		}
		logx.WithContext(l.ctx).Errorw("failed to find URL for deletion", logx.Field("error", findErr.Error()))
		return problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error",
			"failed to look up link for deletion")
	}

	// Hard delete (per user decision: row removed, not soft delete)
	if delErr := l.svcCtx.UrlModel.Delete(l.ctx, url.Id); delErr != nil {
		logx.WithContext(l.ctx).Errorw("failed to delete URL", logx.Field("error", delErr.Error()))
		return problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error",
			"failed to delete link")
	}

	return nil
}

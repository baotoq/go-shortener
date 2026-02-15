// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"
	"errors"

	"go-shortener/pkg/problemdetails"
	"go-shortener/services/analytics-rpc/analyticsclient"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

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

	url, findErr := l.svcCtx.UrlModel.FindOneByShortCode(l.ctx, req.Code)
	if findErr != nil {
		if errors.Is(findErr, model.ErrNotFound) {
			return nil, problemdetails.New(404, problemdetails.TypeNotFound, "Not Found",
				"short code '"+req.Code+"' not found")
		}
		logx.WithContext(l.ctx).Errorw("failed to find URL", logx.Field("error", findErr.Error()))
		return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error",
			"failed to look up link detail")
	}

	// Get click count from Analytics RPC service
	var totalClicks int64
	clickResp, rpcErr := l.svcCtx.AnalyticsRpc.GetClickCount(l.ctx, &analyticsclient.GetClickCountRequest{
		ShortCode: req.Code,
	})
	if rpcErr != nil {
		// Log but don't fail - degrade gracefully, return 0 clicks
		logx.WithContext(l.ctx).Errorw("failed to get click count from analytics rpc, degrading gracefully",
			logx.Field("code", req.Code),
			logx.Field("error", rpcErr.Error()),
		)
	} else {
		totalClicks = clickResp.TotalClicks
	}

	return &types.LinkDetailResponse{
		ShortCode:   url.ShortCode,
		OriginalUrl: url.OriginalUrl,
		CreatedAt:   url.CreatedAt.Unix(),
		TotalClicks: totalClicks,
	}, nil
}

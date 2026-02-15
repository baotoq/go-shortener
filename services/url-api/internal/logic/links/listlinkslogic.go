// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"context"
	"math"

	"go-shortener/pkg/problemdetails"
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
	logx.WithContext(l.ctx).Infow("list links",
		logx.Field("page", req.Page),
		logx.Field("per_page", req.PerPage),
	)

	urls, totalCount, queryErr := l.svcCtx.UrlModel.ListWithPagination(
		l.ctx, req.Page, req.PerPage, req.Search, req.Sort, req.Order,
	)
	if queryErr != nil {
		logx.WithContext(l.ctx).Errorw("failed to list URLs", logx.Field("error", queryErr.Error()))
		return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error", "failed to list links")
	}

	// Map model results to response types
	linkItems := make([]types.LinkItem, 0, len(urls))
	for _, u := range urls {
		linkItems = append(linkItems, types.LinkItem{
			ShortCode:   u.ShortCode,
			OriginalUrl: u.OriginalUrl,
			CreatedAt:   u.CreatedAt.Unix(),
		})
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(req.PerPage)))

	return &types.LinkListResponse{
		Links:      linkItems,
		Page:       req.Page,
		PerPage:    req.PerPage,
		TotalPages: totalPages,
		TotalCount: totalCount,
	}, nil
}

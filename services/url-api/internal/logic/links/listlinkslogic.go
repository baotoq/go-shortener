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
	logx.WithContext(l.ctx).Infow("list links",
		logx.Field("page", req.Page),
		logx.Field("per_page", req.PerPage),
	)

	// Phase 7 stub: Return mock paginated data
	// Phase 8 adds: Database pagination, sorting, searching
	return &types.LinkListResponse{
		Links: []types.LinkItem{
			{ShortCode: "stub0001", OriginalUrl: "https://example.com", CreatedAt: 1700000000},
			{ShortCode: "stub0002", OriginalUrl: "https://go-zero.dev", CreatedAt: 1700000001},
		},
		Page:       req.Page,
		PerPage:    req.PerPage,
		TotalPages: 1,
		TotalCount: 2,
	}, nil
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package redirect

import (
	"context"
	"fmt"

	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RedirectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Redirect to original URL
func NewRedirectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RedirectLogic {
	return &RedirectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RedirectLogic) Redirect(req *types.RedirectRequest) error {
	logx.WithContext(l.ctx).Infow("redirect", logx.Field("code", req.Code))

	// Phase 7 stub: Redirect requires database lookup (implemented in Phase 8)
	// Return not-found error to prove routing and error handling work
	// Phase 8 adds: DB lookup, click event publishing, actual HTTP redirect
	return fmt.Errorf("short code '%s' not found (stub - DB not connected)", req.Code)
}

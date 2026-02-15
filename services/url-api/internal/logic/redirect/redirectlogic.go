// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package redirect

import (
  "context"
  "errors"

  "go-shortener/pkg/problemdetails"
  "go-shortener/services/url-api/internal/svc"
  "go-shortener/services/url-api/internal/types"
  "go-shortener/services/url-api/model"

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

// Redirect looks up the original URL by short code and returns it for HTTP redirect.
// Also increments click count asynchronously (fire-and-forget).
func (l *RedirectLogic) Redirect(req *types.RedirectRequest) (string, error) {
  logx.WithContext(l.ctx).Infow("redirect", logx.Field("code", req.Code))

  url, err := l.svcCtx.UrlModel.FindOneByShortCode(l.ctx, req.Code)
  if err != nil {
    if errors.Is(err, model.ErrNotFound) {
      return "", problemdetails.New(404, problemdetails.TypeNotFound, "Not Found",
        "short code '"+req.Code+"' not found")
    }
    logx.WithContext(l.ctx).Errorw("failed to find URL", logx.Field("error", err.Error()))
    return "", problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error",
      "failed to look up short code")
  }

  // Increment click count asynchronously (fire-and-forget)
  go func() {
    if incErr := l.svcCtx.UrlModel.IncrementClickCount(context.Background(), req.Code); incErr != nil {
      logx.Errorw("failed to increment click count",
        logx.Field("code", req.Code),
        logx.Field("error", incErr.Error()),
      )
    }
  }()

  return url.OriginalUrl, nil
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package redirect

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"go-shortener/common/events"
	"go-shortener/pkg/problemdetails"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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
// Publishes a ClickEvent to Kafka asynchronously (fire-and-forget).
func (l *RedirectLogic) Redirect(req *types.RedirectRequest, r *http.Request) (string, error) {
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

	// Publish click event to Kafka asynchronously (fire-and-forget)
	threading.GoSafe(func() {
		clickEvent := events.ClickEvent{
			ShortCode: req.Code,
			Timestamp: time.Now().Unix(),
			IP:        extractClientIP(r),
			UserAgent: r.UserAgent(),
			Referer:   r.Referer(),
		}

		payload, marshalErr := json.Marshal(clickEvent)
		if marshalErr != nil {
			logx.Errorf("failed to marshal click event: %v", marshalErr)
			return
		}

		if pushErr := l.svcCtx.KqPusher.Push(l.ctx, string(payload)); pushErr != nil {
			logx.Errorw("failed to push click event to Kafka",
				logx.Field("code", req.Code),
				logx.Field("error", pushErr.Error()),
			)
		}
	})

	return url.OriginalUrl, nil
}

// extractClientIP returns the client IP from X-Forwarded-For, X-Real-IP, or RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the comma-separated chain
		if comma := strings.IndexByte(xff, ','); comma != -1 {
			return strings.TrimSpace(xff[:comma])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

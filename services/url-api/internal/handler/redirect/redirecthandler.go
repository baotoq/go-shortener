// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package redirect

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-shortener/services/url-api/internal/logic/redirect"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
)

// Redirect to original URL
func RedirectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RedirectRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := redirect.NewRedirectLogic(r.Context(), svcCtx)
		originalUrl, err := l.Redirect(&req, r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		http.Redirect(w, r, originalUrl, http.StatusFound)
	}
}

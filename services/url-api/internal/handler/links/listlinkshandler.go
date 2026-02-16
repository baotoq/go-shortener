// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"net/http"

	"go-shortener/services/url-api/internal/logic/links"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// List all links with pagination
func ListLinksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LinkListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := links.NewListLinksLogic(r.Context(), svcCtx)
		resp, err := l.ListLinks(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

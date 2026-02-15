// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package links

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-shortener/services/url-api/internal/logic/links"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
)

// Delete link
func DeleteLinkHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteLinkRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := links.NewDeleteLinkLogic(r.Context(), svcCtx)
		err := l.DeleteLink(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}

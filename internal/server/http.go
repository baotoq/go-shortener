package server

import (
	"context"
	nethttp "net/http"

	v1 "go-shortener/api/shortener/v1"
	"go-shortener/internal/conf"
	"go-shortener/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, shortener *service.ShortenerService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterShortenerHTTPServer(srv, shortener)

	// Add redirect handler with 302 redirect
	srv.HandleFunc("/r/{short_code}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		shortCode := r.URL.Path[len("/r/"):]
		resp, err := shortener.RedirectURL(context.Background(), &v1.RedirectURLRequest{
			ShortCode: shortCode,
		})
		if err != nil {
			nethttp.Error(w, err.Error(), 404)
			return
		}
		nethttp.Redirect(w, r, resp.OriginalUrl, 302)
	})

	return srv
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"go-shortener/pkg/problemdetails"
	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/internal/handler"
	"go-shortener/services/url-api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/url.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// Health check endpoint
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/healthz",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}),
	})

	// Custom RFC 7807 error handler
	// Note: go-zero's doHandleError checks if the returned body implements error,
	// and if so writes it as plaintext. We return a problemDetailsBody (non-error)
	// struct to ensure JSON serialization.
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		var pd *problemdetails.ProblemDetail
		if errors.As(err, &pd) {
			return pd.Status, pd.Body()
		}

		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeValidationError,
			"Bad Request",
			err.Error(),
		)
		return problem.Status, problem.Body()
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

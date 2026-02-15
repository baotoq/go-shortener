// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
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

	// Custom RFC 7807 error handler
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		// For Phase 7, return validation errors in Problem Details format
		// Phase 8 will add more sophisticated error parsing and domain error handling
		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeValidationError,
			"Bad Request",
			err.Error(),
		)
		return problem.Status, problem
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

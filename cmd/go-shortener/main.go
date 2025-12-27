package main

import (
	"context"
	"flag"
	"os"

	"go-shortener/internal/biz"
	"go-shortener/internal/conf"
	"go-shortener/internal/domain"
	"go-shortener/internal/infra/eventbus"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(
	logger log.Logger,
	gs *grpc.Server,
	hs *http.Server,
	eventBus *eventbus.EventBus,
	router *eventbus.Router,
	forwarder *eventbus.Forwarder,
	repo domain.URLRepository,
) *kratos.App {
	// Register event handlers
	biz.RegisterEventHandlers(router, repo, logger)

	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
		kratos.BeforeStart(func(ctx context.Context) error {
			// Start the outbox forwarder
			forwarder.Start(ctx)
			// Start the event router in a goroutine
			go func() {
				if err := router.Run(ctx); err != nil {
					log.NewHelper(logger).Errorf("event router error: %v", err)
				}
			}()
			return nil
		}),
		kratos.BeforeStop(func(ctx context.Context) error {
			// Stop the forwarder and router
			forwarder.Stop()
			if err := router.Close(); err != nil {
				log.NewHelper(logger).Errorf("failed to close router: %v", err)
			}
			if err := eventBus.Close(); err != nil {
				log.NewHelper(logger).Errorf("failed to close event bus: %v", err)
			}
			return nil
		}),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

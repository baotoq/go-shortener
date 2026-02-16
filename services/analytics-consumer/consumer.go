package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"go-shortener/services/analytics-consumer/internal/config"
	"go-shortener/services/analytics-consumer/internal/mqs"
	"go-shortener/services/analytics-consumer/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/consumer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Setup service infrastructure (logging, metrics, devserver, etc.)
	c.MustSetUp()

	svcCtx := svc.NewServiceContext(c)

	// Start health check server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		addr := fmt.Sprintf(":%d", c.HealthCheckPort)
		logx.Infof("Health check server listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			logx.Errorf("Health server error: %v", err)
		}
	}()

	group := service.NewServiceGroup()
	defer group.Stop()

	group.Add(kq.MustNewQueue(c.KqConsumerConf, mqs.NewClickEventConsumer(context.Background(), svcCtx)))

	fmt.Printf("Starting analytics consumer, listening on topic %s...\n", c.KqConsumerConf.Topic)
	group.Start()
}

package main

import (
	"context"
	"flag"
	"fmt"

	"go-shortener/services/analytics-consumer/internal/config"
	"go-shortener/services/analytics-consumer/internal/mqs"
	"go-shortener/services/analytics-consumer/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/consumer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	svcCtx := svc.NewServiceContext(c)

	group := service.NewServiceGroup()
	defer group.Stop()

	group.Add(kq.MustNewQueue(c.KqConsumerConf, mqs.NewClickEventConsumer(context.Background(), svcCtx)))

	fmt.Printf("Starting analytics consumer, listening on topic %s...\n", c.KqConsumerConf.Topic)
	group.Start()
}

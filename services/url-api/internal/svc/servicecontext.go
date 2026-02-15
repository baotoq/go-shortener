// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-shortener/services/analytics-rpc/analyticsclient"
	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/model"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	_ "github.com/lib/pq"
)

type ServiceContext struct {
	Config       config.Config
	UrlModel     model.UrlsModel
	KqPusher     *kq.Pusher
	AnalyticsRpc analyticsclient.Analytics
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewSqlConn("postgres", c.DataSource)

	return &ServiceContext{
		Config:       c,
		UrlModel:     model.NewUrlsModel(conn),
		KqPusher:     kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
		AnalyticsRpc: analyticsclient.NewAnalytics(zrpc.MustNewClient(c.AnalyticsRpc)),
	}
}

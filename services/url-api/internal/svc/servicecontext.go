// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"time"

	"go-shortener/services/analytics-rpc/analyticsclient"
	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/model"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
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

	// Configure connection pool
	db, err := conn.RawDB()
	logx.Must(err)
	db.SetMaxOpenConns(c.Pool.MaxOpenConns)
	db.SetMaxIdleConns(c.Pool.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(c.Pool.ConnMaxLifetime) * time.Second)
	logx.Infof("Connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%ds",
		c.Pool.MaxOpenConns, c.Pool.MaxIdleConns, c.Pool.ConnMaxLifetime)

	return &ServiceContext{
		Config:       c,
		UrlModel:     model.NewUrlsModel(conn),
		KqPusher:     kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
		AnalyticsRpc: analyticsclient.NewAnalytics(zrpc.MustNewClient(c.AnalyticsRpc)),
	}
}

package svc

import (
	"go-shortener/services/analytics-consumer/internal/config"
	"go-shortener/services/analytics-rpc/model"

	"github.com/oschwald/geoip2-golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	_ "github.com/lib/pq"
)

type ServiceContext struct {
	Config     config.Config
	ClickModel model.ClicksModel
	GeoDB      *geoip2.Reader
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewSqlConn("postgres", c.DataSource)

	var geoDB *geoip2.Reader
	if c.GeoIPPath != "" {
		db, err := geoip2.Open(c.GeoIPPath)
		if err != nil {
			logx.Infof("GeoIP database not available at %s, falling back to Unknown", c.GeoIPPath)
		} else {
			geoDB = db
		}
	}

	return &ServiceContext{
		Config:     c,
		ClickModel: model.NewClicksModel(conn),
		GeoDB:      geoDB,
	}
}

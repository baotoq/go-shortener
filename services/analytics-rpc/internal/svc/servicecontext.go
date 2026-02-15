package svc

import (
	"go-shortener/services/analytics-rpc/internal/config"
	"go-shortener/services/analytics-rpc/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	_ "github.com/lib/pq"
)

type ServiceContext struct {
	Config     config.Config
	ClickModel model.ClicksModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewSqlConn("postgres", c.DataSource)
	return &ServiceContext{
		Config:     c,
		ClickModel: model.NewClicksModel(conn),
	}
}

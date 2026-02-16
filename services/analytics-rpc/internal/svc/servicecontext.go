package svc

import (
	"time"

	"go-shortener/services/analytics-rpc/internal/config"
	"go-shortener/services/analytics-rpc/model"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config     config.Config
	ClickModel model.ClicksModel
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
		Config:     c,
		ClickModel: model.NewClicksModel(conn),
	}
}

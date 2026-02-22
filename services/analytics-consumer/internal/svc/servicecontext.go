package svc

import (
	"net"
	"time"

	"go-shortener/services/analytics-consumer/internal/config"
	"go-shortener/services/analytics-rpc/model"

	_ "github.com/lib/pq"
	"github.com/oschwald/geoip2-golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// GeoIPReader abstracts country lookup for testability.
// *geoip2.Reader naturally satisfies this interface.
type GeoIPReader interface {
	Country(ipAddress net.IP) (*geoip2.Country, error)
}

type ServiceContext struct {
	Config     config.Config
	ClickModel model.ClicksModel
	GeoDB      GeoIPReader
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

	var geoDB GeoIPReader
	if c.GeoIPPath != "" {
		gdb, geoErr := geoip2.Open(c.GeoIPPath)
		if geoErr != nil {
			logx.Infof("GeoIP database not available at %s, falling back to Unknown", c.GeoIPPath)
		} else {
			geoDB = gdb
		}
	}

	return &ServiceContext{
		Config:     c,
		ClickModel: model.NewClicksModel(conn),
		GeoDB:      geoDB,
	}
}

package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type Config struct {
	service.ServiceConf
	DataSource      string
	Pool            PoolConfig
	KqConsumerConf  kq.KqConf
	GeoIPPath       string `json:",optional"`
	HealthCheckPort int    `json:",default=8082"`
}

type PoolConfig struct {
	MaxOpenConns    int `json:",default=10"`
	MaxIdleConns    int `json:",default=5"`
	ConnMaxLifetime int `json:",default=3600"` // seconds
}

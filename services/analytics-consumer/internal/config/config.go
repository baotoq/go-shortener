package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type Config struct {
	service.ServiceConf
	DataSource    string
	KqConsumerConf kq.KqConf
	GeoIPPath     string `json:",optional"`
}

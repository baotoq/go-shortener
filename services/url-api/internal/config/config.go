// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	BaseUrl      string
	DataSource   string
	Pool         PoolConfig
	KqPusherConf KqPusherConf
	AnalyticsRpc zrpc.RpcClientConf
}

type PoolConfig struct {
	MaxOpenConns    int `json:",default=10"`
	MaxIdleConns    int `json:",default=5"`
	ConnMaxLifetime int `json:",default=3600"` // seconds
}

type KqPusherConf struct {
	Brokers []string
	Topic   string
}

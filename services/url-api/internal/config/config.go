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
	KqPusherConf KqPusherConf
	AnalyticsRpc zrpc.RpcClientConf
}

type KqPusherConf struct {
	Brokers []string
	Topic   string
}

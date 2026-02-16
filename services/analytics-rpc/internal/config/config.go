package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	DataSource string
	Pool       PoolConfig
}

type PoolConfig struct {
	MaxOpenConns    int `json:",default=10"`
	MaxIdleConns    int `json:",default=5"`
	ConnMaxLifetime int `json:",default=3600"` // seconds
}

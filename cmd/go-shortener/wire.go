//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"go-shortener/ent"
	"go-shortener/internal/biz"
	"go-shortener/internal/conf"
	"go-shortener/internal/data"
	"go-shortener/internal/infra/eventbus"
	"go-shortener/internal/server"
	"go-shortener/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// provideEntClient extracts the ent.Client from Data for eventbus providers.
func provideEntClient(d *data.Data) *ent.Client {
	return d.EntClient()
}

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		eventbus.ProviderSet,
		provideEntClient,
		newApp,
	))
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
  "go-shortener/services/url-api/internal/config"
  "go-shortener/services/url-api/model"

  "github.com/zeromicro/go-zero/core/stores/sqlx"
  _ "github.com/lib/pq"
)

type ServiceContext struct {
  Config   config.Config
  UrlModel model.UrlsModel
}

func NewServiceContext(c config.Config) *ServiceContext {
  conn := sqlx.NewSqlConn("postgres", c.DataSource)
  return &ServiceContext{
    Config:   c,
    UrlModel: model.NewUrlsModel(conn),
  }
}

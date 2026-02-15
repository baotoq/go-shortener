package model

import (
  "context"
  "fmt"

  "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ClicksModel = (*customClicksModel)(nil)

type (
  // ClicksModel is an interface to be customized, add more methods here,
  // and implement the added methods in customClicksModel.
  ClicksModel interface {
    clicksModel
    withSession(session sqlx.Session) ClicksModel
    CountByShortCode(ctx context.Context, shortCode string) (int64, error)
  }

  customClicksModel struct {
    *defaultClicksModel
  }
)

// NewClicksModel returns a model for the database table.
func NewClicksModel(conn sqlx.SqlConn) ClicksModel {
  return &customClicksModel{
    defaultClicksModel: newClicksModel(conn),
  }
}

func (m *customClicksModel) withSession(session sqlx.Session) ClicksModel {
  return NewClicksModel(sqlx.NewSqlConnFromSession(session))
}

// CountByShortCode returns the total number of clicks for a given short code.
func (m *customClicksModel) CountByShortCode(ctx context.Context, shortCode string) (int64, error) {
  query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE short_code = $1", m.table)
  var count int64
  err := m.conn.QueryRowCtx(ctx, &count, query, shortCode)
  if err != nil {
    return 0, err
  }
  return count, nil
}

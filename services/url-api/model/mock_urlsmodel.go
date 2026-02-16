package model

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// MockUrlsModel is a test mock for UrlsModel interface.
type MockUrlsModel struct {
	FindOneFunc              func(ctx context.Context, id string) (*Urls, error)
	FindOneByShortCodeFunc   func(ctx context.Context, shortCode string) (*Urls, error)
	InsertFunc               func(ctx context.Context, data *Urls) (sql.Result, error)
	UpdateFunc               func(ctx context.Context, data *Urls) error
	DeleteFunc               func(ctx context.Context, id string) error
	ListWithPaginationFunc   func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error)
	IncrementClickCountFunc  func(ctx context.Context, shortCode string) error
	WithSessionFunc          func(session sqlx.Session) UrlsModel
}

// Ensure MockUrlsModel implements UrlsModel interface
var _ UrlsModel = (*MockUrlsModel)(nil)

func (m *MockUrlsModel) FindOne(ctx context.Context, id string) (*Urls, error) {
	if m.FindOneFunc != nil {
		return m.FindOneFunc(ctx, id)
	}
	panic("MockUrlsModel.FindOneFunc not set")
}

func (m *MockUrlsModel) FindOneByShortCode(ctx context.Context, shortCode string) (*Urls, error) {
	if m.FindOneByShortCodeFunc != nil {
		return m.FindOneByShortCodeFunc(ctx, shortCode)
	}
	panic("MockUrlsModel.FindOneByShortCodeFunc not set")
}

func (m *MockUrlsModel) Insert(ctx context.Context, data *Urls) (sql.Result, error) {
	if m.InsertFunc != nil {
		return m.InsertFunc(ctx, data)
	}
	panic("MockUrlsModel.InsertFunc not set")
}

func (m *MockUrlsModel) Update(ctx context.Context, data *Urls) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, data)
	}
	panic("MockUrlsModel.UpdateFunc not set")
}

func (m *MockUrlsModel) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	panic("MockUrlsModel.DeleteFunc not set")
}

func (m *MockUrlsModel) ListWithPagination(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error) {
	if m.ListWithPaginationFunc != nil {
		return m.ListWithPaginationFunc(ctx, page, pageSize, search, sort, order)
	}
	panic("MockUrlsModel.ListWithPaginationFunc not set")
}

func (m *MockUrlsModel) IncrementClickCount(ctx context.Context, shortCode string) error {
	if m.IncrementClickCountFunc != nil {
		return m.IncrementClickCountFunc(ctx, shortCode)
	}
	panic("MockUrlsModel.IncrementClickCountFunc not set")
}

func (m *MockUrlsModel) withSession(session sqlx.Session) UrlsModel {
	if m.WithSessionFunc != nil {
		return m.WithSessionFunc(session)
	}
	panic("MockUrlsModel.WithSessionFunc not set")
}

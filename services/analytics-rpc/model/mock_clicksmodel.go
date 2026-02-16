package model

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// MockClicksModel is a test mock for ClicksModel interface.
type MockClicksModel struct {
	InsertFunc            func(ctx context.Context, data *Clicks) (sql.Result, error)
	FindOneFunc           func(ctx context.Context, id string) (*Clicks, error)
	UpdateFunc            func(ctx context.Context, data *Clicks) error
	DeleteFunc            func(ctx context.Context, id string) error
	CountByShortCodeFunc  func(ctx context.Context, shortCode string) (int64, error)
	WithSessionFunc       func(session sqlx.Session) ClicksModel
}

// Ensure MockClicksModel implements ClicksModel interface
var _ ClicksModel = (*MockClicksModel)(nil)

func (m *MockClicksModel) Insert(ctx context.Context, data *Clicks) (sql.Result, error) {
	if m.InsertFunc != nil {
		return m.InsertFunc(ctx, data)
	}
	panic("MockClicksModel.InsertFunc not set")
}

func (m *MockClicksModel) FindOne(ctx context.Context, id string) (*Clicks, error) {
	if m.FindOneFunc != nil {
		return m.FindOneFunc(ctx, id)
	}
	panic("MockClicksModel.FindOneFunc not set")
}

func (m *MockClicksModel) Update(ctx context.Context, data *Clicks) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, data)
	}
	panic("MockClicksModel.UpdateFunc not set")
}

func (m *MockClicksModel) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	panic("MockClicksModel.DeleteFunc not set")
}

func (m *MockClicksModel) CountByShortCode(ctx context.Context, shortCode string) (int64, error) {
	if m.CountByShortCodeFunc != nil {
		return m.CountByShortCodeFunc(ctx, shortCode)
	}
	panic("MockClicksModel.CountByShortCodeFunc not set")
}

func (m *MockClicksModel) withSession(session sqlx.Session) ClicksModel {
	if m.WithSessionFunc != nil {
		return m.WithSessionFunc(session)
	}
	panic("MockClicksModel.WithSessionFunc not set")
}

package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UrlsModel = (*customUrlsModel)(nil)

type (
	// UrlsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUrlsModel.
	UrlsModel interface {
		urlsModel
		withSession(session sqlx.Session) UrlsModel
		ListWithPagination(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error)
		IncrementClickCount(ctx context.Context, shortCode string) error
	}

	customUrlsModel struct {
		*defaultUrlsModel
	}
)

// NewUrlsModel returns a model for the database table.
func NewUrlsModel(conn sqlx.SqlConn) UrlsModel {
	return &customUrlsModel{
		defaultUrlsModel: newUrlsModel(conn),
	}
}

func (m *customUrlsModel) withSession(session sqlx.Session) UrlsModel {
	return NewUrlsModel(sqlx.NewSqlConnFromSession(session))
}

// ListWithPagination returns a paginated list of URLs with optional search filtering.
// Uses OFFSET/LIMIT pagination matching the current API contract (page, per_page, sort, order, search).
func (m *customUrlsModel) ListWithPagination(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error) {
	// Build WHERE clause for search
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	argIdx := 1

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("original_url ILIKE $%d", argIdx))
		args = append(args, search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", m.table, whereClause)
	var totalCount int64
	err := m.conn.QueryRowCtx(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Validate sort column (whitelist to prevent SQL injection)
	var sortColumn string
	switch sort {
	case "original_url":
		sortColumn = "original_url"
	default:
		sortColumn = "created_at"
	}

	// Validate order direction
	orderDir := "DESC"
	if strings.ToUpper(order) == "ASC" {
		orderDir = "ASC"
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build data query
	dataQuery := fmt.Sprintf(
		"SELECT %s FROM %s%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		urlsRows, m.table, whereClause, sortColumn, orderDir, argIdx, argIdx+1,
	)
	dataArgs := append(args, pageSize, offset)

	var resp []*Urls
	err = m.conn.QueryRowsCtx(ctx, &resp, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}

	return resp, totalCount, nil
}

// IncrementClickCount atomically increments the click_count for a given short code.
// Uses SET click_count = click_count + 1 to avoid read-modify-write race conditions.
func (m *customUrlsModel) IncrementClickCount(ctx context.Context, shortCode string) error {
	query := fmt.Sprintf("UPDATE %s SET click_count = click_count + 1 WHERE short_code = $1", m.table)
	_, err := m.conn.ExecCtx(ctx, query, shortCode)
	return err
}

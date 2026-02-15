package sqlite

import (
	"context"
	"database/sql"

	"go-shortener/internal/analytics/repository/sqlite/sqlc"
	"go-shortener/internal/analytics/usecase"
)

// ClickRepository implements the usecase.ClickRepository interface using sqlc
type ClickRepository struct {
	queries *sqlc.Queries
}

// NewClickRepository creates a new SQLite-backed click repository
func NewClickRepository(db *sql.DB) *ClickRepository {
	return &ClickRepository{
		queries: sqlc.New(db),
	}
}

// Ensure ClickRepository implements usecase.ClickRepository at compile time
var _ usecase.ClickRepository = (*ClickRepository)(nil)

// InsertClick stores a click event in the database
func (r *ClickRepository) InsertClick(ctx context.Context, shortCode string, clickedAt int64) error {
	return r.queries.InsertClick(ctx, sqlc.InsertClickParams{
		ShortCode: shortCode,
		ClickedAt: clickedAt,
	})
}

// CountByShortCode returns the total number of clicks for a short code
func (r *ClickRepository) CountByShortCode(ctx context.Context, shortCode string) (int64, error) {
	return r.queries.CountClicksByShortCode(ctx, shortCode)
}

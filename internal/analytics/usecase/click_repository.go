package usecase

import "context"

type ClickRepository interface {
	InsertClick(ctx context.Context, shortCode string, clickedAt int64) error
	CountByShortCode(ctx context.Context, shortCode string) (int64, error)
}

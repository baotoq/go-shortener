package usecase

import (
	"context"
	"go-shortener/internal/shared/events"
)

type AnalyticsService struct {
	repo ClickRepository
}

func NewAnalyticsService(repo ClickRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// RecordClick stores a click event
func (s *AnalyticsService) RecordClick(ctx context.Context, event events.ClickEvent) error {
	return s.repo.InsertClick(ctx, event.ShortCode, event.Timestamp.Unix())
}

// GetClickCount returns total clicks for a short code
func (s *AnalyticsService) GetClickCount(ctx context.Context, shortCode string) (int64, error) {
	return s.repo.CountByShortCode(ctx, shortCode)
}

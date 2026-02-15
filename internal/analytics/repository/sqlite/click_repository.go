package sqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
	"strconv"

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

// InsertClick stores a click event with enrichment data in the database
func (r *ClickRepository) InsertClick(ctx context.Context, shortCode string, clickedAt int64, countryCode string, deviceType string, trafficSource string) error {
	return r.queries.InsertEnrichedClick(ctx, sqlc.InsertEnrichedClickParams{
		ShortCode:     shortCode,
		ClickedAt:     clickedAt,
		CountryCode:   countryCode,
		DeviceType:    deviceType,
		TrafficSource: trafficSource,
	})
}

// CountByShortCode returns the total number of clicks for a short code
func (r *ClickRepository) CountByShortCode(ctx context.Context, shortCode string) (int64, error) {
	return r.queries.CountClicksByShortCode(ctx, shortCode)
}

// CountInRange returns total clicks within a time range
func (r *ClickRepository) CountInRange(ctx context.Context, shortCode string, from int64, to int64) (int64, error) {
	return r.queries.CountClicksInRange(ctx, sqlc.CountClicksInRangeParams{
		ShortCode:   shortCode,
		ClickedAt:   from,
		ClickedAt_2: to,
	})
}

// CountByCountryInRange returns click counts grouped by country within a time range
func (r *ClickRepository) CountByCountryInRange(ctx context.Context, shortCode string, from int64, to int64) ([]usecase.GroupCount, error) {
	rows, err := r.queries.CountByCountryInRange(ctx, sqlc.CountByCountryInRangeParams{
		ShortCode:   shortCode,
		ClickedAt:   from,
		ClickedAt_2: to,
	})
	if err != nil {
		return nil, err
	}

	result := make([]usecase.GroupCount, len(rows))
	for i, row := range rows {
		result[i] = usecase.GroupCount{
			Value: row.CountryCode,
			Count: row.Count,
		}
	}
	return result, nil
}

// CountByDeviceInRange returns click counts grouped by device type within a time range
func (r *ClickRepository) CountByDeviceInRange(ctx context.Context, shortCode string, from int64, to int64) ([]usecase.GroupCount, error) {
	rows, err := r.queries.CountByDeviceInRange(ctx, sqlc.CountByDeviceInRangeParams{
		ShortCode:   shortCode,
		ClickedAt:   from,
		ClickedAt_2: to,
	})
	if err != nil {
		return nil, err
	}

	result := make([]usecase.GroupCount, len(rows))
	for i, row := range rows {
		result[i] = usecase.GroupCount{
			Value: row.DeviceType,
			Count: row.Count,
		}
	}
	return result, nil
}

// CountBySourceInRange returns click counts grouped by traffic source within a time range
func (r *ClickRepository) CountBySourceInRange(ctx context.Context, shortCode string, from int64, to int64) ([]usecase.GroupCount, error) {
	rows, err := r.queries.CountBySourceInRange(ctx, sqlc.CountBySourceInRangeParams{
		ShortCode:   shortCode,
		ClickedAt:   from,
		ClickedAt_2: to,
	})
	if err != nil {
		return nil, err
	}

	result := make([]usecase.GroupCount, len(rows))
	for i, row := range rows {
		result[i] = usecase.GroupCount{
			Value: row.TrafficSource,
			Count: row.Count,
		}
	}
	return result, nil
}

// GetClickDetails returns paginated individual click records
func (r *ClickRepository) GetClickDetails(ctx context.Context, shortCode string, cursorTimestamp int64, limit int) (*usecase.PaginatedClicks, error) {
	// Fetch limit+1 to detect if there are more results
	rows, err := r.queries.GetClickDetails(ctx, sqlc.GetClickDetailsParams{
		ShortCode: shortCode,
		ClickedAt: cursorTimestamp,
		Limit:     int64(limit + 1),
	})
	if err != nil {
		return nil, err
	}

	// Check if there are more results
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	// Convert to usecase.ClickDetail
	clicks := make([]usecase.ClickDetail, len(rows))
	for i, row := range rows {
		clicks[i] = usecase.ClickDetail{
			ID:            row.ID,
			ShortCode:     row.ShortCode,
			ClickedAt:     row.ClickedAt,
			CountryCode:   row.CountryCode,
			DeviceType:    row.DeviceType,
			TrafficSource: row.TrafficSource,
		}
	}

	// Generate next cursor if there are more results
	var nextCursor string
	if hasMore && len(clicks) > 0 {
		lastTimestamp := clicks[len(clicks)-1].ClickedAt
		nextCursor = base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(lastTimestamp, 10)))
	}

	return &usecase.PaginatedClicks{
		Clicks:     clicks,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

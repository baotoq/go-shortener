package usecase

import "context"

// ClickDetail represents an individual click record with enrichment data.
type ClickDetail struct {
	ID            int64
	ShortCode     string
	ClickedAt     int64
	CountryCode   string
	DeviceType    string
	TrafficSource string
}

// GroupCount represents a count for a single group value (country, device, source).
type GroupCount struct {
	Value string
	Count int64
}

// PaginatedClicks holds a page of click details with cursor info.
type PaginatedClicks struct {
	Clicks     []ClickDetail
	NextCursor string
	HasMore    bool
}

type ClickRepository interface {
	// InsertClick stores a click with enrichment data.
	InsertClick(ctx context.Context, shortCode string, clickedAt int64, countryCode string, deviceType string, trafficSource string) error
	// CountByShortCode returns total clicks for a short code (backward compat).
	CountByShortCode(ctx context.Context, shortCode string) (int64, error)
	// CountInRange returns total clicks within a time range.
	CountInRange(ctx context.Context, shortCode string, from int64, to int64) (int64, error)
	// CountByCountryInRange returns click counts grouped by country within a time range.
	CountByCountryInRange(ctx context.Context, shortCode string, from int64, to int64) ([]GroupCount, error)
	// CountByDeviceInRange returns click counts grouped by device type within a time range.
	CountByDeviceInRange(ctx context.Context, shortCode string, from int64, to int64) ([]GroupCount, error)
	// CountBySourceInRange returns click counts grouped by traffic source within a time range.
	CountBySourceInRange(ctx context.Context, shortCode string, from int64, to int64) ([]GroupCount, error)
	// GetClickDetails returns paginated individual click records.
	GetClickDetails(ctx context.Context, shortCode string, cursorTimestamp int64, limit int) (*PaginatedClicks, error)
	// DeleteByShortCode removes all click records for a short code.
	DeleteByShortCode(ctx context.Context, shortCode string) error
}

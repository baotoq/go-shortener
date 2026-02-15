package usecase

import (
	"context"
	"go-shortener/internal/shared/events"
)

// GeoIPResolver resolves IP addresses to country codes
type GeoIPResolver interface {
	ResolveCountry(ipStr string) string
}

// DeviceDetector detects device type from User-Agent strings
type DeviceDetector interface {
	DetectDevice(uaString string) string
}

// RefererClassifier classifies traffic sources from referer URLs
type RefererClassifier interface {
	ClassifySource(refererStr string) string
}

// BreakdownItem represents a count with percentage for a single group value
type BreakdownItem struct {
	Value      string
	Count      int64
	Percentage float64 // e.g., 58.3
}

// AnalyticsSummaryResult holds summary analytics with breakdowns
type AnalyticsSummaryResult struct {
	ShortCode      string
	TotalClicks    int64
	Countries      []BreakdownItem
	DeviceTypes    []BreakdownItem
	TrafficSources []BreakdownItem
}

type AnalyticsService struct {
	repo              ClickRepository
	geoIP             GeoIPResolver
	deviceDetector    DeviceDetector
	refererClassifier RefererClassifier
}

func NewAnalyticsService(repo ClickRepository, geoIP GeoIPResolver, deviceDetector DeviceDetector, refererClassifier RefererClassifier) *AnalyticsService {
	return &AnalyticsService{
		repo:              repo,
		geoIP:             geoIP,
		deviceDetector:    deviceDetector,
		refererClassifier: refererClassifier,
	}
}

// RecordEnrichedClick enriches and stores a click event
func (s *AnalyticsService) RecordEnrichedClick(ctx context.Context, event events.ClickEvent) error {
	countryCode := s.geoIP.ResolveCountry(event.ClientIP)
	deviceType := s.deviceDetector.DetectDevice(event.UserAgent)
	trafficSource := s.refererClassifier.ClassifySource(event.Referer)
	return s.repo.InsertClick(ctx, event.ShortCode, event.Timestamp.Unix(), countryCode, deviceType, trafficSource)
}

// GetClickCount returns total clicks for a short code (backward compatibility)
func (s *AnalyticsService) GetClickCount(ctx context.Context, shortCode string) (int64, error) {
	return s.repo.CountByShortCode(ctx, shortCode)
}

// GetAnalyticsSummary returns summary analytics with breakdowns and percentages
func (s *AnalyticsService) GetAnalyticsSummary(ctx context.Context, shortCode string, from int64, to int64) (*AnalyticsSummaryResult, error) {
	// Get total count
	total, err := s.repo.CountInRange(ctx, shortCode, from, to)
	if err != nil {
		return nil, err
	}

	// Get breakdowns
	countries, err := s.repo.CountByCountryInRange(ctx, shortCode, from, to)
	if err != nil {
		return nil, err
	}

	devices, err := s.repo.CountByDeviceInRange(ctx, shortCode, from, to)
	if err != nil {
		return nil, err
	}

	sources, err := s.repo.CountBySourceInRange(ctx, shortCode, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to breakdown items with percentages
	result := &AnalyticsSummaryResult{
		ShortCode:      shortCode,
		TotalClicks:    total,
		Countries:      convertToBreakdownItems(countries, total),
		DeviceTypes:    convertToBreakdownItems(devices, total),
		TrafficSources: convertToBreakdownItems(sources, total),
	}

	return result, nil
}

// GetClickDetails returns paginated individual click records
func (s *AnalyticsService) GetClickDetails(ctx context.Context, shortCode string, cursorTimestamp int64, limit int) (*PaginatedClicks, error) {
	// Validate and default limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetClickDetails(ctx, shortCode, cursorTimestamp, limit)
}

// DeleteClickData removes all click records for a short code
func (s *AnalyticsService) DeleteClickData(ctx context.Context, shortCode string) error {
	return s.repo.DeleteByShortCode(ctx, shortCode)
}

// convertToBreakdownItems converts GroupCounts to BreakdownItems with percentage calculation
func convertToBreakdownItems(groups []GroupCount, total int64) []BreakdownItem {
	if total == 0 {
		return []BreakdownItem{}
	}

	items := make([]BreakdownItem, len(groups))
	for i, group := range groups {
		percentage := (float64(group.Count) / float64(total)) * 100.0
		items[i] = BreakdownItem{
			Value:      group.Value,
			Count:      group.Count,
			Percentage: percentage,
		}
	}
	return items
}

package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go-shortener/pkg/problemdetails"
)

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeProblem writes an RFC 7807 Problem Details response
func writeProblem(w http.ResponseWriter, problem *problemdetails.ProblemDetail) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	json.NewEncoder(w).Encode(problem)
}

// BreakdownResponse represents a single breakdown item with count and percentage
type BreakdownResponse struct {
	Value      string `json:"value"`
	Count      int64  `json:"count"`
	Percentage string `json:"percentage"` // "58.3%"
}

// AnalyticsSummaryResponse holds summary analytics with breakdowns
type AnalyticsSummaryResponse struct {
	ShortCode      string               `json:"short_code"`
	TotalClicks    int64                `json:"total_clicks"`
	Countries      []BreakdownResponse  `json:"countries"`
	DeviceTypes    []BreakdownResponse  `json:"device_types"`
	TrafficSources []BreakdownResponse  `json:"traffic_sources"`
}

// ClickDetailResponse represents a single click record with enrichment data
type ClickDetailResponse struct {
	ShortCode     string `json:"short_code"`
	ClickedAt     int64  `json:"clicked_at"`
	CountryCode   string `json:"country_code"`
	DeviceType    string `json:"device_type"`
	TrafficSource string `json:"traffic_source"`
}

// PaginatedClicksResponse holds paginated click details
type PaginatedClicksResponse struct {
	Clicks     []ClickDetailResponse `json:"clicks"`
	NextCursor string                `json:"next_cursor,omitempty"`
	HasMore    bool                  `json:"has_more"`
}

// formatPercentage formats a float percentage to "XX.X%" format
func formatPercentage(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

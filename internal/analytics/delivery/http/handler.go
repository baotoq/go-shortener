package http

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"go-shortener/internal/analytics/usecase"
	"go-shortener/internal/shared/events"
	"go-shortener/pkg/problemdetails"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler struct {
	analyticsService *usecase.AnalyticsService
	logger           *zap.Logger
}

func NewHandler(analyticsService *usecase.AnalyticsService, logger *zap.Logger) *Handler {
	return &Handler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// AnalyticsResponse is the API response for click count queries.
type AnalyticsResponse struct {
	ShortCode   string `json:"short_code"`
	TotalClicks int64  `json:"total_clicks"`
}

// GetClickCount handles GET /analytics/{code}
func (h *Handler) GetClickCount(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	count, err := h.analyticsService.GetClickCount(r.Context(), code)
	if err != nil {
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to retrieve click count",
		)
		writeProblem(w, problem)
		return
	}

	// Per user decision: zero clicks returns 200 with total_clicks: 0, NOT 404
	resp := AnalyticsResponse{
		ShortCode:   code,
		TotalClicks: count,
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetAnalyticsSummary handles GET /analytics/{code}/summary
func (h *Handler) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Parse time range query parameters
	fromUnix, toUnix, err := parseTimeRange(r)
	if err != nil {
		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeInvalidRequest,
			"Invalid Query Parameters",
			err.Error(),
		)
		writeProblem(w, problem)
		return
	}

	// Get summary from service
	summary, err := h.analyticsService.GetAnalyticsSummary(r.Context(), code, fromUnix, toUnix)
	if err != nil {
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to retrieve analytics summary",
		)
		writeProblem(w, problem)
		return
	}

	// Convert to response format
	resp := convertToSummaryResponse(summary)
	writeJSON(w, http.StatusOK, resp)
}

// GetClickDetails handles GET /analytics/{code}/clicks
func (h *Handler) GetClickDetails(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Parse cursor parameter
	cursorTimestamp, err := parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeInvalidRequest,
			"Invalid Query Parameters",
			"Invalid cursor format",
		)
		writeProblem(w, problem)
		return
	}

	// Parse limit parameter
	limit := parseLimit(r.URL.Query().Get("limit"))

	// Get click details from service
	details, err := h.analyticsService.GetClickDetails(r.Context(), code, cursorTimestamp, limit)
	if err != nil {
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to retrieve click details",
		)
		writeProblem(w, problem)
		return
	}

	// Convert to response format
	resp := convertToClickDetailsResponse(details)
	writeJSON(w, http.StatusOK, resp)
}

// HandleClickEvent processes click events from Dapr pub/sub.
// This is called by the Dapr sidecar when a message arrives on the "clicks" topic.
// Route: POST /events/click
func (h *Handler) HandleClickEvent(w http.ResponseWriter, r *http.Request) {
	// Dapr delivers CloudEvents — we need to extract the data field
	var cloudEvent struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cloudEvent); err != nil {
		h.logger.Error("failed to decode cloud event", zap.Error(err))
		// Return 200 to acknowledge — don't retry malformed events
		w.WriteHeader(http.StatusOK)
		return
	}

	var event events.ClickEvent
	if err := json.Unmarshal(cloudEvent.Data, &event); err != nil {
		h.logger.Error("failed to unmarshal click event data", zap.Error(err))
		// Return 200 to acknowledge — don't retry malformed events
		w.WriteHeader(http.StatusOK)
		return
	}

	// Use RecordEnrichedClick to enrich with GeoIP, UA, and Referer data
	if err := h.analyticsService.RecordEnrichedClick(r.Context(), event); err != nil {
		h.logger.Error("failed to record click",
			zap.String("short_code", event.ShortCode),
			zap.Error(err),
		)
		// Per research recommendation: don't retry on Phase 2, acknowledge the event
		w.WriteHeader(http.StatusOK)
		return
	}

	h.logger.Info("click event recorded",
		zap.String("short_code", event.ShortCode),
	)

	// Return 200 to signal successful processing to Dapr
	w.WriteHeader(http.StatusOK)
}

// HandleLinkDeleted processes link deletion events from Dapr pub/sub.
// This is called by the Dapr sidecar when a message arrives on the "link-deleted" topic.
// Route: POST /events/link-deleted
func (h *Handler) HandleLinkDeleted(w http.ResponseWriter, r *http.Request) {
	var cloudEvent struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cloudEvent); err != nil {
		h.logger.Error("failed to decode link deleted cloud event", zap.Error(err))
		w.WriteHeader(http.StatusOK) // Acknowledge malformed events
		return
	}

	var event events.LinkDeletedEvent
	if err := json.Unmarshal(cloudEvent.Data, &event); err != nil {
		h.logger.Error("failed to unmarshal link deleted event", zap.Error(err))
		w.WriteHeader(http.StatusOK) // Acknowledge malformed events
		return
	}

	if err := h.analyticsService.DeleteClickData(r.Context(), event.ShortCode); err != nil {
		h.logger.Error("failed to delete click data",
			zap.String("short_code", event.ShortCode),
			zap.Error(err),
		)
		// Acknowledge event to prevent infinite retries
	}

	h.logger.Info("deleted click data for link",
		zap.String("short_code", event.ShortCode),
	)

	w.WriteHeader(http.StatusOK)
}

// parseTimeRange parses from and to query parameters
func parseTimeRange(r *http.Request) (from int64, to int64, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Default from to beginning of time if not provided
	from = 0
	if fromStr != "" {
		fromTime, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return 0, 0, err
		}
		from = fromTime.Unix()
	}

	// Default to to current time if not provided
	to = time.Now().Unix()
	if toStr != "" {
		toTime, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return 0, 0, err
		}
		// Add 24 hours minus 1 second to include the entire end date
		to = toTime.Add(24*time.Hour - time.Second).Unix()
	}

	return from, to, nil
}

// parseCursor parses the cursor query parameter
func parseCursor(cursorStr string) (int64, error) {
	if cursorStr == "" {
		return math.MaxInt64, nil // Start from newest
	}

	decoded, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return 0, err
	}

	timestamp, err := strconv.ParseInt(string(decoded), 10, 64)
	if err != nil {
		return 0, err
	}

	return timestamp, nil
}

// parseLimit parses the limit query parameter
func parseLimit(limitStr string) int {
	if limitStr == "" {
		return 20 // Default
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 20
	}

	// Clamp to 1-100
	if limit < 1 {
		return 1
	}
	if limit > 100 {
		return 100
	}

	return limit
}

// convertToSummaryResponse converts service result to HTTP response
func convertToSummaryResponse(summary *usecase.AnalyticsSummaryResult) *AnalyticsSummaryResponse {
	return &AnalyticsSummaryResponse{
		ShortCode:      summary.ShortCode,
		TotalClicks:    summary.TotalClicks,
		Countries:      convertBreakdownItems(summary.Countries),
		DeviceTypes:    convertBreakdownItems(summary.DeviceTypes),
		TrafficSources: convertBreakdownItems(summary.TrafficSources),
	}
}

// convertBreakdownItems converts usecase breakdown items to response format
func convertBreakdownItems(items []usecase.BreakdownItem) []BreakdownResponse {
	resp := make([]BreakdownResponse, len(items))
	for i, item := range items {
		resp[i] = BreakdownResponse{
			Value:      item.Value,
			Count:      item.Count,
			Percentage: formatPercentage(item.Percentage),
		}
	}
	return resp
}

// convertToClickDetailsResponse converts service result to HTTP response
func convertToClickDetailsResponse(details *usecase.PaginatedClicks) *PaginatedClicksResponse {
	clicks := make([]ClickDetailResponse, len(details.Clicks))
	for i, click := range details.Clicks {
		clicks[i] = ClickDetailResponse{
			ShortCode:     click.ShortCode,
			ClickedAt:     click.ClickedAt,
			CountryCode:   click.CountryCode,
			DeviceType:    click.DeviceType,
			TrafficSource: click.TrafficSource,
		}
	}

	return &PaginatedClicksResponse{
		Clicks:     clicks,
		NextCursor: details.NextCursor,
		HasMore:    details.HasMore,
	}
}

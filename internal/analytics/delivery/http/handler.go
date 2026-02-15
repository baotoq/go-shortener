package http

import (
	"encoding/json"
	"net/http"

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

	if err := h.analyticsService.RecordClick(r.Context(), event); err != nil {
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

package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewRouter(handler *Handler, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Health checks
	r.Get("/healthz", handler.Healthz)
	r.Get("/readyz", handler.Readyz)

	// Dapr subscription endpoint — tells Dapr what topics to subscribe to
	r.Get("/dapr/subscribe", func(w http.ResponseWriter, r *http.Request) {
		subscriptions := []map[string]string{
			{
				"pubsubname": "pubsub",
				"topic":      "clicks",
				"route":      "/events/click",
			},
			{
				"pubsubname": "pubsub",
				"topic":      "link-deleted",
				"route":      "/events/link-deleted",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subscriptions)
	})

	// Dapr event delivery endpoints
	r.Post("/events/click", handler.HandleClickEvent)
	r.Post("/events/link-deleted", handler.HandleLinkDeleted)

	// Analytics API
	r.Get("/analytics/{code}", handler.GetClickCount)
	r.Get("/analytics/{code}/summary", handler.GetAnalyticsSummary)
	r.Get("/analytics/{code}/clicks", handler.GetClickDetails)

	return r
}

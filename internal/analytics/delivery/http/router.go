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

	// Dapr subscription endpoint — tells Dapr what topics to subscribe to
	r.Get("/dapr/subscribe", func(w http.ResponseWriter, r *http.Request) {
		subscriptions := []map[string]string{
			{
				"pubsubname": "pubsub",
				"topic":      "clicks",
				"route":      "/events/click",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subscriptions)
	})

	// Dapr event delivery endpoint
	r.Post("/events/click", handler.HandleClickEvent)

	// Analytics API
	r.Get("/analytics/{code}", handler.GetClickCount)

	return r
}

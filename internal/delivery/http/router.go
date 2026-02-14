package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewRouter creates a new Chi router with all middleware and routes
func NewRouter(handler *Handler, logger *zap.Logger, rateLimiter *RateLimiter) http.Handler {
	r := chi.NewRouter()

	// Global middleware chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LoggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(rateLimiter.Middleware)

	// Root-level redirect route
	r.Get("/{code}", handler.Redirect)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/urls", handler.CreateShortURL)
		r.Get("/urls/{code}", handler.GetURLDetails)
	})

	return r
}

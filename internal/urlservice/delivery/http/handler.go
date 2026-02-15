package http

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"go-shortener/internal/shared/events"
	"go-shortener/internal/urlservice/domain"
	"go-shortener/internal/urlservice/usecase"
	"go-shortener/pkg/problemdetails"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for URL operations
type Handler struct {
	service    *usecase.URLService
	baseURL    string
	daprClient dapr.Client  // may be nil if Dapr unavailable
	logger     *zap.Logger
}

// NewHandler creates a new Handler
func NewHandler(service *usecase.URLService, baseURL string, daprClient dapr.Client, logger *zap.Logger) *Handler {
	return &Handler{
		service:    service,
		baseURL:    baseURL,
		daprClient: daprClient,
		logger:     logger,
	}
}

// CreateShortURLRequest represents the request body for creating a short URL
type CreateShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// URLResponse represents the response for URL operations
type URLResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// CreateShortURL handles POST /api/v1/urls
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req CreateShortURLRequest

	// Decode JSON body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeInvalidRequest,
			"Invalid Request",
			"Request body must be valid JSON with 'original_url' field",
		)
		writeProblem(w, problem)
		return
	}

	// Validate original_url is not empty
	if req.OriginalURL == "" {
		problem := problemdetails.New(
			http.StatusBadRequest,
			problemdetails.TypeInvalidURL,
			"Invalid URL",
			"original_url is required",
		)
		writeProblem(w, problem)
		return
	}

	// Create short URL via service
	url, err := h.service.CreateShortURL(r.Context(), req.OriginalURL)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidURL) {
			problem := problemdetails.New(
				http.StatusBadRequest,
				problemdetails.TypeInvalidURL,
				"Invalid URL",
				err.Error(),
			)
			writeProblem(w, problem)
			return
		}

		if errors.Is(err, domain.ErrShortCodeConflict) {
			problem := problemdetails.New(
				http.StatusInternalServerError,
				problemdetails.TypeInternalError,
				"Internal Server Error",
				"Failed to generate short code",
			)
			writeProblem(w, problem)
			return
		}

		// Other errors
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Internal server error",
		)
		writeProblem(w, problem)
		return
	}

	// Build response
	response := URLResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    h.baseURL + "/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
	}

	writeJSON(w, http.StatusCreated, response)
}

// Redirect handles GET /{code}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	url, err := h.service.GetByShortCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			problem := problemdetails.New(
				http.StatusNotFound,
				problemdetails.TypeNotFound,
				"Not Found",
				"Short URL not found: "+code,
			)
			writeProblem(w, problem)
			return
		}

		// Other errors
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Internal server error",
		)
		writeProblem(w, problem)
		return
	}

	// Extract enrichment data BEFORE redirect (r may not be available in goroutine)
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		clientIP = host
	}
	userAgent := r.Header.Get("User-Agent")
	referer := r.Header.Get("Referer")

	// Send redirect response FIRST
	http.Redirect(w, r, url.OriginalURL, http.StatusFound)

	// Fire-and-forget: publish click event after redirect
	// Per user decision: on failure, log error and continue
	if h.daprClient != nil {
		go h.publishClickEvent(code, clientIP, userAgent, referer)
	}
}

// GetURLDetails handles GET /api/v1/urls/{code}
func (h *Handler) GetURLDetails(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	url, err := h.service.GetByShortCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			problem := problemdetails.New(
				http.StatusNotFound,
				problemdetails.TypeNotFound,
				"Not Found",
				"Short URL not found: "+code,
			)
			writeProblem(w, problem)
			return
		}

		// Other errors
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Internal server error",
		)
		writeProblem(w, problem)
		return
	}

	// Build response
	response := URLResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    h.baseURL + "/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
	}

	writeJSON(w, http.StatusOK, response)
}

const (
	pubsubName = "pubsub"
	topicName  = "clicks"
)

func (h *Handler) publishClickEvent(shortCode, clientIP, userAgent, referer string) {
	event := events.ClickEvent{
		ShortCode: shortCode,
		Timestamp: time.Now().UTC(),
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Referer:   referer,
	}

	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal click event",
			zap.String("short_code", shortCode),
			zap.Error(err),
		)
		return
	}

	ctx := context.Background()
	if err := h.daprClient.PublishEvent(ctx, pubsubName, topicName, data); err != nil {
		h.logger.Error("failed to publish click event",
			zap.String("short_code", shortCode),
			zap.Error(err),
		)
		// Click is lost, redirect already succeeded — per user decision
	}
}

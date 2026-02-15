package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"go-shortener/internal/shared/events"
	"go-shortener/internal/urlservice/domain"
	"go-shortener/internal/urlservice/usecase"
	"go-shortener/pkg/problemdetails"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for URL operations
type Handler struct {
	service    *usecase.URLService
	baseURL    string
	daprClient usecase.DaprClient  // may be nil if Dapr unavailable
	logger     *zap.Logger
	db         *sql.DB
}

// NewHandler creates a new Handler
func NewHandler(service *usecase.URLService, baseURL string, daprClient usecase.DaprClient, logger *zap.Logger, db *sql.DB) *Handler {
	return &Handler{
		service:    service,
		baseURL:    baseURL,
		daprClient: daprClient,
		logger:     logger,
		db:         db,
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

// ListLinks handles GET /api/v1/links
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	page := 1
	if p := query.Get("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed >= 1 {
			page = parsed
		}
	}

	perPage := 20
	if pp := query.Get("per_page"); pp != "" {
		if parsed, err := parseInt(pp); err == nil && parsed >= 1 && parsed <= 100 {
			perPage = parsed
		}
	}

	sort := query.Get("sort")
	if sort == "" {
		sort = "created_at"
	}

	order := query.Get("order")
	if order == "" {
		order = "desc"
	}

	// Parse date filters
	var createdAfter, createdBefore time.Time
	if ca := query.Get("created_after"); ca != "" {
		if parsed, err := time.Parse("2006-01-02", ca); err == nil {
			createdAfter = parsed
		}
	}
	if cb := query.Get("created_before"); cb != "" {
		if parsed, err := time.Parse("2006-01-02", cb); err == nil {
			// Add end-of-day to include the entire date
			createdBefore = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	search := query.Get("search")

	// Call service
	result, err := h.service.ListLinks(r.Context(), usecase.ListLinksParams{
		Page:          page,
		PerPage:       perPage,
		Sort:          sort,
		Order:         order,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Search:        search,
	})
	if err != nil {
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to list links",
		)
		writeProblem(w, problem)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetLinkDetail handles GET /api/v1/links/{code}
func (h *Handler) GetLinkDetail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	link, err := h.service.GetLinkDetail(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			problem := problemdetails.New(
				http.StatusNotFound,
				problemdetails.TypeNotFound,
				"Not Found",
				"Link not found: "+code,
			)
			writeProblem(w, problem)
			return
		}

		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to get link details",
		)
		writeProblem(w, problem)
		return
	}

	writeJSON(w, http.StatusOK, link)
}

// DeleteLink handles DELETE /api/v1/links/{code}
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if err := h.service.DeleteLink(r.Context(), code); err != nil {
		problem := problemdetails.New(
			http.StatusInternalServerError,
			problemdetails.TypeInternalError,
			"Internal Server Error",
			"Failed to delete link",
		)
		writeProblem(w, problem)
		return
	}

	// 204 No Content (idempotent - always returns 204 even if link didn't exist)
	w.WriteHeader(http.StatusNoContent)
}

// parseInt parses a string to int, returns error if invalid
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
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

// HealthResponse represents health check response
type HealthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Healthz handles GET /healthz (liveness probe)
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{Status: "ok"}
	writeJSON(w, http.StatusOK, resp)
}

// Readyz handles GET /readyz (readiness probe)
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check database connectivity
	if err := h.db.PingContext(ctx); err != nil {
		resp := HealthResponse{
			Status: "unavailable",
			Reason: "database unavailable: " + err.Error(),
		}
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	// Check Dapr sidecar if client exists
	if h.daprClient != nil {
		httpClient := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:3500/v1.0/healthz", nil)
		if err != nil {
			resp := HealthResponse{
				Status: "unavailable",
				Reason: "dapr health check failed: " + err.Error(),
			}
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}

		httpResp, err := httpClient.Do(req)
		if err != nil {
			resp := HealthResponse{
				Status: "unavailable",
				Reason: "dapr sidecar unavailable: " + err.Error(),
			}
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}
		httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			resp := HealthResponse{
				Status: "unavailable",
				Reason: fmt.Sprintf("dapr sidecar unhealthy: status %d", httpResp.StatusCode),
			}
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}
	}

	resp := HealthResponse{Status: "ready"}
	writeJSON(w, http.StatusOK, resp)
}

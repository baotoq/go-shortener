package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"go-shortener/internal/urlservice/domain"
	"go-shortener/internal/urlservice/usecase"
	"go-shortener/pkg/problemdetails"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for URL operations
type Handler struct {
	service *usecase.URLService
	baseURL string
}

// NewHandler creates a new Handler
func NewHandler(service *usecase.URLService, baseURL string) *Handler {
	return &Handler{
		service: service,
		baseURL: baseURL,
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

	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
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

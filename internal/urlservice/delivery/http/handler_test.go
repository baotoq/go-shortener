package http_test

import (
  "bytes"
  "encoding/json"
  "errors"
  "net/http"
  "net/http/httptest"
  "testing"
  "time"

  "go-shortener/internal/urlservice/domain"
  httphandler "go-shortener/internal/urlservice/delivery/http"
  "go-shortener/internal/urlservice/testutil/mocks"
  "go-shortener/internal/urlservice/usecase"
  "go-shortener/pkg/problemdetails"

  "github.com/go-chi/chi/v5"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
  "github.com/stretchr/testify/require"
  "go.uber.org/zap"
)

// setupTestHandler creates a handler with mocked dependencies for testing
func setupTestHandler(t *testing.T) (*httphandler.Handler, *mocks.MockURLRepository) {
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")
  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), nil)
  return handler, mockRepo
}

// TestCreateShortURL_ValidRequest_Returns201 verifies successful URL creation
func TestCreateShortURL_ValidRequest_Returns201(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  // Prepare request
  reqBody := map[string]string{"original_url": "https://example.com"}
  body, _ := json.Marshal(reqBody)
  req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByOriginalURL(mock.Anything, "https://example.com").Return(nil, domain.ErrURLNotFound)
  mockRepo.EXPECT().Save(mock.Anything, mock.AnythingOfType("string"), "https://example.com").Return(&domain.URL{
    ID:          1,
    ShortCode:   "abc123",
    OriginalURL: "https://example.com",
    CreatedAt:   time.Now(),
  }, nil)

  // Act
  handler.CreateShortURL(rr, req)

  // Assert
  assert.Equal(t, http.StatusCreated, rr.Code)
  assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

  var response httphandler.URLResponse
  err := json.NewDecoder(rr.Body).Decode(&response)
  require.NoError(t, err)
  assert.Equal(t, "abc123", response.ShortCode)
  assert.Equal(t, "http://localhost:8080/abc123", response.ShortURL)
  assert.Equal(t, "https://example.com", response.OriginalURL)
}

// TestCreateShortURL_InvalidJSON_Returns400 verifies malformed JSON handling
func TestCreateShortURL_InvalidJSON_Returns400(t *testing.T) {
  handler, _ := setupTestHandler(t)

  req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader([]byte("invalid json")))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  handler.CreateShortURL(rr, req)

  assert.Equal(t, http.StatusBadRequest, rr.Code)
  assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))

  var problem problemdetails.ProblemDetail
  err := json.NewDecoder(rr.Body).Decode(&problem)
  require.NoError(t, err)
  assert.Equal(t, http.StatusBadRequest, problem.Status)
  assert.Contains(t, problem.Type, problemdetails.TypeInvalidRequest)
}

// TestCreateShortURL_EmptyURL_Returns400 verifies empty URL validation
func TestCreateShortURL_EmptyURL_Returns400(t *testing.T) {
  handler, _ := setupTestHandler(t)

  reqBody := map[string]string{"original_url": ""}
  body, _ := json.Marshal(reqBody)
  req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  handler.CreateShortURL(rr, req)

  assert.Equal(t, http.StatusBadRequest, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInvalidURL)
  assert.Contains(t, problem.Detail, "original_url is required")
}

// TestCreateShortURL_InvalidURL_Returns400 verifies URL scheme validation
func TestCreateShortURL_InvalidURL_Returns400(t *testing.T) {
  handler, _ := setupTestHandler(t)

  reqBody := map[string]string{"original_url": "ftp://example.com"}
  body, _ := json.Marshal(reqBody)
  req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  handler.CreateShortURL(rr, req)

  assert.Equal(t, http.StatusBadRequest, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInvalidURL)
}

// TestCreateShortURL_ServerError_Returns500 verifies unexpected repository error handling
func TestCreateShortURL_ServerError_Returns500(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  reqBody := map[string]string{"original_url": "https://example.com"}
  body, _ := json.Marshal(reqBody)
  req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  // Mock unexpected error
  mockRepo.EXPECT().FindByOriginalURL(mock.Anything, "https://example.com").Return(nil, errors.New("database connection lost"))

  handler.CreateShortURL(rr, req)

  assert.Equal(t, http.StatusInternalServerError, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInternalError)
}

// TestGetURLDetails_ExistingCode_Returns200 verifies URL details retrieval
func TestGetURLDetails_ExistingCode_Returns200(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/urls/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "abc123").Return(&domain.URL{
    ID:          1,
    ShortCode:   "abc123",
    OriginalURL: "https://example.com",
    CreatedAt:   time.Now(),
  }, nil)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/api/v1/urls/{code}", handler.GetURLDetails)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.URLResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, "abc123", response.ShortCode)
  assert.Equal(t, "https://example.com", response.OriginalURL)
}

// TestRedirect_ExistingCode_Returns302 verifies successful redirect
func TestRedirect_ExistingCode_Returns302(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "abc123").Return(&domain.URL{
    ID:          1,
    ShortCode:   "abc123",
    OriginalURL: "https://example.com",
    CreatedAt:   time.Now(),
  }, nil)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/{code}", handler.Redirect)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusFound, rr.Code)
  assert.Equal(t, "https://example.com", rr.Header().Get("Location"))
}

// TestRedirect_NotFound_Returns404 verifies not found handling
func TestRedirect_NotFound_Returns404(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/invalid", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "invalid").Return(nil, domain.ErrURLNotFound)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/{code}", handler.Redirect)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusNotFound, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeNotFound)
  assert.Contains(t, problem.Detail, "invalid")
}

// TestListLinks_DefaultParams_Returns200 verifies listing with default parameters
func TestListLinks_DefaultParams_Returns200(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/links", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindAll(mock.Anything, mock.MatchedBy(func(p usecase.FindAllParams) bool {
    return p.Limit == 20 && p.Offset == 0
  })).Return([]domain.URL{
    {ID: 1, ShortCode: "abc123", OriginalURL: "https://example.com", CreatedAt: time.Now()},
  }, nil)
  mockRepo.EXPECT().Count(mock.Anything, mock.AnythingOfType("usecase.CountParams")).Return(int64(1), nil)

  handler.ListLinks(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
  assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

  var response usecase.LinkListResult
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Len(t, response.Links, 1)
  assert.Equal(t, int64(1), response.Total)
}

// TestListLinks_WithPagination_Returns200 verifies pagination handling
func TestListLinks_WithPagination_Returns200(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/links?page=2&per_page=5", nil)
  rr := httptest.NewRecorder()

  // Mock expectations - page 2 with per_page 5 = offset 5
  mockRepo.EXPECT().FindAll(mock.Anything, mock.MatchedBy(func(p usecase.FindAllParams) bool {
    return p.Limit == 5 && p.Offset == 5
  })).Return([]domain.URL{}, nil)
  mockRepo.EXPECT().Count(mock.Anything, mock.AnythingOfType("usecase.CountParams")).Return(int64(10), nil)

  handler.ListLinks(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestListLinks_ServerError_Returns500 verifies repository error handling
func TestListLinks_ServerError_Returns500(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/links", nil)
  rr := httptest.NewRecorder()

  // Mock error
  mockRepo.EXPECT().FindAll(mock.Anything, mock.Anything).Return(nil, errors.New("database timeout"))

  handler.ListLinks(rr, req)

  assert.Equal(t, http.StatusInternalServerError, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInternalError)
}

// TestGetLinkDetail_Exists_Returns200 verifies link detail retrieval
func TestGetLinkDetail_Exists_Returns200(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/links/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "abc123").Return(&domain.URL{
    ID:          1,
    ShortCode:   "abc123",
    OriginalURL: "https://example.com",
    CreatedAt:   time.Now(),
  }, nil)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/api/v1/links/{code}", handler.GetLinkDetail)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response usecase.LinkWithClicks
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, "abc123", response.ShortCode)
  assert.Equal(t, "https://example.com", response.OriginalURL)
}

// TestGetLinkDetail_NotFound_Returns404 verifies not found handling
func TestGetLinkDetail_NotFound_Returns404(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/api/v1/links/invalid", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "invalid").Return(nil, domain.ErrURLNotFound)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/api/v1/links/{code}", handler.GetLinkDetail)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusNotFound, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeNotFound)
}

// TestDeleteLink_Success_Returns204 verifies successful deletion
func TestDeleteLink_Success_Returns204(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("DELETE", "/api/v1/links/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().Delete(mock.Anything, "abc123").Return(nil)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Delete("/api/v1/links/{code}", handler.DeleteLink)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusNoContent, rr.Code)
}

// TestDeleteLink_Error_Returns500 verifies error handling on delete
func TestDeleteLink_Error_Returns500(t *testing.T) {
  handler, mockRepo := setupTestHandler(t)

  req := httptest.NewRequest("DELETE", "/api/v1/links/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock error
  mockRepo.EXPECT().Delete(mock.Anything, "abc123").Return(errors.New("database lock"))

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Delete("/api/v1/links/{code}", handler.DeleteLink)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusInternalServerError, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInternalError)
}

// TestHealthz_Returns200 verifies liveness probe
func TestHealthz_Returns200(t *testing.T) {
  handler, _ := setupTestHandler(t)

  req := httptest.NewRequest("GET", "/healthz", nil)
  rr := httptest.NewRecorder()

  handler.Healthz(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.HealthResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, "ok", response.Status)
}

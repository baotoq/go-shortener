package http_test

import (
  "database/sql"
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"

  "go-shortener/internal/urlservice/domain"
  httphandler "go-shortener/internal/urlservice/delivery/http"
  "go-shortener/internal/urlservice/testutil/mocks"
  "go-shortener/internal/urlservice/usecase"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
  "github.com/stretchr/testify/require"
  "go.uber.org/zap"
  _ "modernc.org/sqlite"
)

// TestNewRouter_HealthCheckRoute_BypassesRateLimit verifies health checks bypass rate limiter
func TestNewRouter_HealthCheckRoute_BypassesRateLimit(t *testing.T) {
  // Create handler
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")
  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), nil)

  // Create rate limiter with limit=1
  rateLimiter := httphandler.NewRateLimiter(1)
  logger := zap.NewNop()

  // Create router
  router := httphandler.NewRouter(handler, logger, rateLimiter)

  // Use redirect route to exhaust rate limit (simpler than POST which needs multiple mocks)
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "test1").Return(nil, domain.ErrURLNotFound).Once()
  req1 := httptest.NewRequest("GET", "/test1", nil)
  rr1 := httptest.NewRecorder()
  router.ServeHTTP(rr1, req1)

  // Second request should be rate limited
  req2 := httptest.NewRequest("GET", "/test2", nil)
  rr2 := httptest.NewRecorder()
  router.ServeHTTP(rr2, req2)
  assert.Equal(t, http.StatusTooManyRequests, rr2.Code, "Business route should be rate limited")

  // Health check should still work despite rate limit exhausted
  req3 := httptest.NewRequest("GET", "/healthz", nil)
  rr3 := httptest.NewRecorder()
  router.ServeHTTP(rr3, req3)
  assert.Equal(t, http.StatusOK, rr3.Code, "Health check should bypass rate limiter")

  var response httphandler.HealthResponse
  json.NewDecoder(rr3.Body).Decode(&response)
  assert.Equal(t, "ok", response.Status)
}

// TestNewRouter_ReadyzRoute_BypassesRateLimit verifies readiness checks bypass rate limiter
func TestNewRouter_ReadyzRoute_BypassesRateLimit(t *testing.T) {
  // Create handler with in-memory DB
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")

  db, err := sql.Open("sqlite", ":memory:")
  require.NoError(t, err)
  defer db.Close()

  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), db)

  // Create rate limiter with limit=1
  rateLimiter := httphandler.NewRateLimiter(1)
  logger := zap.NewNop()

  // Create router
  router := httphandler.NewRouter(handler, logger, rateLimiter)

  // Exhaust rate limit with redirect route
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "test1").Return(nil, domain.ErrURLNotFound).Once()
  req1 := httptest.NewRequest("GET", "/test1", nil)
  rr1 := httptest.NewRecorder()
  router.ServeHTTP(rr1, req1)

  req2 := httptest.NewRequest("GET", "/test2", nil)
  rr2 := httptest.NewRecorder()
  router.ServeHTTP(rr2, req2)
  assert.Equal(t, http.StatusTooManyRequests, rr2.Code)

  // Readyz should still work
  req3 := httptest.NewRequest("GET", "/readyz", nil)
  rr3 := httptest.NewRecorder()
  router.ServeHTTP(rr3, req3)
  assert.Equal(t, http.StatusOK, rr3.Code, "Readyz should bypass rate limiter")

  var response httphandler.HealthResponse
  json.NewDecoder(rr3.Body).Decode(&response)
  assert.Equal(t, "ready", response.Status)
}

// TestNewRouter_BusinessRoute_RateLimited verifies business routes are rate limited
func TestNewRouter_BusinessRoute_RateLimited(t *testing.T) {
  // Create handler
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")
  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), nil)

  // Create rate limiter with limit=1
  rateLimiter := httphandler.NewRateLimiter(1)
  logger := zap.NewNop()

  // Create router
  router := httphandler.NewRouter(handler, logger, rateLimiter)

  // First request to redirect route
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "test1").Return(nil, domain.ErrURLNotFound).Once()
  req1 := httptest.NewRequest("GET", "/test1", nil)
  rr1 := httptest.NewRecorder()
  router.ServeHTTP(rr1, req1)
  assert.NotEqual(t, http.StatusTooManyRequests, rr1.Code, "First request should not be rate limited")

  // Second request should be rate limited
  req2 := httptest.NewRequest("GET", "/test2", nil)
  rr2 := httptest.NewRecorder()
  router.ServeHTTP(rr2, req2)
  assert.Equal(t, http.StatusTooManyRequests, rr2.Code, "Second request should be rate limited")
}

// TestNewRouter_RedirectRoute_Registered verifies redirect route is registered
func TestNewRouter_RedirectRoute_Registered(t *testing.T) {
  // Create handler
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")
  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), nil)

  rateLimiter := httphandler.NewRateLimiter(100)
  logger := zap.NewNop()

  // Create router
  router := httphandler.NewRouter(handler, logger, rateLimiter)

  // Mock will return error, but we verify route is registered (not 405 Method Not Allowed)
  mockRepo.EXPECT().FindByShortCode(mock.Anything, "abc123").Return(nil, domain.ErrURLNotFound)

  req := httptest.NewRequest("GET", "/abc123", nil)
  rr := httptest.NewRecorder()
  router.ServeHTTP(rr, req)

  // Should get 404 (business logic) not 405 (method not allowed)
  // The handler was called, so it returns application/problem+json
  assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code, "Route should be registered")
  assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"), "Handler should be called")
}

// TestNewRouter_APIRoutes_Registered verifies all API routes are registered
func TestNewRouter_APIRoutes_Registered(t *testing.T) {
  // Create handler
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost:8080")
  handler := httphandler.NewHandler(service, "http://localhost:8080", nil, zap.NewNop(), nil)

  rateLimiter := httphandler.NewRateLimiter(100)
  logger := zap.NewNop()

  // Create router
  router := httphandler.NewRouter(handler, logger, rateLimiter)

  // Test that routes are registered and return something other than 405
  // We use GET /api/v1/links which is simplest (no URL params, no mocks needed for success)
  mockRepo.EXPECT().FindAll(mock.Anything, mock.Anything).Return([]domain.URL{}, nil)
  mockRepo.EXPECT().Count(mock.Anything, mock.Anything).Return(int64(0), nil)

  req := httptest.NewRequest("GET", "/api/v1/links", nil)
  rr := httptest.NewRecorder()
  router.ServeHTTP(rr, req)

  // Should return 200 (success), not 405 (method not allowed)
  assert.Equal(t, http.StatusOK, rr.Code, "GET /api/v1/links should be registered")
  assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

  // Test DELETE route
  mockRepo.EXPECT().Delete(mock.Anything, "abc123").Return(nil)

  req2 := httptest.NewRequest("DELETE", "/api/v1/links/abc123", nil)
  rr2 := httptest.NewRecorder()
  router.ServeHTTP(rr2, req2)

  assert.Equal(t, http.StatusNoContent, rr2.Code, "DELETE /api/v1/links/{code} should be registered")
}

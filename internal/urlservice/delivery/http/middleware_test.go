package http_test

import (
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"

  httphandler "go-shortener/internal/urlservice/delivery/http"
  "go-shortener/pkg/problemdetails"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
  "go.uber.org/zap"
)

// TestRateLimiter_Middleware_WithinLimit_Returns200 verifies requests within limit succeed
func TestRateLimiter_Middleware_WithinLimit_Returns200(t *testing.T) {
  rl := httphandler.NewRateLimiter(100)

  // Create simple handler that always returns 200
  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  handler := rl.Middleware(nextHandler)

  // Send 5 requests from same IP
  for i := 0; i < 5; i++ {
    req := httptest.NewRequest("GET", "/test", nil)
    req.RemoteAddr = "192.168.1.1:12345"
    rr := httptest.NewRecorder()

    handler.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
    assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
    assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
  }
}

// TestRateLimiter_Middleware_ExceedsLimit_Returns429 verifies rate limit enforcement
func TestRateLimiter_Middleware_ExceedsLimit_Returns429(t *testing.T) {
  rl := httphandler.NewRateLimiter(2)

  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  handler := rl.Middleware(nextHandler)

  // Send 3 requests from same IP
  for i := 0; i < 3; i++ {
    req := httptest.NewRequest("GET", "/test", nil)
    req.RemoteAddr = "192.168.1.1:12345"
    rr := httptest.NewRecorder()

    handler.ServeHTTP(rr, req)

    if i < 2 {
      // First 2 requests should succeed
      assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
    } else {
      // Third request should be rate limited
      assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Request %d should be rate limited", i+1)
      assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))
      assert.Equal(t, "2", rr.Header().Get("X-RateLimit-Limit"))
      assert.Equal(t, "0", rr.Header().Get("X-RateLimit-Remaining"))
      assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))

      // Verify Problem Details structure
      var problem problemdetails.ProblemDetail
      err := json.NewDecoder(rr.Body).Decode(&problem)
      require.NoError(t, err)
      assert.Equal(t, http.StatusTooManyRequests, problem.Status)
      assert.Contains(t, problem.Type, "rate-limit-exceeded")
    }
  }
}

// TestRateLimiter_Middleware_DifferentIPs_IndependentLimits verifies per-IP isolation
func TestRateLimiter_Middleware_DifferentIPs_IndependentLimits(t *testing.T) {
  rl := httphandler.NewRateLimiter(1)

  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  handler := rl.Middleware(nextHandler)

  // Request from first IP
  req1 := httptest.NewRequest("GET", "/test", nil)
  req1.RemoteAddr = "10.0.0.1:1234"
  rr1 := httptest.NewRecorder()
  handler.ServeHTTP(rr1, req1)
  assert.Equal(t, http.StatusOK, rr1.Code, "First IP should succeed")

  // Request from second IP (should also succeed despite limit=1)
  req2 := httptest.NewRequest("GET", "/test", nil)
  req2.RemoteAddr = "10.0.0.2:5678"
  rr2 := httptest.NewRecorder()
  handler.ServeHTTP(rr2, req2)
  assert.Equal(t, http.StatusOK, rr2.Code, "Second IP should succeed independently")
}

// TestRateLimiter_Middleware_SetsHeaders verifies rate limit headers
func TestRateLimiter_Middleware_SetsHeaders(t *testing.T) {
  rl := httphandler.NewRateLimiter(10)

  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  })

  handler := rl.Middleware(nextHandler)

  // Test successful request headers
  req := httptest.NewRequest("GET", "/test", nil)
  req.RemoteAddr = "192.168.1.1:12345"
  rr := httptest.NewRecorder()

  handler.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
  assert.Equal(t, "10", rr.Header().Get("X-RateLimit-Limit"))

  // X-RateLimit-Remaining should be set and be a number less than limit
  remaining := rr.Header().Get("X-RateLimit-Remaining")
  assert.NotEmpty(t, remaining)

  // Test 429 response headers by exhausting limit
  rl2 := httphandler.NewRateLimiter(1)
  handler2 := rl2.Middleware(nextHandler)

  // First request succeeds
  req1 := httptest.NewRequest("GET", "/test", nil)
  req1.RemoteAddr = "192.168.1.2:12345"
  rr1 := httptest.NewRecorder()
  handler2.ServeHTTP(rr1, req1)
  assert.Equal(t, http.StatusOK, rr1.Code)

  // Second request gets 429 with Reset header
  req2 := httptest.NewRequest("GET", "/test", nil)
  req2.RemoteAddr = "192.168.1.2:12345"
  rr2 := httptest.NewRecorder()
  handler2.ServeHTTP(rr2, req2)

  assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
  assert.Equal(t, "1", rr2.Header().Get("X-RateLimit-Limit"))
  assert.Equal(t, "0", rr2.Header().Get("X-RateLimit-Remaining"))
  assert.NotEmpty(t, rr2.Header().Get("X-RateLimit-Reset"))
}

// TestLoggerMiddleware_LogsRequest verifies logger middleware calls next handler
func TestLoggerMiddleware_LogsRequest(t *testing.T) {
  logger := zap.NewNop()
  middleware := httphandler.LoggerMiddleware(logger)

  // Track whether next handler was called
  nextCalled := false
  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    nextCalled = true
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  handler := middleware(nextHandler)

  req := httptest.NewRequest("GET", "/test", nil)
  rr := httptest.NewRecorder()

  handler.ServeHTTP(rr, req)

  assert.True(t, nextCalled, "Next handler should be called")
  assert.Equal(t, http.StatusOK, rr.Code)
  assert.Equal(t, "OK", rr.Body.String())
}

// TestLoggerMiddleware_PreservesStatusCode verifies status code preservation
func TestLoggerMiddleware_PreservesStatusCode(t *testing.T) {
  logger := zap.NewNop()
  middleware := httphandler.LoggerMiddleware(logger)

  // Create handler that returns 404
  nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Not Found"))
  })

  handler := middleware(nextHandler)

  req := httptest.NewRequest("GET", "/test", nil)
  rr := httptest.NewRecorder()

  handler.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusNotFound, rr.Code, "Status code should be preserved")
  assert.Equal(t, "Not Found", rr.Body.String())
}

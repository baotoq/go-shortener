package http_test

import (
  "bytes"
  "encoding/base64"
  "encoding/json"
  "errors"
  "net/http"
  "net/http/httptest"
  "testing"
  "time"

  httphandler "go-shortener/internal/analytics/delivery/http"
  "go-shortener/internal/analytics/testutil/mocks"
  "go-shortener/internal/analytics/usecase"
  "go-shortener/internal/shared/events"
  "go-shortener/pkg/problemdetails"

  "github.com/go-chi/chi/v5"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
  "github.com/stretchr/testify/require"
  "go.uber.org/zap"
)

// setupTestAnalyticsHandler creates a handler with all mocked dependencies
func setupTestAnalyticsHandler(t *testing.T) (*httphandler.Handler, *mocks.MockClickRepository, *mocks.MockGeoIPResolver, *mocks.MockDeviceDetector, *mocks.MockRefererClassifier) {
  mockRepo := mocks.NewMockClickRepository(t)
  mockGeoIP := mocks.NewMockGeoIPResolver(t)
  mockDevice := mocks.NewMockDeviceDetector(t)
  mockReferer := mocks.NewMockRefererClassifier(t)
  service := usecase.NewAnalyticsService(mockRepo, mockGeoIP, mockDevice, mockReferer)
  handler := httphandler.NewHandler(service, zap.NewNop(), nil)
  return handler, mockRepo, mockGeoIP, mockDevice, mockReferer
}

// TestGetClickCount_ReturnsCount verifies click count retrieval
func TestGetClickCount_ReturnsCount(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().CountByShortCode(mock.Anything, "abc123").Return(int64(42), nil)

  // Setup chi router for URL param extraction
  r := chi.NewRouter()
  r.Get("/analytics/{code}", handler.GetClickCount)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
  assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

  var response httphandler.AnalyticsResponse
  err := json.NewDecoder(rr.Body).Decode(&response)
  require.NoError(t, err)
  assert.Equal(t, "abc123", response.ShortCode)
  assert.Equal(t, int64(42), response.TotalClicks)
}

// TestGetClickCount_ZeroClicks_Returns200WithZero verifies zero clicks handling per decision
func TestGetClickCount_ZeroClicks_Returns200WithZero(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/xyz789", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().CountByShortCode(mock.Anything, "xyz789").Return(int64(0), nil)

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}", handler.GetClickCount)
  r.ServeHTTP(rr, req)

  // Per decision: zero clicks returns 200, not 404
  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.AnalyticsResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, int64(0), response.TotalClicks)
}

// TestGetClickCount_Error_Returns500 verifies error handling
func TestGetClickCount_Error_Returns500(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123", nil)
  rr := httptest.NewRecorder()

  // Mock error
  mockRepo.EXPECT().CountByShortCode(mock.Anything, "abc123").Return(int64(0), errors.New("database timeout"))

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}", handler.GetClickCount)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusInternalServerError, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInternalError)
}

// TestGetAnalyticsSummary_Returns200 verifies summary with breakdowns
func TestGetAnalyticsSummary_Returns200(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123/summary", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().CountInRange(mock.Anything, "abc123", int64(0), mock.AnythingOfType("int64")).Return(int64(100), nil)
  mockRepo.EXPECT().CountByCountryInRange(mock.Anything, "abc123", int64(0), mock.AnythingOfType("int64")).Return([]usecase.GroupCount{
    {Value: "US", Count: 42},
  }, nil)
  mockRepo.EXPECT().CountByDeviceInRange(mock.Anything, "abc123", int64(0), mock.AnythingOfType("int64")).Return([]usecase.GroupCount{
    {Value: "desktop", Count: 60},
  }, nil)
  mockRepo.EXPECT().CountBySourceInRange(mock.Anything, "abc123", int64(0), mock.AnythingOfType("int64")).Return([]usecase.GroupCount{
    {Value: "search", Count: 50},
  }, nil)

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/summary", handler.GetAnalyticsSummary)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.AnalyticsSummaryResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, "abc123", response.ShortCode)
  assert.Equal(t, int64(100), response.TotalClicks)
  assert.Len(t, response.Countries, 1)
  assert.Equal(t, "42.0%", response.Countries[0].Percentage)
}

// TestGetAnalyticsSummary_WithTimeRange_PassesParams verifies time range parsing
func TestGetAnalyticsSummary_WithTimeRange_PassesParams(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123/summary?from=2024-01-01&to=2024-01-31", nil)
  rr := httptest.NewRecorder()

  // Parse expected timestamps
  fromTime, _ := time.Parse("2006-01-02", "2024-01-01")
  toTime, _ := time.Parse("2006-01-02", "2024-01-31")
  expectedFrom := fromTime.Unix()
  expectedTo := toTime.Add(24*time.Hour - time.Second).Unix()

  // Mock expectations
  mockRepo.EXPECT().CountInRange(mock.Anything, "abc123", expectedFrom, expectedTo).Return(int64(50), nil)
  mockRepo.EXPECT().CountByCountryInRange(mock.Anything, "abc123", expectedFrom, expectedTo).Return([]usecase.GroupCount{}, nil)
  mockRepo.EXPECT().CountByDeviceInRange(mock.Anything, "abc123", expectedFrom, expectedTo).Return([]usecase.GroupCount{}, nil)
  mockRepo.EXPECT().CountBySourceInRange(mock.Anything, "abc123", expectedFrom, expectedTo).Return([]usecase.GroupCount{}, nil)

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/summary", handler.GetAnalyticsSummary)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestGetAnalyticsSummary_InvalidDateFormat_Returns400 verifies date parsing error handling
func TestGetAnalyticsSummary_InvalidDateFormat_Returns400(t *testing.T) {
  handler, _, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123/summary?from=invalid-date", nil)
  rr := httptest.NewRecorder()

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/summary", handler.GetAnalyticsSummary)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusBadRequest, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInvalidRequest)
}

// TestGetClickDetails_Returns200 verifies paginated click details
func TestGetClickDetails_Returns200(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123/clicks", nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().GetClickDetails(mock.Anything, "abc123", mock.AnythingOfType("int64"), 20).Return(&usecase.PaginatedClicks{
    Clicks: []usecase.ClickDetail{
      {ID: 1, ShortCode: "abc123", ClickedAt: 1700000000, CountryCode: "US", DeviceType: "desktop", TrafficSource: "search"},
    },
    NextCursor: "cursor123",
    HasMore:    true,
  }, nil)

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/clicks", handler.GetClickDetails)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.PaginatedClicksResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Len(t, response.Clicks, 1)
  assert.Equal(t, "cursor123", response.NextCursor)
  assert.True(t, response.HasMore)
}

// TestGetClickDetails_WithCursor_ParsesCursor verifies cursor parsing
func TestGetClickDetails_WithCursor_ParsesCursor(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  // Create valid base64 cursor
  cursor := base64.StdEncoding.EncodeToString([]byte("1700000000"))
  req := httptest.NewRequest("GET", "/analytics/abc123/clicks?cursor="+cursor, nil)
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().GetClickDetails(mock.Anything, "abc123", int64(1700000000), 20).Return(&usecase.PaginatedClicks{
    Clicks:  []usecase.ClickDetail{},
    HasMore: false,
  }, nil)

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/clicks", handler.GetClickDetails)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestGetClickDetails_InvalidCursor_Returns400 verifies invalid cursor handling
func TestGetClickDetails_InvalidCursor_Returns400(t *testing.T) {
  handler, _, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/analytics/abc123/clicks?cursor=invalid-base64!!!", nil)
  rr := httptest.NewRecorder()

  // Setup chi router
  r := chi.NewRouter()
  r.Get("/analytics/{code}/clicks", handler.GetClickDetails)
  r.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusBadRequest, rr.Code)

  var problem problemdetails.ProblemDetail
  json.NewDecoder(rr.Body).Decode(&problem)
  assert.Contains(t, problem.Type, problemdetails.TypeInvalidRequest)
  assert.Contains(t, problem.Detail, "Invalid cursor format")
}

// TestHandleClickEvent_ValidEvent_Returns200 verifies CloudEvent processing
func TestHandleClickEvent_ValidEvent_Returns200(t *testing.T) {
  handler, mockRepo, mockGeoIP, mockDevice, mockReferer := setupTestAnalyticsHandler(t)

  // Create CloudEvent wrapper
  event := events.ClickEvent{
    ShortCode: "abc123",
    Timestamp: time.Unix(1700000000, 0),
    ClientIP:  "192.168.1.1",
    UserAgent: "Mozilla/5.0",
    Referer:   "https://google.com",
  }
  cloudEvent := map[string]interface{}{
    "data": event,
  }
  body, _ := json.Marshal(cloudEvent)
  req := httptest.NewRequest("POST", "/events/click", bytes.NewReader(body))
  req.Header.Set("Content-Type", "application/json")
  rr := httptest.NewRecorder()

  // Mock expectations
  mockGeoIP.EXPECT().ResolveCountry("192.168.1.1").Return("US")
  mockDevice.EXPECT().DetectDevice("Mozilla/5.0").Return("desktop")
  mockReferer.EXPECT().ClassifySource("https://google.com").Return("search")
  mockRepo.EXPECT().InsertClick(mock.Anything, "abc123", int64(1700000000), "US", "desktop", "search").Return(nil)

  handler.HandleClickEvent(rr, req)

  // Per decision: event handlers always return 200
  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHandleClickEvent_MalformedJSON_Returns200 verifies malformed event acknowledgment
func TestHandleClickEvent_MalformedJSON_Returns200(t *testing.T) {
  handler, _, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("POST", "/events/click", bytes.NewReader([]byte("invalid json")))
  rr := httptest.NewRecorder()

  handler.HandleClickEvent(rr, req)

  // Per decision: acknowledge malformed events to prevent retry
  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHandleClickEvent_ProcessingError_Returns200 verifies error acknowledgment
func TestHandleClickEvent_ProcessingError_Returns200(t *testing.T) {
  handler, mockRepo, mockGeoIP, mockDevice, mockReferer := setupTestAnalyticsHandler(t)

  event := events.ClickEvent{
    ShortCode: "abc123",
    Timestamp: time.Unix(1700000000, 0),
    ClientIP:  "192.168.1.1",
    UserAgent: "Mozilla/5.0",
    Referer:   "https://google.com",
  }
  cloudEvent := map[string]interface{}{
    "data": event,
  }
  body, _ := json.Marshal(cloudEvent)
  req := httptest.NewRequest("POST", "/events/click", bytes.NewReader(body))
  rr := httptest.NewRecorder()

  // Mock processing error
  mockGeoIP.EXPECT().ResolveCountry("192.168.1.1").Return("US")
  mockDevice.EXPECT().DetectDevice("Mozilla/5.0").Return("desktop")
  mockReferer.EXPECT().ClassifySource("https://google.com").Return("search")
  mockRepo.EXPECT().InsertClick(mock.Anything, "abc123", int64(1700000000), "US", "desktop", "search").Return(errors.New("database locked"))

  handler.HandleClickEvent(rr, req)

  // Per decision: acknowledge error to prevent retry storm
  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHandleLinkDeleted_ValidEvent_DeletesData verifies link deletion event handling
func TestHandleLinkDeleted_ValidEvent_DeletesData(t *testing.T) {
  handler, mockRepo, _, _, _ := setupTestAnalyticsHandler(t)

  event := events.LinkDeletedEvent{
    ShortCode: "abc123",
    DeletedAt: time.Now(),
  }
  cloudEvent := map[string]interface{}{
    "data": event,
  }
  body, _ := json.Marshal(cloudEvent)
  req := httptest.NewRequest("POST", "/events/link-deleted", bytes.NewReader(body))
  rr := httptest.NewRecorder()

  // Mock expectations
  mockRepo.EXPECT().DeleteByShortCode(mock.Anything, "abc123").Return(nil)

  handler.HandleLinkDeleted(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHandleLinkDeleted_MalformedEvent_Returns200 verifies malformed event acknowledgment
func TestHandleLinkDeleted_MalformedEvent_Returns200(t *testing.T) {
  handler, _, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("POST", "/events/link-deleted", bytes.NewReader([]byte("not json")))
  rr := httptest.NewRecorder()

  handler.HandleLinkDeleted(rr, req)

  // Per decision: acknowledge malformed events
  assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHealthz_Returns200 verifies liveness probe
func TestHealthz_Returns200(t *testing.T) {
  handler, _, _, _, _ := setupTestAnalyticsHandler(t)

  req := httptest.NewRequest("GET", "/healthz", nil)
  rr := httptest.NewRecorder()

  handler.Healthz(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)

  var response httphandler.HealthResponse
  json.NewDecoder(rr.Body).Decode(&response)
  assert.Equal(t, "ok", response.Status)
}

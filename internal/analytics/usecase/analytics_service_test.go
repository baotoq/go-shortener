package usecase_test

import (
  "context"
  "errors"
  "testing"
  "time"

  "go-shortener/internal/analytics/testutil/mocks"
  "go-shortener/internal/analytics/usecase"
  "go-shortener/internal/shared/events"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
  "github.com/stretchr/testify/require"
)

// TestRecordEnrichedClick_EnrichesAndStores verifies enrichment services are called and click is stored
func TestRecordEnrichedClick_EnrichesAndStores(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  mockGeoIP := mocks.NewMockGeoIPResolver(t)
  mockDevice := mocks.NewMockDeviceDetector(t)
  mockReferer := mocks.NewMockRefererClassifier(t)
  service := usecase.NewAnalyticsService(mockRepo, mockGeoIP, mockDevice, mockReferer)

  ctx := context.Background()
  event := events.ClickEvent{
    ShortCode: "abc123",
    Timestamp: time.Unix(1700000000, 0),
    ClientIP:  "192.168.1.1",
    UserAgent: "Mozilla/5.0",
    Referer:   "https://google.com",
  }

  // Mock expectations
  mockGeoIP.EXPECT().ResolveCountry("192.168.1.1").Return("US")
  mockDevice.EXPECT().DetectDevice("Mozilla/5.0").Return("desktop")
  mockReferer.EXPECT().ClassifySource("https://google.com").Return("search")
  mockRepo.EXPECT().InsertClick(ctx, "abc123", int64(1700000000), "US", "desktop", "search").Return(nil)

  // Act
  err := service.RecordEnrichedClick(ctx, event)

  // Assert
  require.NoError(t, err)
}

// TestRecordEnrichedClick_RepositoryError_ReturnsError verifies repo errors are propagated
func TestRecordEnrichedClick_RepositoryError_ReturnsError(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  mockGeoIP := mocks.NewMockGeoIPResolver(t)
  mockDevice := mocks.NewMockDeviceDetector(t)
  mockReferer := mocks.NewMockRefererClassifier(t)
  service := usecase.NewAnalyticsService(mockRepo, mockGeoIP, mockDevice, mockReferer)

  ctx := context.Background()
  event := events.ClickEvent{
    ShortCode: "abc123",
    Timestamp: time.Unix(1700000000, 0),
    ClientIP:  "192.168.1.1",
    UserAgent: "Mozilla/5.0",
    Referer:   "https://google.com",
  }

  repoErr := errors.New("database error")

  // Mock expectations
  mockGeoIP.EXPECT().ResolveCountry("192.168.1.1").Return("US")
  mockDevice.EXPECT().DetectDevice("Mozilla/5.0").Return("desktop")
  mockReferer.EXPECT().ClassifySource("https://google.com").Return("search")
  mockRepo.EXPECT().InsertClick(ctx, "abc123", int64(1700000000), "US", "desktop", "search").Return(repoErr)

  // Act
  err := service.RecordEnrichedClick(ctx, event)

  // Assert
  require.Error(t, err)
  assert.Equal(t, repoErr, err)
}

// TestGetClickCount_ReturnsCount verifies click count retrieval
func TestGetClickCount_ReturnsCount(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()

  // Mock expectations
  mockRepo.EXPECT().CountByShortCode(ctx, "abc123").Return(int64(42), nil)

  // Act
  count, err := service.GetClickCount(ctx, "abc123")

  // Assert
  require.NoError(t, err)
  assert.Equal(t, int64(42), count)
}

// TestGetClickCount_NoClicks_ReturnsZero verifies zero clicks handling
func TestGetClickCount_NoClicks_ReturnsZero(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()

  // Mock expectations
  mockRepo.EXPECT().CountByShortCode(ctx, "xyz789").Return(int64(0), nil)

  // Act
  count, err := service.GetClickCount(ctx, "xyz789")

  // Assert
  require.NoError(t, err)
  assert.Equal(t, int64(0), count)
}

// TestGetAnalyticsSummary_ReturnsBreakdowns verifies summary with correct percentage calculation
func TestGetAnalyticsSummary_ReturnsBreakdowns(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  from := int64(1700000000)
  to := int64(1700086400)

  // Mock expectations
  mockRepo.EXPECT().CountInRange(ctx, "abc123", from, to).Return(int64(100), nil)
  mockRepo.EXPECT().CountByCountryInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{
    {Value: "US", Count: 42},
    {Value: "UK", Count: 30},
  }, nil)
  mockRepo.EXPECT().CountByDeviceInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{
    {Value: "desktop", Count: 60},
    {Value: "mobile", Count: 40},
  }, nil)
  mockRepo.EXPECT().CountBySourceInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{
    {Value: "search", Count: 50},
    {Value: "direct", Count: 50},
  }, nil)

  // Act
  result, err := service.GetAnalyticsSummary(ctx, "abc123", from, to)

  // Assert
  require.NoError(t, err)
  assert.NotNil(t, result)
  assert.Equal(t, "abc123", result.ShortCode)
  assert.Equal(t, int64(100), result.TotalClicks)

  // Verify breakdowns and percentages
  assert.Len(t, result.Countries, 2)
  assert.Equal(t, "US", result.Countries[0].Value)
  assert.Equal(t, int64(42), result.Countries[0].Count)
  assert.InDelta(t, 42.0, result.Countries[0].Percentage, 0.01)
  assert.Equal(t, "UK", result.Countries[1].Value)
  assert.Equal(t, int64(30), result.Countries[1].Count)
  assert.InDelta(t, 30.0, result.Countries[1].Percentage, 0.01)

  assert.Len(t, result.DeviceTypes, 2)
  assert.Equal(t, "desktop", result.DeviceTypes[0].Value)
  assert.InDelta(t, 60.0, result.DeviceTypes[0].Percentage, 0.01)

  assert.Len(t, result.TrafficSources, 2)
  assert.Equal(t, "search", result.TrafficSources[0].Value)
  assert.InDelta(t, 50.0, result.TrafficSources[0].Percentage, 0.01)
}

// TestGetAnalyticsSummary_ZeroClicks_ReturnsEmptyBreakdowns verifies empty breakdowns when total is zero
func TestGetAnalyticsSummary_ZeroClicks_ReturnsEmptyBreakdowns(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  from := int64(1700000000)
  to := int64(1700086400)

  // Mock expectations
  mockRepo.EXPECT().CountInRange(ctx, "abc123", from, to).Return(int64(0), nil)
  mockRepo.EXPECT().CountByCountryInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{}, nil)
  mockRepo.EXPECT().CountByDeviceInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{}, nil)
  mockRepo.EXPECT().CountBySourceInRange(ctx, "abc123", from, to).Return([]usecase.GroupCount{}, nil)

  // Act
  result, err := service.GetAnalyticsSummary(ctx, "abc123", from, to)

  // Assert
  require.NoError(t, err)
  assert.NotNil(t, result)
  assert.Equal(t, int64(0), result.TotalClicks)
  assert.Empty(t, result.Countries)
  assert.Empty(t, result.DeviceTypes)
  assert.Empty(t, result.TrafficSources)
}

// TestGetAnalyticsSummary_CountError_ReturnsError verifies error propagation from CountInRange
func TestGetAnalyticsSummary_CountError_ReturnsError(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  from := int64(1700000000)
  to := int64(1700086400)

  repoErr := errors.New("database connection lost")

  // Mock expectations
  mockRepo.EXPECT().CountInRange(ctx, "abc123", from, to).Return(int64(0), repoErr)

  // Act
  result, err := service.GetAnalyticsSummary(ctx, "abc123", from, to)

  // Assert
  require.Error(t, err)
  assert.Nil(t, result)
  assert.Equal(t, repoErr, err)
}

// TestGetClickDetails_ReturnsPaginated verifies paginated click details
func TestGetClickDetails_ReturnsPaginated(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  cursorTimestamp := int64(1700000000)
  limit := 20

  expectedClicks := &usecase.PaginatedClicks{
    Clicks: []usecase.ClickDetail{
      {ID: 1, ShortCode: "abc123", ClickedAt: 1700000100, CountryCode: "US", DeviceType: "desktop", TrafficSource: "search"},
      {ID: 2, ShortCode: "abc123", ClickedAt: 1700000200, CountryCode: "UK", DeviceType: "mobile", TrafficSource: "direct"},
    },
    NextCursor: "base64cursor",
    HasMore:    true,
  }

  // Mock expectations
  mockRepo.EXPECT().GetClickDetails(ctx, "abc123", cursorTimestamp, limit).Return(expectedClicks, nil)

  // Act
  result, err := service.GetClickDetails(ctx, "abc123", cursorTimestamp, limit)

  // Assert
  require.NoError(t, err)
  assert.Equal(t, expectedClicks, result)
  assert.Len(t, result.Clicks, 2)
  assert.True(t, result.HasMore)
}

// TestGetClickDetails_InvalidLimit_DefaultsTo20 verifies limit validation and defaulting
func TestGetClickDetails_InvalidLimit_DefaultsTo20(t *testing.T) {
  testCases := []struct {
    name  string
    limit int
  }{
    {name: "Zero limit", limit: 0},
    {name: "Negative limit", limit: -5},
    {name: "Over max limit", limit: 200},
  }

  for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
      // Setup
      mockRepo := mocks.NewMockClickRepository(t)
      service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

      ctx := context.Background()

      // Mock expectations - should be called with default limit 20
      mockRepo.EXPECT().GetClickDetails(ctx, "abc123", mock.AnythingOfType("int64"), 20).Return(&usecase.PaginatedClicks{
        Clicks:  []usecase.ClickDetail{},
        HasMore: false,
      }, nil)

      // Act
      _, err := service.GetClickDetails(ctx, "abc123", 1700000000, tc.limit)

      // Assert
      require.NoError(t, err)
    })
  }
}

// TestGetClickDetails_RepositoryError_ReturnsError verifies repository error propagation
func TestGetClickDetails_RepositoryError_ReturnsError(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  repoErr := errors.New("query timeout")

  // Mock expectations
  mockRepo.EXPECT().GetClickDetails(ctx, "abc123", int64(1700000000), 20).Return(nil, repoErr)

  // Act
  result, err := service.GetClickDetails(ctx, "abc123", 1700000000, 20)

  // Assert
  require.Error(t, err)
  assert.Nil(t, result)
  assert.Equal(t, repoErr, err)
}

// TestDeleteClickData_Success verifies delete operation
func TestDeleteClickData_Success(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()

  // Mock expectations
  mockRepo.EXPECT().DeleteByShortCode(ctx, "abc123").Return(nil)

  // Act
  err := service.DeleteClickData(ctx, "abc123")

  // Assert
  require.NoError(t, err)
}

// TestDeleteClickData_Error_ReturnsError verifies delete error propagation
func TestDeleteClickData_Error_ReturnsError(t *testing.T) {
  // Setup
  mockRepo := mocks.NewMockClickRepository(t)
  service := usecase.NewAnalyticsService(mockRepo, nil, nil, nil)

  ctx := context.Background()
  repoErr := errors.New("foreign key constraint")

  // Mock expectations
  mockRepo.EXPECT().DeleteByShortCode(ctx, "abc123").Return(repoErr)

  // Act
  err := service.DeleteClickData(ctx, "abc123")

  // Assert
  require.Error(t, err)
  assert.Equal(t, repoErr, err)
}

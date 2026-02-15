package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go-shortener/internal/urlservice/domain"
	"go-shortener/internal/urlservice/testutil/mocks"
	"go-shortener/internal/urlservice/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCreateShortURL_ValidURL_ReturnsNewShortURL tests successful creation of a new short URL
func TestCreateShortURL_ValidURL_ReturnsNewShortURL(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "https://example.com"

	// Mock expectations
	mockRepo.EXPECT().FindByOriginalURL(ctx, originalURL).Return(nil, domain.ErrURLNotFound)
	mockRepo.EXPECT().Save(ctx, mock.AnythingOfType("string"), originalURL).Return(&domain.URL{
		ID:          1,
		ShortCode:   "abc123XY",
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}, nil)

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, originalURL, result.OriginalURL)
	assert.NotEmpty(t, result.ShortCode)
}

// TestCreateShortURL_DuplicateURL_ReturnsExisting tests deduplication
func TestCreateShortURL_DuplicateURL_ReturnsExisting(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "https://example.com"
	existing := &domain.URL{
		ID:          1,
		ShortCode:   "existing1",
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().FindByOriginalURL(ctx, originalURL).Return(existing, nil)

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, existing, result)
	assert.Equal(t, "existing1", result.ShortCode)
}

// TestCreateShortURL_InvalidScheme_ReturnsError tests rejection of ftp:// URLs
func TestCreateShortURL_InvalidScheme_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "ftp://example.com"

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, domain.ErrInvalidURL))
	assert.Contains(t, err.Error(), "url scheme must be http or https")
}

// TestCreateShortURL_EmptyHost_ReturnsError tests rejection of URLs without host
func TestCreateShortURL_EmptyHost_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "https://"

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, domain.ErrInvalidURL))
	assert.Contains(t, err.Error(), "url must have a host")
}

// TestCreateShortURL_ExceedsMaxLength_ReturnsError tests rejection of URLs exceeding 2048 chars
func TestCreateShortURL_ExceedsMaxLength_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	// Create a URL longer than 2048 characters
	longPath := strings.Repeat("a", 2050)
	originalURL := "https://example.com/" + longPath

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, domain.ErrInvalidURL))
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

// TestCreateShortURL_CollisionRetry_SucceedsOnSecondAttempt tests retry logic for short code collisions
func TestCreateShortURL_CollisionRetry_SucceedsOnSecondAttempt(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "https://example.com"

	// Mock expectations: first attempt fails with collision, second succeeds
	mockRepo.EXPECT().FindByOriginalURL(ctx, originalURL).Return(nil, domain.ErrURLNotFound)
	mockRepo.EXPECT().Save(ctx, mock.AnythingOfType("string"), originalURL).
		Return(nil, errors.New("UNIQUE constraint failed: urls.short_code")).
		Once()
	mockRepo.EXPECT().Save(ctx, mock.AnythingOfType("string"), originalURL).
		Return(&domain.URL{
			ID:          1,
			ShortCode:   "retry123",
			OriginalURL: originalURL,
			CreatedAt:   time.Now(),
		}, nil).
		Once()

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "retry123", result.ShortCode)
}

// TestCreateShortURL_RepositoryError_ReturnsError tests handling of repository errors
func TestCreateShortURL_RepositoryError_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	originalURL := "https://example.com"
	repoErr := errors.New("database connection failed")

	// Mock expectations
	mockRepo.EXPECT().FindByOriginalURL(ctx, originalURL).Return(nil, domain.ErrURLNotFound)
	mockRepo.EXPECT().Save(ctx, mock.AnythingOfType("string"), originalURL).Return(nil, repoErr)

	// Act
	result, err := service.CreateShortURL(ctx, originalURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
}

// TestGetByShortCode_Exists_ReturnsURL tests successful retrieval
func TestGetByShortCode_Exists_ReturnsURL(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	shortCode := "abc123XY"
	expectedURL := &domain.URL{
		ID:          1,
		ShortCode:   shortCode,
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().FindByShortCode(ctx, shortCode).Return(expectedURL, nil)

	// Act
	result, err := service.GetByShortCode(ctx, shortCode)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedURL, result)
}

// TestGetByShortCode_NotFound_ReturnsError tests handling of not found
func TestGetByShortCode_NotFound_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	shortCode := "notfound"

	// Mock expectations
	mockRepo.EXPECT().FindByShortCode(ctx, shortCode).Return(nil, domain.ErrURLNotFound)

	// Act
	result, err := service.GetByShortCode(ctx, shortCode)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, domain.ErrURLNotFound))
}

// TestListLinks_ReturnsPagedResults tests pagination math
func TestListLinks_ReturnsPagedResults(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	params := usecase.ListLinksParams{
		Page:    2,
		PerPage: 10,
		Sort:    "created_at",
		Order:   "desc",
	}

	urls := []domain.URL{
		{ID: 1, ShortCode: "code1", OriginalURL: "https://example.com/1", CreatedAt: time.Now()},
		{ID: 2, ShortCode: "code2", OriginalURL: "https://example.com/2", CreatedAt: time.Now()},
	}

	// Mock expectations
	mockRepo.EXPECT().FindAll(ctx, usecase.FindAllParams{
		SortOrder: "desc",
		Limit:     10,
		Offset:    10, // (page 2 - 1) * 10
	}).Return(urls, nil)
	mockRepo.EXPECT().Count(ctx, usecase.CountParams{}).Return(int64(25), nil)

	// Act
	result, err := service.ListLinks(ctx, params)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Links))
	assert.Equal(t, int64(25), result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.PerPage)
	assert.Equal(t, 3, result.TotalPages) // ceil(25/10) = 3
}

// TestListLinks_InvalidSortField_ReturnsError tests rejection of unsupported sort fields
func TestListLinks_InvalidSortField_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	params := usecase.ListLinksParams{
		Page:    1,
		PerPage: 10,
		Sort:    "invalid_field",
		Order:   "desc",
	}

	// Act
	result, err := service.ListLinks(ctx, params)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported sort field")
}

// TestListLinks_DefaultParams_UsesDescOrder tests default ordering
func TestListLinks_DefaultParams_UsesDescOrder(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	params := usecase.ListLinksParams{
		Page:    1,
		PerPage: 10,
		// Sort and Order are empty - should default
	}

	urls := []domain.URL{}

	// Mock expectations - verify "desc" is used as default
	mockRepo.EXPECT().FindAll(ctx, usecase.FindAllParams{
		SortOrder: "desc", // Default
		Limit:     10,
		Offset:    0,
	}).Return(urls, nil)
	mockRepo.EXPECT().Count(ctx, usecase.CountParams{}).Return(int64(0), nil)

	// Act
	result, err := service.ListLinks(ctx, params)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestListLinks_WithClickCounts_EnrichesFromDapr tests click count enrichment via Dapr
func TestListLinks_WithClickCounts_EnrichesFromDapr(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	mockDapr := mocks.NewMockDaprClient(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, mockDapr, logger, "http://localhost:8080")

	ctx := context.Background()
	params := usecase.ListLinksParams{
		Page:    1,
		PerPage: 10,
		Sort:    "created_at",
		Order:   "desc",
	}

	urls := []domain.URL{
		{ID: 1, ShortCode: "code1", OriginalURL: "https://example.com/1", CreatedAt: time.Now()},
	}

	clickResponse, _ := json.Marshal(map[string]interface{}{
		"total_clicks": 42,
	})

	// Mock expectations
	mockRepo.EXPECT().FindAll(ctx, mock.Anything).Return(urls, nil)
	mockRepo.EXPECT().Count(ctx, mock.Anything).Return(int64(1), nil)
	mockDapr.EXPECT().InvokeMethod(mock.Anything, "analytics-service", "analytics/code1", "get").
		Return(clickResponse, nil)

	// Act
	result, err := service.ListLinks(ctx, params)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Links))
	assert.Equal(t, int64(42), result.Links[0].TotalClicks)
}

// TestListLinks_DaprUnavailable_FallsToZeroClicks tests graceful degradation when Dapr is nil
func TestListLinks_DaprUnavailable_FallsToZeroClicks(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080") // nil Dapr client

	ctx := context.Background()
	params := usecase.ListLinksParams{
		Page:    1,
		PerPage: 10,
		Sort:    "created_at",
		Order:   "desc",
	}

	urls := []domain.URL{
		{ID: 1, ShortCode: "code1", OriginalURL: "https://example.com/1", CreatedAt: time.Now()},
	}

	// Mock expectations
	mockRepo.EXPECT().FindAll(ctx, mock.Anything).Return(urls, nil)
	mockRepo.EXPECT().Count(ctx, mock.Anything).Return(int64(1), nil)

	// Act
	result, err := service.ListLinks(ctx, params)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Links))
	assert.Equal(t, int64(0), result.Links[0].TotalClicks) // Falls back to 0
}

// TestGetLinkDetail_ExistingLink_ReturnsWithClicks tests single link retrieval with click count
func TestGetLinkDetail_ExistingLink_ReturnsWithClicks(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	mockDapr := mocks.NewMockDaprClient(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, mockDapr, logger, "http://localhost:8080")

	ctx := context.Background()
	shortCode := "code1"
	urlData := &domain.URL{
		ID:          1,
		ShortCode:   shortCode,
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}

	clickResponse, _ := json.Marshal(map[string]interface{}{
		"total_clicks": 100,
	})

	// Mock expectations
	mockRepo.EXPECT().FindByShortCode(ctx, shortCode).Return(urlData, nil)
	mockDapr.EXPECT().InvokeMethod(mock.Anything, "analytics-service", "analytics/code1", "get").
		Return(clickResponse, nil)

	// Act
	result, err := service.GetLinkDetail(ctx, shortCode)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, shortCode, result.ShortCode)
	assert.Equal(t, int64(100), result.TotalClicks)
}

// TestGetLinkDetail_NotFound_ReturnsError tests handling of not found
func TestGetLinkDetail_NotFound_ReturnsError(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080")

	ctx := context.Background()
	shortCode := "notfound"

	// Mock expectations
	mockRepo.EXPECT().FindByShortCode(ctx, shortCode).Return(nil, domain.ErrURLNotFound)

	// Act
	result, err := service.GetLinkDetail(ctx, shortCode)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, domain.ErrURLNotFound))
}

// TestDeleteLink_Success_DeletesAndPublishesEvent tests successful deletion with event publishing
func TestDeleteLink_Success_DeletesAndPublishesEvent(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	mockDapr := mocks.NewMockDaprClient(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, mockDapr, logger, "http://localhost:8080")

	ctx := context.Background()
	shortCode := "code1"

	// Mock expectations
	mockRepo.EXPECT().Delete(ctx, shortCode).Return(nil)
	mockDapr.EXPECT().PublishEvent(mock.Anything, "pubsub", "link-deleted", mock.Anything).
		Return(nil)

	// Act
	err := service.DeleteLink(ctx, shortCode)

	// Assert
	require.NoError(t, err)

	// Give goroutine time to execute
	time.Sleep(10 * time.Millisecond)
}

// TestDeleteLink_NoDapr_DeletesWithoutEvent tests deletion when Dapr client is nil
func TestDeleteLink_NoDapr_DeletesWithoutEvent(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockURLRepository(t)
	logger := zap.NewNop()
	service := usecase.NewURLService(mockRepo, nil, logger, "http://localhost:8080") // nil Dapr

	ctx := context.Background()
	shortCode := "code1"

	// Mock expectations
	mockRepo.EXPECT().Delete(ctx, shortCode).Return(nil)

	// Act
	err := service.DeleteLink(ctx, shortCode)

	// Assert
	require.NoError(t, err)
	// No panic, no event published
}

package links

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-shortener/services/analytics-rpc/analyticsclient"
	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockAnalyticsClient is a test mock for Analytics interface
type MockAnalyticsClient struct {
	GetClickCountFunc func(ctx context.Context, in *analyticsclient.GetClickCountRequest, opts ...grpc.CallOption) (*analyticsclient.GetClickCountResponse, error)
}

func (m *MockAnalyticsClient) GetClickCount(ctx context.Context, in *analyticsclient.GetClickCountRequest, opts ...grpc.CallOption) (*analyticsclient.GetClickCountResponse, error) {
	if m.GetClickCountFunc != nil {
		return m.GetClickCountFunc(ctx, in, opts...)
	}
	panic("MockAnalyticsClient.GetClickCountFunc not set")
}

func TestGetLinkDetailLogic_Success(t *testing.T) {
	createdAt := time.Now().Add(-24 * time.Hour)

	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			assert.Equal(t, "abc12345", shortCode)
			return &model.Urls{
				Id:          "test-id",
				ShortCode:   "abc12345",
				OriginalUrl: "https://example.com",
				ClickCount:  0,
				CreatedAt:   createdAt,
			}, nil
		},
	}

	mockAnalytics := &MockAnalyticsClient{
		GetClickCountFunc: func(ctx context.Context, in *analyticsclient.GetClickCountRequest, opts ...grpc.CallOption) (*analyticsclient.GetClickCountResponse, error) {
			assert.Equal(t, "abc12345", in.ShortCode)
			return &analyticsclient.GetClickCountResponse{
				ShortCode:   "abc12345",
				TotalClicks: 42,
			}, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:       config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel:     mockModel,
		AnalyticsRpc: mockAnalytics,
	}

	logic := NewGetLinkDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetLinkDetail(&types.LinkDetailRequest{Code: "abc12345"})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "abc12345", resp.ShortCode)
	assert.Equal(t, "https://example.com", resp.OriginalUrl)
	assert.Equal(t, createdAt.Unix(), resp.CreatedAt)
	assert.Equal(t, int64(42), resp.TotalClicks)
}

func TestGetLinkDetailLogic_NotFound(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, model.ErrNotFound
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewGetLinkDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetLinkDetail(&types.LinkDetailRequest{Code: "notfound"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetLinkDetailLogic_RPCFailure(t *testing.T) {
	createdAt := time.Now().Add(-24 * time.Hour)

	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return &model.Urls{
				Id:          "test-id",
				ShortCode:   "abc12345",
				OriginalUrl: "https://example.com",
				ClickCount:  0,
				CreatedAt:   createdAt,
			}, nil
		},
	}

	mockAnalytics := &MockAnalyticsClient{
		GetClickCountFunc: func(ctx context.Context, in *analyticsclient.GetClickCountRequest, opts ...grpc.CallOption) (*analyticsclient.GetClickCountResponse, error) {
			return nil, errors.New("analytics service unavailable")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:       config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel:     mockModel,
		AnalyticsRpc: mockAnalytics,
	}

	logic := NewGetLinkDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetLinkDetail(&types.LinkDetailRequest{Code: "abc12345"})

	// Should succeed with graceful degradation - returns 0 clicks
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "abc12345", resp.ShortCode)
	assert.Equal(t, "https://example.com", resp.OriginalUrl)
	assert.Equal(t, int64(0), resp.TotalClicks, "should return 0 clicks on RPC failure")
}

func TestGetLinkDetailLogic_DBError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, errors.New("database connection error")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewGetLinkDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetLinkDetail(&types.LinkDetailRequest{Code: "abc12345"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Internal Error")
}

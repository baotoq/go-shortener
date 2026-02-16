package redirect

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectLogic_Success(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			assert.Equal(t, "abc12345", shortCode)
			return &model.Urls{
				Id:          "test-id",
				ShortCode:   "abc12345",
				OriginalUrl: "https://example.com",
				ClickCount:  10,
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
		KqPusher: nil, // nil KqPusher is fine - GoSafe handles nil gracefully
	}

	req := httptest.NewRequest("GET", "/abc12345", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	logic := NewRedirectLogic(context.Background(), svcCtx)
	originalUrl, err := logic.Redirect(&types.RedirectRequest{Code: "abc12345"}, req)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", originalUrl)
}

func TestRedirectLogic_NotFound(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, model.ErrNotFound
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
		KqPusher: nil,
	}

	req := httptest.NewRequest("GET", "/notfound", nil)
	logic := NewRedirectLogic(context.Background(), svcCtx)
	originalUrl, err := logic.Redirect(&types.RedirectRequest{Code: "notfound"}, req)

	require.Error(t, err)
	assert.Empty(t, originalUrl)
	assert.Contains(t, err.Error(), "not found")
}

func TestRedirectLogic_DBError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, errors.New("database connection timeout")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
		KqPusher: nil,
	}

	req := httptest.NewRequest("GET", "/abc12345", nil)
	logic := NewRedirectLogic(context.Background(), svcCtx)
	originalUrl, err := logic.Redirect(&types.RedirectRequest{Code: "abc12345"}, req)

	require.Error(t, err)
	assert.Empty(t, originalUrl)
	assert.Contains(t, err.Error(), "Internal Error")
}

func TestExtractClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	ip := extractClientIP(req)
	assert.Equal(t, "1.2.3.4", ip)
}

func TestExtractClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "9.8.7.6")

	ip := extractClientIP(req)
	assert.Equal(t, "9.8.7.6", ip)
}

func TestExtractClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	ip := extractClientIP(req)
	assert.Equal(t, "10.0.0.1", ip)
}

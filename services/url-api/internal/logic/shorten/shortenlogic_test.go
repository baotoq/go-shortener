package shorten

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenLogic_Success(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		InsertFunc: func(ctx context.Context, data *model.Urls) (sql.Result, error) {
			// Verify data structure
			assert.NotEmpty(t, data.Id)
			assert.NotEmpty(t, data.ShortCode)
			assert.Equal(t, "https://example.com", data.OriginalUrl)
			assert.Equal(t, int64(0), data.ClickCount)
			return nil, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewShortenLogic(context.Background(), svcCtx)
	resp, err := logic.Shorten(&types.ShortenRequest{
		OriginalUrl: "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.ShortCode)
	assert.Equal(t, "http://localhost:8080/"+resp.ShortCode, resp.ShortUrl)
	assert.Equal(t, "https://example.com", resp.OriginalUrl)
	assert.Len(t, resp.ShortCode, 8)
}

func TestShortenLogic_InsertError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		InsertFunc: func(ctx context.Context, data *model.Urls) (sql.Result, error) {
			return nil, errors.New("database connection error")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewShortenLogic(context.Background(), svcCtx)
	resp, err := logic.Shorten(&types.ShortenRequest{
		OriginalUrl: "https://example.com",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Internal Error")
}

func TestShortenLogic_CollisionRetry(t *testing.T) {
	attemptCount := 0
	mockModel := &model.MockUrlsModel{
		InsertFunc: func(ctx context.Context, data *model.Urls) (sql.Result, error) {
			attemptCount++
			// First attempt: simulate unique constraint violation
			if attemptCount == 1 {
				return nil, errors.New("duplicate key value violates unique constraint")
			}
			// Second attempt: success
			return nil, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewShortenLogic(context.Background(), svcCtx)
	resp, err := logic.Shorten(&types.ShortenRequest{
		OriginalUrl: "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, attemptCount, "should have retried once")
	assert.NotEmpty(t, resp.ShortCode)
}

func TestShortenLogic_MaxRetriesExceeded(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		InsertFunc: func(ctx context.Context, data *model.Urls) (sql.Result, error) {
			// Always return unique constraint violation
			return nil, errors.New("duplicate key value violates unique constraint")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewShortenLogic(context.Background(), svcCtx)
	resp, err := logic.Shorten(&types.ShortenRequest{
		OriginalUrl: "https://example.com",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "maximum retries")
}

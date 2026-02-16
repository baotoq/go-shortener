package logic

import (
	"context"
	"errors"
	"testing"

	"go-shortener/services/analytics-rpc/analytics"
	"go-shortener/services/analytics-rpc/internal/config"
	"go-shortener/services/analytics-rpc/internal/svc"
	"go-shortener/services/analytics-rpc/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClickCountLogic_Success(t *testing.T) {
	mockModel := &model.MockClicksModel{
		CountByShortCodeFunc: func(ctx context.Context, shortCode string) (int64, error) {
			assert.Equal(t, "abc12345", shortCode)
			return 42, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
	}

	logic := NewGetClickCountLogic(context.Background(), svcCtx)
	resp, err := logic.GetClickCount(&analytics.GetClickCountRequest{
		ShortCode: "abc12345",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "abc12345", resp.ShortCode)
	assert.Equal(t, int64(42), resp.TotalClicks)
}

func TestGetClickCountLogic_Zero(t *testing.T) {
	mockModel := &model.MockClicksModel{
		CountByShortCodeFunc: func(ctx context.Context, shortCode string) (int64, error) {
			assert.Equal(t, "unknown", shortCode)
			return 0, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
	}

	logic := NewGetClickCountLogic(context.Background(), svcCtx)
	resp, err := logic.GetClickCount(&analytics.GetClickCountRequest{
		ShortCode: "unknown",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "unknown", resp.ShortCode)
	assert.Equal(t, int64(0), resp.TotalClicks, "zero clicks should not be an error")
}

func TestGetClickCountLogic_DBError(t *testing.T) {
	mockModel := &model.MockClicksModel{
		CountByShortCodeFunc: func(ctx context.Context, shortCode string) (int64, error) {
			return 0, errors.New("database connection timeout")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
	}

	logic := NewGetClickCountLogic(context.Background(), svcCtx)
	resp, err := logic.GetClickCount(&analytics.GetClickCountRequest{
		ShortCode: "abc12345",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database connection timeout")
}

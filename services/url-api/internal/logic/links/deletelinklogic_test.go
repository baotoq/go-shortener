package links

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-shortener/services/url-api/internal/config"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteLinkLogic_Success(t *testing.T) {
	deletedID := ""

	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			assert.Equal(t, "abc12345", shortCode)
			return &model.Urls{
				Id:          "test-uuid-123",
				ShortCode:   "abc12345",
				OriginalUrl: "https://example.com",
				ClickCount:  10,
				CreatedAt:   time.Now(),
			}, nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewDeleteLinkLogic(context.Background(), svcCtx)
	err := logic.DeleteLink(&types.DeleteLinkRequest{Code: "abc12345"})

	require.NoError(t, err)
	assert.Equal(t, "test-uuid-123", deletedID, "should delete by UUID primary key")
}

func TestDeleteLinkLogic_NotFound(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, model.ErrNotFound
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewDeleteLinkLogic(context.Background(), svcCtx)
	err := logic.DeleteLink(&types.DeleteLinkRequest{Code: "notfound"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteLinkLogic_FindError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return nil, errors.New("database connection error")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewDeleteLinkLogic(context.Background(), svcCtx)
	err := logic.DeleteLink(&types.DeleteLinkRequest{Code: "abc12345"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Internal Error")
}

func TestDeleteLinkLogic_DeleteError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		FindOneByShortCodeFunc: func(ctx context.Context, shortCode string) (*model.Urls, error) {
			return &model.Urls{
				Id:          "test-uuid-123",
				ShortCode:   "abc12345",
				OriginalUrl: "https://example.com",
				ClickCount:  10,
				CreatedAt:   time.Now(),
			}, nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return errors.New("foreign key constraint violation")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewDeleteLinkLogic(context.Background(), svcCtx)
	err := logic.DeleteLink(&types.DeleteLinkRequest{Code: "abc12345"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Internal Error")
}

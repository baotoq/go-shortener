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

func TestListLinksLogic_Success(t *testing.T) {
	createdAt1 := time.Now().Add(-48 * time.Hour)
	createdAt2 := time.Now().Add(-24 * time.Hour)

	mockModel := &model.MockUrlsModel{
		ListWithPaginationFunc: func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*model.Urls, int64, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 10, pageSize)
			assert.Equal(t, "", search)
			assert.Equal(t, "created_at", sort)
			assert.Equal(t, "desc", order)

			return []*model.Urls{
				{
					Id:          "id-1",
					ShortCode:   "abc12345",
					OriginalUrl: "https://example.com",
					ClickCount:  10,
					CreatedAt:   createdAt1,
				},
				{
					Id:          "id-2",
					ShortCode:   "def67890",
					OriginalUrl: "https://another.com",
					ClickCount:  5,
					CreatedAt:   createdAt2,
				},
			}, 2, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewListLinksLogic(context.Background(), svcCtx)
	resp, err := logic.ListLinks(&types.LinkListRequest{
		Page:    1,
		PerPage: 10,
		Search:  "",
		Sort:    "created_at",
		Order:   "desc",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Links))
	assert.Equal(t, "abc12345", resp.Links[0].ShortCode)
	assert.Equal(t, "https://example.com", resp.Links[0].OriginalUrl)
	assert.Equal(t, createdAt1.Unix(), resp.Links[0].CreatedAt)
	assert.Equal(t, "def67890", resp.Links[1].ShortCode)
	assert.Equal(t, int64(2), resp.TotalCount)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 10, resp.PerPage)
}

func TestListLinksLogic_Empty(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		ListWithPaginationFunc: func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*model.Urls, int64, error) {
			return []*model.Urls{}, 0, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewListLinksLogic(context.Background(), svcCtx)
	resp, err := logic.ListLinks(&types.LinkListRequest{
		Page:    1,
		PerPage: 10,
		Search:  "",
		Sort:    "created_at",
		Order:   "desc",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0, len(resp.Links))
	assert.Equal(t, int64(0), resp.TotalCount)
	assert.Equal(t, 0, resp.TotalPages)
}

func TestListLinksLogic_WithSearch(t *testing.T) {
	createdAt := time.Now()

	mockModel := &model.MockUrlsModel{
		ListWithPaginationFunc: func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*model.Urls, int64, error) {
			assert.Equal(t, "example", search)
			return []*model.Urls{
				{
					Id:          "id-1",
					ShortCode:   "abc12345",
					OriginalUrl: "https://example.com",
					ClickCount:  10,
					CreatedAt:   createdAt,
				},
			}, 1, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewListLinksLogic(context.Background(), svcCtx)
	resp, err := logic.ListLinks(&types.LinkListRequest{
		Page:    1,
		PerPage: 10,
		Search:  "example",
		Sort:    "created_at",
		Order:   "desc",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Links))
	assert.Equal(t, int64(1), resp.TotalCount)
}

func TestListLinksLogic_DBError(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		ListWithPaginationFunc: func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*model.Urls, int64, error) {
			return nil, 0, errors.New("database connection error")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewListLinksLogic(context.Background(), svcCtx)
	resp, err := logic.ListLinks(&types.LinkListRequest{
		Page:    1,
		PerPage: 10,
		Search:  "",
		Sort:    "created_at",
		Order:   "desc",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Internal Error")
}

func TestListLinksLogic_Pagination(t *testing.T) {
	mockModel := &model.MockUrlsModel{
		ListWithPaginationFunc: func(ctx context.Context, page, pageSize int, search, sort, order string) ([]*model.Urls, int64, error) {
			// Simulate 25 total items with page size 10
			return []*model.Urls{}, 25, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:   config.Config{BaseUrl: "http://localhost:8080"},
		UrlModel: mockModel,
	}

	logic := NewListLinksLogic(context.Background(), svcCtx)
	resp, err := logic.ListLinks(&types.LinkListRequest{
		Page:    2,
		PerPage: 10,
		Search:  "",
		Sort:    "created_at",
		Order:   "desc",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(25), resp.TotalCount)
	assert.Equal(t, 3, resp.TotalPages, "25 items / 10 per page = 3 pages")
	assert.Equal(t, 2, resp.Page)
}

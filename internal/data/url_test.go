package data

import (
	"context"
	"testing"
	"time"

	"go-shortener/ent/enttest"
	"go-shortener/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestRepo(t *testing.T) (*urlRepo, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	data := &Data{
		db:  client,
		rdb: nil,
	}

	repo := &urlRepo{
		data: data,
		log:  log.NewHelper(log.DefaultLogger),
	}

	cleanup := func() {
		client.Close()
	}

	return repo, cleanup
}

func TestURLRepo_Create(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	u := &biz.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
	}

	created, err := repo.Create(ctx, u)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, u.ShortCode, created.ShortCode)
	assert.Equal(t, u.OriginalURL, created.OriginalURL)
	assert.Zero(t, created.ClickCount)
}

func TestURLRepo_Create_WithExpiry(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	expiresAt := time.Now().Add(24 * time.Hour)

	u := &biz.URL{
		ShortCode:   "expiry",
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
	}

	created, err := repo.Create(ctx, u)
	require.NoError(t, err)
	assert.NotNil(t, created.ExpiresAt)
}

func TestURLRepo_GetByShortCode(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	u := &biz.URL{
		ShortCode:   "gettest",
		OriginalURL: "https://example.com",
	}
	_, err := repo.Create(ctx, u)
	require.NoError(t, err)

	found, err := repo.GetByShortCode(ctx, "gettest")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "gettest", found.ShortCode)
}

func TestURLRepo_GetByShortCode_NotFound(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	found, err := repo.GetByShortCode(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestURLRepo_IncrementClickCount(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	u := &biz.URL{
		ShortCode:   "clicks",
		OriginalURL: "https://example.com",
	}
	_, err := repo.Create(ctx, u)
	require.NoError(t, err)

	err = repo.IncrementClickCount(ctx, "clicks")
	require.NoError(t, err)

	found, err := repo.GetByShortCode(ctx, "clicks")
	require.NoError(t, err)
	assert.Equal(t, int64(1), found.ClickCount)

	_ = repo.IncrementClickCount(ctx, "clicks")
	_ = repo.IncrementClickCount(ctx, "clicks")

	found, err = repo.GetByShortCode(ctx, "clicks")
	require.NoError(t, err)
	assert.Equal(t, int64(3), found.ClickCount)
}

func TestURLRepo_Delete(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	u := &biz.URL{
		ShortCode:   "todelete",
		OriginalURL: "https://example.com",
	}
	_, err := repo.Create(ctx, u)
	require.NoError(t, err)

	err = repo.Delete(ctx, "todelete")
	require.NoError(t, err)

	found, err := repo.GetByShortCode(ctx, "todelete")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestURLRepo_List(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &biz.URL{
			ShortCode:   "list" + string(rune('a'+i)),
			OriginalURL: "https://example.com",
		}
		_, err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	urls, total, err := repo.List(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, urls, 5)
	assert.Equal(t, 5, total)
}

func TestURLRepo_List_Pagination(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		u := &biz.URL{
			ShortCode:   "page" + string(rune('a'+i)),
			OriginalURL: "https://example.com",
		}
		_, err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	urls, total, err := repo.List(ctx, 1, 3)
	require.NoError(t, err)
	assert.Len(t, urls, 3)
	assert.Equal(t, 10, total)

	urls2, _, err := repo.List(ctx, 2, 3)
	require.NoError(t, err)
	assert.Len(t, urls2, 3)
}

func TestURLRepo_ExistsShortCode(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	u := &biz.URL{
		ShortCode:   "exists",
		OriginalURL: "https://example.com",
	}
	_, err := repo.Create(ctx, u)
	require.NoError(t, err)

	exists, err := repo.ExistsShortCode(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsShortCode(ctx, "notexists")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestURLRepo_CacheKey(t *testing.T) {
	repo := &urlRepo{}
	key := repo.cacheKey("test123")
	assert.Equal(t, "url:test123", key)
}

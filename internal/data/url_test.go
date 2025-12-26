package data

import (
	"context"
	"testing"
	"time"

	"go-shortener/ent/enttest"
	"go-shortener/internal/domain"

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

func TestURLRepo_Save(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("test123")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)

	err := repo.Save(ctx, u)
	require.NoError(t, err)
	assert.NotZero(t, u.ID())
	assert.Equal(t, "test123", u.ShortCode().String())
	assert.Equal(t, "https://example.com", u.OriginalURL().String())
	assert.Zero(t, u.ClickCount())
}

func TestURLRepo_Save_WithExpiry(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	expiresAt := time.Now().Add(24 * time.Hour)

	sc, _ := domain.NewShortCode("expiry")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, &expiresAt)

	err := repo.Save(ctx, u)
	require.NoError(t, err)
	assert.NotNil(t, u.ExpiresAt())
}

func TestURLRepo_FindByShortCode(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("gettest")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := repo.Save(ctx, u)
	require.NoError(t, err)

	found, err := repo.FindByShortCode(ctx, sc)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "gettest", found.ShortCode().String())
}

func TestURLRepo_FindByShortCode_NotFound(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("nonexistent")
	found, err := repo.FindByShortCode(ctx, sc)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestURLRepo_IncrementClickCount(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("clicks")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := repo.Save(ctx, u)
	require.NoError(t, err)

	err = repo.IncrementClickCount(ctx, sc)
	require.NoError(t, err)

	found, err := repo.FindByShortCode(ctx, sc)
	require.NoError(t, err)
	assert.Equal(t, int64(1), found.ClickCount())

	_ = repo.IncrementClickCount(ctx, sc)
	_ = repo.IncrementClickCount(ctx, sc)

	found, err = repo.FindByShortCode(ctx, sc)
	require.NoError(t, err)
	assert.Equal(t, int64(3), found.ClickCount())
}

func TestURLRepo_Delete(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("todelete")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := repo.Save(ctx, u)
	require.NoError(t, err)

	err = repo.Delete(ctx, sc)
	require.NoError(t, err)

	found, err := repo.FindByShortCode(ctx, sc)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestURLRepo_FindAll(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		sc, _ := domain.NewShortCode("list" + string(rune('a'+i)))
		ou, _ := domain.NewOriginalURL("https://example.com")
		u := domain.NewURL(sc, ou, nil)
		err := repo.Save(ctx, u)
		require.NoError(t, err)
	}

	urls, total, err := repo.FindAll(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, urls, 5)
	assert.Equal(t, 5, total)
}

func TestURLRepo_FindAll_Pagination(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		sc, _ := domain.NewShortCode("page" + string(rune('a'+i)))
		ou, _ := domain.NewOriginalURL("https://example.com")
		u := domain.NewURL(sc, ou, nil)
		err := repo.Save(ctx, u)
		require.NoError(t, err)
	}

	urls, total, err := repo.FindAll(ctx, 1, 3)
	require.NoError(t, err)
	assert.Len(t, urls, 3)
	assert.Equal(t, 10, total)

	urls2, _, err := repo.FindAll(ctx, 2, 3)
	require.NoError(t, err)
	assert.Len(t, urls2, 3)
}

func TestURLRepo_Exists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	sc, _ := domain.NewShortCode("exists")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := repo.Save(ctx, u)
	require.NoError(t, err)

	exists, err := repo.Exists(ctx, sc)
	require.NoError(t, err)
	assert.True(t, exists)

	scNotExists, _ := domain.NewShortCode("notexists")
	exists, err = repo.Exists(ctx, scNotExists)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestURLRepo_CacheKey(t *testing.T) {
	repo := &urlRepo{}
	key := repo.cacheKey("test123")
	assert.Equal(t, "url:test123", key)
}

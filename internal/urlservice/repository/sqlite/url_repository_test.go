package sqlite

import (
  "context"
  "database/sql"
  "strings"
  "testing"
  "time"

  "go-shortener/internal/urlservice/database"
  "go-shortener/internal/urlservice/domain"
  "go-shortener/internal/urlservice/usecase"

  _ "modernc.org/sqlite"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
  db, err := sql.Open("sqlite", ":memory:")
  require.NoError(t, err)
  t.Cleanup(func() { db.Close() })

  // Run migrations
  err = database.RunMigrations(db)
  require.NoError(t, err)

  return db
}

func TestURLRepository_Save_CreatesRecord(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  url, err := repo.Save(context.Background(), "test123", "https://example.com")

  require.NoError(t, err)
  assert.Equal(t, "test123", url.ShortCode)
  assert.Equal(t, "https://example.com", url.OriginalURL)
  assert.NotZero(t, url.ID)
  assert.False(t, url.CreatedAt.IsZero())
}

func TestURLRepository_FindByShortCode_Exists(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save first
  saved, err := repo.Save(context.Background(), "abc123", "https://example.com")
  require.NoError(t, err)

  // Find by short code
  found, err := repo.FindByShortCode(context.Background(), "abc123")

  require.NoError(t, err)
  assert.Equal(t, saved.ID, found.ID)
  assert.Equal(t, saved.ShortCode, found.ShortCode)
  assert.Equal(t, saved.OriginalURL, found.OriginalURL)
}

func TestURLRepository_FindByShortCode_NotFound(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  found, err := repo.FindByShortCode(context.Background(), "nonexistent")

  assert.ErrorIs(t, err, domain.ErrURLNotFound)
  assert.Nil(t, found)
}

func TestURLRepository_FindByOriginalURL_Exists(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save first
  saved, err := repo.Save(context.Background(), "xyz789", "https://duplicate-test.com")
  require.NoError(t, err)

  // Find by original URL
  found, err := repo.FindByOriginalURL(context.Background(), "https://duplicate-test.com")

  require.NoError(t, err)
  assert.Equal(t, saved.ID, found.ID)
  assert.Equal(t, saved.ShortCode, found.ShortCode)
  assert.Equal(t, saved.OriginalURL, found.OriginalURL)
}

func TestURLRepository_FindByOriginalURL_NotFound(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  found, err := repo.FindByOriginalURL(context.Background(), "https://not-found.com")

  assert.ErrorIs(t, err, domain.ErrURLNotFound)
  assert.Nil(t, found)
}

func TestURLRepository_Save_DuplicateShortCode_ReturnsError(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save first URL
  _, err := repo.Save(context.Background(), "duplicate", "https://first.com")
  require.NoError(t, err)

  // Try to save with same short code
  _, err = repo.Save(context.Background(), "duplicate", "https://second.com")

  assert.Error(t, err)
  assert.True(t, strings.Contains(err.Error(), "UNIQUE"))
}

func TestURLRepository_FindAll_Pagination(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save 5 URLs
  for i := 1; i <= 5; i++ {
    _, err := repo.Save(context.Background(), string(rune('a'+i-1))+string(rune('0'+i)), "https://example.com/"+string(rune('0'+i)))
    require.NoError(t, err)
    time.Sleep(time.Millisecond) // Ensure different timestamps
  }

  // Query page 1 with limit 2
  urls, err := repo.FindAll(context.Background(), usecase.FindAllParams{
    Limit:     2,
    Offset:    0,
    SortOrder: "desc",
  })

  require.NoError(t, err)
  assert.Len(t, urls, 2)
}

func TestURLRepository_FindAll_SortOrder(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save URLs
  _, err := repo.Save(context.Background(), "first", "https://example.com/1")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "second", "https://example.com/2")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "third", "https://example.com/3")
  require.NoError(t, err)

  // Test descending order
  urlsDesc, err := repo.FindAll(context.Background(), usecase.FindAllParams{
    Limit:     10,
    Offset:    0,
    SortOrder: "desc",
  })
  require.NoError(t, err)
  require.Len(t, urlsDesc, 3)

  // Test ascending order
  urlsAsc, err := repo.FindAll(context.Background(), usecase.FindAllParams{
    Limit:     10,
    Offset:    0,
    SortOrder: "asc",
  })
  require.NoError(t, err)
  require.Len(t, urlsAsc, 3)

  // Verify that asc and desc return different orderings (at least first elements differ)
  assert.NotEqual(t, urlsDesc[0].ID, urlsAsc[0].ID, "asc and desc should have different order")
}

func TestURLRepository_FindAll_SearchFilter(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save URLs with different original URLs
  _, err := repo.Save(context.Background(), "code1", "https://google.com")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "code2", "https://example.com")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "code3", "https://google.co.uk")
  require.NoError(t, err)

  // Search for "google"
  urls, err := repo.FindAll(context.Background(), usecase.FindAllParams{
    Limit:     10,
    Offset:    0,
    SortOrder: "desc",
    Search:    "google",
  })

  require.NoError(t, err)
  assert.Len(t, urls, 2)
  for _, url := range urls {
    assert.True(t, strings.Contains(url.OriginalURL, "google"))
  }
}

func TestURLRepository_Count_WithFilters(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save different URLs
  _, err := repo.Save(context.Background(), "old", "https://old.com")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "new1", "https://example.com/new1")
  require.NoError(t, err)

  _, err = repo.Save(context.Background(), "new2", "https://example.com/new2")
  require.NoError(t, err)

  // Count all
  count, err := repo.Count(context.Background(), usecase.CountParams{})
  require.NoError(t, err)
  assert.Equal(t, int64(3), count)

  // Count with search filter
  count, err = repo.Count(context.Background(), usecase.CountParams{
    Search: "example.com",
  })

  require.NoError(t, err)
  assert.Equal(t, int64(2), count)
}

func TestURLRepository_Delete_RemovesRecord(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Save URL
  _, err := repo.Save(context.Background(), "delete-me", "https://example.com")
  require.NoError(t, err)

  // Delete it
  err = repo.Delete(context.Background(), "delete-me")
  require.NoError(t, err)

  // Try to find it
  found, err := repo.FindByShortCode(context.Background(), "delete-me")
  assert.ErrorIs(t, err, domain.ErrURLNotFound)
  assert.Nil(t, found)
}

func TestURLRepository_Delete_Idempotent(t *testing.T) {
  db := setupTestDB(t)
  repo := NewURLRepository(db)

  // Delete non-existent code should succeed
  err := repo.Delete(context.Background(), "nonexistent")
  assert.NoError(t, err)
}

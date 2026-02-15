package sqlite

import (
  "context"
  "database/sql"
  "testing"
  "time"

  "go-shortener/internal/analytics/database"

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

func TestClickRepository_InsertClick_StoresRecord(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()
  err := repo.InsertClick(context.Background(), "abc123", now, "US", "desktop", "direct")

  require.NoError(t, err)

  // Verify via count
  count, err := repo.CountByShortCode(context.Background(), "abc123")
  require.NoError(t, err)
  assert.Equal(t, int64(1), count)
}

func TestClickRepository_CountByShortCode_ReturnsCount(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert 3 clicks
  err := repo.InsertClick(context.Background(), "test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "test", now+1, "UK", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "test", now+2, "CA", "tablet", "social")
  require.NoError(t, err)

  count, err := repo.CountByShortCode(context.Background(), "test")

  require.NoError(t, err)
  assert.Equal(t, int64(3), count)
}

func TestClickRepository_CountByShortCode_NoClicks_ReturnsZero(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  count, err := repo.CountByShortCode(context.Background(), "nonexistent")

  require.NoError(t, err)
  assert.Equal(t, int64(0), count)
}

func TestClickRepository_CountInRange_FiltersCorrectly(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks at different times
  err := repo.InsertClick(context.Background(), "range-test", now-100, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "range-test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "range-test", now+50, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "range-test", now+200, "US", "desktop", "direct")
  require.NoError(t, err)

  // Count clicks in range [now, now+100]
  count, err := repo.CountInRange(context.Background(), "range-test", now, now+100)

  require.NoError(t, err)
  assert.Equal(t, int64(2), count) // Should include clicks at now and now+50
}

func TestClickRepository_CountByCountryInRange_GroupsCorrectly(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks with different countries
  err := repo.InsertClick(context.Background(), "country-test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "country-test", now+1, "US", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "country-test", now+2, "UK", "desktop", "social")
  require.NoError(t, err)

  groups, err := repo.CountByCountryInRange(context.Background(), "country-test", now, now+100)

  require.NoError(t, err)
  assert.Len(t, groups, 2)

  // Find US count
  var usCount int64
  for _, g := range groups {
    if g.Value == "US" {
      usCount = g.Count
    }
  }
  assert.Equal(t, int64(2), usCount)
}

func TestClickRepository_CountByDeviceInRange_GroupsCorrectly(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks with different device types
  err := repo.InsertClick(context.Background(), "device-test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "device-test", now+1, "UK", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "device-test", now+2, "CA", "mobile", "social")
  require.NoError(t, err)

  groups, err := repo.CountByDeviceInRange(context.Background(), "device-test", now, now+100)

  require.NoError(t, err)
  assert.Len(t, groups, 2)

  // Find mobile count
  var mobileCount int64
  for _, g := range groups {
    if g.Value == "mobile" {
      mobileCount = g.Count
    }
  }
  assert.Equal(t, int64(2), mobileCount)
}

func TestClickRepository_CountBySourceInRange_GroupsCorrectly(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks with different traffic sources
  err := repo.InsertClick(context.Background(), "source-test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "source-test", now+1, "UK", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "source-test", now+2, "CA", "tablet", "search")
  require.NoError(t, err)

  groups, err := repo.CountBySourceInRange(context.Background(), "source-test", now, now+100)

  require.NoError(t, err)
  assert.Len(t, groups, 2)

  // Find search count
  var searchCount int64
  for _, g := range groups {
    if g.Value == "search" {
      searchCount = g.Count
    }
  }
  assert.Equal(t, int64(2), searchCount)
}

func TestClickRepository_GetClickDetails_ReturnsPaginated(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert 5 clicks
  for i := 0; i < 5; i++ {
    err := repo.InsertClick(context.Background(), "page-test", now+int64(i), "US", "desktop", "direct")
    require.NoError(t, err)
  }

  // Request limit 3 (should fetch 4 internally to detect hasMore)
  result, err := repo.GetClickDetails(context.Background(), "page-test", now+100, 3)

  require.NoError(t, err)
  assert.Len(t, result.Clicks, 3)
  assert.True(t, result.HasMore)
  assert.NotEmpty(t, result.NextCursor)
}

func TestClickRepository_GetClickDetails_Cursor(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks with distinct timestamps
  err := repo.InsertClick(context.Background(), "cursor-test", now+100, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "cursor-test", now+50, "UK", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "cursor-test", now, "CA", "tablet", "social")
  require.NoError(t, err)

  // First page: cursor = MaxInt64 (start from newest)
  result, err := repo.GetClickDetails(context.Background(), "cursor-test", 9223372036854775807, 2)
  require.NoError(t, err)
  assert.Len(t, result.Clicks, 2)
  assert.True(t, result.HasMore)

  // Clicks should be newest first
  assert.Equal(t, now+100, result.Clicks[0].ClickedAt)
  assert.Equal(t, now+50, result.Clicks[1].ClickedAt)
}

func TestClickRepository_DeleteByShortCode_RemovesAll(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks
  err := repo.InsertClick(context.Background(), "delete-test", now, "US", "desktop", "direct")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "delete-test", now+1, "UK", "mobile", "search")
  require.NoError(t, err)

  // Verify they exist
  count, err := repo.CountByShortCode(context.Background(), "delete-test")
  require.NoError(t, err)
  assert.Equal(t, int64(2), count)

  // Delete all clicks for the code
  err = repo.DeleteByShortCode(context.Background(), "delete-test")
  require.NoError(t, err)

  // Verify they're gone
  count, err = repo.CountByShortCode(context.Background(), "delete-test")
  require.NoError(t, err)
  assert.Equal(t, int64(0), count)
}

func TestClickRepository_DeleteByShortCode_OnlyAffectsTargetCode(t *testing.T) {
  db := setupTestDB(t)
  repo := NewClickRepository(db)

  now := time.Now().Unix()

  // Insert clicks for code1
  err := repo.InsertClick(context.Background(), "code1", now, "US", "desktop", "direct")
  require.NoError(t, err)

  // Insert clicks for code2
  err = repo.InsertClick(context.Background(), "code2", now, "UK", "mobile", "search")
  require.NoError(t, err)

  err = repo.InsertClick(context.Background(), "code2", now+1, "CA", "tablet", "social")
  require.NoError(t, err)

  // Delete code1 clicks
  err = repo.DeleteByShortCode(context.Background(), "code1")
  require.NoError(t, err)

  // Verify code1 clicks are gone
  count, err := repo.CountByShortCode(context.Background(), "code1")
  require.NoError(t, err)
  assert.Equal(t, int64(0), count)

  // Verify code2 clicks remain
  count, err = repo.CountByShortCode(context.Background(), "code2")
  require.NoError(t, err)
  assert.Equal(t, int64(2), count)
}

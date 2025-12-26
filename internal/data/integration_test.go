package data

import (
	"context"
	"testing"
	"time"

	"go-shortener/ent"
	"go-shortener/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

// IntegrationTestSuite is the test suite for integration tests using testcontainers.
type IntegrationTestSuite struct {
	suite.Suite
	ctx             context.Context
	pgContainer     *postgres.PostgresContainer
	redisContainer  *tcredis.RedisContainer
	entClient       *ent.Client
	redisClient     *redis.Client
	repo            domain.URLRepository
	data            *Data
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(s.ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(s.T(), err)
	s.pgContainer = pgContainer

	// Start Redis container
	redisContainer, err := tcredis.Run(s.ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(s.T(), err)
	s.redisContainer = redisContainer

	// Get connection strings
	pgConnStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	redisEndpoint, err := redisContainer.Endpoint(s.ctx, "")
	require.NoError(s.T(), err)

	// Initialize Ent client with PostgreSQL
	s.entClient, err = ent.Open("postgres", pgConnStr)
	require.NoError(s.T(), err)

	// Run migrations
	err = s.entClient.Schema.Create(s.ctx)
	require.NoError(s.T(), err)

	// Initialize Redis client
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: redisEndpoint,
	})

	// Create Data and repository
	s.data = &Data{
		db:  s.entClient,
		rdb: s.redisClient,
	}
	s.repo = NewURLRepo(s.data, log.DefaultLogger)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.entClient != nil {
		s.entClient.Close()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.pgContainer != nil {
		s.pgContainer.Terminate(s.ctx)
	}
	if s.redisContainer != nil {
		s.redisContainer.Terminate(s.ctx)
	}
}

func (s *IntegrationTestSuite) TearDownTest() {
	// Clean up data after each test
	s.entClient.URL.Delete().ExecX(s.ctx)
	s.redisClient.FlushAll(s.ctx)
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestSave_NewURL() {
	// Arrange
	shortCode, _ := domain.NewShortCode("abc123")
	originalURL, _ := domain.NewOriginalURL("https://example.com")
	url := domain.NewURL(shortCode, originalURL, nil)

	// Act
	err := s.repo.Save(s.ctx, url)

	// Assert
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), url.ID())
}

func (s *IntegrationTestSuite) TestSave_WithExpiration() {
	// Arrange
	shortCode, _ := domain.NewShortCode("expire1")
	originalURL, _ := domain.NewOriginalURL("https://example.com")
	expiresAt := time.Now().Add(24 * time.Hour)
	url := domain.NewURL(shortCode, originalURL, &expiresAt)

	// Act
	err := s.repo.Save(s.ctx, url)

	// Assert
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), url.ID())

	// Verify the expiration was saved
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), found.ExpiresAt())
	assert.WithinDuration(s.T(), expiresAt, *found.ExpiresAt(), time.Second)
}

func (s *IntegrationTestSuite) TestFindByShortCode_Exists() {
	// Arrange
	shortCode, _ := domain.NewShortCode("find123")
	originalURL, _ := domain.NewOriginalURL("https://github.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// Act
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)

	// Assert
	require.NoError(s.T(), err)
	require.NotNil(s.T(), found)
	assert.Equal(s.T(), shortCode.String(), found.ShortCode().String())
	assert.Equal(s.T(), originalURL.String(), found.OriginalURL().String())
}

func (s *IntegrationTestSuite) TestFindByShortCode_NotFound() {
	// Arrange
	shortCode, _ := domain.NewShortCode("notexist")

	// Act
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)

	// Assert
	require.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

func (s *IntegrationTestSuite) TestFindByShortCode_UsesCache() {
	// Arrange
	shortCode, _ := domain.NewShortCode("cached1")
	originalURL, _ := domain.NewOriginalURL("https://cached.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// First call - should cache the result
	found1, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), found1)

	// Delete from database directly (not through repo to keep cache)
	s.entClient.URL.DeleteOneID(int(url.ID())).ExecX(s.ctx)

	// Act - should return cached result
	found2, err := s.repo.FindByShortCode(s.ctx, shortCode)

	// Assert
	require.NoError(s.T(), err)
	require.NotNil(s.T(), found2)
	assert.Equal(s.T(), shortCode.String(), found2.ShortCode().String())
}

func (s *IntegrationTestSuite) TestDelete() {
	// Arrange
	shortCode, _ := domain.NewShortCode("delete1")
	originalURL, _ := domain.NewOriginalURL("https://delete.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(s.ctx, shortCode)

	// Assert
	require.NoError(s.T(), err)

	// Verify deletion
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

func (s *IntegrationTestSuite) TestExists() {
	// Arrange
	shortCode, _ := domain.NewShortCode("exists1")
	originalURL, _ := domain.NewOriginalURL("https://exists.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// Act & Assert - exists
	exists, err := s.repo.Exists(s.ctx, shortCode)
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)

	// Act & Assert - does not exist
	nonExistent, _ := domain.NewShortCode("nonexist")
	exists, err = s.repo.Exists(s.ctx, nonExistent)
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)
}

func (s *IntegrationTestSuite) TestIncrementClickCount() {
	// Arrange
	shortCode, _ := domain.NewShortCode("clicks1")
	originalURL, _ := domain.NewOriginalURL("https://clicks.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), url.ClickCount())

	// Act
	err = s.repo.IncrementClickCount(s.ctx, shortCode)
	require.NoError(s.T(), err)
	err = s.repo.IncrementClickCount(s.ctx, shortCode)
	require.NoError(s.T(), err)
	err = s.repo.IncrementClickCount(s.ctx, shortCode)
	require.NoError(s.T(), err)

	// Assert
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), found.ClickCount())
}

func (s *IntegrationTestSuite) TestFindAll() {
	// Arrange - create multiple URLs
	for i := 1; i <= 5; i++ {
		code := "list" + string(rune('a'+i-1)) + "00"
		shortCode, _ := domain.NewShortCode(code)
		originalURL, _ := domain.NewOriginalURL("https://example" + code + ".com")
		url := domain.NewURL(shortCode, originalURL, nil)
		err := s.repo.Save(s.ctx, url)
		require.NoError(s.T(), err)
		time.Sleep(10 * time.Millisecond) // Ensure different creation times
	}

	// Act - get first page
	urls, total, err := s.repo.FindAll(s.ctx, 1, 3)

	// Assert
	require.NoError(s.T(), err)
	assert.Len(s.T(), urls, 3)
	assert.Equal(s.T(), 5, total)
}

func (s *IntegrationTestSuite) TestFindAll_Pagination() {
	// Arrange - create 10 URLs
	for i := 1; i <= 10; i++ {
		code := "page" + string(rune('a'+i-1)) + "0"
		shortCode, _ := domain.NewShortCode(code)
		originalURL, _ := domain.NewOriginalURL("https://page" + code + ".com")
		url := domain.NewURL(shortCode, originalURL, nil)
		err := s.repo.Save(s.ctx, url)
		require.NoError(s.T(), err)
	}

	// Act & Assert - page 1
	page1, total, err := s.repo.FindAll(s.ctx, 1, 5)
	require.NoError(s.T(), err)
	assert.Len(s.T(), page1, 5)
	assert.Equal(s.T(), 10, total)

	// Act & Assert - page 2
	page2, total, err := s.repo.FindAll(s.ctx, 2, 5)
	require.NoError(s.T(), err)
	assert.Len(s.T(), page2, 5)
	assert.Equal(s.T(), 10, total)

	// Act & Assert - page 3 (empty)
	page3, total, err := s.repo.FindAll(s.ctx, 3, 5)
	require.NoError(s.T(), err)
	assert.Len(s.T(), page3, 0)
	assert.Equal(s.T(), 10, total)
}

func (s *IntegrationTestSuite) TestCacheInvalidation_OnUpdate() {
	// Arrange
	shortCode, _ := domain.NewShortCode("update1")
	originalURL, _ := domain.NewOriginalURL("https://update.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// Cache the URL
	_, err = s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)

	// Act - increment click count (should invalidate cache)
	err = s.repo.IncrementClickCount(s.ctx, shortCode)
	require.NoError(s.T(), err)

	// Assert - next fetch should get updated data from DB
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), found.ClickCount())
}

func (s *IntegrationTestSuite) TestCacheInvalidation_OnDelete() {
	// Arrange
	shortCode, _ := domain.NewShortCode("delcache")
	originalURL, _ := domain.NewOriginalURL("https://delcache.com")
	url := domain.NewURL(shortCode, originalURL, nil)
	err := s.repo.Save(s.ctx, url)
	require.NoError(s.T(), err)

	// Cache the URL
	_, err = s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(s.ctx, shortCode)
	require.NoError(s.T(), err)

	// Assert - cache should be invalidated
	found, err := s.repo.FindByShortCode(s.ctx, shortCode)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

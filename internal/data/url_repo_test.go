package data

import (
	"context"
	"testing"
	"time"

	"go-shortener/ent"
	"go-shortener/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

type URLRepoTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *postgres.PostgresContainer
	entClient   *ent.Client
	repo        *urlRepo
}

func (s *URLRepoTestSuite) SetupSuite() {
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

	// Get connection string
	pgConnStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	// Initialize Ent client with PostgreSQL
	s.entClient, err = ent.Open("postgres", pgConnStr)
	require.NoError(s.T(), err)

	// Run migrations
	err = s.entClient.Schema.Create(s.ctx)
	require.NoError(s.T(), err)

	// Create repository (without Redis)
	data := &Data{
		db:  s.entClient,
		rdb: nil,
	}
	s.repo = &urlRepo{
		data: data,
		log:  log.NewHelper(log.DefaultLogger),
	}
}

func (s *URLRepoTestSuite) TearDownSuite() {
	if s.entClient != nil {
		s.entClient.Close()
	}
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}
}

func (s *URLRepoTestSuite) TearDownTest() {
	// Clean up data after each test
	s.entClient.URL.Delete().ExecX(s.ctx)
}

func TestURLRepoTestSuite(t *testing.T) {
	suite.Run(t, new(URLRepoTestSuite))
}

func (s *URLRepoTestSuite) TestSave() {
	// Arrange
	sc, _ := domain.NewShortCode("test123")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)

	// Act
	err := s.repo.Save(s.ctx, u)

	// Assert
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), u.ID())
	assert.Equal(s.T(), "test123", u.ShortCode().String())
	assert.Equal(s.T(), "https://example.com", u.OriginalURL().String())
	assert.Zero(s.T(), u.ClickCount())
}

func (s *URLRepoTestSuite) TestSave_WithExpiry() {
	// Arrange
	expiresAt := time.Now().Add(24 * time.Hour)
	sc, _ := domain.NewShortCode("expiry")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, &expiresAt)

	// Act
	err := s.repo.Save(s.ctx, u)

	// Assert
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), u.ExpiresAt())
}

func (s *URLRepoTestSuite) TestFindByShortCode() {
	// Arrange
	sc, _ := domain.NewShortCode("gettest")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := s.repo.Save(s.ctx, u)
	require.NoError(s.T(), err)

	// Act
	found, err := s.repo.FindByShortCode(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	require.NotNil(s.T(), found)
	assert.Equal(s.T(), "gettest", found.ShortCode().String())
}

func (s *URLRepoTestSuite) TestFindByShortCode_NotFound() {
	// Arrange
	sc, _ := domain.NewShortCode("nonexistent")

	// Act
	found, err := s.repo.FindByShortCode(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

func (s *URLRepoTestSuite) TestIncrementClickCount() {
	// Arrange
	sc, _ := domain.NewShortCode("clicks")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := s.repo.Save(s.ctx, u)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.IncrementClickCount(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	found, err := s.repo.FindByShortCode(s.ctx, sc)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), found.ClickCount())
}

func (s *URLRepoTestSuite) TestIncrementClickCount_Multiple() {
	// Arrange
	sc, _ := domain.NewShortCode("clicks2")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := s.repo.Save(s.ctx, u)
	require.NoError(s.T(), err)

	// Act
	_ = s.repo.IncrementClickCount(s.ctx, sc)
	_ = s.repo.IncrementClickCount(s.ctx, sc)
	_ = s.repo.IncrementClickCount(s.ctx, sc)

	// Assert
	found, err := s.repo.FindByShortCode(s.ctx, sc)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), found.ClickCount())
}

func (s *URLRepoTestSuite) TestDelete() {
	// Arrange
	sc, _ := domain.NewShortCode("todelete")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := s.repo.Save(s.ctx, u)
	require.NoError(s.T(), err)

	// Act
	err = s.repo.Delete(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	found, err := s.repo.FindByShortCode(s.ctx, sc)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

func (s *URLRepoTestSuite) TestFindAll() {
	// Arrange
	for i := 0; i < 5; i++ {
		sc, _ := domain.NewShortCode("list" + string(rune('a'+i)))
		ou, _ := domain.NewOriginalURL("https://example.com")
		u := domain.NewURL(sc, ou, nil)
		err := s.repo.Save(s.ctx, u)
		require.NoError(s.T(), err)
	}

	// Act
	urls, total, err := s.repo.FindAll(s.ctx, 1, 10)

	// Assert
	require.NoError(s.T(), err)
	assert.Len(s.T(), urls, 5)
	assert.Equal(s.T(), 5, total)
}

func (s *URLRepoTestSuite) TestFindAll_Pagination() {
	// Arrange
	for i := 0; i < 10; i++ {
		sc, _ := domain.NewShortCode("page" + string(rune('a'+i)))
		ou, _ := domain.NewOriginalURL("https://example.com")
		u := domain.NewURL(sc, ou, nil)
		err := s.repo.Save(s.ctx, u)
		require.NoError(s.T(), err)
	}

	// Act
	urls, total, err := s.repo.FindAll(s.ctx, 1, 3)

	// Assert
	require.NoError(s.T(), err)
	assert.Len(s.T(), urls, 3)
	assert.Equal(s.T(), 10, total)
}

func (s *URLRepoTestSuite) TestFindAll_Pagination_Page2() {
	// Arrange
	for i := 0; i < 10; i++ {
		sc, _ := domain.NewShortCode("pg2" + string(rune('a'+i)))
		ou, _ := domain.NewOriginalURL("https://example.com")
		u := domain.NewURL(sc, ou, nil)
		err := s.repo.Save(s.ctx, u)
		require.NoError(s.T(), err)
	}

	// Act
	urls, _, err := s.repo.FindAll(s.ctx, 2, 3)

	// Assert
	require.NoError(s.T(), err)
	assert.Len(s.T(), urls, 3)
}

func (s *URLRepoTestSuite) TestExists() {
	// Arrange
	sc, _ := domain.NewShortCode("exists")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.NewURL(sc, ou, nil)
	err := s.repo.Save(s.ctx, u)
	require.NoError(s.T(), err)

	// Act
	exists, err := s.repo.Exists(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)
}

func (s *URLRepoTestSuite) TestExists_NotFound() {
	// Arrange
	sc, _ := domain.NewShortCode("notexists")

	// Act
	exists, err := s.repo.Exists(s.ctx, sc)

	// Assert
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)
}

func (s *URLRepoTestSuite) TestCacheKey() {
	// Arrange & Act
	key := s.repo.cacheKey("test123")

	// Assert
	assert.Equal(s.T(), "url:test123", key)
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "go-shortener/api/shortener/v1"
	"go-shortener/internal/biz"
	"go-shortener/internal/domain"
	"go-shortener/internal/mocks"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// setupUoWMock configures UnitOfWork mock to execute the transaction function
func setupUoWMock(uow *mocks.UnitOfWork) {
	uow.EXPECT().
		Do(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error, _ ...domain.AggregateRoot) error {
			return fn(ctx)
		}).
		Maybe()
}

func TestShortenerService_CreateURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	// Act
	resp, err := svc.CreateURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp.Url)
	assert.Equal(t, "https://example.com", resp.Url.OriginalUrl)
	assert.NotEmpty(t, resp.Url.ShortCode)
}

func TestShortenerService_CreateURL_WithCustomCode(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	customCode := "mycode"
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}

	// Act
	resp, err := svc.CreateURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "mycode", resp.Url.ShortCode)
}

func TestShortenerService_CreateURL_WithExpiry(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	expiresAt := timestamppb.New(time.Now().Add(24 * time.Hour))
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		ExpiresAt:   expiresAt,
	}

	// Act
	resp, err := svc.CreateURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp.Url.ExpiresAt)
}

func TestShortenerService_CreateURL_InvalidURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.CreateURLRequest{
		OriginalUrl: "invalid-url",
	}

	// Act
	_, err := svc.CreateURL(context.Background(), req)

	// Assert
	assert.Error(t, err)
}

func TestShortenerService_GetURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("gettest")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.GetURLRequest{
		ShortCode: "gettest",
	}

	// Act
	resp, err := svc.GetURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "gettest", resp.Url.ShortCode)
}

func TestShortenerService_GetURL_NotFound(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("nonexistent")
	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(nil, nil)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.GetURLRequest{
		ShortCode: "nonexistent",
	}

	// Act
	_, err := svc.GetURL(context.Background(), req)

	// Assert
	assert.Error(t, err)
}

func TestShortenerService_RedirectURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("redirect")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)
	repo.EXPECT().IncrementClickCount(mock.Anything, sc).Return(nil)
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.RedirectURLRequest{
		ShortCode: "redirect",
	}

	// Act
	resp, err := svc.RedirectURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.OriginalUrl)
}

func TestShortenerService_GetURLStats(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("stats1")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 5, nil, time.Now(), time.Now())

	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.GetURLStatsRequest{
		ShortCode: "stats1",
	}

	// Act
	resp, err := svc.GetURLStats(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "stats1", resp.ShortCode)
	assert.Equal(t, int64(5), resp.ClickCount)
}

func TestShortenerService_DeleteURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("todelete")
	repo.EXPECT().Delete(mock.Anything, sc).Return(nil)
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.DeleteURLRequest{
		ShortCode: "todelete",
	}

	// Act
	resp, err := svc.DeleteURL(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestShortenerService_ListURLs(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc1, _ := domain.NewShortCode("lista1")
	ou1, _ := domain.NewOriginalURL("https://example1.com")
	url1 := domain.ReconstructURL(1, sc1, ou1, 0, nil, time.Now(), time.Now())

	sc2, _ := domain.NewShortCode("listb2")
	ou2, _ := domain.NewOriginalURL("https://example2.com")
	url2 := domain.ReconstructURL(2, sc2, ou2, 0, nil, time.Now(), time.Now())

	expectedURLs := []*domain.URL{url1, url2}
	repo.EXPECT().FindAll(mock.Anything, 1, 10).Return(expectedURLs, 2, nil)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.ListURLsRequest{
		Page:     1,
		PageSize: 10,
	}

	// Act
	resp, err := svc.ListURLs(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.Urls, 2)
	assert.Equal(t, int32(2), resp.Total)
}

func TestShortenerService_CreateURL_RepoError(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("database error"))
	setupUoWMock(uow)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	// Act
	_, err := svc.CreateURL(context.Background(), req)

	// Assert
	assert.Error(t, err)
}

func TestShortenerService_toURLInfo(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	uc := biz.NewURLUsecase(repo, uow, log.DefaultLogger)
	svc := NewShortenerService(uc)

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	sc, _ := domain.NewShortCode("testxx")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.ReconstructURL(1, sc, ou, 10, &expiresAt, now, now)

	// Act
	info := svc.toURLInfo(u)

	// Assert
	assert.Equal(t, int64(1), info.Id)
	assert.Equal(t, "testxx", info.ShortCode)
	assert.Equal(t, "https://example.com", info.OriginalUrl)
	assert.Equal(t, int64(10), info.ClickCount)
	assert.NotEmpty(t, info.ShortUrl)
	assert.NotNil(t, info.ExpiresAt)
}

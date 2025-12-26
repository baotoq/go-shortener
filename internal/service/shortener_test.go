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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ShortenerServiceTestSuite struct {
	suite.Suite
	repo *mocks.URLRepository
	uow  *mocks.UnitOfWork
	sut  *ShortenerService
}

func (s *ShortenerServiceTestSuite) SetupTest() {
	s.repo = mocks.NewURLRepository(s.T())
	s.uow = mocks.NewUnitOfWork(s.T())
	uc := biz.NewURLUsecase(s.repo, s.uow, log.DefaultLogger)
	s.sut = NewShortenerService(uc)
}

func (s *ShortenerServiceTestSuite) setupUoWMock() {
	s.uow.EXPECT().
		Do(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error, _ ...domain.AggregateRoot) error {
			return fn(ctx)
		}).
		Maybe()
}

func TestShortenerServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ShortenerServiceTestSuite))
}

func (s *ShortenerServiceTestSuite) TestCreateURL() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	// Act
	resp, err := s.sut.CreateURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(resp.Url)
	s.Equal("https://example.com", resp.Url.OriginalUrl)
	s.NotEmpty(resp.Url.ShortCode)
}

func (s *ShortenerServiceTestSuite) TestCreateURL_WithCustomCode() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	customCode := "mycode"
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}

	// Act
	resp, err := s.sut.CreateURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Equal("mycode", resp.Url.ShortCode)
}

func (s *ShortenerServiceTestSuite) TestCreateURL_WithExpiry() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	expiresAt := timestamppb.New(time.Now().Add(24 * time.Hour))
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		ExpiresAt:   expiresAt,
	}

	// Act
	resp, err := s.sut.CreateURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.NotNil(resp.Url.ExpiresAt)
}

func (s *ShortenerServiceTestSuite) TestCreateURL_InvalidURL() {
	// Arrange
	req := &v1.CreateURLRequest{
		OriginalUrl: "invalid-url",
	}

	// Act
	_, err := s.sut.CreateURL(context.Background(), req)

	// Assert
	s.Error(err)
}

func (s *ShortenerServiceTestSuite) TestGetURL() {
	// Arrange
	sc, _ := domain.NewShortCode("gettest")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

	req := &v1.GetURLRequest{
		ShortCode: "gettest",
	}

	// Act
	resp, err := s.sut.GetURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Equal("gettest", resp.Url.ShortCode)
}

func (s *ShortenerServiceTestSuite) TestGetURL_NotFound() {
	// Arrange
	sc, _ := domain.NewShortCode("nonexistent")
	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(nil, nil)

	req := &v1.GetURLRequest{
		ShortCode: "nonexistent",
	}

	// Act
	_, err := s.sut.GetURL(context.Background(), req)

	// Assert
	s.Error(err)
}

func (s *ShortenerServiceTestSuite) TestRedirectURL() {
	// Arrange
	sc, _ := domain.NewShortCode("redirect")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)
	s.setupUoWMock()

	req := &v1.RedirectURLRequest{
		ShortCode: "redirect",
	}

	// Act
	resp, err := s.sut.RedirectURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Equal("https://example.com", resp.OriginalUrl)
}

func (s *ShortenerServiceTestSuite) TestGetURLStats() {
	// Arrange
	sc, _ := domain.NewShortCode("stats1")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 5, nil, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

	req := &v1.GetURLStatsRequest{
		ShortCode: "stats1",
	}

	// Act
	resp, err := s.sut.GetURLStats(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Equal("stats1", resp.ShortCode)
	s.Equal(int64(5), resp.ClickCount)
}

func (s *ShortenerServiceTestSuite) TestDeleteURL() {
	// Arrange
	sc, _ := domain.NewShortCode("todelete")
	s.repo.EXPECT().Delete(mock.Anything, sc).Return(nil)
	s.setupUoWMock()

	req := &v1.DeleteURLRequest{
		ShortCode: "todelete",
	}

	// Act
	resp, err := s.sut.DeleteURL(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.True(resp.Success)
}

func (s *ShortenerServiceTestSuite) TestListURLs() {
	// Arrange
	sc1, _ := domain.NewShortCode("lista1")
	ou1, _ := domain.NewOriginalURL("https://example1.com")
	url1 := domain.ReconstructURL(1, sc1, ou1, 0, nil, time.Now(), time.Now())

	sc2, _ := domain.NewShortCode("listb2")
	ou2, _ := domain.NewOriginalURL("https://example2.com")
	url2 := domain.ReconstructURL(2, sc2, ou2, 0, nil, time.Now(), time.Now())

	expectedURLs := []*domain.URL{url1, url2}
	s.repo.EXPECT().FindAll(mock.Anything, 1, 10).Return(expectedURLs, 2, nil)

	req := &v1.ListURLsRequest{
		Page:     1,
		PageSize: 10,
	}

	// Act
	resp, err := s.sut.ListURLs(context.Background(), req)

	// Assert
	s.Require().NoError(err)
	s.Len(resp.Urls, 2)
	s.Equal(int32(2), resp.Total)
}

func (s *ShortenerServiceTestSuite) TestCreateURL_RepoError() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("database error"))
	s.setupUoWMock()

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	// Act
	_, err := s.sut.CreateURL(context.Background(), req)

	// Assert
	s.Error(err)
}

func (s *ShortenerServiceTestSuite) TestToURLInfo() {
	// Arrange
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	sc, _ := domain.NewShortCode("testxx")
	ou, _ := domain.NewOriginalURL("https://example.com")
	u := domain.ReconstructURL(1, sc, ou, 10, &expiresAt, now, now)

	// Act
	info := s.sut.toURLInfo(u)

	// Assert
	s.Equal(int64(1), info.Id)
	s.Equal("testxx", info.ShortCode)
	s.Equal("https://example.com", info.OriginalUrl)
	s.Equal(int64(10), info.ClickCount)
	s.NotEmpty(info.ShortUrl)
	s.NotNil(info.ExpiresAt)
}

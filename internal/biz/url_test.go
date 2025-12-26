package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-shortener/internal/domain"
	"go-shortener/internal/mocks"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type URLUsecaseTestSuite struct {
	suite.Suite
	repo *mocks.URLRepository
	uow  *mocks.UnitOfWork
	sut  *URLUsecase
}

func (s *URLUsecaseTestSuite) SetupTest() {
	s.repo = mocks.NewURLRepository(s.T())
	s.uow = mocks.NewUnitOfWork(s.T())
	s.sut = NewURLUsecase(s.repo, s.uow, log.DefaultLogger)
}

func (s *URLUsecaseTestSuite) setupUoWMock() {
	s.uow.EXPECT().
		Do(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error, _ ...domain.AggregateRoot) error {
			return fn(ctx)
		}).
		Maybe()
}

func TestURLUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(URLUsecaseTestSuite))
}

func (s *URLUsecaseTestSuite) TestCreateURL_ValidURLWithoutCustomCode() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	// Act
	url, err := s.sut.CreateURL(context.Background(), "https://example.com", nil, nil)

	// Assert
	s.Require().NoError(err)
	s.Equal("https://example.com", url.OriginalURL().String())
	s.NotEmpty(url.ShortCode().String())
}

func (s *URLUsecaseTestSuite) TestCreateURL_ValidURLWithCustomCode() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	customCode := "mycode"

	// Act
	url, err := s.sut.CreateURL(context.Background(), "https://example.com", &customCode, nil)

	// Assert
	s.Require().NoError(err)
	s.Equal("mycode", url.ShortCode().String())
}

func (s *URLUsecaseTestSuite) TestCreateURL_InvalidURL_Empty() {
	// Act
	_, err := s.sut.CreateURL(context.Background(), "", nil, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrInvalidURL, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_InvalidURL_NoScheme() {
	// Act
	_, err := s.sut.CreateURL(context.Background(), "example.com", nil, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrInvalidURL, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_InvalidURL_FTPScheme() {
	// Act
	_, err := s.sut.CreateURL(context.Background(), "ftp://example.com", nil, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrInvalidURL, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_InvalidCustomCode_TooShort() {
	// Arrange
	customCode := "ab"

	// Act
	_, err := s.sut.CreateURL(context.Background(), "https://example.com", &customCode, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrInvalidCode, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_InvalidCustomCode_SpecialChars() {
	// Arrange
	customCode := "my@code"

	// Act
	_, err := s.sut.CreateURL(context.Background(), "https://example.com", &customCode, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrInvalidCode, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_DuplicateCustomCode() {
	// Arrange - first creation succeeds
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Once()
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	s.setupUoWMock()

	customCode := "existing"

	// Act - first creation succeeds
	_, err := s.sut.CreateURL(context.Background(), "https://example.com", &customCode, nil)
	s.Require().NoError(err)

	// Arrange - second call fails
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(true, nil).Once()

	// Act - second creation fails
	_, err = s.sut.CreateURL(context.Background(), "https://example2.com", &customCode, nil)

	// Assert
	s.Error(err)
	s.Equal(ErrShortCodeExists, err)
}

func (s *URLUsecaseTestSuite) TestCreateURL_RepoError() {
	// Arrange
	s.repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
	s.repo.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("database error"))
	s.setupUoWMock()

	// Act
	_, err := s.sut.CreateURL(context.Background(), "https://example.com", nil, nil)

	// Assert
	s.Error(err)
}

func (s *URLUsecaseTestSuite) TestGetURL_Existing() {
	// Arrange
	sc, _ := domain.NewShortCode("testcode")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

	// Act
	url, err := s.sut.GetURL(context.Background(), "testcode")

	// Assert
	s.Require().NoError(err)
	s.Equal("testcode", url.ShortCode().String())
}

func (s *URLUsecaseTestSuite) TestGetURL_NotFound() {
	// Arrange
	sc, _ := domain.NewShortCode("nonexistent")
	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(nil, nil)

	// Act
	_, err := s.sut.GetURL(context.Background(), "nonexistent")

	// Assert
	s.Error(err)
	s.Equal(ErrURLNotFound, err)
}

func (s *URLUsecaseTestSuite) TestGetURL_Expired() {
	// Arrange
	expiredTime := time.Now().Add(-1 * time.Hour)
	sc, _ := domain.NewShortCode("expired")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expiredURL := domain.ReconstructURL(1, sc, ou, 0, &expiredTime, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expiredURL, nil)

	// Act
	_, err := s.sut.GetURL(context.Background(), "expired")

	// Assert
	s.Error(err)
	s.Equal(ErrURLExpired, err)
}

func (s *URLUsecaseTestSuite) TestRedirectURL() {
	// Arrange
	sc, _ := domain.NewShortCode("redirect")
	ou, _ := domain.NewOriginalURL("https://example.com")
	urlEntity := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	s.repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(urlEntity, nil)
	s.setupUoWMock()

	// Act
	result, err := s.sut.RedirectURL(context.Background(), "redirect")

	// Assert
	s.Require().NoError(err)
	s.Equal("https://example.com", result)
}

func (s *URLUsecaseTestSuite) TestDeleteURL() {
	// Arrange
	s.repo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
	s.setupUoWMock()

	// Act
	err := s.sut.DeleteURL(context.Background(), "todelete")

	// Assert
	s.Require().NoError(err)
}

func (s *URLUsecaseTestSuite) TestListURLs() {
	// Arrange
	sc1, _ := domain.NewShortCode("code1")
	ou1, _ := domain.NewOriginalURL("https://example1.com")
	url1 := domain.ReconstructURL(1, sc1, ou1, 0, nil, time.Now(), time.Now())

	sc2, _ := domain.NewShortCode("code2")
	ou2, _ := domain.NewOriginalURL("https://example2.com")
	url2 := domain.ReconstructURL(2, sc2, ou2, 0, nil, time.Now(), time.Now())

	expectedURLs := []*domain.URL{url1, url2}

	s.repo.EXPECT().FindAll(mock.Anything, 1, 10).Return(expectedURLs, 2, nil)

	// Act
	urls, total, err := s.sut.ListURLs(context.Background(), 1, 10)

	// Assert
	s.Require().NoError(err)
	s.Len(urls, 2)
	s.Equal(2, total)
}

func (s *URLUsecaseTestSuite) TestListURLs_Page0DefaultsTo1() {
	// Arrange
	s.repo.EXPECT().FindAll(mock.Anything, 1, 10).Return([]*domain.URL{}, 0, nil)

	// Act
	urls, _, err := s.sut.ListURLs(context.Background(), 0, 10)

	// Assert
	s.NoError(err)
	s.NotNil(urls)
}

func (s *URLUsecaseTestSuite) TestListURLs_PageSize0DefaultsTo20() {
	// Arrange
	s.repo.EXPECT().FindAll(mock.Anything, 1, 20).Return([]*domain.URL{}, 0, nil)

	// Act
	urls, _, err := s.sut.ListURLs(context.Background(), 1, 0)

	// Assert
	s.NoError(err)
	s.NotNil(urls)
}

func (s *URLUsecaseTestSuite) TestListURLs_PageSizeOver100DefaultsTo20() {
	// Arrange
	s.repo.EXPECT().FindAll(mock.Anything, 1, 20).Return([]*domain.URL{}, 0, nil)

	// Act
	urls, _, err := s.sut.ListURLs(context.Background(), 1, 200)

	// Assert
	s.NoError(err)
	s.NotNil(urls)
}

func (s *URLUsecaseTestSuite) TestGetShortURL() {
	// Act
	shortURL := s.sut.GetShortURL("abc123")

	// Assert
	s.Equal("http://localhost:8000/r/abc123", shortURL)
}

package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-shortener/internal/domain"
	"go-shortener/internal/mocks"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestURLUsecase_CreateURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		customCode  *string
		expiresAt   *time.Time
		wantErr     bool
		errType     error
	}{
		{
			name:        "valid url without custom code",
			originalURL: "https://example.com",
			customCode:  nil,
			wantErr:     false,
		},
		{
			name:        "valid url with custom code",
			originalURL: "https://example.com",
			customCode:  strPtr("mycode"),
			wantErr:     false,
		},
		{
			name:        "invalid url - empty",
			originalURL: "",
			customCode:  nil,
			wantErr:     true,
			errType:     ErrInvalidURL,
		},
		{
			name:        "invalid url - no scheme",
			originalURL: "example.com",
			customCode:  nil,
			wantErr:     true,
			errType:     ErrInvalidURL,
		},
		{
			name:        "invalid url - ftp scheme",
			originalURL: "ftp://example.com",
			customCode:  nil,
			wantErr:     true,
			errType:     ErrInvalidURL,
		},
		{
			name:        "invalid custom code - too short",
			originalURL: "https://example.com",
			customCode:  strPtr("ab"),
			wantErr:     true,
			errType:     ErrInvalidCode,
		},
		{
			name:        "invalid custom code - special chars",
			originalURL: "https://example.com",
			customCode:  strPtr("my@code"),
			wantErr:     true,
			errType:     ErrInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := mocks.NewURLRepository(t)
			uow := mocks.NewUnitOfWork(t)

			if !tt.wantErr {
				if tt.customCode != nil {
					repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
				} else {
					repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Maybe()
				}
				repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				setupUoWMock(uow)
			}

			uc := NewURLUsecase(repo, uow, log.DefaultLogger)

			// Act
			url, err := uc.CreateURL(context.Background(), tt.originalURL, tt.customCode, tt.expiresAt)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.originalURL, url.OriginalURL().String())

			if tt.customCode != nil {
				assert.Equal(t, *tt.customCode, url.ShortCode().String())
			} else {
				assert.NotEmpty(t, url.ShortCode().String())
			}
		})
	}
}

func TestURLUsecase_CreateURL_DuplicateCustomCode(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)
	customCode := "existing"

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	setupUoWMock(uow)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act - first creation succeeds
	_, err := uc.CreateURL(context.Background(), "https://example.com", &customCode, nil)

	// Assert
	require.NoError(t, err)

	// Arrange - second call
	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(true, nil).Once()

	// Act - second creation fails
	_, err = uc.CreateURL(context.Background(), "https://example2.com", &customCode, nil)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrShortCodeExists, err)
}

func TestURLUsecase_GetURL(t *testing.T) {
	t.Run("existing url", func(t *testing.T) {
		// Arrange
		repo := mocks.NewURLRepository(t)
		uow := mocks.NewUnitOfWork(t)

		sc, _ := domain.NewShortCode("testcode")
		ou, _ := domain.NewOriginalURL("https://example.com")
		expectedURL := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

		repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expectedURL, nil)

		uc := NewURLUsecase(repo, uow, log.DefaultLogger)

		// Act
		url, err := uc.GetURL(context.Background(), "testcode")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "testcode", url.ShortCode().String())
	})

	t.Run("non-existing url", func(t *testing.T) {
		// Arrange
		repo := mocks.NewURLRepository(t)
		uow := mocks.NewUnitOfWork(t)

		sc, _ := domain.NewShortCode("nonexistent")
		repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(nil, nil)

		uc := NewURLUsecase(repo, uow, log.DefaultLogger)

		// Act
		_, err := uc.GetURL(context.Background(), "nonexistent")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrURLNotFound, err)
	})
}

func TestURLUsecase_GetURL_Expired(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	expiredTime := time.Now().Add(-1 * time.Hour)
	sc, _ := domain.NewShortCode("expired")
	ou, _ := domain.NewOriginalURL("https://example.com")
	expiredURL := domain.ReconstructURL(1, sc, ou, 0, &expiredTime, time.Now(), time.Now())

	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(expiredURL, nil)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	_, err := uc.GetURL(context.Background(), "expired")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrURLExpired, err)
}

func TestURLUsecase_RedirectURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc, _ := domain.NewShortCode("redirect")
	ou, _ := domain.NewOriginalURL("https://example.com")
	urlEntity := domain.ReconstructURL(1, sc, ou, 0, nil, time.Now(), time.Now())

	repo.EXPECT().FindByShortCode(mock.Anything, sc).Return(urlEntity, nil)
	repo.EXPECT().IncrementClickCount(mock.Anything, mock.Anything).Return(nil)
	setupUoWMock(uow)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	result, err := uc.RedirectURL(context.Background(), "redirect")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestURLUsecase_DeleteURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
	setupUoWMock(uow)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	err := uc.DeleteURL(context.Background(), "todelete")

	// Assert
	require.NoError(t, err)
}

func TestURLUsecase_ListURLs(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	sc1, _ := domain.NewShortCode("code1")
	ou1, _ := domain.NewOriginalURL("https://example1.com")
	url1 := domain.ReconstructURL(1, sc1, ou1, 0, nil, time.Now(), time.Now())

	sc2, _ := domain.NewShortCode("code2")
	ou2, _ := domain.NewOriginalURL("https://example2.com")
	url2 := domain.ReconstructURL(2, sc2, ou2, 0, nil, time.Now(), time.Now())

	expectedURLs := []*domain.URL{url1, url2}

	repo.EXPECT().FindAll(mock.Anything, 1, 10).Return(expectedURLs, 2, nil)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	urls, total, err := uc.ListURLs(context.Background(), 1, 10)

	// Assert
	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Equal(t, 2, total)
}

func TestURLUsecase_ListURLs_Pagination(t *testing.T) {
	t.Run("page 0 defaults to 1", func(t *testing.T) {
		// Arrange
		repo := mocks.NewURLRepository(t)
		uow := mocks.NewUnitOfWork(t)

		repo.EXPECT().FindAll(mock.Anything, 1, 10).Return([]*domain.URL{}, 0, nil)

		uc := NewURLUsecase(repo, uow, log.DefaultLogger)

		// Act
		urls, _, err := uc.ListURLs(context.Background(), 0, 10)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, urls)
	})

	t.Run("pageSize 0 defaults to 20", func(t *testing.T) {
		// Arrange
		repo := mocks.NewURLRepository(t)
		uow := mocks.NewUnitOfWork(t)

		repo.EXPECT().FindAll(mock.Anything, 1, 20).Return([]*domain.URL{}, 0, nil)

		uc := NewURLUsecase(repo, uow, log.DefaultLogger)

		// Act
		urls, _, err := uc.ListURLs(context.Background(), 1, 0)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, urls)
	})

	t.Run("pageSize > 100 defaults to 20", func(t *testing.T) {
		// Arrange
		repo := mocks.NewURLRepository(t)
		uow := mocks.NewUnitOfWork(t)

		repo.EXPECT().FindAll(mock.Anything, 1, 20).Return([]*domain.URL{}, 0, nil)

		uc := NewURLUsecase(repo, uow, log.DefaultLogger)

		// Act
		urls, _, err := uc.ListURLs(context.Background(), 1, 200)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, urls)
	})
}

func TestURLUsecase_GetShortURL(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	shortURL := uc.GetShortURL("abc123")

	// Assert
	assert.Equal(t, "http://localhost:8000/r/abc123", shortURL)
}

func TestURLUsecase_CreateURL_RepoError(t *testing.T) {
	// Arrange
	repo := mocks.NewURLRepository(t)
	uow := mocks.NewUnitOfWork(t)

	repo.EXPECT().Exists(mock.Anything, mock.Anything).Return(false, nil)
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("database error"))
	setupUoWMock(uow)

	uc := NewURLUsecase(repo, uow, log.DefaultLogger)

	// Act
	_, err := uc.CreateURL(context.Background(), "https://example.com", nil, nil)

	// Assert
	assert.Error(t, err)
}

func strPtr(s string) *string {
	return &s
}

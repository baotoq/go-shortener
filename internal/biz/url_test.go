package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockURLRepo struct {
	urls      map[string]*URL
	createErr error
	getErr    error
	deleteErr error
	listErr   error
	existsErr error
	incrErr   error
}

func newMockRepo() *mockURLRepo {
	return &mockURLRepo{
		urls: make(map[string]*URL),
	}
}

func (m *mockURLRepo) Create(ctx context.Context, url *URL) (*URL, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	url.ID = int64(len(m.urls) + 1)
	url.CreatedAt = time.Now()
	url.UpdatedAt = time.Now()
	m.urls[url.ShortCode] = url
	return url, nil
}

func (m *mockURLRepo) GetByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.urls[shortCode], nil
}

func (m *mockURLRepo) IncrementClickCount(ctx context.Context, shortCode string) error {
	if m.incrErr != nil {
		return m.incrErr
	}
	if url, ok := m.urls[shortCode]; ok {
		url.ClickCount++
	}
	return nil
}

func (m *mockURLRepo) Delete(ctx context.Context, shortCode string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.urls, shortCode)
	return nil
}

func (m *mockURLRepo) List(ctx context.Context, page, pageSize int) ([]*URL, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	urls := make([]*URL, 0, len(m.urls))
	for _, u := range m.urls {
		urls = append(urls, u)
	}
	return urls, len(urls), nil
}

func (m *mockURLRepo) ExistsShortCode(ctx context.Context, shortCode string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	_, exists := m.urls[shortCode]
	return exists, nil
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
			repo := newMockRepo()
			uc := NewURLUsecase(repo, log.DefaultLogger)

			url, err := uc.CreateURL(context.Background(), tt.originalURL, tt.customCode, tt.expiresAt)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.originalURL, url.OriginalURL)

			if tt.customCode != nil {
				assert.Equal(t, *tt.customCode, url.ShortCode)
			} else {
				assert.NotEmpty(t, url.ShortCode)
			}
		})
	}
}

func TestURLUsecase_CreateURL_DuplicateCustomCode(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	customCode := "existing"
	_, err := uc.CreateURL(context.Background(), "https://example.com", &customCode, nil)
	require.NoError(t, err)

	_, err = uc.CreateURL(context.Background(), "https://example2.com", &customCode, nil)
	assert.Error(t, err)
}

func TestURLUsecase_GetURL(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	customCode := "testcode"
	created, err := uc.CreateURL(context.Background(), "https://example.com", &customCode, nil)
	require.NoError(t, err)

	tests := []struct {
		name      string
		shortCode string
		wantErr   bool
	}{
		{
			name:      "existing url",
			shortCode: created.ShortCode,
			wantErr:   false,
		},
		{
			name:      "non-existing url",
			shortCode: "nonexistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := uc.GetURL(context.Background(), tt.shortCode)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.shortCode, url.ShortCode)
		})
	}
}

func TestURLUsecase_GetURL_Expired(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	expiredTime := time.Now().Add(-1 * time.Hour)
	customCode := "expired"

	repo.urls[customCode] = &URL{
		ID:          1,
		ShortCode:   customCode,
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiredTime,
		CreatedAt:   time.Now(),
	}

	_, err := uc.GetURL(context.Background(), customCode)
	assert.Error(t, err)
}

func TestURLUsecase_RedirectURL(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	customCode := "redirect"
	originalURL := "https://example.com"
	_, err := uc.CreateURL(context.Background(), originalURL, &customCode, nil)
	require.NoError(t, err)

	result, err := uc.RedirectURL(context.Background(), customCode)
	require.NoError(t, err)
	assert.Equal(t, originalURL, result)
	assert.Equal(t, int64(1), repo.urls[customCode].ClickCount)
}

func TestURLUsecase_DeleteURL(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	customCode := "todelete"
	_, err := uc.CreateURL(context.Background(), "https://example.com", &customCode, nil)
	require.NoError(t, err)

	err = uc.DeleteURL(context.Background(), customCode)
	require.NoError(t, err)

	_, err = uc.GetURL(context.Background(), customCode)
	assert.Error(t, err)
}

func TestURLUsecase_ListURLs(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	for i := 0; i < 5; i++ {
		code := "code" + string(rune('a'+i))
		_, err := uc.CreateURL(context.Background(), "https://example.com", &code, nil)
		require.NoError(t, err)
	}

	urls, total, err := uc.ListURLs(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Len(t, urls, 5)
	assert.Equal(t, 5, total)
}

func TestURLUsecase_ListURLs_Pagination(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	urls, _, err := uc.ListURLs(context.Background(), 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, urls)

	urls, _, err = uc.ListURLs(context.Background(), 1, 0)
	assert.NoError(t, err)
	assert.NotNil(t, urls)

	urls, _, err = uc.ListURLs(context.Background(), 1, 200)
	assert.NoError(t, err)
	assert.NotNil(t, urls)
}

func TestURLUsecase_GetShortURL(t *testing.T) {
	repo := newMockRepo()
	uc := NewURLUsecase(repo, log.DefaultLogger)

	shortURL := uc.GetShortURL("abc123")
	assert.Equal(t, "http://localhost:8000/r/abc123", shortURL)
}

func TestURLUsecase_CreateURL_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errors.New("database error")
	uc := NewURLUsecase(repo, log.DefaultLogger)

	_, err := uc.CreateURL(context.Background(), "https://example.com", nil, nil)
	assert.Error(t, err)
}

func strPtr(s string) *string {
	return &s
}

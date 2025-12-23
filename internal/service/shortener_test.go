package service

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "go-shortener/api/shortener/v1"
	"go-shortener/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockURLRepo struct {
	urls      map[string]*biz.URL
	createErr error
	getErr    error
	deleteErr error
	listErr   error
	existsErr error
	incrErr   error
}

func newMockRepo() *mockURLRepo {
	return &mockURLRepo{
		urls: make(map[string]*biz.URL),
	}
}

func (m *mockURLRepo) Create(ctx context.Context, url *biz.URL) (*biz.URL, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	url.ID = int64(len(m.urls) + 1)
	url.CreatedAt = time.Now()
	url.UpdatedAt = time.Now()
	m.urls[url.ShortCode] = url
	return url, nil
}

func (m *mockURLRepo) GetByShortCode(ctx context.Context, shortCode string) (*biz.URL, error) {
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

func (m *mockURLRepo) List(ctx context.Context, page, pageSize int) ([]*biz.URL, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	urls := make([]*biz.URL, 0, len(m.urls))
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

func setupService() (*ShortenerService, *mockURLRepo) {
	repo := newMockRepo()
	uc := biz.NewURLUsecase(repo, log.DefaultLogger)
	svc := NewShortenerService(uc)
	return svc, repo
}

func TestShortenerService_CreateURL(t *testing.T) {
	svc, _ := setupService()

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	resp, err := svc.CreateURL(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Url)
	assert.Equal(t, "https://example.com", resp.Url.OriginalUrl)
	assert.NotEmpty(t, resp.Url.ShortCode)
}

func TestShortenerService_CreateURL_WithCustomCode(t *testing.T) {
	svc, _ := setupService()

	customCode := "mycode"
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}

	resp, err := svc.CreateURL(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "mycode", resp.Url.ShortCode)
}

func TestShortenerService_CreateURL_WithExpiry(t *testing.T) {
	svc, _ := setupService()

	expiresAt := timestamppb.New(time.Now().Add(24 * time.Hour))
	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		ExpiresAt:   expiresAt,
	}

	resp, err := svc.CreateURL(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp.Url.ExpiresAt)
}

func TestShortenerService_CreateURL_InvalidURL(t *testing.T) {
	svc, _ := setupService()

	req := &v1.CreateURLRequest{
		OriginalUrl: "invalid-url",
	}

	_, err := svc.CreateURL(context.Background(), req)
	assert.Error(t, err)
}

func TestShortenerService_GetURL(t *testing.T) {
	svc, _ := setupService()

	customCode := "gettest"
	createReq := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}
	_, err := svc.CreateURL(context.Background(), createReq)
	require.NoError(t, err)

	req := &v1.GetURLRequest{
		ShortCode: "gettest",
	}

	resp, err := svc.GetURL(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "gettest", resp.Url.ShortCode)
}

func TestShortenerService_GetURL_NotFound(t *testing.T) {
	svc, _ := setupService()

	req := &v1.GetURLRequest{
		ShortCode: "nonexistent",
	}

	_, err := svc.GetURL(context.Background(), req)
	assert.Error(t, err)
}

func TestShortenerService_RedirectURL(t *testing.T) {
	svc, repo := setupService()

	customCode := "redirect"
	createReq := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}
	_, err := svc.CreateURL(context.Background(), createReq)
	require.NoError(t, err)

	req := &v1.RedirectURLRequest{
		ShortCode: "redirect",
	}

	resp, err := svc.RedirectURL(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.OriginalUrl)
	assert.Equal(t, int64(1), repo.urls["redirect"].ClickCount)
}

func TestShortenerService_GetURLStats(t *testing.T) {
	svc, repo := setupService()

	customCode := "stats"
	createReq := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}
	_, err := svc.CreateURL(context.Background(), createReq)
	require.NoError(t, err)

	repo.urls["stats"].ClickCount = 42

	req := &v1.GetURLStatsRequest{
		ShortCode: "stats",
	}

	resp, err := svc.GetURLStats(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "stats", resp.ShortCode)
	assert.Equal(t, int64(42), resp.ClickCount)
}

func TestShortenerService_DeleteURL(t *testing.T) {
	svc, _ := setupService()

	customCode := "todelete"
	createReq := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
		CustomCode:  &customCode,
	}
	_, err := svc.CreateURL(context.Background(), createReq)
	require.NoError(t, err)

	req := &v1.DeleteURLRequest{
		ShortCode: "todelete",
	}

	resp, err := svc.DeleteURL(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	getReq := &v1.GetURLRequest{ShortCode: "todelete"}
	_, err = svc.GetURL(context.Background(), getReq)
	assert.Error(t, err)
}

func TestShortenerService_ListURLs(t *testing.T) {
	svc, _ := setupService()

	for i := 0; i < 5; i++ {
		code := "list" + string(rune('a'+i))
		createReq := &v1.CreateURLRequest{
			OriginalUrl: "https://example.com",
			CustomCode:  &code,
		}
		_, err := svc.CreateURL(context.Background(), createReq)
		require.NoError(t, err)
	}

	req := &v1.ListURLsRequest{
		Page:     1,
		PageSize: 10,
	}

	resp, err := svc.ListURLs(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Urls, 5)
	assert.Equal(t, int32(5), resp.Total)
}

func TestShortenerService_CreateURL_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errors.New("database error")
	uc := biz.NewURLUsecase(repo, log.DefaultLogger)
	svc := NewShortenerService(uc)

	req := &v1.CreateURLRequest{
		OriginalUrl: "https://example.com",
	}

	_, err := svc.CreateURL(context.Background(), req)
	assert.Error(t, err)
}

func TestShortenerService_toURLInfo(t *testing.T) {
	svc, _ := setupService()

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	u := &biz.URL{
		ID:          1,
		ShortCode:   "test",
		OriginalURL: "https://example.com",
		ClickCount:  10,
		CreatedAt:   now,
		ExpiresAt:   &expiresAt,
	}

	info := svc.toURLInfo(u)

	assert.Equal(t, int64(1), info.Id)
	assert.Equal(t, "test", info.ShortCode)
	assert.Equal(t, "https://example.com", info.OriginalUrl)
	assert.Equal(t, int64(10), info.ClickCount)
	assert.NotEmpty(t, info.ShortUrl)
	assert.NotNil(t, info.ExpiresAt)
}

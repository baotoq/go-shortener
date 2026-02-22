package mqs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"go-shortener/common/events"
	"go-shortener/services/analytics-consumer/internal/config"
	"go-shortener/services/analytics-consumer/internal/svc"
	"go-shortener/services/analytics-rpc/model"

	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGeoIPReader implements svc.GeoIPReader for testing.
type mockGeoIPReader struct {
	countryFunc func(ip net.IP) (*geoip2.Country, error)
}

func (m *mockGeoIPReader) Country(ip net.IP) (*geoip2.Country, error) {
	return m.countryFunc(ip)
}

func TestClickEventConsumer_Success(t *testing.T) {
	var insertedClick *model.Clicks

	mockModel := &model.MockClicksModel{
		InsertFunc: func(ctx context.Context, data *model.Clicks) (sql.Result, error) {
			insertedClick = data
			return nil, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
		GeoDB:      nil, // No GeoIP database
	}

	consumer := NewClickEventConsumer(context.Background(), svcCtx)

	event := events.ClickEvent{
		ShortCode: "abc12345",
		Timestamp: time.Now().Unix(),
		IP:        "1.2.3.4",
		UserAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		Referer:   "https://google.com/search",
	}

	payload, _ := json.Marshal(event)
	err := consumer.Consume(context.Background(), "", string(payload))

	require.NoError(t, err)
	require.NotNil(t, insertedClick)
	assert.Equal(t, "abc12345", insertedClick.ShortCode)
	assert.Equal(t, "XX", insertedClick.CountryCode, "should fallback to XX without GeoIP")
	assert.Equal(t, "Bot", insertedClick.DeviceType)
	assert.Equal(t, "Search", insertedClick.TrafficSource)
	assert.NotEmpty(t, insertedClick.Id)
}

func TestClickEventConsumer_InvalidJSON(t *testing.T) {
	mockModel := &model.MockClicksModel{
		InsertFunc: func(ctx context.Context, data *model.Clicks) (sql.Result, error) {
			t.Fatal("Insert should not be called for invalid JSON")
			return nil, nil
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
		GeoDB:      nil,
	}

	consumer := NewClickEventConsumer(context.Background(), svcCtx)

	// Malformed JSON
	err := consumer.Consume(context.Background(), "", "{invalid json")

	// Should return nil (skip, don't retry)
	assert.NoError(t, err, "malformed JSON should return nil to skip message")
}

func TestClickEventConsumer_DuplicateKey(t *testing.T) {
	mockModel := &model.MockClicksModel{
		InsertFunc: func(ctx context.Context, data *model.Clicks) (sql.Result, error) {
			return nil, errors.New("duplicate key value violates unique constraint")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
		GeoDB:      nil,
	}

	consumer := NewClickEventConsumer(context.Background(), svcCtx)

	event := events.ClickEvent{
		ShortCode: "abc12345",
		Timestamp: time.Now().Unix(),
		IP:        "1.2.3.4",
		UserAgent: "Mozilla/5.0",
		Referer:   "",
	}

	payload, _ := json.Marshal(event)
	err := consumer.Consume(context.Background(), "", string(payload))

	// Should return nil (idempotent handling)
	assert.NoError(t, err, "duplicate key should return nil for idempotency")
}

func TestClickEventConsumer_DBError(t *testing.T) {
	mockModel := &model.MockClicksModel{
		InsertFunc: func(ctx context.Context, data *model.Clicks) (sql.Result, error) {
			return nil, errors.New("database connection error")
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		ClickModel: mockModel,
		GeoDB:      nil,
	}

	consumer := NewClickEventConsumer(context.Background(), svcCtx)

	event := events.ClickEvent{
		ShortCode: "abc12345",
		Timestamp: time.Now().Unix(),
		IP:        "1.2.3.4",
		UserAgent: "Mozilla/5.0",
		Referer:   "",
	}

	payload, _ := json.Marshal(event)
	err := consumer.Consume(context.Background(), "", string(payload))

	// Should return error (retry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")
}

func TestResolveDeviceType(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "Empty user agent",
			userAgent: "",
			expected:  "Unknown",
		},
		{
			name:      "Bot Googlebot",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			expected:  "Bot",
		},
		{
			name:      "Desktop Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected:  "Desktop",
		},
		{
			name:      "Desktop Safari",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
			expected:  "Desktop",
		},
		{
			name:      "iPhone (detected as Desktop due to library limitation)",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
			expected:  "Desktop",
		},
		{
			name:      "Android (detected as Desktop due to library limitation)",
			userAgent: "Mozilla/5.0 (Linux; Android 10; SM-G973F)",
			expected:  "Desktop",
		},
		{
			name:      "curl (not detected as bot)",
			userAgent: "curl/7.64.1",
			expected:  "Desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveDeviceType(tt.userAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveTrafficSource(t *testing.T) {
	tests := []struct {
		name     string
		referer  string
		expected string
	}{
		{
			name:     "Empty referer",
			referer:  "",
			expected: "Direct",
		},
		{
			name:     "Google search",
			referer:  "https://www.google.com/search?q=test",
			expected: "Search",
		},
		{
			name:     "Bing search",
			referer:  "https://www.bing.com/search?q=test",
			expected: "Search",
		},
		{
			name:     "DuckDuckGo search",
			referer:  "https://duckduckgo.com/?q=test",
			expected: "Search",
		},
		{
			name:     "Facebook",
			referer:  "https://www.facebook.com/",
			expected: "Social",
		},
		{
			name:     "Twitter",
			referer:  "https://twitter.com/user/status/123",
			expected: "Social",
		},
		{
			name:     "t.co (Twitter short link)",
			referer:  "https://t.co/abc123",
			expected: "Social",
		},
		{
			name:     "LinkedIn",
			referer:  "https://www.linkedin.com/feed/",
			expected: "Social",
		},
		{
			name:     "Reddit",
			referer:  "https://www.reddit.com/r/programming/",
			expected: "Social",
		},
		{
			name:     "Instagram",
			referer:  "https://www.instagram.com/",
			expected: "Social",
		},
		{
			name:     "YouTube",
			referer:  "https://www.youtube.com/watch?v=abc",
			expected: "Social",
		},
		{
			name:     "TikTok",
			referer:  "https://www.tiktok.com/@user",
			expected: "Social",
		},
		{
			name:     "Other website",
			referer:  "https://example.com/page",
			expected: "Referral",
		},
		{
			name:     "News site",
			referer:  "https://news.ycombinator.com/",
			expected: "Referral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTrafficSource(tt.referer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveCountry_NilGeoDB(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		GeoDB: nil,
	}

	result := resolveCountry(svcCtx, "1.2.3.4")
	assert.Equal(t, "XX", result, "should return XX when GeoDB is nil")
}

func TestResolveCountry_EmptyIP(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		GeoDB: nil,
	}

	result := resolveCountry(svcCtx, "")
	assert.Equal(t, "XX", result, "should return XX for empty IP")
}

func TestResolveCountry_InvalidIP(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		GeoDB: nil,
	}

	result := resolveCountry(svcCtx, "not-an-ip")
	assert.Equal(t, "XX", result, "should return XX for invalid IP")
}

func TestResolveCountry_WithGeoDB_Success(t *testing.T) {
	mock := &mockGeoIPReader{
		countryFunc: func(ip net.IP) (*geoip2.Country, error) {
			c := &geoip2.Country{}
			c.Country.IsoCode = "US"
			return c, nil
		},
	}
	svcCtx := &svc.ServiceContext{GeoDB: mock}
	result := resolveCountry(svcCtx, "8.8.8.8")
	assert.Equal(t, "US", result)
}

func TestResolveCountry_WithGeoDB_LookupError(t *testing.T) {
	mock := &mockGeoIPReader{
		countryFunc: func(ip net.IP) (*geoip2.Country, error) {
			return nil, errors.New("lookup failed")
		},
	}
	svcCtx := &svc.ServiceContext{GeoDB: mock}
	result := resolveCountry(svcCtx, "8.8.8.8")
	assert.Equal(t, "XX", result)
}

func TestResolveCountry_WithGeoDB_EmptyIsoCode(t *testing.T) {
	mock := &mockGeoIPReader{
		countryFunc: func(ip net.IP) (*geoip2.Country, error) {
			return &geoip2.Country{}, nil
		},
	}
	svcCtx := &svc.ServiceContext{GeoDB: mock}
	result := resolveCountry(svcCtx, "8.8.8.8")
	assert.Equal(t, "XX", result)
}

func TestResolveCountry_WithGeoDB_InvalidIP(t *testing.T) {
	mock := &mockGeoIPReader{
		countryFunc: func(ip net.IP) (*geoip2.Country, error) {
			t.Fatal("Country should not be called for invalid IP")
			return nil, nil
		},
	}
	svcCtx := &svc.ServiceContext{GeoDB: mock}
	result := resolveCountry(svcCtx, "not-an-ip")
	assert.Equal(t, "XX", result)
}

func TestResolveDeviceType_Mobile(t *testing.T) {
	// UA must contain "Mobile" token for mssola/useragent to detect it
	result := resolveDeviceType("Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.86 Mobile Safari/537.36")
	assert.Equal(t, "Mobile", result)
}

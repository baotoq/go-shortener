package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOriginalURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr error
	}{
		{
			name:    "valid https url",
			rawURL:  "https://example.com",
			wantErr: nil,
		},
		{
			name:    "valid http url",
			rawURL:  "http://example.com",
			wantErr: nil,
		},
		{
			name:    "valid url with path",
			rawURL:  "https://example.com/path/to/page",
			wantErr: nil,
		},
		{
			name:    "valid url with query",
			rawURL:  "https://example.com?foo=bar&baz=qux",
			wantErr: nil,
		},
		{
			name:    "valid url with port",
			rawURL:  "https://example.com:8080/path",
			wantErr: nil,
		},
		{
			name:    "empty url",
			rawURL:  "",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid url - no scheme",
			rawURL:  "example.com",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid url - ftp scheme",
			rawURL:  "ftp://example.com",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid url - no host",
			rawURL:  "https://",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid url - javascript scheme",
			rawURL:  "javascript:alert(1)",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := NewOriginalURL(tt.rawURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.True(t, url.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.rawURL, url.String())
				assert.False(t, url.IsEmpty())
			}
		})
	}
}

func TestOriginalURL_Host(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		wantHost string
	}{
		{
			name:     "simple host",
			rawURL:   "https://example.com",
			wantHost: "example.com",
		},
		{
			name:     "host with port",
			rawURL:   "https://example.com:8080",
			wantHost: "example.com:8080",
		},
		{
			name:     "subdomain",
			rawURL:   "https://www.example.com",
			wantHost: "www.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := NewOriginalURL(tt.rawURL)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, url.Host())
		})
	}
}

func TestOriginalURL_Scheme(t *testing.T) {
	httpsURL, err := NewOriginalURL("https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "https", httpsURL.Scheme())

	httpURL, err := NewOriginalURL("http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "http", httpURL.Scheme())
}

func TestOriginalURL_Equals(t *testing.T) {
	url1, _ := NewOriginalURL("https://example.com")
	url2, _ := NewOriginalURL("https://example.com")
	url3, _ := NewOriginalURL("https://other.com")

	assert.True(t, url1.Equals(url2))
	assert.False(t, url1.Equals(url3))
}

func TestOriginalURL_EmptyHost(t *testing.T) {
	var emptyURL OriginalURL
	assert.Equal(t, "", emptyURL.Host())
	assert.Equal(t, "", emptyURL.Scheme())
}

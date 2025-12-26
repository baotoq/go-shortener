package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShortCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		{
			name:    "valid alphanumeric code",
			code:    "abc123",
			wantErr: nil,
		},
		{
			name:    "valid code with underscore",
			code:    "my_code",
			wantErr: nil,
		},
		{
			name:    "valid code with hyphen",
			code:    "my-code",
			wantErr: nil,
		},
		{
			name:    "minimum length",
			code:    "abc",
			wantErr: nil,
		},
		{
			name:    "maximum length",
			code:    "12345678901234567890",
			wantErr: nil,
		},
		{
			name:    "empty code",
			code:    "",
			wantErr: ErrInvalidCode,
		},
		{
			name:    "too short",
			code:    "ab",
			wantErr: ErrInvalidCode,
		},
		{
			name:    "too long",
			code:    "123456789012345678901",
			wantErr: ErrInvalidCode,
		},
		{
			name:    "invalid characters - space",
			code:    "my code",
			wantErr: ErrInvalidCode,
		},
		{
			name:    "invalid characters - special",
			code:    "my@code",
			wantErr: ErrInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := NewShortCode(tt.code)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.True(t, sc.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.code, sc.String())
				assert.False(t, sc.IsEmpty())
			}
		})
	}
}

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "default length",
			length: 0,
		},
		{
			name:   "custom length 8",
			length: 8,
		},
		{
			name:   "custom length 10",
			length: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := GenerateShortCode(tt.length)
			require.NoError(t, err)
			assert.False(t, sc.IsEmpty())

			expectedLen := tt.length
			if expectedLen <= 0 {
				expectedLen = DefaultShortCodeLength
			}
			assert.Len(t, sc.String(), expectedLen)
		})
	}
}

func TestShortCode_Equals(t *testing.T) {
	sc1, _ := NewShortCode("test123")
	sc2, _ := NewShortCode("test123")
	sc3, _ := NewShortCode("other")

	assert.True(t, sc1.Equals(sc2))
	assert.False(t, sc1.Equals(sc3))
}

func TestShortCode_String(t *testing.T) {
	sc, err := NewShortCode("mycode")
	require.NoError(t, err)
	assert.Equal(t, "mycode", sc.String())
}

func TestGenerateShortCode_Uniqueness(t *testing.T) {
	codes := make(map[string]bool)
	for range 100 {
		sc, err := GenerateShortCode(DefaultShortCodeLength)
		require.NoError(t, err)
		assert.False(t, codes[sc.String()], "duplicate code generated")
		codes[sc.String()] = true
	}
}

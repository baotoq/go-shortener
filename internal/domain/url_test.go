package domain

import (
	"testing"
	"time"

	"go-shortener/internal/domain/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewURL(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	url := NewURL(shortCode, originalURL, nil)

	assert.Equal(t, int64(0), url.ID())
	assert.Equal(t, shortCode, url.ShortCode())
	assert.Equal(t, originalURL, url.OriginalURL())
	assert.Equal(t, int64(0), url.ClickCount())
	assert.Nil(t, url.ExpiresAt())
	assert.False(t, url.CreatedAt().IsZero())
	assert.False(t, url.UpdatedAt().IsZero())
}

func TestNewURL_WithExpiration(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")
	expiresAt := time.Now().Add(24 * time.Hour)

	url := NewURL(shortCode, originalURL, &expiresAt)

	assert.NotNil(t, url.ExpiresAt())
	assert.True(t, url.HasExpiration())
}

func TestReconstructURL(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")
	expiresAt := time.Now().Add(24 * time.Hour)
	createdAt := time.Now().Add(-1 * time.Hour)
	updatedAt := time.Now()

	url := ReconstructURL(
		42,
		shortCode,
		originalURL,
		100,
		&expiresAt,
		createdAt,
		updatedAt,
	)

	assert.Equal(t, int64(42), url.ID())
	assert.Equal(t, shortCode, url.ShortCode())
	assert.Equal(t, originalURL, url.OriginalURL())
	assert.Equal(t, int64(100), url.ClickCount())
	assert.NotNil(t, url.ExpiresAt())
	assert.Equal(t, createdAt, url.CreatedAt())
	assert.Equal(t, updatedAt, url.UpdatedAt())
}

func TestURL_IsExpired(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	t.Run("no expiration", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)
		assert.False(t, url.IsExpired())
	})

	t.Run("not expired", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		url := NewURL(shortCode, originalURL, &expiresAt)
		assert.False(t, url.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)
		url := NewURL(shortCode, originalURL, &expiresAt)
		assert.True(t, url.IsExpired())
	})
}

func TestURL_CanRedirect(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	t.Run("can redirect when not expired", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)
		err := url.CanRedirect()
		assert.NoError(t, err)
	})

	t.Run("cannot redirect when expired", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)
		url := NewURL(shortCode, originalURL, &expiresAt)
		err := url.CanRedirect()
		assert.ErrorIs(t, err, ErrURLExpired)
	})
}

func TestURL_RecordClick(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	url := NewURL(shortCode, originalURL, nil)
	url.ClearEvents() // Clear URLCreated event
	initialUpdatedAt := url.UpdatedAt()

	time.Sleep(1 * time.Millisecond) // Ensure time difference
	url.RecordClick("Mozilla/5.0", "192.168.1.1", "https://google.com")

	assert.Equal(t, int64(1), url.ClickCount())
	assert.True(t, url.UpdatedAt().After(initialUpdatedAt))

	// Verify URLClicked event was raised
	events := url.Events()
	assert.Len(t, events, 1)
	clickEvent, ok := events[0].(event.URLClicked)
	assert.True(t, ok)
	assert.Equal(t, "url.clicked", clickEvent.EventName())
	assert.Equal(t, "Mozilla/5.0", clickEvent.UserAgent)
	assert.Equal(t, "192.168.1.1", clickEvent.IPAddress)

	url.ClearEvents()
	url.RecordClick("", "", "")
	assert.Equal(t, int64(2), url.ClickCount())
}

func TestURL_Redirect(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	t.Run("successful redirect", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)
		redirectURL, err := url.Redirect("Mozilla/5.0", "192.168.1.1", "")

		require.NoError(t, err)
		assert.Equal(t, "https://example.com", redirectURL)
		assert.Equal(t, int64(1), url.ClickCount())
	})

	t.Run("redirect fails when expired", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)
		url := NewURL(shortCode, originalURL, &expiresAt)
		redirectURL, err := url.Redirect("Mozilla/5.0", "192.168.1.1", "")

		assert.ErrorIs(t, err, ErrURLExpired)
		assert.Empty(t, redirectURL)
		assert.Equal(t, int64(0), url.ClickCount())
	})
}

func TestURL_SetID(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	url := NewURL(shortCode, originalURL, nil)
	assert.Equal(t, int64(0), url.ID())

	url.SetID(42)
	assert.Equal(t, int64(42), url.ID())
}

func TestURL_HasExpiration(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	t.Run("no expiration", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)
		assert.False(t, url.HasExpiration())
	})

	t.Run("with expiration", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		url := NewURL(shortCode, originalURL, &expiresAt)
		assert.True(t, url.HasExpiration())
	})
}

func TestURL_Events(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	t.Run("URLCreated event is raised on creation", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)

		events := url.Events()
		require.Len(t, events, 1)

		createdEvent, ok := events[0].(event.URLCreated)
		require.True(t, ok)
		assert.Equal(t, "url.created", createdEvent.EventName())
		assert.Equal(t, shortCode.String(), createdEvent.AggregateID())
		assert.Equal(t, shortCode.String(), createdEvent.ShortCode)
		assert.Equal(t, originalURL.String(), createdEvent.OriginalURL)
	})

	t.Run("ClearEvents removes all events", func(t *testing.T) {
		url := NewURL(shortCode, originalURL, nil)
		assert.Len(t, url.Events(), 1)

		url.ClearEvents()
		assert.Len(t, url.Events(), 0)
	})
}

func TestURL_ClickMilestone(t *testing.T) {
	shortCode, _ := NewShortCode("test123")
	originalURL, _ := NewOriginalURL("https://example.com")

	// Create URL with 99 clicks (just before milestone)
	url := ReconstructURL(1, shortCode, originalURL, 99, nil, time.Now(), time.Now())

	url.RecordClick("", "", "")

	events := url.Events()
	require.Len(t, events, 2) // URLClicked + ClickMilestoneReached

	milestoneEvent, ok := events[1].(event.ClickMilestoneReached)
	require.True(t, ok)
	assert.Equal(t, "url.milestone_reached", milestoneEvent.EventName())
	assert.Equal(t, int64(100), milestoneEvent.Milestone)
	assert.Equal(t, int64(100), milestoneEvent.ClickCount)
}

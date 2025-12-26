package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLCreated(t *testing.T) {
	e := NewURLCreated("test123", "https://example.com", nil)

	assert.Equal(t, "url.created", e.EventName())
	assert.Equal(t, "test123", e.AggregateID())
	assert.Equal(t, "test123", e.ShortCode)
	assert.Equal(t, "https://example.com", e.OriginalURL)
	assert.Nil(t, e.ExpiresAt)
	assert.False(t, e.OccurredAt().IsZero())
}

func TestURLClicked(t *testing.T) {
	e := NewURLClicked("test123", 42, "Mozilla/5.0", "192.168.1.1", "https://google.com")

	assert.Equal(t, "url.clicked", e.EventName())
	assert.Equal(t, "test123", e.AggregateID())
	assert.Equal(t, int64(42), e.ClickCount)
	assert.Equal(t, "Mozilla/5.0", e.UserAgent)
	assert.Equal(t, "192.168.1.1", e.IPAddress)
	assert.Equal(t, "https://google.com", e.Referrer)
}

func TestURLExpired(t *testing.T) {
	expiredAt := time.Now().UTC()
	e := NewURLExpired("test123", expiredAt)

	assert.Equal(t, "url.expired", e.EventName())
	assert.Equal(t, "test123", e.AggregateID())
	assert.Equal(t, expiredAt, e.ExpiredAt)
}

func TestURLDeleted(t *testing.T) {
	e := NewURLDeleted("test123")

	assert.Equal(t, "url.deleted", e.EventName())
	assert.Equal(t, "test123", e.AggregateID())
}

func TestClickMilestoneReached(t *testing.T) {
	e := NewClickMilestoneReached("test123", 1000, 1000)

	assert.Equal(t, "url.milestone_reached", e.EventName())
	assert.Equal(t, "test123", e.AggregateID())
	assert.Equal(t, int64(1000), e.Milestone)
	assert.Equal(t, int64(1000), e.ClickCount)
}

func TestCheckMilestone(t *testing.T) {
	tests := []struct {
		name          string
		previousCount int64
		currentCount  int64
		wantMilestone int64
	}{
		{"reaches 100", 99, 100, 100},
		{"reaches 1000", 999, 1000, 1000},
		{"no milestone", 50, 51, 0},
		{"skips milestone", 90, 150, 100},
		{"already past", 100, 101, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			milestone := CheckMilestone(tt.previousCount, tt.currentCount)
			assert.Equal(t, tt.wantMilestone, milestone)
		})
	}
}

func TestDispatcher(t *testing.T) {
	t.Run("dispatch to handler", func(t *testing.T) {
		dispatcher := NewDispatcher()
		var received Event

		dispatcher.Register("url.created", &mockHandler{
			fn: func(e Event) error {
				received = e
				return nil
			},
		})

		e := NewURLCreated("test", "https://example.com", nil)
		err := dispatcher.Dispatch(e)

		require.NoError(t, err)
		assert.NotNil(t, received)
	})

	t.Run("dispatch unregistered event", func(t *testing.T) {
		dispatcher := NewDispatcher()
		e := NewURLDeleted("test")

		err := dispatcher.Dispatch(e)
		assert.NoError(t, err)
	})

	t.Run("dispatch all", func(t *testing.T) {
		dispatcher := NewDispatcher()
		count := 0

		dispatcher.Register("url.clicked", &mockHandler{
			fn: func(e Event) error {
				count++
				return nil
			},
		})

		events := []Event{
			NewURLClicked("test", 1, "", "", ""),
			NewURLClicked("test", 2, "", "", ""),
		}

		err := dispatcher.DispatchAll(events)

		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

type mockHandler struct {
	fn func(Event) error
}

func (m *mockHandler) Handle(e Event) error {
	return m.fn(e)
}

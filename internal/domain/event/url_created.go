package event

import "time"

// URLCreated is raised when a new short URL is created.
type URLCreated struct {
	Base
	ShortCode   string
	OriginalURL string
	ExpiresAt   *time.Time
}

// NewURLCreated creates a new URLCreated event.
func NewURLCreated(shortCode, originalURL string, expiresAt *time.Time) URLCreated {
	return URLCreated{
		Base:        NewBase(shortCode),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		ExpiresAt:   expiresAt,
	}
}

// EventName returns the event name.
func (e URLCreated) EventName() string {
	return "url.created"
}

package event

import "time"

// Compile-time interface check
var _ Event = URLCreated{}

// URLCreated is raised when a new short URL is created.
type URLCreated struct {
	Base
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
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

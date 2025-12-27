package event

import "time"

// Compile-time interface check
var _ Event = URLExpired{}

// URLExpired is raised when a URL has expired.
type URLExpired struct {
	Base
	ShortCode string    `json:"short_code"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewURLExpired creates a new URLExpired event.
func NewURLExpired(shortCode string, expiredAt time.Time) URLExpired {
	return URLExpired{
		Base:      NewBase(shortCode),
		ShortCode: shortCode,
		ExpiredAt: expiredAt,
	}
}

// EventName returns the event name.
func (e URLExpired) EventName() string {
	return "url.expired"
}

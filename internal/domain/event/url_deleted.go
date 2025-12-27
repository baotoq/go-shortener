package event

// Compile-time interface check
var _ Event = URLDeleted{}

// URLDeleted is raised when a URL is deleted.
type URLDeleted struct {
	Base
	ShortCode string `json:"short_code"`
}

// NewURLDeleted creates a new URLDeleted event.
func NewURLDeleted(shortCode string) URLDeleted {
	return URLDeleted{
		Base:      NewBase(shortCode),
		ShortCode: shortCode,
	}
}

// EventName returns the event name.
func (e URLDeleted) EventName() string {
	return "url.deleted"
}

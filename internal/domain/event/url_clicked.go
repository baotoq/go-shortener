package event

// URLClicked is raised when a short URL is accessed for redirection.
type URLClicked struct {
	Base
	ShortCode  string
	ClickCount int64
	UserAgent  string
	IPAddress  string
	Referrer   string
}

// NewURLClicked creates a new URLClicked event.
func NewURLClicked(shortCode string, clickCount int64, userAgent, ipAddress, referrer string) URLClicked {
	return URLClicked{
		Base:       NewBase(shortCode),
		ShortCode:  shortCode,
		ClickCount: clickCount,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		Referrer:   referrer,
	}
}

// EventName returns the event name.
func (e URLClicked) EventName() string {
	return "url.clicked"
}

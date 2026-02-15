package events

import "time"

// ClickEvent represents a URL redirect click for analytics tracking.
// Published by URL Service, consumed by Analytics Service.
type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	Timestamp time.Time `json:"timestamp"`
}

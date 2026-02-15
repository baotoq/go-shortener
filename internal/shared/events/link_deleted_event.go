package events

import "time"

// LinkDeletedEvent represents a link deletion for analytics cleanup.
// Published by URL Service, consumed by Analytics Service.
type LinkDeletedEvent struct {
	ShortCode string    `json:"short_code"`
	DeletedAt time.Time `json:"deleted_at"`
}

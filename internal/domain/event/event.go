package event

import "time"

// Event is the base interface for all domain events.
type Event interface {
	// EventName returns the name of the event.
	EventName() string
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// AggregateID returns the ID of the aggregate that raised the event.
	AggregateID() string
}

// Base contains common fields for all events.
type Base struct {
	occurredAt  time.Time
	aggregateID string
}

// NewBase creates a new base event.
func NewBase(aggregateID string) Base {
	return Base{
		occurredAt:  time.Now().UTC(),
		aggregateID: aggregateID,
	}
}

// OccurredAt returns when the event occurred.
func (e Base) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the aggregate ID.
func (e Base) AggregateID() string {
	return e.aggregateID
}

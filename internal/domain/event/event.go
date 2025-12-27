package event

import (
	"time"

	"github.com/google/uuid"
)

// Event is the base interface for all domain events.
type Event interface {
	// EventID returns the unique identifier of the event.
	EventID() string
	// EventName returns the name of the event.
	EventName() string
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// AggregateID returns the ID of the aggregate that raised the event.
	AggregateID() string
}

// Base contains common fields for all events.
type Base struct {
	ID          string    `json:"event_id"`
	OccurredAtT time.Time `json:"occurred_at"`
	AggregateId string    `json:"aggregate_id"`
}

// NewBase creates a new base event.
func NewBase(aggregateID string) Base {
	return Base{
		ID:          uuid.Must(uuid.NewV7()).String(),
		OccurredAtT: time.Now().UTC(),
		AggregateId: aggregateID,
	}
}

// EventID returns the unique identifier of the event.
func (e Base) EventID() string {
	return e.ID
}

// OccurredAt returns when the event occurred.
func (e Base) OccurredAt() time.Time {
	return e.OccurredAtT
}

// AggregateID returns the aggregate ID.
func (e Base) AggregateID() string {
	return e.AggregateId
}

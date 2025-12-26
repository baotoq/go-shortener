package domain

import "go-shortener/internal/domain/event"

// AggregateRoot is the interface for domain aggregates that can raise events.
type AggregateRoot interface {
	// Events returns all uncommitted domain events.
	Events() []event.Event
	// ClearEvents clears all domain events after dispatch.
	ClearEvents()
}

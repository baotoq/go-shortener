package domain

import (
	"context"

	"go-shortener/internal/domain/event"
)

// UnitOfWork manages database transactions and domain event dispatching.
type UnitOfWork interface {
	// Do executes the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// If successful, domain events from provided aggregates are dispatched.
	Do(ctx context.Context, fn func(ctx context.Context) error, aggregates ...AggregateRoot) error
}

// AggregateRoot is the interface for domain aggregates that can raise events.
type AggregateRoot interface {
	// Events returns all uncommitted domain events.
	Events() []event.Event
	// ClearEvents clears all domain events after dispatch.
	ClearEvents()
}

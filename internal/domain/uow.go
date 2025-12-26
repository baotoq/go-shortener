package domain

//go:generate mockery --name=UnitOfWork --output=../mocks --outpkg=mocks --with-expecter

import "context"

// UnitOfWork manages database transactions and domain event dispatching.
type UnitOfWork interface {
	// Do executes the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// If successful, domain events from provided aggregates are dispatched.
	Do(ctx context.Context, fn func(ctx context.Context) error, aggregates ...AggregateRoot) error
}

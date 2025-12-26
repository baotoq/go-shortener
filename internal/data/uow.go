package data

import (
	"context"

	"go-shortener/ent"
	"go-shortener/internal/domain"
	"go-shortener/internal/domain/event"

	"github.com/go-kratos/kratos/v2/log"
)

type txKey struct{}

// unitOfWork implements domain.UnitOfWork with transaction support.
type unitOfWork struct {
	db         *ent.Client
	dispatcher *event.Dispatcher
	log        *log.Helper
}

// NewUnitOfWork creates a new UnitOfWork.
func NewUnitOfWork(data *Data, dispatcher *event.Dispatcher, logger log.Logger) domain.UnitOfWork {
	return &unitOfWork{
		db:         data.db,
		dispatcher: dispatcher,
		log:        log.NewHelper(logger),
	}
}

// Do executes the function within a database transaction.
// Events are dispatched only after successful commit.
func (u *unitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error, aggregates ...domain.AggregateRoot) error {
	tx, err := u.db.Tx(ctx)
	if err != nil {
		return err
	}

	// Store tx in context for repositories to use
	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			u.log.WithContext(ctx).Errorf("rollback failed: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Dispatch events after successful commit
	u.dispatchEvents(aggregates)

	return nil
}

// dispatchEvents dispatches all events from aggregates.
func (u *unitOfWork) dispatchEvents(aggregates []domain.AggregateRoot) {
	for _, aggregate := range aggregates {
		events := aggregate.Events()
		if len(events) == 0 {
			continue
		}

		if err := u.dispatcher.DispatchAll(events); err != nil {
			u.log.Errorf("failed to dispatch events: %v", err)
		}

		aggregate.ClearEvents()
	}
}

// TxFromContext retrieves the transaction from context.
func TxFromContext(ctx context.Context) *ent.Tx {
	tx, _ := ctx.Value(txKey{}).(*ent.Tx)
	return tx
}

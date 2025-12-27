package data

import (
	"context"

	"go-shortener/ent"
	"go-shortener/internal/domain"
	"go-shortener/internal/domain/event"
	"go-shortener/internal/infra/eventbus"

	"github.com/go-kratos/kratos/v2/log"
)

// Compile-time interface check
var _ domain.UnitOfWork = (*unitOfWork)(nil)

type txKey struct{}

// unitOfWork implements domain.UnitOfWork with transaction support and outbox pattern.
type unitOfWork struct {
	db     *ent.Client
	outbox *eventbus.OutboxPublisher
	log    *log.Helper
}

// NewUnitOfWork creates a new UnitOfWork.
func NewUnitOfWork(data *Data, outbox *eventbus.OutboxPublisher, logger log.Logger) domain.UnitOfWork {
	return &unitOfWork{
		db:     data.db,
		outbox: outbox,
		log:    log.NewHelper(logger),
	}
}

// Do executes the function within a database transaction.
// Events are stored in the outbox table within the same transaction.
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

	// Store events in outbox within the same transaction
	if err := u.storeEventsInOutbox(txCtx, tx, aggregates); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			u.log.WithContext(ctx).Errorf("rollback failed: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Clear events after successful commit
	for _, aggregate := range aggregates {
		aggregate.ClearEvents()
	}

	return nil
}

// storeEventsInOutbox stores all events from aggregates in the outbox table.
func (u *unitOfWork) storeEventsInOutbox(ctx context.Context, tx *ent.Tx, aggregates []domain.AggregateRoot) error {
	var events []event.Event
	for _, aggregate := range aggregates {
		events = append(events, aggregate.Events()...)
	}

	if len(events) == 0 {
		return nil
	}

	return u.outbox.PublishInTx(ctx, tx, events)
}

// TxFromContext retrieves the transaction from context.
func TxFromContext(ctx context.Context) *ent.Tx {
	tx, _ := ctx.Value(txKey{}).(*ent.Tx)
	return tx
}

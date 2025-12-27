package eventbus

import (
	"context"

	"go-shortener/ent"
	"go-shortener/internal/domain/event"

	"github.com/ThreeDotsLabs/watermill/message"
)

// OutboxPublisher publishes events to the outbox table within a transaction.
type OutboxPublisher struct {
	db *ent.Client
}

// NewOutboxPublisher creates a new outbox publisher.
func NewOutboxPublisher(db *ent.Client) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// PublishInTx stores events in the outbox table using the provided transaction.
func (p *OutboxPublisher) PublishInTx(ctx context.Context, tx *ent.Tx, events []event.Event) error {
	for _, e := range events {
		msg, err := EventToMessage(e)
		if err != nil {
			return err
		}

		if err := p.storeMessage(ctx, tx, msg); err != nil {
			return err
		}
	}
	return nil
}

// storeMessage stores a Watermill message in the outbox table.
func (p *OutboxPublisher) storeMessage(ctx context.Context, tx *ent.Tx, msg *message.Message) error {
	metadata := make(map[string]string)
	for k, v := range msg.Metadata {
		metadata[k] = v
	}

	return tx.OutboxMessage.Create().
		SetUUID(msg.UUID).
		SetPayload(msg.Payload).
		SetMetadata(metadata).
		Exec(ctx)
}

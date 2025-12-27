package eventbus

import (
	"context"
	"sync"
	"time"

	"go-shortener/ent"
	"go-shortener/ent/outboxmessage"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	defaultPollInterval = 100 * time.Millisecond
	defaultBatchSize    = 100
)

// Forwarder reads messages from the outbox table and forwards them to the event bus.
type Forwarder struct {
	db           *ent.Client
	publisher    message.Publisher
	topic        string
	pollInterval time.Duration
	batchSize    int
	logger       watermill.LoggerAdapter

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewForwarder creates a new outbox forwarder.
func NewForwarder(
	db *ent.Client,
	publisher message.Publisher,
	logger watermill.LoggerAdapter,
) *Forwarder {
	return &Forwarder{
		db:           db,
		publisher:    publisher,
		topic:        URLEventsTopic,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		logger:       logger,
	}
}

// Start begins forwarding messages from the outbox.
func (f *Forwarder) Start(ctx context.Context) {
	f.ctx, f.cancel = context.WithCancel(ctx)
	f.wg.Add(1)
	go f.run()
	f.logger.Info("outbox forwarder started", nil)
}

// Stop stops the forwarder gracefully.
func (f *Forwarder) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()
	f.logger.Info("outbox forwarder stopped", nil)
}

func (f *Forwarder) run() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			f.forwardBatch()
		}
	}
}

func (f *Forwarder) forwardBatch() {
	messages, err := f.db.OutboxMessage.Query().
		Order(ent.Asc(outboxmessage.FieldCreatedAt)).
		Limit(f.batchSize).
		All(f.ctx)

	if err != nil {
		f.logger.Error("failed to query outbox messages", err, nil)
		return
	}

	for _, om := range messages {
		if err := f.forwardMessage(om); err != nil {
			f.logger.Error("failed to forward message", err, watermill.LogFields{
				"uuid": om.UUID,
			})
			continue
		}

		// Delete the message after successful forwarding
		if err := f.db.OutboxMessage.DeleteOne(om).Exec(f.ctx); err != nil {
			f.logger.Error("failed to delete outbox message", err, watermill.LogFields{
				"uuid": om.UUID,
			})
		}
	}
}

func (f *Forwarder) forwardMessage(om *ent.OutboxMessage) error {
	msg := message.NewMessage(om.UUID, om.Payload)
	for k, v := range om.Metadata {
		msg.Metadata.Set(k, v)
	}

	if err := f.publisher.Publish(f.topic, msg); err != nil {
		return err
	}

	f.logger.Debug("forwarded message", watermill.LogFields{
		"uuid":       om.UUID,
		"event_name": om.Metadata["event_name"],
	})

	return nil
}

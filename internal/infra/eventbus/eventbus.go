package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"go-shortener/internal/domain/event"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

const (
	// URLEventsTopic is the topic for all URL-related events.
	URLEventsTopic = "url.events"
)

// EventBus wraps Watermill pub/sub for domain events.
type EventBus struct {
	pubsub    *gochannel.GoChannel
	publisher message.Publisher
	logger    watermill.LoggerAdapter
}

// NewEventBus creates a new event bus using Go channels.
func NewEventBus(logger watermill.LoggerAdapter) *EventBus {
	pubsub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: 100,
			Persistent:          false,
		},
		logger,
	)

	return &EventBus{
		pubsub:    pubsub,
		publisher: pubsub,
		logger:    logger,
	}
}

// Publisher returns the Watermill publisher.
func (b *EventBus) Publisher() message.Publisher {
	return b.publisher
}

// Subscriber returns the Watermill subscriber.
func (b *EventBus) Subscriber() message.Subscriber {
	return b.pubsub
}

// Publish publishes a domain event to the event bus.
func (b *EventBus) Publish(ctx context.Context, e event.Event) error {
	msg, err := EventToMessage(e)
	if err != nil {
		return err
	}
	return b.publisher.Publish(URLEventsTopic, msg)
}

// PublishAll publishes multiple domain events.
func (b *EventBus) PublishAll(ctx context.Context, events []event.Event) error {
	for _, e := range events {
		if err := b.Publish(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the event bus.
func (b *EventBus) Close() error {
	return b.pubsub.Close()
}

// EventEnvelope wraps a domain event for serialization.
type EventEnvelope struct {
	EventID     string          `json:"event_id"`
	EventName   string          `json:"event_name"`
	AggregateID string          `json:"aggregate_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Payload     json.RawMessage `json:"payload"`
}

// EventToMessage converts a domain event to a Watermill message.
func EventToMessage(e event.Event) (*message.Message, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	envelope := EventEnvelope{
		EventID:     e.EventID(),
		EventName:   e.EventName(),
		AggregateID: e.AggregateID(),
		OccurredAt:  e.OccurredAt(),
		Payload:     payload,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(e.EventID(), data)
	msg.Metadata.Set("event_name", e.EventName())
	msg.Metadata.Set("aggregate_id", e.AggregateID())

	return msg, nil
}

// MessageToEnvelope extracts the event envelope from a Watermill message.
func MessageToEnvelope(msg *message.Message) (*EventEnvelope, error) {
	var envelope EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

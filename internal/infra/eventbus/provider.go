package eventbus

import (
	"go-shortener/ent"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is eventbus providers.
var ProviderSet = wire.NewSet(
	NewKratosLoggerAdapter,
	NewEventBus,
	NewRouter,
	ProvideOutboxPublisher,
	ProvideForwarder,
)

// ProvideOutboxPublisher creates an OutboxPublisher from Data's db client.
func ProvideOutboxPublisher(db *ent.Client) *OutboxPublisher {
	return NewOutboxPublisher(db)
}

// ProvideForwarder creates a Forwarder.
func ProvideForwarder(db *ent.Client, eventBus *EventBus, logger log.Logger) *Forwarder {
	return NewForwarder(db, eventBus.Publisher(), NewKratosLoggerAdapter(logger))
}

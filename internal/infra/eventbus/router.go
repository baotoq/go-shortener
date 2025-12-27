package eventbus

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// EventHandler handles events from the event bus.
type EventHandler interface {
	// HandlerName returns the name of the handler.
	HandlerName() string
	// EventName returns the event name this handler handles.
	EventName() string
	// Handle processes the event envelope.
	Handle(ctx context.Context, envelope *EventEnvelope) error
}

// Router routes messages to event handlers.
type Router struct {
	router   *message.Router
	eventBus *EventBus
	handlers []EventHandler
	logger   watermill.LoggerAdapter
}

// NewRouter creates a new event router.
func NewRouter(eventBus *EventBus, logger watermill.LoggerAdapter) (*Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	return &Router{
		router:   router,
		eventBus: eventBus,
		handlers: make([]EventHandler, 0),
		logger:   logger,
	}, nil
}

// AddHandler registers an event handler.
func (r *Router) AddHandler(handler EventHandler) {
	r.handlers = append(r.handlers, handler)

	r.router.AddNoPublisherHandler(
		handler.HandlerName(),
		URLEventsTopic,
		r.eventBus.Subscriber(),
		r.createHandlerFunc(handler),
	)
}

// createHandlerFunc creates a Watermill handler function for an event handler.
func (r *Router) createHandlerFunc(handler EventHandler) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		envelope, err := MessageToEnvelope(msg)
		if err != nil {
			r.logger.Error("failed to parse message", err, nil)
			return nil // Don't retry on parse errors
		}

		// Only handle events matching the handler's event name
		if envelope.EventName != handler.EventName() {
			return nil
		}

		if err := handler.Handle(msg.Context(), envelope); err != nil {
			r.logger.Error("failed to handle event", err, watermill.LogFields{
				"handler":    handler.HandlerName(),
				"event_name": envelope.EventName,
				"event_id":   envelope.EventID,
			})
			return err
		}

		return nil
	}
}

// Run starts the router.
func (r *Router) Run(ctx context.Context) error {
	return r.router.Run(ctx)
}

// Running returns a channel that is closed when the router is running.
func (r *Router) Running() chan struct{} {
	return r.router.Running()
}

// Close stops the router.
func (r *Router) Close() error {
	return r.router.Close()
}

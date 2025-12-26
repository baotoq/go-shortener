package data

import (
	"go-shortener/internal/domain/event"

	"github.com/go-kratos/kratos/v2/log"
)

// LoggingEventHandler logs all domain events.
type LoggingEventHandler struct {
	log *log.Helper
}

// NewLoggingEventHandler creates a new logging event handler.
func NewLoggingEventHandler(logger log.Logger) *LoggingEventHandler {
	return &LoggingEventHandler{
		log: log.NewHelper(logger),
	}
}

// Handle logs the event details.
func (h *LoggingEventHandler) Handle(e event.Event) error {
	switch evt := e.(type) {
	case event.URLCreated:
		h.log.Infof("[Event] URL created: %s -> %s", evt.ShortCode, evt.OriginalURL)
	case event.URLClicked:
		h.log.Infof("[Event] URL clicked: %s (count: %d, ip: %s)", evt.ShortCode, evt.ClickCount, evt.IPAddress)
	case event.URLDeleted:
		h.log.Infof("[Event] URL deleted: %s", evt.ShortCode)
	case event.URLExpired:
		h.log.Infof("[Event] URL expired: %s at %s", evt.ShortCode, evt.ExpiredAt)
	case event.ClickMilestoneReached:
		h.log.Infof("[Event] Milestone reached: %s hit %d clicks!", evt.ShortCode, evt.Milestone)
	default:
		h.log.Infof("[Event] %s: %s", e.EventName(), e.AggregateID())
	}
	return nil
}

// NewEventDispatcher creates and configures an event dispatcher with handlers.
func NewEventDispatcher(logger log.Logger) *event.Dispatcher {
	dispatcher := event.NewDispatcher()
	loggingHandler := NewLoggingEventHandler(logger)

	// Register logging handler for all event types
	dispatcher.Register("url.created", loggingHandler)
	dispatcher.Register("url.clicked", loggingHandler)
	dispatcher.Register("url.deleted", loggingHandler)
	dispatcher.Register("url.expired", loggingHandler)
	dispatcher.Register("url.milestone_reached", loggingHandler)

	return dispatcher
}

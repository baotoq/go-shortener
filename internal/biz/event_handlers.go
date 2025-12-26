package biz

import (
	"context"

	"go-shortener/internal/domain"
	"go-shortener/internal/domain/event"

	"github.com/go-kratos/kratos/v2/log"
)

// Compile-time interface checks
var (
	_ event.Handler = (*LoggingEventHandler)(nil)
	_ event.Handler = (*ClickEventHandler)(nil)
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

// ClickEventHandler handles URLClicked events and increments click count.
type ClickEventHandler struct {
	repo domain.URLRepository
	log  *log.Helper
}

// NewClickEventHandler creates a new click event handler.
func NewClickEventHandler(repo domain.URLRepository, logger log.Logger) *ClickEventHandler {
	return &ClickEventHandler{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Handle increments the click count when a URLClicked event is received.
func (h *ClickEventHandler) Handle(e event.Event) error {
	evt, ok := e.(event.URLClicked)
	if !ok {
		return nil
	}

	sc, err := domain.NewShortCode(evt.ShortCode)
	if err != nil {
		h.log.Warnf("Invalid short code in URLClicked event: %s", evt.ShortCode)
		return nil
	}

	if err := h.repo.IncrementClickCount(context.Background(), sc); err != nil {
		h.log.Warnf("Failed to increment click count for %s: %v", evt.ShortCode, err)
		return err
	}

	return nil
}

// NewEventDispatcher creates and configures an event dispatcher with handlers.
func NewEventDispatcher(repo domain.URLRepository, logger log.Logger) *event.Dispatcher {
	dispatcher := event.NewDispatcher()
	loggingHandler := NewLoggingEventHandler(logger)
	clickHandler := NewClickEventHandler(repo, logger)

	// Register logging handler for all event types
	dispatcher.Register("url.created", loggingHandler)
	dispatcher.Register("url.clicked", loggingHandler)
	dispatcher.Register("url.deleted", loggingHandler)
	dispatcher.Register("url.expired", loggingHandler)
	dispatcher.Register("url.milestone_reached", loggingHandler)

	// Register click handler for click events
	dispatcher.Register("url.clicked", clickHandler)

	return dispatcher
}

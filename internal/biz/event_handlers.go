package biz

import (
	"context"
	"encoding/json"

	"go-shortener/internal/domain"
	"go-shortener/internal/domain/event"
	"go-shortener/internal/infra/eventbus"

	"github.com/go-kratos/kratos/v2/log"
)

// Compile-time interface checks
var (
	_ eventbus.EventHandler = (*LoggingEventHandler)(nil)
	_ eventbus.EventHandler = (*ClickEventHandler)(nil)
)

// LoggingEventHandler logs all domain events.
type LoggingEventHandler struct {
	log       *log.Helper
	eventName string
}

// NewLoggingEventHandler creates a new logging event handler.
func NewLoggingEventHandler(logger log.Logger, eventName string) *LoggingEventHandler {
	return &LoggingEventHandler{
		log:       log.NewHelper(logger),
		eventName: eventName,
	}
}

func (h *LoggingEventHandler) HandlerName() string {
	return "logging_handler_" + h.eventName
}

func (h *LoggingEventHandler) EventName() string {
	return h.eventName
}

// Handle logs the event details.
func (h *LoggingEventHandler) Handle(ctx context.Context, envelope *eventbus.EventEnvelope) error {
	switch envelope.EventName {
	case "url.created":
		var evt event.URLCreated
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			return err
		}
		h.log.Infof("[Event] URL created: %s -> %s", evt.ShortCode, evt.OriginalURL)
	case "url.clicked":
		var evt event.URLClicked
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			return err
		}
		h.log.Infof("[Event] URL clicked: %s (count: %d, ip: %s)", evt.ShortCode, evt.ClickCount, evt.IPAddress)
	case "url.deleted":
		var evt event.URLDeleted
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			return err
		}
		h.log.Infof("[Event] URL deleted: %s", evt.ShortCode)
	case "url.expired":
		var evt event.URLExpired
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			return err
		}
		h.log.Infof("[Event] URL expired: %s at %s", evt.ShortCode, evt.ExpiredAt)
	case "url.milestone_reached":
		var evt event.ClickMilestoneReached
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			return err
		}
		h.log.Infof("[Event] Milestone reached: %s hit %d clicks!", evt.ShortCode, evt.Milestone)
	default:
		h.log.Infof("[Event] %s: %s", envelope.EventName, envelope.AggregateID)
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

func (h *ClickEventHandler) HandlerName() string {
	return "click_handler"
}

func (h *ClickEventHandler) EventName() string {
	return "url.clicked"
}

// Handle increments the click count when a URLClicked event is received.
func (h *ClickEventHandler) Handle(ctx context.Context, envelope *eventbus.EventEnvelope) error {
	var evt event.URLClicked
	if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
		h.log.Warnf("failed to unmarshal URLClicked event: %v", err)
		return nil
	}

	sc, err := domain.NewShortCode(evt.ShortCode)
	if err != nil {
		h.log.Warnf("Invalid short code in URLClicked event: %s", evt.ShortCode)
		return nil
	}

	if err := h.repo.IncrementClickCount(ctx, sc); err != nil {
		h.log.Warnf("Failed to increment click count for %s: %v", evt.ShortCode, err)
		return err
	}

	return nil
}

// RegisterEventHandlers registers all event handlers with the router.
func RegisterEventHandlers(router *eventbus.Router, repo domain.URLRepository, logger log.Logger) {
	eventNames := []string{
		"url.created",
		"url.clicked",
		"url.deleted",
		"url.expired",
		"url.milestone_reached",
	}

	// Register logging handlers for all event types
	for _, eventName := range eventNames {
		router.AddHandler(NewLoggingEventHandler(logger, eventName))
	}

	// Register click handler
	router.AddHandler(NewClickEventHandler(repo, logger))
}

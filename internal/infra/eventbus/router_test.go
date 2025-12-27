package eventbus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-shortener/internal/domain/event"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/suite"
)

var handlerCounter atomic.Int64

type RouterTestSuite struct {
	suite.Suite
	eventBus *EventBus
	sut      *Router
	logger   watermill.LoggerAdapter
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}

func (s *RouterTestSuite) SetupTest() {
	s.logger = watermill.NopLogger{}
	s.eventBus = NewEventBus(s.logger)

	var err error
	s.sut, err = NewRouter(s.eventBus, s.logger)
	s.Require().NoError(err)
}

func (s *RouterTestSuite) TearDownTest() {
	if s.sut != nil {
		s.sut.Close()
	}
	if s.eventBus != nil {
		s.eventBus.Close()
	}
}

// mockHandler is a test event handler.
type mockHandler struct {
	name      string
	eventName string
	received  []*EventEnvelope
	mu        sync.Mutex
	wg        sync.WaitGroup
}

func newMockHandler(eventName string) *mockHandler {
	id := handlerCounter.Add(1)
	return &mockHandler{
		name:      "mock_handler_" + eventName + "_" + string(rune(id)),
		eventName: eventName,
		received:  make([]*EventEnvelope, 0),
	}
}

func (h *mockHandler) HandlerName() string {
	return h.name
}

func (h *mockHandler) EventName() string {
	return h.eventName
}

func (h *mockHandler) Handle(ctx context.Context, envelope *EventEnvelope) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.received = append(h.received, envelope)
	h.wg.Done()
	return nil
}

func (h *mockHandler) ReceivedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.received)
}

func (h *mockHandler) ExpectMessages(count int) {
	h.wg.Add(count)
}

func (h *mockHandler) Wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *RouterTestSuite) TestAddHandler() {
	// Arrange
	handler := newMockHandler("url.created")

	// Act
	s.sut.AddHandler(handler)

	// Assert - no panic means success
}

func (s *RouterTestSuite) TestRouterHandlesEvent() {
	// Arrange
	handler := newMockHandler("url.created")
	handler.ExpectMessages(1)
	s.sut.AddHandler(handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.sut.Run(ctx)
	<-s.sut.Running()
	evt := event.NewURLCreated("abc123", "https://example.com", nil)

	// Act
	err := s.eventBus.Publish(ctx, evt)

	// Assert
	s.Require().NoError(err)
	received := handler.Wait(2 * time.Second)
	s.True(received, "handler should receive the event")
	s.Equal(1, handler.ReceivedCount())
}

func (s *RouterTestSuite) TestRouterFiltersEventsByName() {
	// Arrange
	createdHandler := newMockHandler("url.created")
	clickedHandler := newMockHandler("url.clicked")
	createdHandler.ExpectMessages(1)
	s.sut.AddHandler(createdHandler)
	s.sut.AddHandler(clickedHandler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.sut.Run(ctx)
	<-s.sut.Running()
	evt := event.NewURLCreated("abc123", "https://example.com", nil)

	// Act
	err := s.eventBus.Publish(ctx, evt)

	// Assert
	s.Require().NoError(err)
	received := createdHandler.Wait(2 * time.Second)
	s.True(received)
	time.Sleep(100 * time.Millisecond)
	s.Equal(1, createdHandler.ReceivedCount())
	s.Equal(0, clickedHandler.ReceivedCount())
}

func (s *RouterTestSuite) TestMultipleHandlersForSameEvent() {
	// Arrange
	handler1 := newMockHandler("url.clicked")
	handler2 := newMockHandler("url.clicked")
	handler1.ExpectMessages(1)
	handler2.ExpectMessages(1)
	s.sut.AddHandler(handler1)
	s.sut.AddHandler(handler2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.sut.Run(ctx)
	<-s.sut.Running()
	evt := event.NewURLClicked("abc123", 1, "Mozilla", "127.0.0.1", "")

	// Act
	err := s.eventBus.Publish(ctx, evt)

	// Assert
	s.Require().NoError(err)
	s.True(handler1.Wait(2 * time.Second))
	s.True(handler2.Wait(2 * time.Second))
	s.Equal(1, handler1.ReceivedCount())
	s.Equal(1, handler2.ReceivedCount())
}

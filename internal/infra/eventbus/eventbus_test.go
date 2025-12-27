package eventbus

import (
	"context"
	"testing"
	"time"

	"go-shortener/internal/domain/event"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/suite"
)

type EventBusTestSuite struct {
	suite.Suite
	sut    *EventBus
	logger watermill.LoggerAdapter
}

func TestEventBusTestSuite(t *testing.T) {
	suite.Run(t, new(EventBusTestSuite))
}

func (s *EventBusTestSuite) SetupTest() {
	s.logger = watermill.NopLogger{}
	s.sut = NewEventBus(s.logger)
}

func (s *EventBusTestSuite) TearDownTest() {
	if s.sut != nil {
		s.sut.Close()
	}
}

func (s *EventBusTestSuite) TestPublish() {
	// Arrange
	ctx := context.Background()
	evt := event.NewURLCreated("abc123", "https://example.com", nil)

	// Act
	err := s.sut.Publish(ctx, evt)

	// Assert
	s.NoError(err)
}

func (s *EventBusTestSuite) TestPublishAll() {
	// Arrange
	ctx := context.Background()
	events := []event.Event{
		event.NewURLCreated("abc123", "https://example.com", nil),
		event.NewURLClicked("abc123", 1, "Mozilla", "127.0.0.1", ""),
	}

	// Act
	err := s.sut.PublishAll(ctx, events)

	// Assert
	s.NoError(err)
}

func (s *EventBusTestSuite) TestEventToMessage() {
	// Arrange
	evt := event.NewURLCreated("abc123", "https://example.com", nil)

	// Act
	msg, err := EventToMessage(evt)

	// Assert
	s.NoError(err)
	s.NotNil(msg)
	s.Equal(evt.EventID(), msg.UUID)
	s.Equal("url.created", msg.Metadata.Get("event_name"))
	s.Equal("abc123", msg.Metadata.Get("aggregate_id"))
}

func (s *EventBusTestSuite) TestMessageToEnvelope() {
	// Arrange
	evt := event.NewURLCreated("abc123", "https://example.com", nil)
	msg, err := EventToMessage(evt)
	s.Require().NoError(err)

	// Act
	envelope, err := MessageToEnvelope(msg)

	// Assert
	s.NoError(err)
	s.NotNil(envelope)
	s.Equal(evt.EventID(), envelope.EventID)
	s.Equal("url.created", envelope.EventName)
	s.Equal("abc123", envelope.AggregateID)
}

func (s *EventBusTestSuite) TestPublishAndSubscribe() {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	messages, err := s.sut.Subscriber().Subscribe(ctx, URLEventsTopic)
	s.Require().NoError(err)
	evt := event.NewURLCreated("test123", "https://example.com", nil)

	// Act
	err = s.sut.Publish(ctx, evt)
	s.Require().NoError(err)

	// Assert
	select {
	case msg := <-messages:
		envelope, err := MessageToEnvelope(msg)
		s.NoError(err)
		s.Equal("url.created", envelope.EventName)
		s.Equal("test123", envelope.AggregateID)
		msg.Ack()
	case <-ctx.Done():
		s.Fail("timeout waiting for message")
	}
}

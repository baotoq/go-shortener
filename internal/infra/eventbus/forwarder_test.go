package eventbus

import (
	"context"
	"testing"
	"time"

	"go-shortener/ent"
	"go-shortener/ent/enttest"
	"go-shortener/internal/domain/event"

	"github.com/ThreeDotsLabs/watermill"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type ForwarderTestSuite struct {
	suite.Suite
	client   *ent.Client
	eventBus *EventBus
	sut      *Forwarder
	outbox   *OutboxPublisher
	logger   watermill.LoggerAdapter
}

func TestForwarderTestSuite(t *testing.T) {
	suite.Run(t, new(ForwarderTestSuite))
}

func (s *ForwarderTestSuite) SetupTest() {
	s.client = enttest.Open(s.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	s.logger = watermill.NopLogger{}
	s.eventBus = NewEventBus(s.logger)
	s.outbox = NewOutboxPublisher(s.client)
	s.sut = NewForwarder(s.client, s.eventBus.Publisher(), s.logger)
}

func (s *ForwarderTestSuite) TearDownTest() {
	if s.sut != nil {
		s.sut.Stop()
	}
	if s.eventBus != nil {
		s.eventBus.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
}

func (s *ForwarderTestSuite) TestForwarderForwardsMessages() {
	// Arrange
	ctx := context.Background()
	messages, err := s.eventBus.Subscriber().Subscribe(ctx, URLEventsTopic)
	s.Require().NoError(err)
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	evt := event.NewURLCreated("abc123", "https://example.com", nil)
	err = s.outbox.PublishInTx(ctx, tx, []event.Event{evt})
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)

	// Act
	s.sut.Start(ctx)

	// Assert
	select {
	case msg := <-messages:
		envelope, err := MessageToEnvelope(msg)
		s.NoError(err)
		s.Equal("url.created", envelope.EventName)
		s.Equal("abc123", envelope.AggregateID)
		msg.Ack()
	case <-time.After(2 * time.Second):
		s.Fail("timeout waiting for forwarded message")
	}
	time.Sleep(200 * time.Millisecond)
	outboxMessages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(outboxMessages, 0)
}

func (s *ForwarderTestSuite) TestForwarderDeletesAfterForwarding() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	events := []event.Event{
		event.NewURLCreated("test1", "https://example1.com", nil),
		event.NewURLCreated("test2", "https://example2.com", nil),
	}
	err = s.outbox.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)
	outboxMessages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(outboxMessages, 2)

	// Act
	s.sut.Start(ctx)
	time.Sleep(500 * time.Millisecond)

	// Assert
	outboxMessages, err = s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(outboxMessages, 0)
}

func (s *ForwarderTestSuite) TestForwarderStartStop() {
	// Arrange
	ctx := context.Background()

	// Act & Assert - no panic means success
	s.sut.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	s.sut.Stop()
}

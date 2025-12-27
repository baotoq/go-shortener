package eventbus

import (
	"context"
	"testing"

	"go-shortener/ent"
	"go-shortener/ent/enttest"
	"go-shortener/internal/domain/event"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type OutboxTestSuite struct {
	suite.Suite
	client *ent.Client
	sut    *OutboxPublisher
}

func TestOutboxTestSuite(t *testing.T) {
	suite.Run(t, new(OutboxTestSuite))
}

func (s *OutboxTestSuite) SetupTest() {
	s.client = enttest.Open(s.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	s.sut = NewOutboxPublisher(s.client)
}

func (s *OutboxTestSuite) TearDownTest() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *OutboxTestSuite) TestPublishInTx_SingleEvent() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	events := []event.Event{
		event.NewURLCreated("abc123", "https://example.com", nil),
	}

	// Act
	err = s.sut.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)

	// Assert
	messages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(messages, 1)
	s.NotEmpty(messages[0].UUID)
	s.NotEmpty(messages[0].Payload)
	s.Equal("url.created", messages[0].Metadata["event_name"])
}

func (s *OutboxTestSuite) TestPublishInTx_MultipleEvents() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	events := []event.Event{
		event.NewURLCreated("abc123", "https://example.com", nil),
		event.NewURLClicked("abc123", 1, "Mozilla", "127.0.0.1", ""),
		event.NewURLDeleted("abc123"),
	}

	// Act
	err = s.sut.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)

	// Assert
	messages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(messages, 3)
}

func (s *OutboxTestSuite) TestPublishInTx_RollbackClearsEvents() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	events := []event.Event{
		event.NewURLCreated("abc123", "https://example.com", nil),
	}

	// Act
	err = s.sut.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Rollback()
	s.Require().NoError(err)

	// Assert
	messages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(messages, 0)
}

func (s *OutboxTestSuite) TestPublishInTx_EmptyEvents() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	events := []event.Event{}

	// Act
	err = s.sut.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)

	// Assert
	messages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(messages, 0)
}

func (s *OutboxTestSuite) TestPublishInTx_PreservesEventMetadata() {
	// Arrange
	ctx := context.Background()
	tx, err := s.client.Tx(ctx)
	s.Require().NoError(err)
	evt := event.NewURLClicked("test123", 42, "Chrome", "192.168.1.1", "https://google.com")
	events := []event.Event{evt}

	// Act
	err = s.sut.PublishInTx(ctx, tx, events)
	s.Require().NoError(err)
	err = tx.Commit()
	s.Require().NoError(err)

	// Assert
	messages, err := s.client.OutboxMessage.Query().All(ctx)
	s.NoError(err)
	s.Len(messages, 1)
	s.Equal(evt.EventID(), messages[0].UUID)
	s.Equal("url.clicked", messages[0].Metadata["event_name"])
	s.Equal("test123", messages[0].Metadata["aggregate_id"])
}

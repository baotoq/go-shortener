package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OutboxMessage holds the schema definition for the OutboxMessage entity.
// This is used by Watermill's SQL forwarder for the outbox pattern.
type OutboxMessage struct {
	ent.Schema
}

// Fields of the OutboxMessage.
func (OutboxMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("uuid").
			Unique().
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(36)",
				dialect.SQLite:   "varchar(36)",
			}),
		field.Bytes("payload").
			NotEmpty(),
		field.JSON("metadata", map[string]string{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Indexes of the OutboxMessage.
func (OutboxMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
	}
}

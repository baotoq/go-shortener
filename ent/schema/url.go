package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// URL holds the schema definition for the URL entity.
type URL struct {
	ent.Schema
}

// Fields of the URL.
func (URL) Fields() []ent.Field {
	return []ent.Field{
		field.String("short_code").
			Unique().
			NotEmpty().
			MaxLen(20).
			Comment("The unique short code for the URL"),
		field.String("original_url").
			NotEmpty().
			MaxLen(2048).
			Comment("The original long URL"),
		field.Int64("click_count").
			Default(0).
			NonNegative().
			Comment("Number of times this URL has been accessed"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("Optional expiration time for the URL"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the URL was created"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("When the URL was last updated"),
	}
}

// Edges of the URL.
func (URL) Edges() []ent.Edge {
	return nil
}

// Indexes of the URL.
func (URL) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("short_code").Unique(),
		index.Fields("created_at"),
	}
}

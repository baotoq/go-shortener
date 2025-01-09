package data

import (
	"context"
	"go-shortener/internal/conf"
	"go-shortener/internal/data/ent"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	_ "github.com/mattn/go-sqlite3"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUrlRepo)

// Data .
type Data struct {
	db *ent.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)

	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	d := &Data{
		db: client,
	}

	cleanup := func() {
		log.Info("message", "closing the data resources")
		if err := d.db.Close(); err != nil {
			log.Error(err)
		}
	}

	return d, cleanup, nil
}

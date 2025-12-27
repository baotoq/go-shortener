package data

import (
	"context"

	"go-shortener/ent"
	"go-shortener/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	_ "github.com/mattn/go-sqlite3"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewURLRepo, NewURLCache, NewCachedURLRepository, NewUnitOfWork)

// Data holds database and cache clients
type Data struct {
	db  *ent.Client
	rdb *redis.Client
}

// RedisClient returns the Redis client, which may be nil if Redis is unavailable.
func (d *Data) RedisClient() *redis.Client {
	return d.rdb
}

// EntClient returns the Ent database client.
func (d *Data) EntClient() *ent.Client {
	return d.db
}

// NewURLCache creates a new URL cache from Data.
func NewURLCache(data *Data, logger log.Logger) URLCache {
	return NewRedisURLCache(data.RedisClient(), logger)
}

// NewData creates a new Data instance with database and redis connections
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)

	// Initialize Ent client with SQLite
	client, err := ent.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		return nil, nil, err
	}

	// Run auto migration
	if err := client.Schema.Create(context.Background()); err != nil {
		client.Close()
		return nil, nil, err
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})

	// Test Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warnf("Redis connection failed: %v, caching will be disabled", err)
		rdb = nil
	}

	cleanup := func() {
		log.Info("closing the data resources")
		if err := client.Close(); err != nil {
			log.Errorf("failed to close ent client: %v", err)
		}
		if rdb != nil {
			if err := rdb.Close(); err != nil {
				log.Errorf("failed to close redis client: %v", err)
			}
		}
	}

	return &Data{db: client, rdb: rdb}, cleanup, nil
}

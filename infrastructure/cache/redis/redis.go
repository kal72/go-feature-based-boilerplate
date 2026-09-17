package redis

import (
	"context"
	"fmt"
	"time"

	"go-feature-based-boilerplate/infrastructure/config"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// New creates a Redis client and verifies connectivity via a ping.
func New(cfg *config.Config) (*redis.Client, error) {
	redisCfg := cfg.Redis

	client := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	if cfg.Telemetry.Enabled {
		if err := redisotel.InstrumentTracing(client); err != nil {
			return nil, fmt.Errorf("redis: otel tracing: %w", err)
		}
	}

	return client, nil
}

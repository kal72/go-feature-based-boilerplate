package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements Storage backed by a Redis cluster or standalone instance.
// It guarantees distributed consistency and atomic lock acquisition across multiple server nodes.
type RedisStorage struct {
	client *redis.Client
	prefix string
}

// NewRedisStorage initializes and returns a new Redis-backed idempotency storage.
// The default key prefix is "idempotency:".
func NewRedisStorage(client *redis.Client, prefix ...string) *RedisStorage {
	p := "idempotency:"
	if len(prefix) > 0 && prefix[0] != "" {
		p = prefix[0]
	}
	return &RedisStorage{
		client: client,
		prefix: p,
	}
}

func (r *RedisStorage) formatKey(key string) string {
	return fmt.Sprintf("%s%s", r.prefix, key)
}

// Lock atomically acquires a lock for key via Redis SetNX with the specified TTL.
func (r *RedisStorage) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	record := &Record{
		Status:    StatusInProgress,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("idempotency: marshal lock record: %w", err)
	}

	return r.client.SetNX(ctx, r.formatKey(key), data, ttl).Result()
}

// Set stores the completed record for key with the specified TTL.
func (r *RedisStorage) Set(ctx context.Context, key string, record *Record, ttl time.Duration) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("idempotency: marshal record: %w", err)
	}

	return r.client.Set(ctx, r.formatKey(key), data, ttl).Err()
}

// Get retrieves and deserializes the Record from Redis for key.
func (r *RedisStorage) Get(ctx context.Context, key string) (*Record, error) {
	val, err := r.client.Get(ctx, r.formatKey(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Key does not exist
		}
		return nil, fmt.Errorf("idempotency: redis get: %w", err)
	}

	var record Record
	if err := json.Unmarshal(val, &record); err != nil {
		return nil, fmt.Errorf("idempotency: unmarshal record: %w", err)
	}

	return &record, nil
}

// Delete removes the key from Redis.
func (r *RedisStorage) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.formatKey(key)).Err()
}

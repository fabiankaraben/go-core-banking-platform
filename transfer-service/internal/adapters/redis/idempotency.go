package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

// IdempotencyStore implements ports.IdempotencyStore backed by Redis.
type IdempotencyStore struct {
	client *redis.Client
}

// NewIdempotencyStore creates a new IdempotencyStore.
func NewIdempotencyStore(addr string) *IdempotencyStore {
	client := redis.NewClient(&redis.Options{Addr: addr})
	return &IdempotencyStore{client: client}
}

// Get retrieves the cached response for an idempotency key.
// Returns nil, nil if the key does not exist.
func (s *IdempotencyStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, redisKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis GET idempotency key: %w", err)
	}
	return val, nil
}

// Set stores a serialized response under the idempotency key for 24 hours.
func (s *IdempotencyStore) Set(ctx context.Context, key string, value []byte) error {
	if err := s.client.Set(ctx, redisKey(key), value, idempotencyTTL).Err(); err != nil {
		return fmt.Errorf("redis SET idempotency key: %w", err)
	}
	return nil
}

// Close shuts down the Redis client.
func (s *IdempotencyStore) Close() error {
	return s.client.Close()
}

func redisKey(key string) string {
	return "idempotency:" + key
}

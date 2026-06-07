package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

// IdempotencyStore uses Redis SETNX for payment-level idempotency.
type IdempotencyStore struct {
	client *redis.Client
}

// NewIdempotencyStore connects to Redis at addr.
func NewIdempotencyStore(addr string) (*IdempotencyStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis.NewIdempotencyStore ping: %w", err)
	}
	return &IdempotencyStore{client: client}, nil
}

// Acquire attempts to set the key. Returns true if this caller won the lock.
func (s *IdempotencyStore) Acquire(ctx context.Context, key string) (bool, error) {
	ok, err := s.client.SetNX(ctx, key, "1", idempotencyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("redis.Acquire: %w", err)
	}
	return ok, nil
}

// Release deletes the idempotency key (e.g., after a failed gateway call).
func (s *IdempotencyStore) Release(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Close closes the Redis connection.
func (s *IdempotencyStore) Close() error {
	return s.client.Close()
}

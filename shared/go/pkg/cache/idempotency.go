package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	// IdempotencyTTL is the default TTL for idempotency keys.
	IdempotencyTTL = 24 * time.Hour
	// TokenBlacklistPrefix is the key prefix for blacklisted JWT IDs.
	TokenBlacklistPrefix = "token:blacklist:"
	// IdempotencyPrefix is the key prefix for payment idempotency keys.
	IdempotencyPrefix = "idempotency:"
)

// IdempotencyStore provides idempotency key management for the payment service.
type IdempotencyStore struct {
	client *Client
}

// NewIdempotencyStore creates an IdempotencyStore backed by client.
func NewIdempotencyStore(client *Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

// Acquire tries to claim an idempotency key. Returns (true, nil) on first call.
// Returns (false, nil) if the key already exists (duplicate request).
func (s *IdempotencyStore) Acquire(ctx context.Context, key string) (bool, error) {
	fullKey := IdempotencyPrefix + key
	ok, err := s.client.SetNX(ctx, fullKey, []byte("processing"), IdempotencyTTL)
	if err != nil {
		return false, fmt.Errorf("idempotency.Acquire %s: %w", key, err)
	}
	return ok, nil
}

// SetResult stores the final result for an idempotency key.
func (s *IdempotencyStore) SetResult(ctx context.Context, key string, result []byte) error {
	fullKey := IdempotencyPrefix + key
	if err := s.client.Set(ctx, fullKey, result, IdempotencyTTL); err != nil {
		return fmt.Errorf("idempotency.SetResult %s: %w", key, err)
	}
	return nil
}

// GetResult returns the stored result. Returns ErrCacheMiss if not found.
func (s *IdempotencyStore) GetResult(ctx context.Context, key string) ([]byte, error) {
	fullKey := IdempotencyPrefix + key
	return s.client.Get(ctx, fullKey)
}

// BlacklistToken stores a JTI with remaining TTL (for logout).
func (s *IdempotencyStore) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	fullKey := TokenBlacklistPrefix + jti
	if err := s.client.Set(ctx, fullKey, []byte("1"), ttl); err != nil {
		return fmt.Errorf("idempotency.BlacklistToken %s: %w", jti, err)
	}
	return nil
}

// IsTokenBlacklisted returns true if the JTI was blacklisted.
func (s *IdempotencyStore) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	fullKey := TokenBlacklistPrefix + jti
	_, err := s.client.Get(ctx, fullKey)
	if err == ErrCacheMiss {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("idempotency.IsTokenBlacklisted %s: %w", jti, err)
	}
	return true, nil
}

package redis

import (
	"context"
	"fmt"
	"time"

	sharedcache "github.com/xyn-pos/shared/pkg/cache"
)

// TokenBlacklist stores logged-out JWT IDs to prevent token reuse.
type TokenBlacklist struct {
	store *sharedcache.IdempotencyStore
}

// NewTokenBlacklist wraps the shared IdempotencyStore.
func NewTokenBlacklist(store *sharedcache.IdempotencyStore) *TokenBlacklist {
	return &TokenBlacklist{store: store}
}

// Blacklist adds a JTI to the blacklist, expiring after ttl.
func (b *TokenBlacklist) Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	if err := b.store.BlacklistToken(ctx, jti, ttl); err != nil {
		return fmt.Errorf("tokenBlacklist.Blacklist: %w", err)
	}
	return nil
}

// IsBlacklisted reports whether the given JTI has been revoked.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	ok, err := b.store.IsTokenBlacklisted(ctx, jti)
	if err != nil {
		return false, fmt.Errorf("tokenBlacklist.IsBlacklisted: %w", err)
	}
	return ok, nil
}

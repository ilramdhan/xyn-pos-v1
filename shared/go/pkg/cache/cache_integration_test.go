//go:build integration

package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/xyn-pos/shared/pkg/cache"
)

func startRedis(t *testing.T) *cache.Client {
	t.Helper()
	ctx := context.Background()
	container, err := tcredis.Run(ctx, "redis:8.8-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	addr, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	client, err := cache.NewClient(cache.Config{Addr: addr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestIntegration_SetAndGet(t *testing.T) {
	c := startRedis(t)
	ctx := context.Background()

	err := c.Set(ctx, "test-key", []byte("hello"), time.Minute)
	require.NoError(t, err)

	val, err := c.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), val)
}

func TestIntegration_Get_MissingKey_ReturnsCacheMiss(t *testing.T) {
	c := startRedis(t)
	_, err := c.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestIntegration_SetNX_SecondCallReturnsFalse(t *testing.T) {
	c := startRedis(t)
	ctx := context.Background()

	ok1, err := c.SetNX(ctx, "idempotency:abc", []byte("processing"), time.Hour)
	require.NoError(t, err)
	assert.True(t, ok1)

	ok2, err := c.SetNX(ctx, "idempotency:abc", []byte("result"), time.Hour)
	require.NoError(t, err)
	assert.False(t, ok2)

	val, _ := c.Get(ctx, "idempotency:abc")
	assert.Equal(t, []byte("processing"), val)
}

func TestIntegration_TTL_KeyExpires(t *testing.T) {
	c := startRedis(t)
	ctx := context.Background()

	err := c.Set(ctx, "expiring-key", []byte("value"), 100*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	_, err = c.Get(ctx, "expiring-key")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestIntegration_Delete_RemovesKey(t *testing.T) {
	c := startRedis(t)
	ctx := context.Background()

	_ = c.Set(ctx, "del-key", []byte("v"), time.Minute)
	err := c.Delete(ctx, "del-key")
	require.NoError(t, err)

	_, err = c.Get(ctx, "del-key")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestIntegration_IdempotencyStore_Acquire(t *testing.T) {
	c := startRedis(t)
	store := cache.NewIdempotencyStore(c)
	ctx := context.Background()

	ok1, err := store.Acquire(ctx, "pay-001")
	require.NoError(t, err)
	assert.True(t, ok1)

	ok2, err := store.Acquire(ctx, "pay-001")
	require.NoError(t, err)
	assert.False(t, ok2)
}

func TestIntegration_IdempotencyStore_SetAndGetResult(t *testing.T) {
	c := startRedis(t)
	store := cache.NewIdempotencyStore(c)
	ctx := context.Background()

	_, _ = store.Acquire(ctx, "pay-002")
	err := store.SetResult(ctx, "pay-002", []byte(`{"snap_url":"https://example.com"}`))
	require.NoError(t, err)

	result, err := store.GetResult(ctx, "pay-002")
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"snap_url":"https://example.com"}`), result)
}

func TestIntegration_IdempotencyStore_BlacklistToken(t *testing.T) {
	c := startRedis(t)
	store := cache.NewIdempotencyStore(c)
	ctx := context.Background()

	jti := "jti-abc-123"
	blacklisted, err := store.IsTokenBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.False(t, blacklisted)

	err = store.BlacklistToken(ctx, jti, time.Hour)
	require.NoError(t, err)

	blacklisted, err = store.IsTokenBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.True(t, blacklisted)
}

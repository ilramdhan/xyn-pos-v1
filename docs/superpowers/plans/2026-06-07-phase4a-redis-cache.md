# Phase 4a — Redis + Shared Cache Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Redis to local dev docker-compose and create `shared/go/pkg/cache` — a thin go-redis/v9 wrapper with typed Config, a connection client, and idempotency helpers used by the payment service.

**Architecture:** Standalone Redis 8.8 in docker-compose (not Cluster). The `cache` package wraps `github.com/redis/go-redis/v9` with a typed `Client` that exposes only the operations we need: `Set`, `SetNX`, `Get`, `Delete`. Tests use testcontainers-go for a real Redis instance.

**Tech Stack:** Go 1.26.4, go-redis/v9 (v9.9.0+), testcontainers-go v0.42.0, testcontainers-go/modules/redis.

**Branch:** `feat/phase4-backend-mvp`

---

## File Structure

```
shared/go/pkg/cache/
├── config.go        ← Config struct + validation
├── cache.go         ← Client struct, NewClient, Set/SetNX/Get/Delete
└── cache_test.go    ← unit tests (mock) + integration tests (testcontainers)
```

Modify:
- `docker-compose.yml` — add `redis` service + `redis-data` volume
- `.env.example` — add `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`
- `shared/go/go.mod` — add go-redis/v9

---

### Task 1: Add Redis to docker-compose.yml and .env.example

**Files:**
- Modify: `docker-compose.yml`
- Modify: `.env.example`

- [ ] **Step 1: Add redis service block to docker-compose.yml**

Find the `services:` section of `docker-compose.yml` and add before the final `volumes:` block:

```yaml
  redis:
    image: redis:8.8-alpine
    container_name: xyn-redis
    ports:
      - "6379:6379"
    command: >
      redis-server
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
      --save 60 1
      --loglevel warning
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - xyn-net
```

Also add `redis-data:` under the top-level `volumes:` section.

- [ ] **Step 2: Add env vars to .env.example**

Append to `.env.example`:

```
# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

- [ ] **Step 3: Start Redis and verify**

```bash
docker compose up redis -d
docker exec xyn-redis redis-cli ping
```

Expected output: `PONG`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.example
git commit -m "feat(infra): add Redis 8.8 to docker-compose (story 3.4.4)"
```

---

### Task 2: Add go-redis/v9 to shared/go/go.mod

**Files:**
- Modify: `shared/go/go.mod`
- Modify: `shared/go/go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd shared/go && go get github.com/redis/go-redis/v9@latest
```

Expected: `go: added github.com/redis/go-redis/v9 v9.x.x`

- [ ] **Step 2: Also add testcontainers redis module**

```bash
go get github.com/testcontainers/testcontainers-go/modules/redis@v0.42.0
```

- [ ] **Step 3: Sync workspace**

```bash
cd ../.. && go work sync
```

- [ ] **Step 4: Commit**

```bash
git add shared/go/go.mod shared/go/go.sum go.work.sum
git commit -m "chore(shared): add go-redis/v9 and testcontainers redis module"
```

---

### Task 3: Create cache/config.go

**Files:**
- Create: `shared/go/pkg/cache/config.go`

- [ ] **Step 1: Write the failing test**

Create `shared/go/pkg/cache/cache_test.go`:

```go
package cache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xyn-pos/shared/pkg/cache"
)

func TestConfig_Validate_EmptyAddr_ReturnsError(t *testing.T) {
	cfg := cache.Config{}
	err := cfg.Validate()
	assert.ErrorIs(t, err, cache.ErrInvalidAddr)
}

func TestConfig_Validate_ValidAddr_ReturnsNil(t *testing.T) {
	cfg := cache.Config{Addr: "localhost:6379"}
	assert.NoError(t, cfg.Validate())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd shared/go && go test ./pkg/cache/... -run TestConfig -v
```

Expected: FAIL (package not found / type not defined)

- [ ] **Step 3: Create config.go**

```go
package cache

import "errors"

// ErrInvalidAddr is returned when Addr is empty.
var ErrInvalidAddr = errors.New("cache: addr cannot be empty")

// Config holds Redis connection settings.
type Config struct {
	Addr       string // host:port, e.g. "localhost:6379"
	Password   string // empty string = no auth
	DB         int    // Redis database number (0–15)
	MaxRetries int    // default 3
}

// Validate returns ErrInvalidAddr if Addr is empty.
func (c Config) Validate() error {
	if c.Addr == "" {
		return ErrInvalidAddr
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd shared/go && go test ./pkg/cache/... -run TestConfig -v
```

Expected: PASS

---

### Task 4: Create cache/cache.go with Client

**Files:**
- Create: `shared/go/pkg/cache/cache.go`

- [ ] **Step 1: Write the failing unit tests (add to cache_test.go)**

```go
func TestNewClient_InvalidConfig_ReturnsError(t *testing.T) {
	_, err := cache.NewClient(cache.Config{})
	assert.ErrorIs(t, err, cache.ErrInvalidAddr)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd shared/go && go test ./pkg/cache/... -run TestNewClient -v
```

Expected: FAIL

- [ ] **Step 3: Create cache.go**

```go
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client and exposes only the operations this project needs.
type Client struct {
	rdb *redis.Client
}

// NewClient creates and pings a Redis connection. Returns ErrInvalidAddr for bad config.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:       cfg.Addr,
		Password:   cfg.Password,
		DB:         cfg.DB,
		MaxRetries: maxRetries,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("cache.NewClient ping %s: %w", cfg.Addr, err)
	}

	return &Client{rdb: rdb}, nil
}

// Close releases the connection pool.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Set stores key=value with TTL. TTL=0 means no expiry.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// SetNX stores key=value only if key does not exist. Returns true if the key was set.
func (c *Client) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

// Get retrieves the value for key. Returns ErrCacheMiss if key is absent.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	return val, err
}

// Delete removes one or more keys. Missing keys are silently ignored.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// ErrCacheMiss is returned by Get when the key does not exist.
var ErrCacheMiss = errors.New("cache: key not found")
```

Add `"errors"` to the import block.

- [ ] **Step 4: Run unit test**

```bash
cd shared/go && go test ./pkg/cache/... -run TestNewClient_InvalidConfig -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add shared/go/pkg/cache/config.go shared/go/pkg/cache/cache.go shared/go/pkg/cache/cache_test.go
git commit -m "feat(shared/cache): add Config + Client with Set/SetNX/Get/Delete"
```

---

### Task 5: Create cache/idempotency.go + Integration Tests

**Files:**
- Create: `shared/go/pkg/cache/idempotency.go`
- Modify: `shared/go/pkg/cache/cache_test.go` (add integration tests)

The `IdempotencyStore` wraps `Client` with domain-specific key prefixes and TTLs.

- [ ] **Step 1: Write integration tests (add to cache_test.go)**

```go
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
	t.Cleanup(func() { container.Terminate(ctx) })

	addr, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	client, err := cache.NewClient(cache.Config{Addr: addr})
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })
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

	// Value should still be the first one
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
```

- [ ] **Step 2: Run integration tests to verify they fail**

```bash
cd shared/go && go test ./pkg/cache/... -tags=integration -v -run TestIntegration
```

Expected: FAIL (idempotency.go not yet needed, but startRedis may fail if modules not downloaded)

- [ ] **Step 3: Create idempotency.go**

```go
package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	// IdempotencyTTL is the default TTL for idempotency keys (24 hours).
	IdempotencyTTL = 24 * time.Hour
	// TokenBlacklistPrefix is the key prefix for blacklisted JWT IDs.
	TokenBlacklistPrefix = "token:blacklist:"
	// IdempotencyPrefix is the key prefix for payment idempotency keys.
	IdempotencyPrefix = "idempotency:"
	// QRISStatusPrefix is the key prefix for cached QRIS status.
	QRISStatusPrefix = "qris:status:"
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

// SetResult stores the final result for an idempotency key, replacing "processing".
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

// IsTokenBlacklisted returns true if the JTI was blacklisted (user logged out).
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
```

- [ ] **Step 4: Run integration tests**

```bash
cd shared/go && go test ./pkg/cache/... -tags=integration -v -run TestIntegration -count=1
```

Expected: all PASS (Docker must be running)

- [ ] **Step 5: Run unit tests (no Docker)**

```bash
cd shared/go && go test ./pkg/cache/... -v -count=1
```

Expected: TestConfig_* and TestNewClient_InvalidConfig PASS. Integration tests skipped (no build tag).

- [ ] **Step 6: Lint**

```bash
cd shared/go && golangci-lint run ./pkg/cache/...
```

Expected: no issues.

- [ ] **Step 7: Commit**

```bash
git add shared/go/pkg/cache/
git commit -m "feat(shared/cache): add IdempotencyStore with BlacklistToken + QRIS helpers"
```

---

## Self-Review

- Spec §2 coverage: Redis docker-compose ✅, shared/go/pkg/cache/ ✅, Config ✅, SetNX idempotency ✅, token blacklist ✅
- Key naming conventions from spec: `idempotency:{key}`, `token:blacklist:{jti}`, `qris:status:{id}` — all defined as constants ✅
- No placeholder code ✅
- `ErrCacheMiss` and `ErrInvalidAddr` are both exported sentinel errors ✅
- Integration tests use real Redis via testcontainers, not mocks ✅
- Build tag `integration` separates fast unit tests from slow integration tests ✅

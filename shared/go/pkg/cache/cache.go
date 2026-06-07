package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned by Get when the key does not exist.
var ErrCacheMiss = errors.New("cache: key not found")

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
		_ = rdb.Close()
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

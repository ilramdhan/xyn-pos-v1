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

func TestNewClient_InvalidConfig_ReturnsError(t *testing.T) {
	_, err := cache.NewClient(cache.Config{})
	assert.ErrorIs(t, err, cache.ErrInvalidAddr)
}

package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	shareddb "github.com/xyn-pos/shared/pkg/database"
)

func TestNewPool_InvalidDSN_ReturnsError(t *testing.T) {
	_, err := shareddb.NewPool(context.Background(), shareddb.Config{DSN: "not-a-valid-dsn"})
	assert.Error(t, err)
}

func TestNewPool_UnreachableHost_ReturnsError(t *testing.T) {
	_, err := shareddb.NewPool(context.Background(), shareddb.Config{
		DSN: "postgres://test:test@localhost:1/testdb?sslmode=disable&connect_timeout=1",
	})
	assert.Error(t, err)
}

package tenant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/xyn-pos/services/tenant"
)

func TestLoadConfig_MissingDatabaseURL_ReturnsError(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := tenant.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoadConfig_ValidEnv_ReturnsConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/tenant?sslmode=disable")
	t.Setenv("GRPC_PORT", "50052")

	cfg, err := tenant.LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 50052, cfg.GRPCPort)
	assert.Equal(t, "postgres://user:pass@localhost:5432/tenant?sslmode=disable", cfg.DatabaseURL)
}

func TestLoadConfig_InvalidGRPCPort_ReturnsError(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/tenant?sslmode=disable")
	t.Setenv("GRPC_PORT", "not-a-number")
	_, err := tenant.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GRPC_PORT")
}

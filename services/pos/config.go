package pos

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all environment-driven configuration for the pos service.
type Config struct {
	ServiceName  string
	Version      string
	Env          string
	DatabaseURL  string
	GRPCPort     int
	OTLPEndpoint string
	PASETOKeyHex string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServiceName:  getEnv("SERVICE_NAME", "pos-service"),
		Version:      getEnv("SERVICE_VERSION", "0.0.1"),
		Env:          getEnv("ENVIRONMENT", "dev"),
		OTLPEndpoint: getEnv("OTLP_ENDPOINT", "localhost:4317"),
		PASETOKeyHex: getEnv("PASETO_KEY_HEX", ""),
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	cfg.DatabaseURL = dbURL

	if cfg.PASETOKeyHex == "" && cfg.Env != "dev" {
		return nil, fmt.Errorf("config: PASETO_KEY_HEX is required in non-dev environments")
	}

	portStr := getEnv("GRPC_PORT", "50052")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("config: invalid GRPC_PORT %q: %w", portStr, err)
	}
	cfg.GRPCPort = port

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package inventory

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime configuration for the inventory service.
type Config struct {
	DatabaseURL  string
	KafkaBrokers []string
	GRPCPort     string
	PASETOKeyHex string
	Env          string
}

// ConfigFromEnv reads Config from environment variables.
func ConfigFromEnv() (*Config, error) {
	dbURL := os.Getenv("INVENTORY_DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("INVENTORY_DATABASE_URL is required")
	}
	env := envOrDefault("ENVIRONMENT", "dev")
	pasetoKey := os.Getenv("PASETO_KEY_HEX")
	if pasetoKey == "" && env != "dev" {
		return nil, fmt.Errorf("PASETO_KEY_HEX is required in non-dev environments")
	}
	brokersRaw := envOrDefault("KAFKA_BROKERS", "localhost:9092")
	return &Config{
		DatabaseURL:  dbURL,
		KafkaBrokers: strings.Split(brokersRaw, ","),
		GRPCPort:     envOrDefault("INVENTORY_GRPC_PORT", "50054"),
		PASETOKeyHex: pasetoKey,
		Env:          env,
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

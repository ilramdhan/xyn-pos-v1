package payment

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration for the payment service.
type Config struct {
	DatabaseURL          string
	RedisAddr            string
	KafkaBrokers         []string
	MidtransServerKey    string
	MidtransClientKey    string
	MidtransIsProduction bool
	GRPCPort             string
	HTTPPort             string
	PASETOKeyHex         string
}

// ConfigFromEnv reads Config from environment variables.
func ConfigFromEnv() (*Config, error) {
	dbURL := os.Getenv("PAYMENT_DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("PAYMENT_DATABASE_URL is required")
	}
	midKey := os.Getenv("MIDTRANS_SERVER_KEY")
	if midKey == "" {
		return nil, fmt.Errorf("MIDTRANS_SERVER_KEY is required")
	}
	pasetoKey := os.Getenv("PASETO_KEY_HEX")
	if pasetoKey == "" && os.Getenv("ENVIRONMENT") != "dev" {
		return nil, fmt.Errorf("PASETO_KEY_HEX is required in non-dev environments")
	}
	brokersRaw := envOrDefault("KAFKA_BROKERS", "localhost:9092")
	return &Config{
		DatabaseURL:          dbURL,
		RedisAddr:            envOrDefault("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:         strings.Split(brokersRaw, ","),
		MidtransServerKey:    midKey,
		MidtransClientKey:    os.Getenv("MIDTRANS_CLIENT_KEY"),
		MidtransIsProduction: os.Getenv("MIDTRANS_ENV") == "production",
		GRPCPort:             envOrDefault("PAYMENT_GRPC_PORT", "50053"),
		HTTPPort:             envOrDefault("PAYMENT_HTTP_PORT", "8083"),
		PASETOKeyHex:         pasetoKey,
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

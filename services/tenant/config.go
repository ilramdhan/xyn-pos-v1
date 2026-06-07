package tenant

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all environment-driven configuration for the tenant service.
type Config struct {
	ServiceName string
	Version     string
	Env         string

	DatabaseURL  string
	GRPCPort     int
	OTLPEndpoint string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Keycloak
	KeycloakBaseURL      string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
	KeycloakAdminToken   string

	// PASETOKeyHex is the hex-encoded 32-byte symmetric key for PASETO v4 local tokens.
	PASETOKeyHex string
}

// LoadConfig reads configuration from environment variables.
// Returns an error if required variables are missing.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServiceName:  getEnv("SERVICE_NAME", "tenant-service"),
		Version:      getEnv("SERVICE_VERSION", "0.0.1"),
		Env:          getEnv("ENVIRONMENT", "dev"),
		OTLPEndpoint: getEnv("OTLP_ENDPOINT", "localhost:4317"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,

		KeycloakBaseURL:      getEnv("KEYCLOAK_BASE_URL", "http://localhost:8080"),
		KeycloakRealm:        getEnv("KEYCLOAK_REALM", "xyn-pos"),
		KeycloakClientID:     getEnv("KEYCLOAK_CLIENT_ID", "xyn-backend"),
		KeycloakClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
		KeycloakAdminToken:   getEnv("KEYCLOAK_ADMIN_TOKEN", ""),

		PASETOKeyHex: getEnv("PASETO_KEY_HEX", ""),
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	cfg.DatabaseURL = dbURL

	portStr := getEnv("GRPC_PORT", "50051")
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

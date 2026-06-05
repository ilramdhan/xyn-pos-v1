module github.com/xyn-pos/shared

go 1.26

require (
    // Error handling
    github.com/samber/oops v1.14.1

    // gRPC + protobuf
    google.golang.org/grpc v1.71.1
    google.golang.org/protobuf v1.36.5

    // Database
    github.com/jackc/pgx/v5 v5.7.4

    // Kafka
    github.com/twmb/franz-go v1.18.1

    // Auth — PASETO tokens
    github.com/o1egg/paseto v1.0.2

    // Observability
    go.opentelemetry.io/otel v1.44.0
    go.opentelemetry.io/otel/trace v1.44.0
    go.opentelemetry.io/otel/metric v1.44.0
    go.opentelemetry.io/otel/sdk v1.44.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0

    // Logging
    golang.org/x/exp v0.0.0-20240909161429-701f63a606c0

    // UUID
    github.com/google/uuid v1.6.0

    // Config
    github.com/spf13/viper v1.20.1
)

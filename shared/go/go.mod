module github.com/xyn-pos/shared

go 1.26

require (
	aidanwoods.dev/go-paseto v1.5.3
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/samber/lo v1.49.1
	github.com/samber/oops v1.17.0
	github.com/samber/slog-zap/v2 v2.6.2
	github.com/twmb/franz-go v1.18.1
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/sdk/metric v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.81.1
)

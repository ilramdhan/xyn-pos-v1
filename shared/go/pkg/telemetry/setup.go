package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds OTEL setup configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string // e.g., "otel-collector:4317"
	Environment    string // "dev", "staging", "prod"
}

// Setup initializes the global OTEL tracer provider.
// Returns a shutdown function to be called on service exit.
func Setup(ctx context.Context, cfg Config) (shutdown func(), err error) {
	conn, err := grpc.NewClient(
		cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry.Setup grpc dial: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("telemetry.Setup exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
	)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("telemetry.Setup resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return func() {
		_ = tp.Shutdown(context.Background())
		_ = conn.Close()
	}, nil
}

// RecordError marks a span as errored and records the error.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
}

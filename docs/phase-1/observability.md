# Observability — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Stack: OpenTelemetry Go 1.44.0 | Grafana 13.0.2 | Prometheus 3.12.0 | Loki | Jaeger

---

## 1. Observability Philosophy

**Three Pillars:**
```
Traces  — Distributed request tracing (OpenTelemetry → Jaeger)
Metrics — Numeric time-series (Prometheus → Grafana)
Logs    — Structured text events (slog → Loki → Grafana)
```

**Correlation:** Every log line must contain the `trace_id` so you can jump from a log to the distributed trace in one click. OpenTelemetry handles this automatically if you propagate context correctly.

**The Golden Signal Rule (SRE):** For every service, measure these four signals:
- **Latency** — How long do requests take? (p50, p95, p99)
- **Traffic** — How many requests are we handling? (req/s)
- **Errors** — What percentage of requests fail? (error_rate)
- **Saturation** — How full is the system? (CPU, memory, DB pool)

---

## 2. OpenTelemetry Setup

### 2.1 Tracer Provider Initialization

```go
// shared/go/pkg/telemetry/tracer.go
package telemetry

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type Config struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
    OTELEndpoint   string    // e.g. "localhost:4317"
    SamplingRate   float64   // 0.0–1.0
    Insecure       bool
}

// NewTracerProvider initializes OTEL and returns a shutdown function.
func NewTracerProvider(ctx context.Context, cfg Config) (func(context.Context) error, error) {
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion(cfg.ServiceVersion),
            semconv.DeploymentEnvironment(cfg.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("create OTEL resource: %w", err)
    }

    opts := []grpc.DialOption{}
    if cfg.Insecure {
        opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
    }

    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.OTELEndpoint),
        otlptracegrpc.WithDialOption(opts...),
    )
    if err != nil {
        return nil, fmt.Errorf("create OTLP exporter: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.ParentBased(
            sdktrace.TraceIDRatioBased(cfg.SamplingRate),
        )),
    )

    // Set global tracer provider and propagator
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp.Shutdown, nil
}
```

### 2.2 Instrumented gRPC Server

```go
// shared/go/pkg/middleware/tracing.go
package middleware

import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "google.golang.org/grpc"
)

// InstrumentedGRPCServer creates a gRPC server with OTEL tracing, metrics, and logging.
func InstrumentedGRPCServer(additionalOpts ...grpc.ServerOption) *grpc.Server {
    baseOpts := []grpc.ServerOption{
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
        grpc.ChainUnaryInterceptor(
            RecoveryInterceptor(),
            LoggingInterceptor(),
            AuthInterceptor(),
        ),
        grpc.ChainStreamInterceptor(
            StreamRecoveryInterceptor(),
            StreamAuthInterceptor(),
        ),
    }
    return grpc.NewServer(append(baseOpts, additionalOpts...)...)
}
```

### 2.3 Manual Span Creation

```go
// When you need a custom span for a significant operation:
func (h *CheckoutHandler) Handle(ctx context.Context, cmd CheckoutCommand) error {
    tracer := otel.Tracer("pos-service")

    ctx, span := tracer.Start(ctx, "checkout.handle",
        trace.WithAttributes(
            attribute.String("order.id", cmd.OrderID.String()),
            attribute.String("tenant.id", cmd.TenantID.String()),
            attribute.Int("item.count", len(cmd.Items)),
        ),
    )
    defer span.End()

    // Add events to the span (like log annotations)
    span.AddEvent("stock_check_started")
    if err := h.checkStock(ctx, cmd.Items); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    span.AddEvent("stock_check_passed")

    // ... rest of logic
    return nil
}
```

### 2.4 Structured Logging with Trace Correlation

```go
// shared/go/pkg/logger/logger.go
package logger

import (
    "context"
    "log/slog"
    "os"

    "go.opentelemetry.io/otel/trace"
)

// New creates a structured slog logger that automatically injects trace context.
func New(format, level string) *slog.Logger {
    var handler slog.Handler
    opts := &slog.HandlerOptions{
        Level:     parseLevel(level),
        AddSource: true,
    }

    if format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    return slog.New(handler)
}

// FromContext returns a logger that automatically includes the OTEL trace ID.
// Use this inside request handlers so every log line is traceable.
func FromContext(ctx context.Context) *slog.Logger {
    span := trace.SpanFromContext(ctx)
    if !span.SpanContext().IsValid() {
        return slog.Default()
    }
    return slog.Default().With(
        slog.String("trace_id", span.SpanContext().TraceID().String()),
        slog.String("span_id", span.SpanContext().SpanID().String()),
    )
}
```

---

## 3. Prometheus Metrics

### 3.1 Standard Service Metrics

Every Go service automatically exposes these metrics via the OpenTelemetry + Prometheus bridge:

```
# Default metrics from otelgrpc:
grpc_server_started_total          — gRPC requests started
grpc_server_handled_total          — gRPC requests completed (by code)
grpc_server_handling_seconds       — gRPC request latency histogram

# Default Go runtime metrics:
go_goroutines                      — Active goroutines
go_gc_duration_seconds             — GC pause duration
go_memstats_alloc_bytes            — Memory allocated
process_cpu_seconds_total          — CPU usage
```

### 3.2 Custom Business Metrics

```go
// shared/go/pkg/metrics/metrics.go
package metrics

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

type POSMetrics struct {
    OrdersCreated       metric.Int64Counter
    OrdersCompleted     metric.Int64Counter
    OrdersCancelled     metric.Int64Counter
    CheckoutDuration    metric.Float64Histogram
    PaymentSuccessRate  metric.Float64ObservableGauge
    ActiveShifts        metric.Int64ObservableGauge
    StockAlerts         metric.Int64Counter
}

func NewPOSMetrics() (*POSMetrics, error) {
    meter := otel.GetMeterProvider().Meter("xyn-pos")
    m := &POSMetrics{}
    var err error

    m.OrdersCreated, err = meter.Int64Counter("pos.orders.created",
        metric.WithDescription("Total number of orders created"),
        metric.WithUnit("{orders}"),
    )
    if err != nil { return nil, err }

    m.CheckoutDuration, err = meter.Float64Histogram("pos.checkout.duration",
        metric.WithDescription("Time to complete a checkout in seconds"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.1, 0.5, 1.0, 2.0, 5.0, 10.0),
    )
    if err != nil { return nil, err }

    m.StockAlerts, err = meter.Int64Counter("inventory.stock.alerts",
        metric.WithDescription("Number of low stock alerts triggered"),
        metric.WithUnit("{alerts}"),
    )
    if err != nil { return nil, err }

    return m, nil
}

// Usage in handler:
func (h *CheckoutHandler) Handle(ctx context.Context, cmd CheckoutCommand) error {
    start := time.Now()
    defer func() {
        h.metrics.CheckoutDuration.Record(ctx, time.Since(start).Seconds(),
            metric.WithAttributes(
                attribute.String("tenant.id", cmd.TenantID.String()),
                attribute.String("branch.id", cmd.BranchID.String()),
                attribute.String("payment.method", cmd.PaymentMethod),
            ),
        )
    }()
    // ... checkout logic
}
```

### 3.3 Prometheus Scrape Config

```yaml
# infra/prometheus/prometheus.yml (Prometheus 3.12.0)
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    environment: production

rule_files:
  - "rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

scrape_configs:
  - job_name: 'pos-service'
    static_configs:
      - targets: ['pos-service:2113']
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance

  - job_name: 'payment-service'
    static_configs:
      - targets: ['payment-service:2114']

  - job_name: 'inventory-service'
    static_configs:
      - targets: ['inventory-service:2115']

  # Kubernetes service discovery (production)
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: "true"
```

---

## 4. Grafana Dashboards

### 4.1 Dashboard Structure

```
Grafana 13 Dashboards:
├── Platform Overview           ← Cross-service health at a glance
├── POS Operations
│   ├── Sales Dashboard         ← Revenue, order count, avg transaction
│   ├── Cashier Performance     ← Per-cashier metrics, shift summary
│   └── Product Performance     ← Top sellers, dead stock
├── Service Health
│   ├── API Gateway (KrakenD)   ← Request rates, error rates, latency
│   ├── POS Service             ← gRPC metrics, DB pool, Kafka lag
│   ├── Payment Service         ← Payment success rate, gateway health
│   └── Inventory Service       ← Stock mutation rate, alert count
├── Infrastructure
│   ├── PostgreSQL 18           ← Query performance, connection pool, replication lag
│   ├── Redis 8                 ← Hit rate, memory usage, eviction rate
│   ├── Kafka 4.3               ← Consumer lag, throughput, partition balance
│   └── K8s Resources           ← Pod CPU/memory, HPA activity
└── SLO Dashboard               ← SLO compliance, error budget burn rate
```

### 4.2 Sales Dashboard Panels

```
Row 1: Today's KPIs (auto-refresh: 1m)
├── Total Revenue (today)          stat panel
├── Total Orders (today)           stat panel
├── Average Transaction Value      stat panel
├── Payment Success Rate           gauge (target: > 99%)
└── Active Cashiers                stat panel

Row 2: Revenue Over Time
├── Hourly Revenue (bar chart, today vs yesterday)
└── Order Volume (bar chart, with moving average)

Row 3: Payment Methods Breakdown
├── Payment Method Distribution    pie chart
└── QRIS vs Cash vs Card Trend     time series

Row 4: Product Performance
├── Top 10 Products by Revenue     table
└── Stock Depletion Rate           time series
```

### 4.3 SLO Dashboard — Prometheus Recording Rules

```yaml
# infra/prometheus/rules/slo.yml
groups:
  - name: pos_slo
    interval: 1m
    rules:
      # Checkout success rate (target: 99.5%)
      - record: pos:checkout_success_rate:5m
        expr: |
          sum(rate(grpc_server_handled_total{grpc_service="xyn.payment.v1.PaymentService",
            grpc_method="ProcessPayment", grpc_code="OK"}[5m]))
          /
          sum(rate(grpc_server_handled_total{grpc_service="xyn.payment.v1.PaymentService",
            grpc_method="ProcessPayment"}[5m]))

      # Checkout p95 latency (target: < 800ms)
      - record: pos:checkout_latency_p95:5m
        expr: |
          histogram_quantile(0.95,
            sum(rate(grpc_server_handling_seconds_bucket{
              grpc_service="xyn.payment.v1.PaymentService",
              grpc_method="ProcessPayment"}[5m])) by (le))

      # Error budget: how much of our 0.5% error budget is consumed this month
      - record: pos:checkout_error_budget_consumed:30d
        expr: |
          1 - (sum(rate(grpc_server_handled_total{
            grpc_service="xyn.payment.v1.PaymentService",
            grpc_method="ProcessPayment", grpc_code="OK"}[30d]))
          /
          sum(rate(grpc_server_handled_total{
            grpc_service="xyn.payment.v1.PaymentService",
            grpc_method="ProcessPayment"}[30d]))) / 0.005
```

### 4.4 Alert Rules

```yaml
# infra/prometheus/rules/alerts.yml
groups:
  - name: pos_critical
    rules:
      - alert: PaymentServiceDown
        expr: up{job="payment-service"} == 0
        for: 1m
        labels:
          severity: critical
          team: backend
        annotations:
          summary: "Payment service is down"
          description: "Payment service {{ $labels.instance }} has been down for 1 minute."
          runbook: "https://docs.xyn.app/runbooks/payment-service-down"

      - alert: HighCheckoutErrorRate
        expr: pos:checkout_success_rate:5m < 0.995
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Checkout error rate above SLO ({{ printf \"%.2f\" $value }})"
          description: "Checkout success rate has been below 99.5% for 5 minutes."

      - alert: HighCheckoutLatency
        expr: pos:checkout_latency_p95:5m > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Checkout p95 latency above SLO ({{ printf \"%.0f\" (mul $value 1000) }}ms)"

      - alert: KafkaConsumerLagHigh
        expr: kafka_consumer_group_lag{group="inventory-service"} > 10000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Inventory service Kafka consumer lag is high ({{ $value }})"
          description: "Stock deductions may be delayed by {{ printf \"%.0f\" (div $value 1000) }}s"

      - alert: DatabasePoolExhaustion
        expr: |
          (pgxpool_acquired_conns / pgxpool_max_conns) > 0.9
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool {{ printf \"%.0f\" (mul $value 100) }}% full"

      - alert: LowStockCritical
        expr: inventory_stock_quantity < inventory_stock_reorder_point * 0.5
        labels:
          severity: info
        annotations:
          summary: "Critical stock level for {{ $labels.product_name }}"
```

---

## 5. Log Management (Loki)

### 5.1 Log Structure Requirements

All services log in JSON format with these required fields:

```json
{
  "time": "2026-06-05T14:00:00.000Z",
  "level": "INFO",
  "msg": "order created",
  "service": "pos-service",
  "version": "1.2.0",
  "environment": "production",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "tenant_id": "018e1234-5678-7abc-def0-123456789abc",
  "order_id": "018e5678-1234-7abc-def0-123456789abc",
  "user_id": "018e9999-0000-7abc-def0-123456789abc"
}
```

### 5.2 Promtail Configuration

```yaml
# infra/promtail/promtail.yml
server:
  http_listen_port: 9080

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: kubernetes_pods
    kubernetes_sd_configs:
      - role: pod
    pipeline_stages:
      - json:
          expressions:
            level: level
            trace_id: trace_id
            tenant_id: tenant_id
            service: service
      - labels:
          level:
          trace_id:
          tenant_id:
          service:
      - timestamp:
          source: time
          format: RFC3339Nano
```

### 5.3 Loki Query Examples

```logql
# All errors in the last 15 minutes
{service="pos-service"} | json | level = "ERROR" | line_format "{{.msg}} tenant={{.tenant_id}}"

# Trace a specific request (jump from Jaeger trace to logs)
{environment="production"} | json | trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"

# Slow checkouts (from logs) — requests that took > 2s
{service="payment-service"} | json | msg = "request completed" | duration > 2000

# Error rate per tenant (for multi-tenant debugging)
sum by (tenant_id) (
  rate({service="pos-service"} | json | level = "ERROR" [5m])
)

# Payment failures by gateway
{service="payment-service"} | json | msg = "payment gateway error"
  | line_format "{{.gateway}} - {{.error_code}}"
```

---

## 6. Distributed Tracing (Jaeger)

### 6.1 Key Traces to Instrument

| Operation | Trace Name | Key Attributes |
|---|---|---|
| Order checkout | `checkout.handle` | order_id, tenant_id, item_count, payment_method |
| Payment processing | `payment.process` | payment_id, order_id, gateway, amount |
| Stock deduction | `inventory.deduct_stock` | product_id, warehouse_id, quantity |
| KDS ticket creation | `kitchen.create_ticket` | order_id, branch_id, station |
| Sync operation | `sync.process_outbox` | device_id, event_count, conflict_count |

### 6.2 Cross-Service Trace Propagation

Context propagation is automatic when you use the gRPC interceptors. For Kafka:

```go
// When publishing to Kafka, inject trace context into message headers
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
    headers := []kafka.Header{}

    // Inject OTEL trace context into Kafka message headers
    carrier := kafkaHeaderCarrier{headers: &headers}
    otel.GetTextMapPropagator().Inject(ctx, carrier)

    msg := kafka.Message{
        Topic:   topic,
        Value:   payload,
        Headers: headers,
    }
    return p.writer.WriteMessages(ctx, msg)
}

// When consuming from Kafka, extract trace context from headers
func (c *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) {
    // Extract context from message headers to link to the producer's trace
    carrier := kafkaHeaderCarrier{headers: &msg.Headers}
    ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

    ctx, span := otel.Tracer("kafka-consumer").Start(ctx, "consume."+msg.Topic)
    defer span.End()

    // Process with full distributed trace
    c.handler(ctx, msg.Value)
}
```

---

## 7. Health Checks

Every service exposes:

```
GET /health       — Liveness: returns 200 if process is alive
GET /ready        — Readiness: returns 200 only when all deps are healthy
GET /metrics      — Prometheus metrics endpoint
```

```go
// shared/go/pkg/health/health.go
type HealthChecker struct {
    checks []Check
}

type Check struct {
    Name string
    Fn   func(ctx context.Context) error
}

func (h *HealthChecker) Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        results := make(map[string]string)
        allHealthy := true

        for _, check := range h.checks {
            if err := check.Fn(ctx); err != nil {
                results[check.Name] = "unhealthy: " + err.Error()
                allHealthy = false
            } else {
                results[check.Name] = "healthy"
            }
        }

        if allHealthy {
            w.WriteHeader(http.StatusOK)
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(results)
    })
}

// Usage in provider.go:
healthChecker := health.New()
healthChecker.Add("database", func(ctx context.Context) error {
    return pool.Ping(ctx)
})
healthChecker.Add("kafka", func(ctx context.Context) error {
    return publisher.Ping(ctx)
})
healthChecker.Add("redis", func(ctx context.Context) error {
    return redisClient.Ping(ctx).Err()
})
```

---

## 8. On-Call Runbooks Index

| Alert | Runbook |
|---|---|
| `PaymentServiceDown` | `docs/runbooks/payment-service-down.md` |
| `HighCheckoutErrorRate` | `docs/runbooks/high-checkout-errors.md` |
| `KafkaConsumerLagHigh` | `docs/runbooks/kafka-consumer-lag.md` |
| `DatabasePoolExhaustion` | `docs/runbooks/db-pool-exhaustion.md` |
| `PostgreSQLReplicationLag` | `docs/runbooks/postgres-replication-lag.md` |

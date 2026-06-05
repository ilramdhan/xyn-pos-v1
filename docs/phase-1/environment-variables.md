# Environment Variables — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Authoritative Reference  
> Rule: Zero secrets in source code. Every secret lives in Vault or K8s Sealed Secrets.

---

## 1. Secret Management Tiers

```
Local Dev:   .env.local files (git-ignored)
CI/CD:       GitHub Actions Secrets (encrypted, per-environment)
Staging:     Kubernetes Sealed Secrets → mounted as env vars
Production:  HashiCorp Vault → Kubernetes External Secrets Operator
```

**Rule:** `.env.local` is in `.gitignore`. Never commit a file with real secrets.  
**Rule:** `.env.example` is committed — it shows all required vars with placeholder values.

---

## 2. Root-Level Environment Variables

```bash
# .env.example (root)

# ── Environment ───────────────────────────────────────────────────────
APP_ENV=development            # development | staging | production
APP_VERSION=0.1.0              # Set by CI/CD pipeline
LOG_LEVEL=debug                # debug | info | warn | error
LOG_FORMAT=json                # json | text (text for dev readability)

# ── PostgreSQL 18.4 ───────────────────────────────────────────────────
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=xyn_pos
POSTGRES_USER=xyn_app          # Application user (RLS enforced)
POSTGRES_PASSWORD=CHANGE_ME    # REQUIRED
POSTGRES_SSL_MODE=disable       # disable (local) | require (staging/prod)
POSTGRES_MAX_CONNS=10
POSTGRES_MIN_CONNS=2
POSTGRES_MAX_CONN_LIFETIME=30m
POSTGRES_MAX_CONN_IDLE_TIME=5m

# Admin user (migrations, cross-tenant ops — never for application queries)
POSTGRES_ADMIN_USER=xyn_admin
POSTGRES_ADMIN_PASSWORD=CHANGE_ME  # REQUIRED

# ── Redis 8.8 ─────────────────────────────────────────────────────────
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=                    # Optional (required in staging/prod)
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s

# Redis Cluster (production)
REDIS_CLUSTER_NODES=             # comma-separated: host1:6379,host2:6379,host3:6379

# ── Kafka 4.3.0 KRaft (NO ZooKeeper) ─────────────────────────────────
KAFKA_BROKERS=localhost:9092     # comma-separated in production
KAFKA_SECURITY_PROTOCOL=PLAINTEXT  # PLAINTEXT (local) | SASL_SSL (prod)
KAFKA_SASL_MECHANISM=            # PLAIN | SCRAM-SHA-512 (prod)
KAFKA_SASL_USERNAME=             # REQUIRED in staging/prod
KAFKA_SASL_PASSWORD=             # REQUIRED in staging/prod
KAFKA_CONSUMER_GROUP_ID=xyn-pos-service  # per service
KAFKA_AUTO_OFFSET_RESET=earliest
KAFKA_ENABLE_IDEMPOTENCE=true    # Required for exactly-once producer semantics
KAFKA_COMPRESSION_TYPE=snappy
KAFKA_BATCH_SIZE=16384
KAFKA_LINGER_MS=5

# ── OpenTelemetry ─────────────────────────────────────────────────────
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317  # Jaeger/Collector gRPC endpoint
OTEL_EXPORTER_OTLP_INSECURE=true                   # false in staging/prod
OTEL_SERVICE_NAME=pos-service                       # per service
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=development
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=1.0                        # 1.0 = 100% (lower in prod: 0.1)

# ── Authentication ────────────────────────────────────────────────────
KEYCLOAK_URL=http://localhost:8180
KEYCLOAK_REALM=xyn
KEYCLOAK_CLIENT_ID=xyn-backend
KEYCLOAK_CLIENT_SECRET=CHANGE_ME   # REQUIRED

# PASETO v4 token signing
PASETO_SECRET_KEY=CHANGE_ME        # REQUIRED — 32-byte hex-encoded secret
PASETO_TOKEN_DURATION=24h
PASETO_REFRESH_TOKEN_DURATION=168h  # 7 days
```

---

## 3. Per-Service Environment Variables

### 3.1 Tenant Service

```bash
# services/tenant/.env.example

SERVICE_NAME=tenant-service
GRPC_PORT=9001
HTTP_PORT=8001   # gRPC-Gateway REST
METRICS_PORT=2112  # Prometheus /metrics

# Database (separate from root — service owns its schema)
DATABASE_URL=postgresql://xyn_app:CHANGE_ME@localhost:5432/xyn_tenant?sslmode=disable

# Subscription & Billing
STRIPE_SECRET_KEY=sk_test_CHANGE_ME           # REQUIRED for billing
STRIPE_WEBHOOK_SECRET=whsec_CHANGE_ME         # REQUIRED for webhook validation
XENDIT_SECRET_KEY=xnd_CHANGE_ME              # Alternative payment provider (ID market)
XENDIT_WEBHOOK_TOKEN=CHANGE_ME

# Tenant provisioning
MAX_BRANCHES_FREE=1
MAX_BRANCHES_GROWTH=5
MAX_BRANCHES_ENTERPRISE=-1    # -1 = unlimited

# Email (for tenant invitation, password reset)
SMTP_HOST=localhost
SMTP_PORT=1025                # MailHog for local dev
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_ADDRESS=no-reply@xyn.app
SMTP_FROM_NAME=xyn POS
```

### 3.2 POS Service

```bash
# services/pos/.env.example

SERVICE_NAME=pos-service
GRPC_PORT=9002
HTTP_PORT=8002
METRICS_PORT=2113

DATABASE_URL=postgresql://xyn_app:CHANGE_ME@localhost:5432/xyn_pos?sslmode=disable

# Internal service endpoints (gRPC)
INVENTORY_SERVICE_URL=localhost:9004     # For stock availability checks
PAYMENT_SERVICE_URL=localhost:9003       # For payment initiation
MARKETING_SERVICE_URL=localhost:9006     # For promo application

# Order configuration
ORDER_DRAFT_TTL=2h           # Auto-cancel unpaid draft orders after 2 hours
MAX_ITEMS_PER_ORDER=100
ENABLE_TIME_BASED_PRICING=true

# Tax configuration (can be overridden per tenant in DB)
DEFAULT_TAX_TYPE=PPN          # PPN | PB1 | NONE
DEFAULT_TAX_RATE=0.11         # 11% PPN (Indonesia 2025)
DEFAULT_SERVICE_CHARGE=0.05   # 5% service charge (F&B)
```

### 3.3 Payment Service

```bash
# services/payment/.env.example

SERVICE_NAME=payment-service
GRPC_PORT=9003
HTTP_PORT=8003
METRICS_PORT=2114

DATABASE_URL=postgresql://xyn_app:CHANGE_ME@localhost:5432/xyn_payment?sslmode=disable

# Payment Gateways
## Midtrans (Indonesia)
MIDTRANS_SERVER_KEY=Mid-server-CHANGE_ME      # REQUIRED
MIDTRANS_CLIENT_KEY=Mid-client-CHANGE_ME      # REQUIRED
MIDTRANS_ENV=sandbox                           # sandbox | production
MIDTRANS_NOTIFICATION_URL=https://api.xyn.app/webhooks/payment/midtrans

## Xendit (Indonesia backup)
XENDIT_SECRET_KEY=xnd_CHANGE_ME
XENDIT_WEBHOOK_TOKEN=CHANGE_ME
XENDIT_ENV=development                         # development | production

# Idempotency
IDEMPOTENCY_TTL=24h          # How long to store idempotency keys in Redis
IDEMPOTENCY_KEY_PREFIX=idem:payment:

# Void window
VOID_WINDOW_HOURS=24         # Can only void within same day by default

# Reconciliation
RECONCILIATION_CRON=0 2 * * *   # Run at 2 AM daily
```

### 3.4 Inventory Service

```bash
# services/inventory/.env.example

SERVICE_NAME=inventory-service
GRPC_PORT=9004
HTTP_PORT=8004
METRICS_PORT=2115

DATABASE_URL=postgresql://xyn_app:CHANGE_ME@localhost:5432/xyn_inventory?sslmode=disable

# Stock configuration
LOW_STOCK_THRESHOLD_DEFAULT=10   # Alert when stock <= this value
NEGATIVE_STOCK_ALLOWED=false     # Allow backorders?
STOCK_CACHE_TTL=30s              # Redis cache TTL for stock reads

# Kafka topics (consumed)
KAFKA_TOPIC_ORDER_PAID=xyn.payment.completed    # Trigger stock deduction
KAFKA_TOPIC_PO_RECEIVED=xyn.inventory.po_received

# Kafka topics (produced)
KAFKA_TOPIC_STOCK_UPDATED=xyn.inventory.stock_updated
KAFKA_TOPIC_LOW_STOCK_ALERT=xyn.inventory.low_stock_alert
```

### 3.5 Kitchen Service

```bash
# services/kitchen/.env.example

SERVICE_NAME=kitchen-service
GRPC_PORT=9005
HTTP_PORT=8005
METRICS_PORT=2116

DATABASE_URL=postgresql://xyn_app:CHANGE_ME@localhost:5432/xyn_kitchen?sslmode=disable

# KDS configuration
KDS_STREAM_KEEPALIVE=30s        # gRPC stream keepalive interval
KDS_TICKET_WARN_MINUTES=5       # Yellow alert threshold
KDS_TICKET_URGENT_MINUTES=10    # Red alert threshold

# Staff
CLOCK_IN_GRACE_PERIOD=15m       # Allow clock-in up to 15 min before shift
COMMISSION_CALCULATION_CRON=0 23 * * *  # Daily at 11 PM
```

### 3.6 Analytics Service

```bash
# services/analytics/.env.example

SERVICE_NAME=analytics-service
GRPC_PORT=9007
HTTP_PORT=8007
METRICS_PORT=2118

# ClickHouse 26.5
CLICKHOUSE_URL=clickhouse://localhost:9000/xyn_analytics
CLICKHOUSE_USERNAME=xyn_analytics
CLICKHOUSE_PASSWORD=CHANGE_ME  # REQUIRED
CLICKHOUSE_MAX_CONN=10
CLICKHOUSE_READ_TIMEOUT=30s    # Analytics queries can be slow

# Kafka (consumed topics for ClickHouse ingestion)
KAFKA_TOPIC_ORDERS=xyn.pos.order_paid
KAFKA_TOPIC_PAYMENTS=xyn.payment.completed
KAFKA_TOPIC_STOCK=xyn.inventory.stock_updated

# AI/ML features
AI_FORECASTING_ENABLED=false    # Enable when AI service is ready
AI_SERVICE_URL=                 # Internal AI inference service
```

---

## 4. Web App Environment Variables

```bash
# apps/web/.env.example (Next.js 16)

# Public (exposed to browser — never put secrets here)
NEXT_PUBLIC_API_URL=http://localhost:8080      # KrakenD API Gateway
NEXT_PUBLIC_WS_URL=ws://localhost:8080
NEXT_PUBLIC_APP_ENV=development
NEXT_PUBLIC_SENTRY_DSN=                        # Sentry error tracking (optional)
NEXT_PUBLIC_POSTHOG_KEY=                       # PostHog analytics (optional)

# Server-only (BFF API routes — never exposed to browser)
API_GATEWAY_INTERNAL_URL=http://localhost:8080
KEYCLOAK_URL=http://localhost:8180
KEYCLOAK_REALM=xyn
KEYCLOAK_CLIENT_ID=xyn-web
KEYCLOAK_CLIENT_SECRET=CHANGE_ME               # REQUIRED
NEXTAUTH_SECRET=CHANGE_ME                      # REQUIRED — 32+ char random string
NEXTAUTH_URL=http://localhost:3000

# Feature flags
NEXT_PUBLIC_ENABLE_KDS=true
NEXT_PUBLIC_ENABLE_ANALYTICS=false             # ClickHouse analytics dashboard
NEXT_PUBLIC_ENABLE_AI_FEATURES=false
```

---

## 5. Mobile App Environment Variables

Flutter uses `--dart-define` or `flutter_dotenv` package:

```bash
# apps/mobile/.env.example

API_BASE_URL=http://10.0.2.2:8080    # Android emulator localhost
# API_BASE_URL=http://localhost:8080  # iOS simulator
# API_BASE_URL=https://api.xyn.app    # Production

WS_BASE_URL=ws://10.0.2.2:8080
APP_ENV=development

# Keycloak
AUTH_DISCOVERY_URL=http://10.0.2.2:8180/realms/xyn/.well-known/openid-configuration
AUTH_CLIENT_ID=xyn-mobile

# Feature flags
ENABLE_OFFLINE_MODE=true
ENABLE_BLUETOOTH_PRINTER=true
ENABLE_CAMERA_SCANNER=true
SYNC_INTERVAL_SECONDS=30            # Background sync frequency
CATALOG_SYNC_INTERVAL_HOURS=1       # Product catalog refresh
```

```dart
// apps/mobile/lib/core/config/app_config.dart
class AppConfig {
  static const apiBaseUrl = String.fromEnvironment('API_BASE_URL',
      defaultValue: 'http://localhost:8080');
  static const appEnv = String.fromEnvironment('APP_ENV',
      defaultValue: 'development');
  static const enableOfflineMode = bool.fromEnvironment('ENABLE_OFFLINE_MODE',
      defaultValue: true);

  static bool get isProduction => appEnv == 'production';
  static bool get isDevelopment => appEnv == 'development';
}
```

---

## 6. Infrastructure Environment Variables

```bash
# infra/.env.example

# ── MinIO (Object Storage — S3-compatible) ────────────────────────────
MINIO_ROOT_USER=xyn_admin
MINIO_ROOT_PASSWORD=CHANGE_ME        # REQUIRED — min 8 chars
MINIO_BUCKET_RECEIPTS=xyn-receipts
MINIO_BUCKET_PRODUCTS=xyn-product-images
MINIO_BUCKET_REPORTS=xyn-reports
MINIO_ENDPOINT=localhost:9000
MINIO_USE_SSL=false                  # true in staging/prod

# ── Keycloak 26.6.3 ───────────────────────────────────────────────────
KC_DB=postgres
KC_DB_URL=jdbc:postgresql://localhost:5432/keycloak
KC_DB_USERNAME=keycloak
KC_DB_PASSWORD=CHANGE_ME             # REQUIRED
KC_HOSTNAME=localhost
KC_HTTP_PORT=8180
KC_BOOTSTRAP_ADMIN_USERNAME=admin
KC_BOOTSTRAP_ADMIN_PASSWORD=CHANGE_ME  # REQUIRED — change immediately after first boot

# ── KrakenD 2.13.7 ────────────────────────────────────────────────────
# KrakenD uses a JSON config file (not env vars) — see infra/krakend/krakend.json
# But runtime overrides:
KRAKEND_PORT=8080
KRAKEND_DEBUG=false                  # true in development
KRAKEND_LOG_LEVEL=WARNING            # DEBUG | INFO | WARNING | ERROR

# ── Grafana 13.0.2 ────────────────────────────────────────────────────
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=CHANGE_ME  # REQUIRED
GF_SERVER_DOMAIN=grafana.xyn.local
GF_SMTP_ENABLED=false                 # Enable for alert email notifications
GF_INSTALL_PLUGINS=grafana-piechart-panel,grafana-worldmap-panel
```

---

## 7. Kafka Topic Naming Convention

```
xyn.{service}.{entity}.{event}

Examples:
xyn.pos.order.created
xyn.pos.order.paid
xyn.payment.payment.completed
xyn.payment.refund.processed
xyn.inventory.stock.updated
xyn.inventory.product.created
xyn.tenant.user.registered
xyn.kitchen.ticket.ready
xyn.marketing.member.points_added
```

**Topic configuration (set via Kafka AdminClient on startup):**
```
Partitions:  12 (default — scale by tenant count)
Replication: 3 (production) | 1 (local dev)
Retention:   7 days (default) | 30 days (financial topics)
Cleanup:     delete (not compact — we want replay capability)
```

---

## 8. Secret Rotation Procedure

When a secret is compromised or needs rotation:

```bash
# 1. Generate new secret
openssl rand -hex 32  # For symmetric keys
openssl genrsa 4096   # For RSA keys

# 2. Update in Vault (production)
vault kv put secret/xyn-pos/payment MIDTRANS_SERVER_KEY=new_value

# 3. K8s External Secrets Operator auto-syncs (within 1 minute)
kubectl get externalsecrets -n xyn-production

# 4. Rolling restart (zero-downtime with HPA)
kubectl rollout restart deployment/payment-service -n xyn-production

# 5. Verify new secret is active
kubectl logs -l app=payment-service -n xyn-production | grep "config loaded"

# 6. Revoke old secret at the provider (Stripe, Xendit, etc.)

# 7. Document in incident log: what, when, who, why
```

---

## 9. Environment-Specific Overrides

| Variable | Local Dev | Staging | Production |
|---|---|---|---|
| `LOG_LEVEL` | `debug` | `info` | `warn` |
| `LOG_FORMAT` | `text` | `json` | `json` |
| `POSTGRES_SSL_MODE` | `disable` | `require` | `require` |
| `OTEL_TRACES_SAMPLER_ARG` | `1.0` (100%) | `0.5` (50%) | `0.1` (10%) |
| `KAFKA_SECURITY_PROTOCOL` | `PLAINTEXT` | `SASL_SSL` | `SASL_SSL` |
| `MIDTRANS_ENV` | `sandbox` | `sandbox` | `production` |
| `NEGATIVE_STOCK_ALLOWED` | `true` | `false` | `false` |
| `VOID_WINDOW_HOURS` | `720` (30 days) | `24` | `24` |

# Deployment Guide — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Stack: Docker | K3s 1.36.1 | ArgoCD 3.4.3 | Terraform 1.15.5 | Kafka 4.3.0 KRaft

---

## 1. Environments

| Environment | Infrastructure | Deployment Method | Purpose |
|---|---|---|---|
| **Local Dev** | Docker Compose | `make infra-up` | Daily development |
| **Staging** | K3s (single-node VM) | ArgoCD GitOps | Pre-release testing |
| **Production** | K3s / K8s (multi-node) | ArgoCD GitOps | Live traffic |
| **Home Server** | K3s (Raspberry Pi / baremetal) | ArgoCD GitOps | Personal staging |

---

## 2. Docker Setup

### 2.1 Multi-Stage Dockerfile (All Go Services)

```dockerfile
# services/pos/Dockerfile

# ── Stage 1: Build ────────────────────────────────────────────────────
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/pos-service \
    ./cmd/server

# ── Stage 2: Security scan (optional — add in CI) ────────────────────
# FROM aquasec/trivy:latest AS scanner
# COPY --from=builder /app/bin/pos-service /bin/pos-service
# RUN trivy filesystem --exit-code 1 --severity HIGH,CRITICAL /bin/pos-service

# ── Stage 3: Runtime (minimal, non-root) ─────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/pos-service /pos-service

# Embed migrations in binary — no external files needed at runtime
# (handled via embed.FS in Go code)

EXPOSE 9002   # gRPC
EXPOSE 8002   # HTTP/REST (gRPC-Gateway)
EXPOSE 2113   # Prometheus metrics

USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/pos-service", "healthcheck"]

ENTRYPOINT ["/pos-service"]
```

### 2.2 Docker Compose (Local Dev)

```yaml
# docker-compose.yml
name: xyn-pos-v1

services:
  # ── PostgreSQL 18.4 ───────────────────────────────────────────────────
  postgres:
    image: postgres:18.4-alpine
    environment:
      POSTGRES_USER: xyn_admin
      POSTGRES_PASSWORD: dev_password_change_me
      POSTGRES_DB: xyn_pos
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./infra/postgres/init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U xyn_admin -d xyn_pos"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ── Redis 8.8 ─────────────────────────────────────────────────────────
  redis:
    image: redis:8.8-alpine
    command: redis-server --save 60 1 --loglevel warning
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s

  # ── Kafka 4.3.0 KRaft (NO ZooKeeper) ─────────────────────────────────
  kafka:
    image: apache/kafka:4.3.0
    environment:
      # KRaft mode — ZooKeeper removed in Kafka 4.x
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093,EXTERNAL://0.0.0.0:29092
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092,EXTERNAL://localhost:29092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT,EXTERNAL:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: false
      CLUSTER_ID: MkU3OEVBNTcwNTJENDM2Qg   # Fixed cluster ID for local dev
    ports:
      - "29092:29092"  # External (from host)
    volumes:
      - kafka_data:/var/lib/kafka/data
    healthcheck:
      test: ["CMD-SHELL", "/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka:9092 > /dev/null 2>&1"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s

  # Kafka topic initialization (run once)
  kafka-init:
    image: apache/kafka:4.3.0
    depends_on:
      kafka:
        condition: service_healthy
    command: >
      bash -c "
        /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic xyn.pos.order.created --partitions 12 --replication-factor 1 &&
        /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic xyn.payment.completed --partitions 12 --replication-factor 1 &&
        /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic xyn.inventory.stock_updated --partitions 12 --replication-factor 1 &&
        echo 'Kafka topics created successfully'
      "
    restart: "no"

  # ── ClickHouse 26.5 ───────────────────────────────────────────────────
  clickhouse:
    image: clickhouse/clickhouse-server:26.5
    environment:
      CLICKHOUSE_DB: xyn_analytics
      CLICKHOUSE_USER: xyn_analytics
      CLICKHOUSE_PASSWORD: dev_password_change_me
    ports:
      - "8123:8123"   # HTTP
      - "9000:9000"   # Native
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    healthcheck:
      test: ["CMD", "clickhouse-client", "--query", "SELECT 1"]
      interval: 10s

  # ── MinIO 8.x (S3-compatible object storage) ──────────────────────────
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: xyn_admin
      MINIO_ROOT_PASSWORD: dev_password_change_me
    ports:
      - "9000:9000"   # S3 API
      - "9001:9001"   # Web console (http://localhost:9001)
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 30s

  # ── Keycloak 26.6.3 ───────────────────────────────────────────────────
  keycloak:
    image: quay.io/keycloak/keycloak:26.6.3
    command: start-dev --import-realm
    environment:
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://postgres:5432/keycloak
      KC_DB_USERNAME: xyn_admin
      KC_DB_PASSWORD: dev_password_change_me
      KC_BOOTSTRAP_ADMIN_USERNAME: admin
      KC_BOOTSTRAP_ADMIN_PASSWORD: admin_change_me
      KC_HTTP_PORT: 8180
      KC_HOSTNAME_STRICT: false
    ports:
      - "8180:8180"
    volumes:
      - ./infra/keycloak/realm-export.json:/opt/keycloak/data/import/realm.json
    depends_on:
      postgres:
        condition: service_healthy

  # ── Observability Stack ───────────────────────────────────────────────
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP

  prometheus:
    image: prom/prometheus:v3.12.0
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=15d'
      - '--web.enable-lifecycle'
    ports:
      - "9090:9090"
    volumes:
      - ./infra/prometheus:/etc/prometheus
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:13.0.2
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin_change_me
      GF_INSTALL_PLUGINS: grafana-piechart-panel
    ports:
      - "3100:3000"  # Grafana UI (not 3000 — conflicts with Next.js)
    volumes:
      - grafana_data:/var/lib/grafana
      - ./infra/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./infra/grafana/datasources:/etc/grafana/provisioning/datasources
    depends_on:
      - prometheus

  loki:
    image: grafana/loki:latest
    ports:
      - "3200:3100"
    volumes:
      - loki_data:/loki

volumes:
  postgres_data:
  redis_data:
  kafka_data:
  clickhouse_data:
  minio_data:
  prometheus_data:
  grafana_data:
  loki_data:
```

---

## 3. K3s Setup (Home Server / Staging)

### 3.1 Install K3s 1.36.1

```bash
# On the server node (Ubuntu 24.04 LTS recommended)
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.36.1+k3s1" sh -s - \
    --disable traefik \          # We'll install Traefik manually for more control
    --disable servicelb \        # Use MetalLB instead
    --write-kubeconfig-mode 644

# Get kubeconfig for local access
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get nodes  # Should show Ready

# For multi-node (add agent nodes):
# SERVER_IP=$(hostname -I | awk '{print $1}')
# K3S_TOKEN=$(cat /var/lib/rancher/k3s/server/node-token)
# On each agent: curl -sfL https://get.k3s.io | K3S_URL=https://$SERVER_IP:6443 K3S_TOKEN=$K3S_TOKEN sh -
```

### 3.2 Namespace Structure

```bash
kubectl create namespace xyn-infra      # Kafka, PostgreSQL, Redis
kubectl create namespace xyn-services   # Go microservices
kubectl create namespace xyn-web        # Next.js
kubectl create namespace xyn-observability  # Grafana, Prometheus, Loki, Jaeger
kubectl create namespace cert-manager
kubectl create namespace argocd
```

### 3.3 Install ArgoCD 3.4.3

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f \
    https://raw.githubusercontent.com/argoproj/argo-cd/v3.4.3/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=300s \
    deployment/argocd-server -n argocd

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
    -o jsonpath="{.data.password}" | base64 -d

# Port-forward for initial setup (then configure ingress)
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Access: https://localhost:8080
```

### 3.4 ArgoCD Application Definition

```yaml
# infra/k8s/argocd/apps/pos-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: pos-service
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/xyn/xyn-pos-v1.git
    targetRevision: main
    path: infra/k8s/services/pos
  destination:
    server: https://kubernetes.default.svc
    namespace: xyn-services
  syncPolicy:
    automated:
      prune: true      # Delete resources removed from Git
      selfHeal: true   # Re-apply if someone manually changes in cluster
    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

---

## 4. Kubernetes Manifests

### 4.1 Service Deployment

```yaml
# infra/k8s/services/pos/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pos-service
  namespace: xyn-services
  labels:
    app: pos-service
    version: "1.0.0"
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pos-service
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0   # Zero-downtime rolling update
  template:
    metadata:
      labels:
        app: pos-service
        version: "1.0.0"
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "2113"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: pos-service
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532     # nonroot user (distroless)
        fsGroup: 65532
      containers:
        - name: pos-service
          image: ghcr.io/xyn/pos-service:1.0.0
          ports:
            - containerPort: 9002   # gRPC
              name: grpc
            - containerPort: 8002   # HTTP
              name: http
            - containerPort: 2113   # Metrics
              name: metrics
          env:
            - name: APP_ENV
              value: production
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: pos-service-secrets
                  key: database-url
            - name: KAFKA_BROKERS
              valueFrom:
                configMapKeyRef:
                  name: kafka-config
                  key: brokers
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
          livenessProbe:
            grpc:
              port: 9002
            initialDelaySeconds: 10
            periodSeconds: 30
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: 8002
            initialDelaySeconds: 5
            periodSeconds: 10
            failureThreshold: 3
          startupProbe:
            httpGet:
              path: /health
              port: 8002
            failureThreshold: 30
            periodSeconds: 5   # Allow 30 × 5s = 150s for startup
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: pos-service-pdb
  namespace: xyn-services
spec:
  minAvailable: 1    # Always at least 1 pod running
  selector:
    matchLabels:
      app: pos-service
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: pos-service-hpa
  namespace: xyn-services
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: pos-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60    # Scale up quickly
      policies:
        - type: Pods
          value: 2
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300   # Scale down slowly (5 min)
      policies:
        - type: Pods
          value: 1
          periodSeconds: 120
```

### 4.2 Sealed Secrets (for sensitive config)

```bash
# Install sealed-secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.27.0/controller.yaml

# Create a secret and seal it
kubectl create secret generic pos-service-secrets \
    --from-literal=database-url="postgresql://..." \
    --dry-run=client \
    -o yaml | \
    kubeseal --format yaml > infra/k8s/services/pos/sealed-secret.yaml

# Commit sealed-secret.yaml to git — it's safe (encrypted with cluster key)
# The controller decrypts it and creates a regular Secret in the cluster
```

---

## 5. Blue-Green Deployment

For zero-downtime major releases (schema migrations, breaking changes):

```yaml
# infra/k8s/services/pos/blue-green.yaml
# Both blue and green are always deployed. Traffic switches via the Service selector.

apiVersion: apps/v1
kind: Deployment
metadata:
  name: pos-service-blue
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pos-service
      slot: blue
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pos-service-green
spec:
  replicas: 0   # Scaled to 0 when inactive
  selector:
    matchLabels:
      app: pos-service
      slot: green
---
apiVersion: v1
kind: Service
metadata:
  name: pos-service
spec:
  selector:
    app: pos-service
    slot: blue   # Switch to 'green' to cut over
  ports:
    - port: 9002
      name: grpc
    - port: 8002
      name: http
```

**Blue-Green cutover procedure:**
```bash
# 1. Deploy new version to green slot
kubectl set image deployment/pos-service-green pos-service=ghcr.io/xyn/pos-service:2.0.0

# 2. Scale up green
kubectl scale deployment/pos-service-green --replicas=2

# 3. Wait for green to be ready
kubectl rollout status deployment/pos-service-green

# 4. Run smoke tests against green (internal endpoint)
./tests/smoke/run.sh pos-service-green

# 5. Switch traffic to green
kubectl patch service pos-service -p '{"spec":{"selector":{"slot":"green"}}}'

# 6. Monitor error rate for 5 minutes
# 7. If OK: scale down blue
kubectl scale deployment/pos-service-blue --replicas=0

# 8. If problems: rollback in seconds (switch back to blue)
kubectl patch service pos-service -p '{"spec":{"selector":{"slot":"blue"}}}'
```

---

## 6. CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go 1.26.4
        uses: actions/setup-go@v5
        with:
          go-version: '1.26.4'

      - name: Run tests
        run: make test

      - name: Run lint
        run: make lint

      - name: Security scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          severity: 'HIGH,CRITICAL'
          exit-code: '1'

      - name: Build and push Docker images
        uses: docker/build-push-action@v6
        with:
          context: services/pos
          push: true
          tags: |
            ghcr.io/xyn/pos-service:${{ github.sha }}
            ghcr.io/xyn/pos-service:latest
          build-args: |
            VERSION=${{ github.sha }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}

  update-manifests:
    needs: build-and-push
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.ARGOCD_DEPLOY_TOKEN }}

      # Update image tag in K8s manifest
      - name: Update image tag
        run: |
          sed -i "s|ghcr.io/xyn/pos-service:.*|ghcr.io/xyn/pos-service:${{ github.sha }}|" \
            infra/k8s/services/pos/deployment.yaml
          git config user.email "ci@xyn.app"
          git config user.name "CI Bot"
          git add infra/k8s/services/pos/deployment.yaml
          git commit -m "chore(deploy): update pos-service to ${{ github.sha }}"
          git push

      # ArgoCD detects the Git change and syncs automatically (GitOps)
```

---

## 7. Rollback Procedure

### Emergency Rollback (< 2 minutes)

```bash
# Option 1: Kubernetes rollback (fastest)
kubectl rollout undo deployment/pos-service -n xyn-services

# Option 2: ArgoCD rollback to specific revision
argocd app rollback pos-service --revision <REVISION_NUMBER>

# Option 3: Revert Git commit (triggers ArgoCD auto-sync)
git revert HEAD --no-edit
git push origin main

# Check rollback status
kubectl rollout status deployment/pos-service -n xyn-services
kubectl get pods -l app=pos-service -n xyn-services
```

### Database Migration Rollback

```bash
# Roll back last migration for a service
make migrate-down SERVICE=pos

# Roll back to specific version
goose -dir services/pos/internal/infrastructure/postgres/migrations \
    postgres "$DATABASE_URL" down-to 20260605000000
```

---

## 8. Home Server (Baremetal K3s) Setup

```bash
# Install K3s on home server (Raspberry Pi 5 / mini PC)
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.36.1+k3s1" sh -s - \
    --disable traefik \
    --node-name home-server

# Install MetalLB for LoadBalancer support on baremetal
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.9/config/manifests/metallb-native.yaml

# Configure IP address pool for your home network
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: home-pool
  namespace: metallb-system
spec:
  addresses:
  - 192.168.1.200-192.168.1.210  # Reserve these IPs on your router
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: home-l2
  namespace: metallb-system
EOF

# Install cert-manager for TLS (optional for home server)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml

# Storage: Use local-path provisioner (built into K3s)
kubectl get storageclass  # Should show "local-path" as default
```

---

## 9. Deployment Checklist

```
Before deploying to production:
✅ All tests pass (unit + integration)
✅ Docker image scanned for vulnerabilities (Trivy)
✅ Database migrations reviewed (zero-downtime strategy confirmed)
✅ Rollback plan documented
✅ SLO metrics being tracked (checkout success rate, latency)
✅ On-call engineer notified
✅ Deploy during low-traffic window (if major change)

After deploying to production:
✅ Deployment rollout status confirmed (kubectl rollout status)
✅ Health checks passing (/health, /ready)
✅ Error rate not elevated (check Grafana)
✅ Latency within SLO (check Grafana)
✅ Kafka consumer lag normal
✅ No increase in Jaeger error traces
✅ Smoke test passed (critical checkout flow)
```

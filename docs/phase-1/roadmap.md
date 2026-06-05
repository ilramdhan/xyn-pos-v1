# Project Roadmap — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer  
> Format: Epics → Stories → Tasks (Kanban-ready)

---

## Overview

| Phase | Name | Status | Est. Duration |
|---|---|---|---|
| Phase 1 | Architecture & Blueprinting | ✅ Done | 1–2 weeks |
| Phase 2 | Environment & Infrastructure Setup | ✅ Done | 1–2 weeks |
| Phase 3 | Core Domain & Microservices Boilerplate | ⬜ Planned | 2–3 weeks |
| Phase 4 | Backend MVP Implementation | ⬜ Planned | 6–8 weeks |
| Phase 5 | Frontend Web & Mobile Implementation | ⬜ Planned | 6–8 weeks |
| Phase 6 | Advanced Features & DevOps | ⬜ Planned | Ongoing |

---

## Phase 1 — Architecture & Blueprinting

### Epic 1.1: Architecture Documentation

| # | Story | Status |
|---|---|---|
| 1.1.1 | Write `architecture.md` (bounded contexts, tech stack decisions, ERD) | ✅ Done |
| 1.1.2 | Write `database-strategy.md` (multi-tenancy, sync strategy) | ✅ Done |
| 1.1.3 | Write `hardware-integration.md` (ESC/POS, barcode, cash drawer) | ✅ Done |
| 1.1.4 | Write `rules.md` (coding standards, git workflow, testing) | ✅ Done |
| 1.1.5 | Write `design.md` (UI/UX design system) | ✅ Done |
| 1.1.6 | Write `api-contracts.md` (gRPC protobuf standards, idempotency) | ✅ Done |
| 1.1.7 | Write `roadmap.md` (this document) | ✅ Done |
| 1.1.8 | Create Architecture Decision Records (ADR) directory | ✅ Done |
| 1.1.9 | Write CLAUDE.md (AI assistant context file) | ✅ Done |
| 1.1.10 | Write `response-standards.md` (BaseResponse[T] generic pattern, OpenAPI rules, error codes) | ✅ Done |
| 1.1.11 | Write `error-handling.md` (error taxonomy, samber/oops, gRPC mapping, recovery) | ✅ Done |
| 1.1.12 | Write `testing-strategy.md` (pyramid, testcontainers, k6 stress tests, SLOs) | ✅ Done |
| 1.1.13 | Write `contributing.md` (setup, Manual DI template, pre-commit, gotchas) | ✅ Done |
| 1.1.14 | Write `environment-variables.md` (all env vars per service, secret rotation) | ✅ Done |
| 1.1.15 | Write `observability.md` (OTEL, Prometheus, Grafana, Loki, Jaeger, alerting) | ✅ Done |
| 1.1.16 | Write `mobile-architecture.md` (Flutter, Riverpod 3.x, Drift, offline sync) | ✅ Done |
| 1.1.17 | Write `deployment-guide.md` (Docker, K3s, ArgoCD, blue-green, CI/CD) | ✅ Done |

---

## Phase 2 — Environment & Infrastructure Setup

### Epic 2.1: Local Development Environment

| # | Story | Tasks | Priority | Status |
|---|---|---|---|---|
| 2.1.1 | Initialize monorepo structure | `services/`, `apps/`, `proto/`, `shared/`, `infra/`, `docs/`, `tests/` | 🔴 High | ✅ Done |
| 2.1.2 | Docker Compose for local dev | PostgreSQL 18.4, Redis 8.8, Kafka 4.3 KRaft (NO ZooKeeper), MinIO, Keycloak, full observability stack | 🔴 High | ✅ Done |
| 2.1.3 | Go workspace setup | `go.work`, `shared/go/go.mod`, `shared/go/pkg` skeleton | 🔴 High | ✅ Done |
| 2.1.4 | Buf setup | `buf.yaml`, `buf.gen.yaml` (Go + TS + Dart codegen), common protos (Money, Pagination, Error) | 🔴 High | ✅ Done |
| 2.1.5 | Golangci-lint setup | `.golangci.yml` v2 format, depguard DDD layer rules, Makefile | 🟡 Medium | ✅ Done |
| 2.1.6 | Pre-commit hooks | `.pre-commit-config.yaml`: gitleaks, golangci-lint, buf lint, yamllint | 🟡 Medium | ✅ Done |
| 2.1.7 | Turborepo setup | `turbo.json` for affected builds, proto-gen caching | 🟡 Medium | ✅ Done |
| 2.1.8 | Root config files | `.env.example`, `.gitignore`, `Makefile` with all dev targets | 🔴 High | ✅ Done |
| 2.1.9 | Infrastructure configs | `infra/prometheus/`, `infra/grafana/`, `infra/promtail/` provisioning | 🟡 Medium | ✅ Done |
| 2.1.10 | Service Dockerfile template | `scripts/templates/Dockerfile.service` — multi-stage, distroless, multi-arch | 🔴 High | ✅ Done |
| 2.1.11 | PostgreSQL init script | `scripts/postgres/init.sql` — databases per bounded context, RLS roles, helper function | 🔴 High | ✅ Done |

### Epic 2.2: CI/CD Pipeline

| # | Story | Tasks | Priority | Status |
|---|---|---|---|---|
| 2.2.1 | GitHub Actions: Go CI | Build, test (unit + integration), lint, security scan | 🔴 High | ✅ Done |
| 2.2.2 | GitHub Actions: Docker build | Affected-only multi-arch builds, push to GHCR, Trivy scan | 🟡 Medium | ✅ Done |
| 2.2.3 | GitHub Actions: Buf lint | Lint, format check, breaking change detection for protos | 🟡 Medium | ✅ Done |
| 2.2.4 | GitHub Actions: Security scan | `gosec`, `govulncheck` in Go CI; `trivy` in Docker build | 🟡 Medium | ✅ Done |
| 2.2.5 | ArgoCD setup | `argocd-values.yml`, App-of-Apps CRD, ApplicationSet for all 6 services | 🟢 Low | ✅ Done |

### Epic 2.3: K3s Home Server / Staging Environment

| # | Story | Tasks | Priority | Status |
|---|---|---|---|---|
| 2.3.1 | K3s installation | `scripts/k3s/install.sh` — K3s 1.36.1, kubeconfig setup | 🟡 Medium | ✅ Done |
| 2.3.2 | Ingress + TLS | `traefik-values.yml` + `cluster-issuer.yml` (Let's Encrypt prod + staging) | 🟡 Medium | ✅ Done |
| 2.3.3 | Persistent volumes | `storage-classes.yml` — xyn-fast (Retain) + xyn-bulk, PVCs for all infra | 🟡 Medium | ✅ Done |
| 2.3.4 | Namespace structure | `namespaces.yml` — xyn-dev, xyn-staging, xyn-prod, xyn-infra, monitoring | 🟡 Medium | ✅ Done |
| 2.3.5 | Sealed Secrets | `post-install.sh` auto-backup, `README.md` with full usage guide | 🟡 Medium | ✅ Done |

---

## Phase 3 — Core Domain & Microservices Boilerplate

### Epic 3.1: Shared Go Packages

| # | Story | Tasks | Priority |
|---|---|---|---|
| 3.1.1 | Error handling package | `shared/go/pkg/errors`: domain errors, oops wrapper, gRPC error helpers | 🔴 High |
| 3.1.2 | Logger package | `shared/go/pkg/logger`: slog structured logger, context extraction | 🔴 High |
| 3.1.3 | Auth package | `shared/go/pkg/auth`: PASETO token generation/verification, claims struct | 🔴 High |
| 3.1.4 | Database package | `shared/go/pkg/database`: pgxpool factory, tenant context helper | 🔴 High |
| 3.1.5 | Telemetry package | `shared/go/pkg/telemetry`: OpenTelemetry setup, span helpers | 🟡 Medium |
| 3.1.6 | Events package | `shared/go/pkg/events`: Kafka producer/consumer factory | 🟡 Medium |
| 3.1.7 | Middleware package | `shared/go/pkg/middleware`: gRPC interceptors (auth, logging, tracing, recovery) | 🔴 High |

### Epic 3.2: Common Proto Definitions

| # | Story | Tasks | Priority |
|---|---|---|---|
| 3.2.1 | Common types proto | `Money`, `Pagination`, `PaginationMeta`, `Address` | 🔴 High |
| 3.2.2 | Error details proto | Standard error structures | 🔴 High |
| 3.2.3 | Generate Go + TS + Dart code | Configure buf.gen.yaml, run generation, commit generated code | 🔴 High |

### Epic 3.3: First Microservice — Tenant Service (Boilerplate)

Use the Tenant service as the template — all other services follow this pattern.

| # | Story | Tasks | Priority |
|---|---|---|---|
| 3.3.1 | Tenant domain model | `Tenant` aggregate, `Branch` entity, value objects, domain events | 🔴 High |
| 3.3.2 | Tenant repository interface | Port definition in domain layer | 🔴 High |
| 3.3.3 | Tenant use cases | `CreateTenant`, `CreateBranch`, `GetTenant` command/query handlers | 🔴 High |
| 3.3.4 | PostgreSQL repository | Implement repository with pgx, goose migrations | 🔴 High |
| 3.3.5 | gRPC handler | Translate proto ↔ domain, call application layer | 🔴 High |
| 3.3.6 | Manual DI wiring | Write `provider.go` with constructor chaining (see ADR-0001) | 🔴 High |
| 3.3.7 | Dockerfile | Multi-stage build, distroless runtime | 🟡 Medium |
| 3.3.8 | Unit tests for domain | Table-driven tests for aggregate behavior | 🔴 High |
| 3.3.9 | Integration tests | Testcontainers PostgreSQL, test real repository | 🟡 Medium |
| 3.3.10 | Goose migrations | Initial schema, RLS policies | 🔴 High |

### Epic 3.4: Database Infrastructure

| # | Story | Tasks | Priority |
|---|---|---|---|
| 3.4.1 | PostgreSQL schema design | Run through ERD from architecture.md, finalize all tables | 🔴 High |
| 3.4.2 | RLS policy templates | `enable_rls.sql` template function for all tables | 🔴 High |
| 3.4.3 | PgBouncer config | Docker Compose + K8s deployment | 🟡 Medium |
| 3.4.4 | Redis setup | Redis Cluster config for local dev | 🟡 Medium |
| 3.4.5 | ClickHouse setup | Docker Compose, initial schema for analytics tables | 🟢 Low |

---

## Phase 4 — Backend MVP Implementation

### Epic 4.1: Identity & Auth Service

| # | Story | Tasks | Priority |
|---|---|---|---|
| 4.1.1 | Keycloak configuration | Realm setup, client credentials, RBAC roles | 🔴 High |
| 4.1.2 | User registration flow | `RegisterUser` command: create in Keycloak + PostgreSQL | 🔴 High |
| 4.1.3 | Login + token issuance | Exchange Keycloak token for internal PASETO token | 🔴 High |
| 4.1.4 | RBAC permission model | Define roles: `owner`, `manager`, `cashier`, `kitchen_staff` | 🔴 High |
| 4.1.5 | PIN verification | For sensitive operations: void, open drawer, discount override | 🔴 High |
| 4.1.6 | Multi-branch user scope | Users scoped to one or all branches | 🟡 Medium |
| 4.1.7 | Subscription model | `SubscriptionPlan` entity, plan limits enforcement | 🟡 Medium |

### Epic 4.2: Product & Menu Service

| # | Story | Tasks | Priority |
|---|---|---|---|
| 4.2.1 | Product domain model | `Product` aggregate, `Category` entity, `Variant` entity | 🔴 High |
| 4.2.2 | Product CRUD | Create, update, archive products | 🔴 High |
| 4.2.3 | Category management | Create, reorder, nest categories | 🔴 High |
| 4.2.4 | Menu configuration | Enable/disable products per branch, branch-specific pricing | 🟡 Medium |
| 4.2.5 | Add-on groups | Define add-ons (toppings, size, etc.) linked to products | 🟡 Medium |
| 4.2.6 | Time-based pricing | Happy hour pricing, time-based rules | 🟢 Low |
| 4.2.7 | Barcode lookup | `LookupBySKU` endpoint for scanner integration | 🔴 High |

### Epic 4.3: POS Core Service

| # | Story | Tasks | Priority |
|---|---|---|---|
| 4.3.1 | Order aggregate | Full domain model with state machine | 🔴 High |
| 4.3.2 | Create order | With table number, order type | 🔴 High |
| 4.3.3 | Add / remove items | With add-ons and notes | 🔴 High |
| 4.3.4 | Tax calculation | PPN 11%, PB1 10% configurable per branch | 🔴 High |
| 4.3.5 | Discount application | Fixed amount, percentage, per-item vs total | 🟡 Medium |
| 4.3.6 | Cancel order | Business rules: only before payment | 🔴 High |
| 4.3.7 | Shift management | Open/close shift, cash count, Z-report | 🟡 Medium |
| 4.3.8 | Pending orders | Park an order, come back to it | 🟡 Medium |
| 4.3.9 | Order events to Kafka | Publish `OrderCreated`, `OrderPaid` etc. | 🔴 High |

### Epic 4.4: Payment Service

| # | Story | Tasks | Priority |
|---|---|---|---|
| 4.4.1 | Payment domain model | `Payment` aggregate, idempotency | 🔴 High |
| 4.4.2 | Cash payment | Process with change calculation | 🔴 High |
| 4.4.3 | QRIS payment | Midtrans/Xendit QRIS integration | 🔴 High |
| 4.4.4 | Card payment | EDC integration stub (manual confirmation) | 🟡 Medium |
| 4.4.5 | Split payment | Multiple methods in one transaction | 🟡 Medium |
| 4.4.6 | Void payment | Reverse a completed payment (with PIN) | 🔴 High |
| 4.4.7 | Refund | Partial or full refund | 🟡 Medium |
| 4.4.8 | Receipt generation | Generate receipt data model, publish for printing | 🔴 High |
| 4.4.9 | Webhook handling | Midtrans/Xendit payment notification handling | 🔴 High |

### Epic 4.5: Inventory Service

| # | Story | Tasks | Priority |
|---|---|---|---|
| 4.5.1 | Stock ledger domain model | `StockLedger`, `StockMutation`, `Warehouse` | 🔴 High |
| 4.5.2 | Stock deduction on sale | Consume `OrderPaid` event, deduct stock | 🔴 High |
| 4.5.3 | Stock receipt on purchase | Receive stock from PO | 🟡 Medium |
| 4.5.4 | Low stock alerts | Threshold-based alerts via event | 🟡 Medium |
| 4.5.5 | Stock adjustment | Manual correction with reason code | 🟡 Medium |
| 4.5.6 | Stock transfer | Between warehouses with approval flow | 🟢 Low |
| 4.5.7 | Stock opname | Physical count reconciliation | 🟢 Low |
| 4.5.8 | BOM / Recipe | Bill of Materials for F&B products | 🟢 Low |

---

## Phase 5 — Frontend Web & Mobile Implementation

### Epic 5.1: Web Foundation

| # | Story | Tasks | Priority |
|---|---|---|---|
| 5.1.1 | Next.js project setup | App router, TypeScript, Tailwind CSS 4, shadcn/ui | 🔴 High |
| 5.1.2 | Auth flow | Login page, PASETO token management, protected routes | 🔴 High |
| 5.1.3 | gRPC-Web client setup | Connect to backend via gRPC-Web, generated TS clients | 🔴 High |
| 5.1.4 | TanStack Query setup | QueryClient, global error handling, optimistic updates | 🔴 High |
| 5.1.5 | Layout system | Sidebar, TopBar, page containers, responsive breakpoints | 🔴 High |
| 5.1.6 | Design tokens | CSS variables, Tailwind config, dark mode | 🟡 Medium |

### Epic 5.2: POS Interface

| # | Story | Tasks | Priority |
|---|---|---|---|
| 5.2.1 | Product panel | Category filter, product grid, search | 🔴 High |
| 5.2.2 | Cart panel | Item list, quantity controls, totals | 🔴 High |
| 5.2.3 | Payment modal | Method selection, amount input, change display | 🔴 High |
| 5.2.4 | Receipt preview | Pre-print preview, print trigger | 🔴 High |
| 5.2.5 | Barcode scanner | Keyboard HID listener, camera fallback | 🟡 Medium |
| 5.2.6 | Printer integration | WebUSB + network bridge adapter | 🟡 Medium |
| 5.2.7 | Cash drawer trigger | Post-payment auto-open | 🟡 Medium |
| 5.2.8 | Keyboard shortcuts | Full keyboard navigation for POS workflow | 🟡 Medium |
| 5.2.9 | Split payment UI | Multi-method payment flow | 🟢 Low |
| 5.2.10 | Pending orders | Park/resume order UI | 🟢 Low |

### Epic 5.3: Dashboard (Admin)

| # | Story | Tasks | Priority |
|---|---|---|---|
| 5.3.1 | Sales overview page | KPI cards, daily revenue chart | 🔴 High |
| 5.3.2 | Product management | CRUD pages for products, categories | 🔴 High |
| 5.3.3 | Order history | Filterable list, detail view, receipt reprint | 🔴 High |
| 5.3.4 | Inventory dashboard | Stock levels, low stock alerts | 🟡 Medium |
| 5.3.5 | User & role management | Invite user, assign role, set PIN | 🟡 Medium |
| 5.3.6 | Branch management | Create branches, configure settings | 🟡 Medium |
| 5.3.7 | Shift reports | Z-report, X-report, cash reconciliation | 🟡 Medium |

### Epic 5.4: KDS (Kitchen Display System)

| # | Story | Tasks | Priority |
|---|---|---|---|
| 5.4.1 | KDS board view | Ticket cards with age-based color coding | 🟡 Medium |
| 5.4.2 | Real-time updates | gRPC server-streaming connection | 🟡 Medium |
| 5.4.3 | Mark item done | Touch-to-complete item workflow | 🟡 Medium |
| 5.4.4 | Station filtering | Filter by kitchen station (grill, salad, etc.) | 🟢 Low |

### Epic 5.5: Mobile App (Flutter)

| # | Story | Tasks | Priority |
|---|---|---|---|
| 5.5.1 | Flutter project setup | Riverpod, Drift, go_router, freezed | 🔴 High |
| 5.5.2 | Auth flow | Login, token storage, biometric unlock | 🔴 High |
| 5.5.3 | Mobile POS screen | Product grid, cart, checkout | 🔴 High |
| 5.5.4 | Camera barcode scanner | mobile_scanner integration | 🟡 Medium |
| 5.5.5 | Bluetooth printer | esc_pos_utils + flutter_pos_printer_platform | 🟡 Medium |
| 5.5.6 | Offline-first data layer | Drift schema, repository pattern | 🔴 High |
| 5.5.7 | Background sync engine | Outbox pattern, sync protocol | 🔴 High |
| 5.5.8 | Conflict resolution UI | Show sync conflicts, let user choose | 🟡 Medium |
| 5.5.9 | Offline indicator | Connection status banner | 🔴 High |
| 5.5.10 | Push notifications | FCM integration (order ready, low stock) | 🟢 Low |

---

## Phase 6 — Advanced Features & DevOps

### Epic 6.1: Marketing & CRM

| # | Story | Tasks | Priority |
|---|---|---|---|
| 6.1.1 | Membership tiers | Bronze/Silver/Gold tier definitions | 🟡 Medium |
| 6.1.2 | Loyalty points | Earn on purchase, redeem at checkout | 🟡 Medium |
| 6.1.3 | Promo engine | Discount stacking rules, eligibility conditions | 🟡 Medium |
| 6.1.4 | Customer profiles | Purchase history, contact info | 🟡 Medium |
| 6.1.5 | WhatsApp notifications | Struk digital, promo blast, birthday greeting | 🟢 Low |
| 6.1.6 | GoFood/GrabFood integration | O2O order ingestion | 🟢 Low |

### Epic 6.2: Analytics

| # | Story | Tasks | Priority |
|---|---|---|---|
| 6.2.1 | Debezium CDC setup | PostgreSQL → Kafka → ClickHouse pipeline | 🟡 Medium |
| 6.2.2 | ClickHouse schema | Analytics tables, materialized views | 🟡 Medium |
| 6.2.3 | Real-time sales dashboard | Revenue, orders, avg transaction via ClickHouse | 🟡 Medium |
| 6.2.4 | Product performance | Top sellers, dead stock, sell-through rate | 🟢 Low |
| 6.2.5 | COGS & profitability | Gross margin per product (requires BOM) | 🟢 Low |
| 6.2.6 | AI demand forecasting | Time-series forecasting for stock replenishment | 🟢 Low |
| 6.2.7 | Customer churn prediction | ML model based on purchase frequency | 🟢 Low |

### Epic 6.3: Observability

| # | Story | Tasks | Priority |
|---|---|---|---|
| 6.3.1 | OpenTelemetry setup | Instrument all Go services with OTEL SDK | 🔴 High |
| 6.3.2 | Jaeger distributed tracing | Deploy Jaeger, connect OTEL exporter | 🟡 Medium |
| 6.3.3 | Prometheus metrics | Custom business metrics (orders/min, payment failures) | 🟡 Medium |
| 6.3.4 | Grafana dashboards | Service health, business KPIs, infrastructure | 🟡 Medium |
| 6.3.5 | Log aggregation | Promtail + Loki, log correlation with trace IDs | 🟡 Medium |
| 6.3.6 | Alerting rules | PagerDuty/Slack alerts for SLO breaches | 🟢 Low |

### Epic 6.4: Production Readiness (Kubernetes)

| # | Story | Tasks | Priority |
|---|---|---|---|
| 6.4.1 | Helm charts | Templated K8s manifests for all services | 🟡 Medium |
| 6.4.2 | HPA / VPA | Auto-scaling policies per service | 🟡 Medium |
| 6.4.3 | PodDisruptionBudget | Ensure availability during node maintenance | 🟡 Medium |
| 6.4.4 | Network policies | Restrict inter-service communication | 🟡 Medium |
| 6.4.5 | Load testing | k6 scenarios: checkout flow, inventory queries | 🟡 Medium |
| 6.4.6 | Blue-Green deployment | Zero-downtime deployment strategy via ArgoCD | 🟢 Low |
| 6.4.7 | Disaster recovery drill | Test backup restore, simulate database failure | 🟢 Low |

---

## Definition of Done (DoD)

A Story is **Done** when:
- [ ] Code passes all linting checks
- [ ] Unit tests written and passing (≥ coverage targets)
- [ ] Integration tests passing (if data access involved)
- [ ] PR reviewed and approved by ≥1 person
- [ ] Deployed to staging and smoke-tested
- [ ] No critical or high security findings from automated scans
- [ ] Documentation updated (if public API changed)

---

## Priority Legend

| Symbol | Meaning |
|---|---|
| 🔴 High | MVP blocker — must be done before product is usable |
| 🟡 Medium | Important feature — needed for full MVP but not day-1 |
| 🟢 Low | Nice-to-have — can be deferred to later phases |

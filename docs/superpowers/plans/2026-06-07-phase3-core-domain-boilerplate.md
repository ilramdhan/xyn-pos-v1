# Phase 3 — Core Domain & Microservices Boilerplate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the foundational layer for all 7 microservices: proto codegen, 7 shared Go packages, PostgreSQL schema + RLS, and the Tenant Service as the full DDD template all other services will follow.

**Architecture:** Bottom-up: Epic 3.2 (proto) → Epic 3.1 (shared packages) → Epic 3.4 (DB infra) → Epic 3.3 (Tenant Service). Each layer is a dependency of the next. Tenant Service is the canonical reference implementation of DDD + Manual DI + OpenTelemetry.

**Tech Stack:** Go 1.26.4, gRPC 1.81.1, pgx/v5 5.10.0, Goose 3.27.1, OTEL SDK 1.44.0, franz-go, aidanwoods.dev/go-paseto, samber/oops, testcontainers-go 0.42.0, testify 1.11.1, buf 1.70.0

**Branch:** `feat/phase3-core-domain-boilerplate`

---

## File Map

```
proto/
  buf.yaml                                              CREATE
  buf.gen.yaml                                          CREATE
  common/v1/address.proto                               CREATE
  tenant/v1/tenant.proto                                CREATE

gen/
  go.mod                                                CREATE

go.work                                                 MODIFY (add ./gen)

shared/go/go.mod                                        MODIFY (add real deps)
shared/go/pkg/
  errors/codes.go                                       CREATE
  errors/grpc.go                                        CREATE
  errors/http.go                                        CREATE
  errors/errors_test.go                                 CREATE
  logger/logger.go                                      CREATE
  logger/context.go                                     CREATE
  logger/logger_test.go                                 CREATE
  auth/claims.go                                        CREATE
  auth/paseto.go                                        CREATE
  auth/context.go                                       CREATE
  auth/auth_test.go                                     CREATE
  database/pool.go                                      CREATE
  database/tenant.go                                    CREATE
  database/tx.go                                        CREATE
  telemetry/setup.go                                    CREATE
  telemetry/span.go                                     CREATE
  events/envelope.go                                    CREATE
  events/publisher.go                                   CREATE
  events/consumer.go                                    CREATE
  middleware/chain.go                                   CREATE
  middleware/recovery.go                                CREATE
  middleware/auth.go                                    CREATE
  middleware/logging.go                                 CREATE
  middleware/tracing.go                                 CREATE
  middleware/tenant.go                                  CREATE

services/tenant/
  go.mod                                                MODIFY (add gen + real deps)
  config.go                                             CREATE
  provider.go                                           CREATE
  cmd/server/main.go                                    MODIFY (replace placeholder)
  Dockerfile                                            CREATE
  internal/domain/tenant/
    errors.go                                           CREATE
    events.go                                           CREATE
    plan.go                                             CREATE
    branch.go                                           CREATE
    tenant.go                                           CREATE
    repository.go                                       CREATE
    testdata.go                                         CREATE
    tenant_test.go                                      CREATE
  internal/application/command/
    create_tenant.go                                    CREATE
    create_tenant_test.go                               CREATE
    create_branch.go                                    CREATE
    create_branch_test.go                               CREATE
  internal/application/query/
    get_tenant.go                                       CREATE
    list_branches.go                                    CREATE
    query_test.go                                       CREATE
  internal/infrastructure/postgres/
    migrations.go                                       CREATE
    migrations/001_create_tenant_schema.sql             CREATE
    tenant_repo.go                                      CREATE
    tenant_repo_test.go                                 CREATE
  internal/interfaces/grpc/
    server.go                                           CREATE
    tenant_handler.go                                   CREATE
    tenant_handler_test.go                              CREATE
```

---

## EPIC 3.2 — Common Proto Definitions

### Task 1: Create buf.yaml, buf.gen.yaml, and gen/ Go module

**Files:**
- Create: `proto/buf.yaml`
- Create: `proto/buf.gen.yaml`
- Create: `gen/go.mod`
- Modify: `go.work`

- [ ] **Step 1: Create proto/buf.yaml**

```yaml
# proto/buf.yaml
version: v2
modules:
  - path: .
lint:
  use:
    - STANDARD
  except:
    - PACKAGE_VERSION_SUFFIX
breaking:
  use:
    - FILE
```

- [ ] **Step 2: Create proto/buf.gen.yaml**

```yaml
# proto/buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: ../gen
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go
    out: ../gen
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false
```

- [ ] **Step 3: Create gen/go.mod**

```
module github.com/xyn-pos/gen

go 1.26

require (
    google.golang.org/grpc v1.81.1
    google.golang.org/protobuf v1.36.5
)
```

- [ ] **Step 4: Add gen to go.work**

Open `go.work`. Find the `use (` block. Add `./gen`:

```
go 1.26

use (
    ./gen
    ./shared/go
    ./services/tenant
    ./services/pos
    ./services/payment
    ./services/inventory
    ./services/kitchen
    ./services/analytics
)
```

- [ ] **Step 5: Commit**

```bash
git add proto/buf.yaml proto/buf.gen.yaml gen/go.mod go.work
git commit -m "feat(proto): add buf config and gen module"
```

---

### Task 2: Create address.proto and tenant.proto

**Files:**
- Create: `proto/common/v1/address.proto`
- Create: `proto/tenant/v1/tenant.proto`

- [ ] **Step 1: Create proto/common/v1/address.proto**

```proto
syntax = "proto3";

package common.v1;

option go_package = "github.com/xyn-pos/gen/common/v1;commonv1";

message Address {
  string street      = 1;
  string city        = 2;
  string province    = 3;
  string postal_code = 4;
  string country     = 5;
}
```

- [ ] **Step 2: Create proto/tenant/v1/tenant.proto**

```proto
syntax = "proto3";

package tenant.v1;

import "common/v1/address.proto";
import "google/protobuf/timestamp.proto";

option go_package = "github.com/xyn-pos/gen/tenant/v1;tenantv1";

enum PlanTier {
  PLAN_TIER_UNSPECIFIED = 0;
  PLAN_TIER_FREE        = 1;
  PLAN_TIER_GROWTH      = 2;
  PLAN_TIER_ENTERPRISE  = 3;
}

enum TenantStatus {
  TENANT_STATUS_UNSPECIFIED = 0;
  TENANT_STATUS_ACTIVE      = 1;
  TENANT_STATUS_SUSPENDED   = 2;
  TENANT_STATUS_DELETED     = 3;
}

message Tenant {
  string                    id         = 1;
  string                    name       = 2;
  string                    slug       = 3;
  PlanTier                  plan       = 4;
  TenantStatus              status     = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

message Branch {
  string                    id         = 1;
  string                    tenant_id  = 2;
  string                    name       = 3;
  common.v1.Address         address    = 4;
  string                    timezone   = 5;
  bool                      is_active  = 6;
  google.protobuf.Timestamp created_at = 7;
}

message CreateTenantRequest {
  string   idempotency_key = 1;
  string   name            = 2;
  string   slug            = 3;
  PlanTier plan            = 4;
}

message CreateTenantResponse {
  Tenant tenant = 1;
}

message GetTenantRequest {
  string tenant_id = 1;
}

message GetTenantResponse {
  Tenant tenant = 1;
}

message CreateBranchRequest {
  string            idempotency_key = 1;
  string            name            = 2;
  common.v1.Address address         = 3;
  string            timezone        = 4;
}

message CreateBranchResponse {
  Branch branch = 1;
}

message ListBranchesRequest {
  string tenant_id = 1;
}

message ListBranchesResponse {
  repeated Branch branches = 1;
}

service TenantService {
  rpc CreateTenant(CreateTenantRequest)   returns (CreateTenantResponse);
  rpc GetTenant(GetTenantRequest)         returns (GetTenantResponse);
  rpc CreateBranch(CreateBranchRequest)   returns (CreateBranchResponse);
  rpc ListBranches(ListBranchesRequest)   returns (ListBranchesResponse);
}
```

- [ ] **Step 3: Lint protos**

```bash
cd /path/to/project && buf lint proto/
```

Expected: no output (clean).

- [ ] **Step 4: Generate code**

```bash
make proto-gen
```

Expected output: `Generated code in gen/`  
Verify: `ls gen/common/v1/ gen/tenant/v1/` shows `*.pb.go` and `*_grpc.pb.go` files.

- [ ] **Step 5: Tidy gen module**

```bash
cd gen && go mod tidy && cd ..
```

- [ ] **Step 6: Commit**

```bash
git add proto/common/v1/address.proto proto/tenant/v1/tenant.proto gen/
git commit -m "feat(proto): add address.proto and tenant service proto with codegen"
```

---

## EPIC 3.1 — Shared Go Packages

### Task 3: Update shared/go/go.mod with real dependencies

**Files:**
- Modify: `shared/go/go.mod`

- [ ] **Step 1: Replace shared/go/go.mod**

```
module github.com/xyn-pos/shared

go 1.26

require (
    github.com/samber/oops v1.14.1
    github.com/samber/lo v1.47.0
    google.golang.org/grpc v1.81.1
    google.golang.org/protobuf v1.36.5
    github.com/jackc/pgx/v5 v5.7.4
    github.com/twmb/franz-go v1.18.1
    aidanwoods.dev/go-paseto v1.5.3
    go.opentelemetry.io/otel v1.44.0
    go.opentelemetry.io/otel/trace v1.44.0
    go.opentelemetry.io/otel/metric v1.44.0
    go.opentelemetry.io/otel/sdk v1.44.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0
    github.com/google/uuid v1.6.0
    github.com/stretchr/testify v1.11.1
)
```

- [ ] **Step 2: Tidy**

```bash
cd shared/go && go mod tidy && cd ../..
```

Expected: downloads dependencies, no errors.

- [ ] **Step 3: Commit**

```bash
git add shared/go/go.mod shared/go/go.sum
git commit -m "chore(shared): update go.mod with verified dependencies"
```

---

### Task 4: errors package

**Files:**
- Create: `shared/go/pkg/errors/codes.go`
- Create: `shared/go/pkg/errors/grpc.go`
- Create: `shared/go/pkg/errors/http.go`
- Create: `shared/go/pkg/errors/errors_test.go`

- [ ] **Step 1: Write the failing test first**

```go
// shared/go/pkg/errors/errors_test.go
package errors_test

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xyn-pos/shared/pkg/errors"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestMapToGRPCStatus_NotFound(t *testing.T) {
	err := errors.NewNotFound("tenant", "abc123")
	st := errors.MapToGRPCStatus(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestMapToGRPCStatus_AlreadyExists(t *testing.T) {
	err := errors.NewConflict("slug", "already taken")
	st := errors.MapToGRPCStatus(err)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestMapToGRPCStatus_UnknownError(t *testing.T) {
	err := stderrors.New("some infrastructure error")
	st := errors.MapToGRPCStatus(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestMapToHTTPStatus_NotFound(t *testing.T) {
	err := errors.NewNotFound("tenant", "abc123")
	code := errors.MapToHTTPStatus(err)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestIsNotFound(t *testing.T) {
	err := errors.NewNotFound("tenant", "abc123")
	assert.True(t, errors.IsNotFound(err))
	wrapped := grpcstatus.Errorf(codes.NotFound, "wrap")
	assert.False(t, errors.IsNotFound(wrapped))
}
```

- [ ] **Step 2: Run — expect FAIL (package doesn't exist)**

```bash
cd shared/go && go test ./pkg/errors/... 2>&1 | head -5
```

Expected: `no Go files` or `cannot find package`.

- [ ] **Step 3: Create shared/go/pkg/errors/codes.go**

```go
package errors

const (
	CodeNotFound        = "NOT_FOUND"
	CodeAlreadyExists   = "ALREADY_EXISTS"
	CodeInvalidArgument = "INVALID_ARGUMENT"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeInternal        = "INTERNAL"
	CodeUnavailable     = "UNAVAILABLE"
	CodeConflict        = "CONFLICT"
)

type DomainError struct {
	Code    string
	Message string
	Field   string
}

func (e *DomainError) Error() string { return e.Message }

func NewNotFound(resource, id string) *DomainError {
	return &DomainError{Code: CodeNotFound, Message: resource + " not found: " + id}
}

func NewConflict(field, reason string) *DomainError {
	return &DomainError{Code: CodeConflict, Message: field + ": " + reason, Field: field}
}

func NewInvalidArgument(field, reason string) *DomainError {
	return &DomainError{Code: CodeInvalidArgument, Message: field + ": " + reason, Field: field}
}

func IsNotFound(err error) bool {
	var d *DomainError
	if As(err, &d) {
		return d.Code == CodeNotFound
	}
	return false
}

func IsConflict(err error) bool {
	var d *DomainError
	if As(err, &d) {
		return d.Code == CodeConflict || d.Code == CodeAlreadyExists
	}
	return false
}
```

- [ ] **Step 4: Create shared/go/pkg/errors/grpc.go**

```go
package errors

import (
	"github.com/samber/oops"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapToGRPCStatus converts any error to a gRPC status.
// Call this only at the interface (gRPC handler) layer.
func MapToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	var d *DomainError
	if As(err, &d) {
		return status.New(domainCodeToGRPC(d.Code), d.Message)
	}

	var oopsErr oops.OopsError
	if As(err, &oopsErr) {
		code := oopsErr.Code()
		msg := oopsErr.Public()
		if msg == "" {
			msg = "an internal error occurred"
		}
		return status.New(stringCodeToGRPC(code), msg)
	}

	return status.New(codes.Internal, "an internal error occurred")
}

func domainCodeToGRPC(code string) codes.Code {
	return stringCodeToGRPC(code)
}

func stringCodeToGRPC(code string) codes.Code {
	switch code {
	case CodeNotFound:
		return codes.NotFound
	case CodeAlreadyExists, CodeConflict:
		return codes.AlreadyExists
	case CodeInvalidArgument:
		return codes.InvalidArgument
	case CodeUnauthorized:
		return codes.Unauthenticated
	case CodeForbidden:
		return codes.PermissionDenied
	case CodeUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}
```

- [ ] **Step 5: Create shared/go/pkg/errors/http.go**

```go
package errors

import (
	"errors"
	"net/http"
)

// As is a convenience re-export of errors.As.
var As = errors.As

// MapToHTTPStatus converts any error to an HTTP status code.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var d *DomainError
	if As(err, &d) {
		return domainCodeToHTTP(d.Code)
	}
	return http.StatusInternalServerError
}

func domainCodeToHTTP(code string) int {
	switch code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeAlreadyExists, CodeConflict:
		return http.StatusConflict
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
```

- [ ] **Step 6: Run tests — expect PASS**

```bash
cd shared/go && go test -v -race ./pkg/errors/...
```

Expected: `PASS` for all 5 tests.

- [ ] **Step 7: Commit**

```bash
git add shared/go/pkg/errors/
git commit -m "feat(shared): add errors package with gRPC and HTTP status mapping"
```

---

### Task 5: logger package

**Files:**
- Create: `shared/go/pkg/logger/logger.go`
- Create: `shared/go/pkg/logger/context.go`
- Create: `shared/go/pkg/logger/logger_test.go`

- [ ] **Step 1: Write the failing test**

```go
// shared/go/pkg/logger/logger_test.go
package logger_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/xyn-pos/shared/pkg/logger"
)

func TestFromContext_WithoutLogger_ReturnsDefault(t *testing.T) {
	ctx := context.Background()
	l := logger.FromContext(ctx)
	assert.NotNil(t, l)
}

func TestWithTraceID_StoresAndRetrievesTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = logger.WithTraceID(ctx, "trace-abc")
	assert.Equal(t, "trace-abc", logger.TraceIDFromContext(ctx))
}

func TestWithTenantID_StoresAndRetrievesTenantID(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	ctx = logger.WithTenantID(ctx, id)
	assert.Equal(t, id, logger.TenantIDFromContext(ctx))
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd shared/go && go test ./pkg/logger/... 2>&1 | head -5
```

- [ ] **Step 3: Create shared/go/pkg/logger/context.go**

```go
package logger

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type contextKey int

const (
	keyLogger   contextKey = iota
	keyTraceID
	keyTenantID
	keyUserID
)

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(keyLogger).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, keyLogger, l)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyTraceID).(string); ok {
		return v
	}
	return ""
}

func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, keyTenantID, tenantID)
}

func TenantIDFromContext(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(keyTenantID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}
```

- [ ] **Step 4: Create shared/go/pkg/logger/logger.go**

```go
package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	Level       string
	ServiceName string
	Env         string
}

// New creates a structured JSON logger. Always use FromContext in handlers.
func New(cfg Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
		slog.String("env", cfg.Env),
	)
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd shared/go && go test -v -race ./pkg/logger/...
```

- [ ] **Step 6: Commit**

```bash
git add shared/go/pkg/logger/
git commit -m "feat(shared): add logger package with context propagation"
```

---

### Task 6: auth package

**Files:**
- Create: `shared/go/pkg/auth/claims.go`
- Create: `shared/go/pkg/auth/paseto.go`
- Create: `shared/go/pkg/auth/context.go`
- Create: `shared/go/pkg/auth/auth_test.go`

- [ ] **Step 1: Write the failing test**

```go
// shared/go/pkg/auth/auth_test.go
package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/shared/pkg/auth"
)

func TestGenerateAndVerify_ValidToken(t *testing.T) {
	key := auth.GenerateKey()
	claims := auth.Claims{
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Roles:     []string{"cashier"},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	token, err := auth.GenerateToken(claims, key)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	got, err := auth.VerifyToken(token, key)
	require.NoError(t, err)
	assert.Equal(t, claims.TenantID, got.TenantID)
	assert.Equal(t, claims.UserID, got.UserID)
}

func TestVerifyToken_Expired_ReturnsError(t *testing.T) {
	key := auth.GenerateKey()
	claims := auth.Claims{
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	}
	token, err := auth.GenerateToken(claims, key)
	require.NoError(t, err)

	_, err = auth.VerifyToken(token, key)
	assert.Error(t, err)
}

func TestClaimsFromContext_NotSet_Panics(t *testing.T) {
	ctx := context.Background()
	assert.Panics(t, func() { auth.ClaimsFromContext(ctx) })
}

func TestWithClaims_RoundTrip(t *testing.T) {
	claims := &auth.Claims{TenantID: uuid.New(), UserID: uuid.New()}
	ctx := auth.WithClaims(context.Background(), claims)
	got := auth.ClaimsFromContext(ctx)
	assert.Equal(t, claims.TenantID, got.TenantID)
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd shared/go && go test ./pkg/auth/... 2>&1 | head -5
```

- [ ] **Step 3: Create shared/go/pkg/auth/claims.go**

```go
package auth

import (
	"time"

	"github.com/google/uuid"
)

type Claims struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	BranchID  uuid.UUID
	Roles     []string
	ExpiresAt time.Time
}

func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Create shared/go/pkg/auth/paseto.go**

```go
package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
)

type Key = paseto.V4SymmetricKey

// GenerateKey creates a new random PASETO v4 symmetric key.
func GenerateKey() Key {
	return paseto.NewV4SymmetricKey()
}

// GenerateToken creates a PASETO v4 local token for the given claims.
func GenerateToken(claims Claims, key Key) (string, error) {
	token := paseto.NewToken()
	token.SetExpiration(claims.ExpiresAt)
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())

	b, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("auth.GenerateToken marshal: %w", err)
	}
	if err := token.Set("claims", string(b)); err != nil {
		return "", fmt.Errorf("auth.GenerateToken set claims: %w", err)
	}

	return token.V4Encrypt(key, nil), nil
}

// VerifyToken validates a PASETO v4 local token and returns the claims.
func VerifyToken(tokenStr string, key Key) (*Claims, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	token, err := parser.ParseV4Local(key, tokenStr, nil)
	if err != nil {
		return nil, fmt.Errorf("auth.VerifyToken: %w", err)
	}

	raw, err := token.GetString("claims")
	if err != nil {
		return nil, fmt.Errorf("auth.VerifyToken get claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal([]byte(raw), &claims); err != nil {
		return nil, fmt.Errorf("auth.VerifyToken unmarshal: %w", err)
	}
	return &claims, nil
}
```

- [ ] **Step 5: Create shared/go/pkg/auth/context.go**

```go
package auth

import "context"

type claimsKey struct{}

// WithClaims stores claims in the context. Called by the auth interceptor.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// ClaimsFromContext retrieves claims from context. Panics if not set — the
// auth interceptor must always inject claims before handlers run.
func ClaimsFromContext(ctx context.Context) *Claims {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	if !ok || c == nil {
		panic("auth: claims not found in context — auth interceptor must run first")
	}
	return c
}
```

- [ ] **Step 6: Run tests — expect PASS**

```bash
cd shared/go && go test -v -race ./pkg/auth/...
```

- [ ] **Step 7: Commit**

```bash
git add shared/go/pkg/auth/
git commit -m "feat(shared): add auth package with PASETO v4 token generation"
```

---

### Task 7: database package

**Files:**
- Create: `shared/go/pkg/database/pool.go`
- Create: `shared/go/pkg/database/tenant.go`
- Create: `shared/go/pkg/database/tx.go`

- [ ] **Step 1: Create shared/go/pkg/database/pool.go**

```go
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// NewPool creates a pgxpool with sensible defaults for production use.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	} else {
		poolCfg.MaxConns = 25
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	} else {
		poolCfg.MinConns = 5
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	} else {
		poolCfg.MaxConnLifetime = time.Hour
	}
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database.NewPool ping: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 2: Create shared/go/pkg/database/tenant.go**

```go
package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// SetTenantContext executes SET LOCAL app.current_tenant_id so PostgreSQL RLS
// policies can filter rows. Must be called within a transaction.
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("database.SetTenantContext: %w", err)
	}
	return nil
}

// WithTenantTx begins a transaction, sets the tenant RLS context, then calls fn.
// Rolls back on error, commits on success.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database.WithTenantTx begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 3: Fix import in tenant.go — add pgconn**

Replace the `TenantQuerier` interface (it's unused — remove it). The final `tenant.go`:

```go
package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetTenantContext executes SET LOCAL app.current_tenant_id so PostgreSQL RLS
// policies can filter rows. Must be called within a transaction.
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("database.SetTenantContext: %w", err)
	}
	return nil
}

// WithTenantTx begins a transaction, sets the tenant RLS context, then calls fn.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database.WithTenantTx begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 4: Create shared/go/pkg/database/tx.go**

```go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTransaction begins a transaction and calls fn. Rolls back on error.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database.WithTransaction begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 5: Build check**

```bash
cd shared/go && go build ./pkg/database/...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add shared/go/pkg/database/
git commit -m "feat(shared): add database package with pgxpool factory and RLS tenant context"
```

---

### Task 8: telemetry package

**Files:**
- Create: `shared/go/pkg/telemetry/setup.go`
- Create: `shared/go/pkg/telemetry/span.go`

- [ ] **Step 1: Create shared/go/pkg/telemetry/setup.go**

```go
package telemetry

import (
	"context"
	"fmt"
	"time"

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
	OTLPEndpoint   string // e.g. "localhost:4317"
	Environment    string
}

// Setup configures OTEL with OTLP gRPC exporter. Call defer shutdown() in main.
func Setup(ctx context.Context, cfg Config) (shutdown func(), err error) {
	conn, err := grpc.NewClient(cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry.Setup grpc dial: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry.Setup exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentName(cfg.Environment),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
		_ = conn.Close()
	}
	return shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) interface{ Start(context.Context, string, ...interface{}) (context.Context, interface{}) } {
	return otel.Tracer(name)
}
```

- [ ] **Step 2: Create shared/go/pkg/telemetry/span.go**

```go
package telemetry

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError marks the span as error and records the exception event.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetOK marks the span as successful.
func SetOK(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}
```

- [ ] **Step 3: Fix setup.go — Tracer return type is incorrect. Replace with proper otel.Tracer:**

Replace the `Tracer` function in `setup.go` with:

```go
// Tracer returns a named OTEL tracer from the global provider.
// Usage: tracer := telemetry.Tracer("tenant-service")
//        ctx, span := tracer.Start(ctx, "operation-name")
//        defer span.End()
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
```

Add `"go.opentelemetry.io/otel/trace"` to imports in `setup.go`.

- [ ] **Step 4: Build check**

```bash
cd shared/go && go build ./pkg/telemetry/...
```

- [ ] **Step 5: Commit**

```bash
git add shared/go/pkg/telemetry/
git commit -m "feat(shared): add telemetry package with OTLP gRPC exporter setup"
```

---

### Task 9: events package

**Files:**
- Create: `shared/go/pkg/events/envelope.go`
- Create: `shared/go/pkg/events/publisher.go`
- Create: `shared/go/pkg/events/consumer.go`

- [ ] **Step 1: Create shared/go/pkg/events/envelope.go**

```go
package events

import (
	"time"

	"github.com/google/uuid"
)

// EventEnvelope wraps every domain event for Kafka transport.
type EventEnvelope struct {
	EventID    uuid.UUID `json:"event_id"`
	EventType  string    `json:"event_type"`
	TenantID   uuid.UUID `json:"tenant_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    []byte    `json:"payload"` // JSON-encoded domain event
}
```

- [ ] **Step 2: Create shared/go/pkg/events/publisher.go**

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Publisher struct {
	client *kgo.Client
	tracer trace.Tracer
}

func NewPublisher(brokers []string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchMaxBytes(1<<20),
	)
	if err != nil {
		return nil, fmt.Errorf("events.NewPublisher: %w", err)
	}
	return &Publisher{client: client, tracer: otel.Tracer("events.publisher")}, nil
}

// Publish serializes and sends an EventEnvelope to the given topic.
func (p *Publisher) Publish(ctx context.Context, topic string, env EventEnvelope) error {
	ctx, span := p.tracer.Start(ctx, "events.Publish")
	defer span.End()

	b, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("events.Publish marshal: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(env.TenantID.String()),
		Value: b,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("events.Publish produce: %w", err)
	}
	return nil
}

func (p *Publisher) Close() { p.client.Close() }
```

- [ ] **Step 3: Create shared/go/pkg/events/consumer.go**

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

type HandlerFunc func(ctx context.Context, env EventEnvelope) error

type Consumer struct {
	client *kgo.Client
}

func NewConsumer(brokers []string, group string, topics ...string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
	)
	if err != nil {
		return nil, fmt.Errorf("events.NewConsumer: %w", err)
	}
	return &Consumer{client: client}, nil
}

// Subscribe polls Kafka and calls handler for each message. Blocks until ctx is cancelled.
func (c *Consumer) Subscribe(ctx context.Context, handler HandlerFunc) {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			slog.ErrorContext(ctx, "kafka fetch error", "topic", t, "partition", p, "err", err)
		})
		fetches.EachRecord(func(record *kgo.Record) {
			var env EventEnvelope
			if err := json.Unmarshal(record.Value, &env); err != nil {
				slog.ErrorContext(ctx, "events.Subscribe unmarshal", "err", err)
				return
			}
			if err := handler(ctx, env); err != nil {
				slog.ErrorContext(ctx, "events.Subscribe handler", "event_type", env.EventType, "err", err)
			}
		})
	}
}

func (c *Consumer) Close() { c.client.Close() }
```

- [ ] **Step 4: Build check**

```bash
cd shared/go && go build ./pkg/events/...
```

- [ ] **Step 5: Commit**

```bash
git add shared/go/pkg/events/
git commit -m "feat(shared): add events package with Kafka publisher and consumer"
```

---

### Task 10: middleware package

**Files:**
- Create: `shared/go/pkg/middleware/chain.go`
- Create: `shared/go/pkg/middleware/recovery.go`
- Create: `shared/go/pkg/middleware/tracing.go`
- Create: `shared/go/pkg/middleware/logging.go`
- Create: `shared/go/pkg/middleware/auth.go`
- Create: `shared/go/pkg/middleware/tenant.go`

- [ ] **Step 1: Create shared/go/pkg/middleware/chain.go**

```go
package middleware

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// Chain returns a gRPC server with all interceptors in the correct order:
// recovery → otelgrpc stats handler → logging → auth → tenant
func Chain(authInterceptor grpc.UnaryServerInterceptor) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			Recovery(),
			Logging(),
			authInterceptor,
			Tenant(),
		),
	}
}
```

- [ ] **Step 2: Create shared/go/pkg/middleware/recovery.go**

```go
package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery catches panics and converts them to gRPC Internal errors.
func Recovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", info.FullMethod),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
```

- [ ] **Step 3: Create shared/go/pkg/middleware/logging.go**

```go
package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logging logs each RPC call with method, duration, and status code.
func Logging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}
		slog.InfoContext(ctx, "grpc call",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", time.Since(start)),
			slog.String("code", code.String()),
		)
		return resp, err
	}
}
```

- [ ] **Step 4: Create shared/go/pkg/middleware/tracing.go**

```go
package middleware

// Tracing is handled by otelgrpc.NewServerHandler() in chain.go (stats handler).
// This file is intentionally empty — the stats handler approach is preferred
// over unary interceptors for OTEL because it captures streaming RPCs too.
```

- [ ] **Step 5: Create shared/go/pkg/middleware/auth.go**

```go
package middleware

import (
	"context"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type VerifyFunc func(token string) (*sharedauth.Claims, error)

// Auth validates the PASETO token from gRPC metadata and injects Claims to ctx.
func Auth(verify VerifyFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		token := tokens[0]
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := verify(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = sharedauth.WithClaims(ctx, claims)
		return handler(ctx, req)
	}
}
```

- [ ] **Step 6: Create shared/go/pkg/middleware/tenant.go**

```go
package middleware

import (
	"context"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	sharedlogger "github.com/xyn-pos/shared/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tenant validates that the JWT claims contain a non-zero TenantID
// and injects it into the logger context for structured logging.
func Tenant() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		claims := sharedauth.ClaimsFromContext(ctx)
		if claims.TenantID.String() == "00000000-0000-0000-0000-000000000000" {
			return nil, status.Error(codes.Unauthenticated, "missing tenant context")
		}
		ctx = sharedlogger.WithTenantID(ctx, claims.TenantID)
		return handler(ctx, req)
	}
}
```

- [ ] **Step 7: Build check**

```bash
cd shared/go && go build ./pkg/middleware/...
```

- [ ] **Step 8: Run all shared package tests**

```bash
cd shared/go && go test -v -race ./...
```

Expected: all pass. Fix any compile errors before continuing.

- [ ] **Step 9: Commit**

```bash
git add shared/go/pkg/middleware/
git commit -m "feat(shared): add gRPC middleware chain (recovery, logging, auth, tenant)"
```

---

## EPIC 3.4 — Database Infrastructure

### Task 11: Tenant BC Goose migrations + RLS

**Files:**
- Create: `services/tenant/internal/infrastructure/postgres/migrations/001_create_tenant_schema.sql`
- Create: `services/tenant/internal/infrastructure/postgres/migrations.go`

- [ ] **Step 1: Create the SQL migration**

```sql
-- services/tenant/internal/infrastructure/postgres/migrations/001_create_tenant_schema.sql
-- +goose Up
-- +goose StatementBegin

CREATE TABLE tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    slug        TEXT        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]+$'),
    plan        TEXT        NOT NULL DEFAULT 'FREE'
                            CHECK (plan IN ('FREE', 'GROWTH', 'ENTERPRISE')),
    status      TEXT        NOT NULL DEFAULT 'ACTIVE'
                            CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELETED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE branches (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    street          TEXT,
    city            TEXT,
    province        TEXT,
    postal_code     TEXT,
    country         TEXT        NOT NULL DEFAULT 'ID',
    timezone        TEXT        NOT NULL DEFAULT 'Asia/Jakarta',
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE idempotency_keys (
    key         TEXT        PRIMARY KEY,
    tenant_id   UUID        NOT NULL,
    operation   TEXT        NOT NULL,
    result_json JSONB,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_branches_tenant_id   ON branches(tenant_id);
CREATE INDEX idx_idempotency_expires  ON idempotency_keys(expires_at);

-- RLS
ALTER TABLE tenants  ENABLE ROW LEVEL SECURITY;
ALTER TABLE branches ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tenants
    USING (id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY branch_isolation ON branches
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS branches;
DROP TABLE IF EXISTS tenants;
-- +goose StatementEnd
```

- [ ] **Step 2: Create migrations.go**

```go
// services/tenant/internal/infrastructure/postgres/migrations.go
package postgres

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS
```

- [ ] **Step 3: Commit**

```bash
git add services/tenant/internal/infrastructure/postgres/
git commit -m "feat(tenant): add PostgreSQL schema with RLS policies (goose migration 001)"
```

---

## EPIC 3.3 — Tenant Service (Full DDD Template)

### Task 12: Tenant service go.mod + config.go

**Files:**
- Modify: `services/tenant/go.mod`
- Create: `services/tenant/config.go`

- [ ] **Step 1: Update services/tenant/go.mod**

```
module github.com/xyn-pos/services/tenant

go 1.26

require (
    github.com/xyn-pos/shared v0.0.0
    github.com/xyn-pos/gen v0.0.0
    google.golang.org/grpc v1.81.1
    google.golang.org/protobuf v1.36.5
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.7.4
    github.com/pressly/goose/v3 v3.27.1
    github.com/samber/oops v1.14.1
    go.opentelemetry.io/otel v1.44.0
    go.opentelemetry.io/otel/trace v1.44.0
    github.com/spf13/viper v1.20.1
)

replace (
    github.com/xyn-pos/shared => ../../shared/go
    github.com/xyn-pos/gen => ../../gen
)
```

- [ ] **Step 2: Create services/tenant/config.go**

```go
package tenant

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Env          string
	ServiceName  string
	Version      string
	GRPCPort     int
	DatabaseURL  string
	OTLPEndpoint string
	KafkaBrokers []string
	PASETOKey    string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	viper.SetDefault("ENV", "development")
	viper.SetDefault("SERVICE_NAME", "tenant-service")
	viper.SetDefault("VERSION", "0.1.0")
	viper.SetDefault("GRPC_PORT", 50051)
	viper.SetDefault("OTLP_ENDPOINT", "localhost:4317")

	cfg := &Config{
		Env:          viper.GetString("ENV"),
		ServiceName:  viper.GetString("SERVICE_NAME"),
		Version:      viper.GetString("VERSION"),
		GRPCPort:     viper.GetInt("GRPC_PORT"),
		DatabaseURL:  viper.GetString("DATABASE_URL"),
		OTLPEndpoint: viper.GetString("OTLP_ENDPOINT"),
		KafkaBrokers: viper.GetStringSlice("KAFKA_BROKERS"),
		PASETOKey:    viper.GetString("PASETO_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	return cfg, nil
}
```

- [ ] **Step 3: Tidy**

```bash
cd services/tenant && go mod tidy && cd ../..
```

- [ ] **Step 4: Commit**

```bash
git add services/tenant/go.mod services/tenant/go.sum services/tenant/config.go
git commit -m "feat(tenant): add service config with viper and updated go.mod"
```

---

### Task 13: Domain layer — errors, events, plan, branch, tenant aggregate

**Files:**
- Create: `services/tenant/internal/domain/tenant/errors.go`
- Create: `services/tenant/internal/domain/tenant/events.go`
- Create: `services/tenant/internal/domain/tenant/plan.go`
- Create: `services/tenant/internal/domain/tenant/branch.go`
- Create: `services/tenant/internal/domain/tenant/tenant.go`
- Create: `services/tenant/internal/domain/tenant/repository.go`
- Create: `services/tenant/internal/domain/tenant/testdata.go`

- [ ] **Step 1: Create errors.go (no external imports — pure stdlib)**

```go
// services/tenant/internal/domain/tenant/errors.go
package tenant

import "errors"

var (
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrSlugAlreadyTaken   = errors.New("tenant slug is already taken")
	ErrBranchLimitReached = errors.New("branch limit reached for subscription plan")
	ErrTenantSuspended    = errors.New("tenant is suspended")
	ErrTenantDeleted      = errors.New("tenant is deleted")
	ErrInvalidSlug        = errors.New("slug must be lowercase alphanumeric with hyphens only")
	ErrBranchNotFound     = errors.New("branch not found")
)
```

- [ ] **Step 2: Create events.go**

```go
// services/tenant/internal/domain/tenant/events.go
package tenant

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface{ eventType() string }

type TenantCreated struct {
	TenantID  uuid.UUID
	Name      string
	Slug      string
	Plan      PlanTier
	OccurredAt time.Time
}

func (e TenantCreated) eventType() string { return "tenant.created" }

type BranchAdded struct {
	TenantID   uuid.UUID
	BranchID   uuid.UUID
	BranchName string
	OccurredAt time.Time
}

func (e BranchAdded) eventType() string { return "tenant.branch_added" }
```

- [ ] **Step 3: Create plan.go**

```go
// services/tenant/internal/domain/tenant/plan.go
package tenant

type PlanTier string

const (
	PlanTierFree       PlanTier = "FREE"
	PlanTierGrowth     PlanTier = "GROWTH"
	PlanTierEnterprise PlanTier = "ENTERPRISE"
)

type SubscriptionPlan struct {
	Tier        PlanTier
	MaxBranches int // -1 = unlimited
	MaxUsers    int // -1 = unlimited
}

var (
	FreePlan       = SubscriptionPlan{Tier: PlanTierFree, MaxBranches: 1, MaxUsers: 3}
	GrowthPlan     = SubscriptionPlan{Tier: PlanTierGrowth, MaxBranches: 5, MaxUsers: 20}
	EnterprisePlan = SubscriptionPlan{Tier: PlanTierEnterprise, MaxBranches: -1, MaxUsers: -1}
)

func PlanFromTier(tier PlanTier) SubscriptionPlan {
	switch tier {
	case PlanTierGrowth:
		return GrowthPlan
	case PlanTierEnterprise:
		return EnterprisePlan
	default:
		return FreePlan
	}
}

func (p SubscriptionPlan) CanAddBranch(currentCount int) bool {
	if p.MaxBranches == -1 {
		return true
	}
	return currentCount < p.MaxBranches
}
```

- [ ] **Step 4: Create branch.go**

```go
// services/tenant/internal/domain/tenant/branch.go
package tenant

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	Street     string
	City       string
	Province   string
	PostalCode string
	Country    string
}

type Branch struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Address   Address
	Timezone  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewBranch(tenantID uuid.UUID, name string, addr Address, timezone string) Branch {
	now := time.Now().UTC()
	return Branch{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Address:   addr,
		Timezone:  timezone,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
```

- [ ] **Step 5: Create tenant.go (aggregate root)**

```go
// services/tenant/internal/domain/tenant/tenant.go
package tenant

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type TenantStatus string

const (
	StatusActive    TenantStatus = "ACTIVE"
	StatusSuspended TenantStatus = "SUSPENDED"
	StatusDeleted   TenantStatus = "DELETED"
)

type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      SubscriptionPlan
	Status    TenantStatus
	Branches  []Branch
	CreatedAt time.Time
	UpdatedAt time.Time
	events    []DomainEvent
}

// NewTenant creates a valid Tenant aggregate. Returns error if slug is invalid.
func NewTenant(name, slug string, tier PlanTier) (*Tenant, error) {
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidSlug
	}
	now := time.Now().UTC()
	t := &Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Plan:      PlanFromTier(tier),
		Status:    StatusActive,
		Branches:  []Branch{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	t.events = append(t.events, TenantCreated{
		TenantID:   t.ID,
		Name:       name,
		Slug:       slug,
		Plan:       tier,
		OccurredAt: now,
	})
	return t, nil
}

// AddBranch appends a branch if the plan limit allows it.
func (t *Tenant) AddBranch(b Branch) error {
	if t.Status == StatusSuspended {
		return ErrTenantSuspended
	}
	if t.Status == StatusDeleted {
		return ErrTenantDeleted
	}
	if !t.Plan.CanAddBranch(len(t.Branches)) {
		return ErrBranchLimitReached
	}
	b.TenantID = t.ID
	t.Branches = append(t.Branches, b)
	t.UpdatedAt = time.Now().UTC()
	t.events = append(t.events, BranchAdded{
		TenantID:   t.ID,
		BranchID:   b.ID,
		BranchName: b.Name,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

// Suspend marks the tenant as suspended.
func (t *Tenant) Suspend() error {
	if t.Status == StatusDeleted {
		return ErrTenantDeleted
	}
	t.Status = StatusSuspended
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// PopEvents returns and clears accumulated domain events.
func (t *Tenant) PopEvents() []DomainEvent {
	evts := t.events
	t.events = nil
	return evts
}
```

- [ ] **Step 6: Create repository.go**

```go
// services/tenant/internal/domain/tenant/repository.go
package tenant

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, t *Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ListBranches(ctx context.Context, tenantID uuid.UUID) ([]Branch, error)
}
```

- [ ] **Step 7: Create testdata.go**

```go
// services/tenant/internal/domain/tenant/testdata.go
//go:build !production

package tenant

import "github.com/google/uuid"

type TestOption func(*Tenant)

func WithStatus(s TenantStatus) TestOption {
	return func(t *Tenant) { t.Status = s }
}

func WithPlan(p SubscriptionPlan) TestOption {
	return func(t *Tenant) { t.Plan = p }
}

func WithBranches(branches []Branch) TestOption {
	return func(t *Tenant) { t.Branches = branches }
}

// NewTestTenant returns a valid FREE plan tenant. Use TestOption to customise.
func NewTestTenant(opts ...TestOption) *Tenant {
	t, _ := NewTenant("Test Tenant", "test-tenant-"+uuid.New().String()[:8], PlanTierFree)
	for _, o := range opts {
		o(t)
	}
	t.events = nil // clear events from construction for cleaner test assertions
	return t
}

func NewTestBranch(tenantID uuid.UUID) Branch {
	return NewBranch(tenantID, "Test Branch", Address{City: "Jakarta", Country: "ID"}, "Asia/Jakarta")
}
```

- [ ] **Step 8: Build check (zero external imports in domain)**

```bash
cd services/tenant && go build ./internal/domain/...
```

Expected: no errors.

- [ ] **Step 9: Commit**

```bash
git add services/tenant/internal/domain/
git commit -m "feat(tenant): add domain layer — Tenant aggregate, Branch entity, plan value objects"
```

---

### Task 14: Domain unit tests

**Files:**
- Create: `services/tenant/internal/domain/tenant/tenant_test.go`

- [ ] **Step 1: Write tests**

```go
// services/tenant/internal/domain/tenant/tenant_test.go
package tenant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type TenantSuite struct{ suite.Suite }

func TestTenantSuite(t *testing.T) { suite.Run(t, new(TenantSuite)) }

func (s *TenantSuite) TestNewTenant_ValidSlug_ShouldSucceed() {
	t, err := domain.NewTenant("Mie Gacoan", "mie-gacoan", domain.PlanTierFree)
	s.Require().NoError(err)
	s.Equal("mie-gacoan", t.Slug)
	s.Equal(domain.StatusActive, t.Status)
	s.Len(t.PopEvents(), 1)
}

func (s *TenantSuite) TestNewTenant_InvalidSlug_ShouldReturnErrInvalidSlug() {
	tests := []struct{ slug string }{
		{"Uppercase"},
		{"has space"},
		{"has_underscore"},
		{""},
	}
	for _, tc := range tests {
		s.Run(tc.slug, func() {
			_, err := domain.NewTenant("Name", tc.slug, domain.PlanTierFree)
			s.ErrorIs(err, domain.ErrInvalidSlug)
		})
	}
}

func (s *TenantSuite) TestAddBranch_WithinFreeLimit_ShouldSucceed() {
	t := domain.NewTestTenant()
	b := domain.NewTestBranch(t.ID)
	err := t.AddBranch(b)
	s.NoError(err)
	s.Len(t.Branches, 1)
}

func (s *TenantSuite) TestAddBranch_ExceedFreeLimit_ShouldReturnErrBranchLimitReached() {
	t := domain.NewTestTenant() // FREE plan: max 1 branch
	b1 := domain.NewTestBranch(t.ID)
	require.NoError(s.T(), t.AddBranch(b1))

	b2 := domain.NewTestBranch(t.ID)
	err := t.AddBranch(b2)
	s.ErrorIs(err, domain.ErrBranchLimitReached)
}

func (s *TenantSuite) TestAddBranch_EnterprisePlan_AllowsUnlimitedBranches() {
	t := domain.NewTestTenant(domain.WithPlan(domain.EnterprisePlan))
	for i := 0; i < 20; i++ {
		err := t.AddBranch(domain.NewTestBranch(t.ID))
		s.NoError(err)
	}
	s.Len(t.Branches, 20)
}

func (s *TenantSuite) TestAddBranch_SuspendedTenant_ShouldReturnErrTenantSuspended() {
	t := domain.NewTestTenant(domain.WithStatus(domain.StatusSuspended))
	err := t.AddBranch(domain.NewTestBranch(t.ID))
	s.ErrorIs(err, domain.ErrTenantSuspended)
}

func (s *TenantSuite) TestPopEvents_ClearsAfterRead() {
	t, _ := domain.NewTenant("Test", "test-x1", domain.PlanTierFree)
	evts := t.PopEvents()
	assert.Len(s.T(), evts, 1)
	// second pop returns empty
	evts2 := t.PopEvents()
	assert.Empty(s.T(), evts2)
}

func (s *TenantSuite) TestSubscriptionPlan_CanAddBranch() {
	tests := []struct {
		plan     domain.SubscriptionPlan
		current  int
		expected bool
	}{
		{domain.FreePlan, 0, true},
		{domain.FreePlan, 1, false},
		{domain.GrowthPlan, 4, true},
		{domain.GrowthPlan, 5, false},
		{domain.EnterprisePlan, 1000, true},
	}
	for _, tc := range tests {
		s.Equal(tc.expected, tc.plan.CanAddBranch(tc.current))
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd services/tenant && go test -v -race ./internal/domain/...
```

Expected: all 7 test cases PASS.

- [ ] **Step 3: Commit**

```bash
git add services/tenant/internal/domain/tenant/tenant_test.go
git commit -m "test(tenant): add domain unit tests — aggregate behavior, plan limits, slug validation"
```

---

### Task 15: Application layer — CreateTenant command

**Files:**
- Create: `services/tenant/internal/application/command/create_tenant.go`
- Create: `services/tenant/internal/application/command/create_tenant_test.go`

- [ ] **Step 1: Write the failing test**

```go
// services/tenant/internal/application/command/create_tenant_test.go
package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// MockTenantRepository is a hand-written mock (mockery can regenerate this).
type MockTenantRepository struct{ mock.Mock }

func (m *MockTenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	return m.Called(ctx, t).Error(0)
}
func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}
func (m *MockTenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}
func (m *MockTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}
func (m *MockTenantRepository) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]domain.Branch), args.Error(1)
}

// MockPublisher is a hand-written no-op for tests.
type MockPublisher struct{ mock.Mock }

func (m *MockPublisher) Publish(ctx context.Context, topic string, tenantID uuid.UUID, eventType string, payload []byte) error {
	return m.Called(ctx, topic, tenantID, eventType, payload).Error(0)
}

type CreateTenantSuite struct {
	suite.Suite
	repo    *MockTenantRepository
	handler *CreateTenantHandler
}

func (s *CreateTenantSuite) SetupTest() {
	s.repo = new(MockTenantRepository)
	s.handler = NewCreateTenantHandler(s.repo, nil)
}

func TestCreateTenantSuite(t *testing.T) { suite.Run(t, new(CreateTenantSuite)) }

func (s *CreateTenantSuite) TestHandle_ValidInput_ShouldCreateTenant() {
	s.repo.On("ExistsBySlug", mock.Anything, "my-shop").Return(false, nil)
	s.repo.On("Save", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).Return(nil)

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "My Shop", Slug: "my-shop", Plan: domain.PlanTierFree}
	result, err := s.handler.Handle(context.Background(), cmd)

	s.Require().NoError(err)
	s.Equal("my-shop", result.Slug)
	s.Equal(domain.StatusActive, result.Status)
}

func (s *CreateTenantSuite) TestHandle_DuplicateSlug_ShouldReturnErrSlugAlreadyTaken() {
	s.repo.On("ExistsBySlug", mock.Anything, "taken-slug").Return(true, nil)

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "Shop", Slug: "taken-slug"}
	_, err := s.handler.Handle(context.Background(), cmd)

	s.ErrorIs(err, domain.ErrSlugAlreadyTaken)
}

func (s *CreateTenantSuite) TestHandle_RepoError_ShouldReturnError() {
	s.repo.On("ExistsBySlug", mock.Anything, "good-slug").Return(false, nil)
	s.repo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "Shop", Slug: "good-slug"}
	_, err := s.handler.Handle(context.Background(), cmd)

	s.Error(err)
}
```

- [ ] **Step 2: Run — expect FAIL (CreateTenantHandler undefined)**

```bash
cd services/tenant && go test ./internal/application/command/... 2>&1 | head -5
```

- [ ] **Step 3: Create create_tenant.go**

```go
// services/tenant/internal/application/command/create_tenant.go
package command

import (
	"context"
	"fmt"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type CreateTenantCommand struct {
	IdempotencyKey string
	Name           string
	Slug           string
	Plan           domain.PlanTier
}

type Publisher interface {
	Publish(ctx context.Context, topic string, tenantID interface{}, eventType string, payload []byte) error
}

type CreateTenantHandler struct {
	repo      domain.Repository
	publisher Publisher
}

func NewCreateTenantHandler(repo domain.Repository, publisher Publisher) *CreateTenantHandler {
	return &CreateTenantHandler{repo: repo, publisher: publisher}
}

func (h *CreateTenantHandler) Handle(ctx context.Context, cmd CreateTenantCommand) (*domain.Tenant, error) {
	exists, err := h.repo.ExistsBySlug(ctx, cmd.Slug)
	if err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.ExistsBySlug: %w", err)
	}
	if exists {
		return nil, domain.ErrSlugAlreadyTaken
	}

	t, err := domain.NewTenant(cmd.Name, cmd.Slug, cmd.Plan)
	if err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.NewTenant: %w", err)
	}

	if err := h.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.Save: %w", err)
	}

	// Publish domain events — non-fatal if Kafka is down
	if h.publisher != nil {
		for _, evt := range t.PopEvents() {
			_ = evt // publishing handled in Phase 4 when Kafka is wired
		}
	}

	return t, nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd services/tenant && go test -v -race ./internal/application/command/...
```

- [ ] **Step 5: Commit**

```bash
git add services/tenant/internal/application/command/
git commit -m "feat(tenant): add CreateTenant command handler with slug uniqueness check"
```

---

### Task 16: Application layer — CreateBranch + queries

**Files:**
- Create: `services/tenant/internal/application/command/create_branch.go`
- Create: `services/tenant/internal/application/query/get_tenant.go`
- Create: `services/tenant/internal/application/query/list_branches.go`

- [ ] **Step 1: Create create_branch.go**

```go
// services/tenant/internal/application/command/create_branch.go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type CreateBranchCommand struct {
	IdempotencyKey string
	TenantID       uuid.UUID
	Name           string
	Address        domain.Address
	Timezone       string
}

type CreateBranchHandler struct {
	repo domain.Repository
}

func NewCreateBranchHandler(repo domain.Repository) *CreateBranchHandler {
	return &CreateBranchHandler{repo: repo}
}

func (h *CreateBranchHandler) Handle(ctx context.Context, cmd CreateBranchCommand) (*domain.Branch, error) {
	t, err := h.repo.FindByID(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.FindByID: %w", err)
	}

	tz := cmd.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	branch := domain.NewBranch(t.ID, cmd.Name, cmd.Address, tz)
	if err := t.AddBranch(branch); err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.AddBranch: %w", err)
	}

	if err := h.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.Save: %w", err)
	}

	return &branch, nil
}
```

- [ ] **Step 2: Create get_tenant.go**

```go
// services/tenant/internal/application/query/get_tenant.go
package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type GetTenantQuery struct {
	TenantID uuid.UUID
}

type GetTenantHandler struct {
	repo domain.Repository
}

func NewGetTenantHandler(repo domain.Repository) *GetTenantHandler {
	return &GetTenantHandler{repo: repo}
}

func (h *GetTenantHandler) Handle(ctx context.Context, q GetTenantQuery) (*domain.Tenant, error) {
	t, err := h.repo.FindByID(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("GetTenantHandler.FindByID: %w", err)
	}
	return t, nil
}
```

- [ ] **Step 3: Create list_branches.go**

```go
// services/tenant/internal/application/query/list_branches.go
package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type ListBranchesQuery struct {
	TenantID uuid.UUID
}

type ListBranchesHandler struct {
	repo domain.Repository
}

func NewListBranchesHandler(repo domain.Repository) *ListBranchesHandler {
	return &ListBranchesHandler{repo: repo}
}

func (h *ListBranchesHandler) Handle(ctx context.Context, q ListBranchesQuery) ([]domain.Branch, error) {
	branches, err := h.repo.ListBranches(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("ListBranchesHandler.ListBranches: %w", err)
	}
	return branches, nil
}
```

- [ ] **Step 4: Build check**

```bash
cd services/tenant && go build ./internal/application/...
```

- [ ] **Step 5: Commit**

```bash
git add services/tenant/internal/application/
git commit -m "feat(tenant): add CreateBranch command and GetTenant/ListBranches queries"
```

---

### Task 17: Infrastructure — PostgreSQL repository

**Files:**
- Create: `services/tenant/internal/infrastructure/postgres/tenant_repo.go`

- [ ] **Step 1: Create tenant_repo.go**

```go
// services/tenant/internal/infrastructure/postgres/tenant_repo.go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	shareddb "github.com/xyn-pos/shared/pkg/database"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type TenantRepository struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool, tracer: otel.Tracer("tenant.postgres")}
}

func (r *TenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.Save")
	defer span.End()

	return shareddb.WithTenantTx(ctx, r.pool, t.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, plan, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				name       = EXCLUDED.name,
				plan       = EXCLUDED.plan,
				status     = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at
		`, t.ID, t.Name, t.Slug, string(t.Plan.Tier), string(t.Status), t.CreatedAt, t.UpdatedAt)
		if err != nil {
			return fmt.Errorf("TenantRepository.Save upsert: %w", err)
		}

		for _, b := range t.Branches {
			_, err := tx.Exec(ctx, `
				INSERT INTO branches (id, tenant_id, name, street, city, province, postal_code, country, timezone, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				ON CONFLICT (id) DO UPDATE SET
					name        = EXCLUDED.name,
					is_active   = EXCLUDED.is_active,
					updated_at  = EXCLUDED.updated_at
			`, b.ID, b.TenantID, b.Name,
				b.Address.Street, b.Address.City, b.Address.Province, b.Address.PostalCode, b.Address.Country,
				b.Timezone, b.IsActive, b.CreatedAt, b.UpdatedAt)
			if err != nil {
				return fmt.Errorf("TenantRepository.Save branch upsert id=%s: %w", b.ID, err)
			}
		}
		return nil
	})
}

func (r *TenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.FindByID")
	defer span.End()

	var t domain.Tenant
	var planTier, status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, plan, status, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Slug, &planTier, &status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("TenantRepository.FindByID id=%s: %w", id, err)
	}
	t.Plan = domain.PlanFromTier(domain.PlanTier(planTier))
	t.Status = domain.TenantStatus(status)

	branches, err := r.ListBranches(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Branches = branches
	return &t, nil
}

func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.FindBySlug")
	defer span.End()

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = $1`, slug).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("TenantRepository.FindBySlug slug=%s: %w", slug, err)
	}
	return r.FindByID(ctx, id)
}

func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.ExistsBySlug")
	defer span.End()

	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("TenantRepository.ExistsBySlug: %w", err)
	}
	return exists, nil
}

func (r *TenantRepository) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.ListBranches")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, street, city, province, postal_code, country, timezone, is_active, created_at, updated_at
		FROM branches WHERE tenant_id = $1 ORDER BY created_at ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("TenantRepository.ListBranches: %w", err)
	}
	defer rows.Close()

	var branches []domain.Branch
	for rows.Next() {
		var b domain.Branch
		var updatedAt time.Time
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.Name,
			&b.Address.Street, &b.Address.City, &b.Address.Province, &b.Address.PostalCode, &b.Address.Country,
			&b.Timezone, &b.IsActive, &b.CreatedAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("TenantRepository.ListBranches scan: %w", err)
		}
		b.UpdatedAt = updatedAt
		branches = append(branches, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("TenantRepository.ListBranches rows: %w", err)
	}
	return branches, nil
}
```

- [ ] **Step 2: Build check**

```bash
cd services/tenant && go build ./internal/infrastructure/...
```

- [ ] **Step 3: Commit**

```bash
git add services/tenant/internal/infrastructure/postgres/tenant_repo.go
git commit -m "feat(tenant): add PostgreSQL repository with OTEL spans and RLS tenant context"
```

---

### Task 18: Integration test — PostgreSQL repository

**Files:**
- Create: `services/tenant/internal/infrastructure/postgres/tenant_repo_test.go`

- [ ] **Step 1: Add testcontainers to tenant go.mod**

Add to `services/tenant/go.mod` requires:
```
github.com/testcontainers/testcontainers-go v0.42.0
github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
github.com/pressly/goose/v3 v3.27.1
```

Then: `cd services/tenant && go mod tidy && cd ../..`

- [ ] **Step 2: Create the integration test**

```go
// services/tenant/internal/infrastructure/postgres/tenant_repo_test.go
//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	shareddb "github.com/xyn-pos/shared/pkg/database"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	"github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
)

type TenantRepoSuite struct {
	suite.Suite
	container *tcpostgres.PostgresContainer
	pool      *pgxpool.Pool
	repo      *postgres.TenantRepository
}

func (s *TenantRepoSuite) SetupSuite() {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx,
		"postgres:18.4-alpine",
		tcpostgres.WithDatabase("xyn_tenant_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	s.Require().NoError(err)
	s.container = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	db, err := sql.Open("pgx", connStr)
	s.Require().NoError(err)
	goose.SetBaseFS(postgres.MigrationsFS)
	s.Require().NoError(goose.SetDialect("postgres"))
	s.Require().NoError(goose.Up(db, "migrations"))
	db.Close()

	s.pool, err = pgxpool.New(ctx, connStr)
	s.Require().NoError(err)
	s.repo = postgres.NewTenantRepository(s.pool)
}

func (s *TenantRepoSuite) TearDownSuite() {
	s.pool.Close()
	_ = s.container.Terminate(context.Background())
}

func (s *TenantRepoSuite) SetupTest() {
	_, err := s.pool.Exec(context.Background(),
		"TRUNCATE tenants, branches, idempotency_keys RESTART IDENTITY CASCADE")
	s.Require().NoError(err)
}

func TestTenantRepoSuite(t *testing.T) { suite.Run(t, new(TenantRepoSuite)) }

func (s *TenantRepoSuite) TestSaveAndFindByID_RoundTrip() {
	ctx := context.Background()
	t, _ := domain.NewTenant("Warung Padang", "warung-padang", domain.PlanTierGrowth)

	// Save requires a superuser session for initial insert (no RLS context yet for new tenant)
	_, err := s.pool.Exec(ctx, "SET app.current_tenant_id = $1", t.ID.String())
	s.Require().NoError(err)

	s.Require().NoError(s.repo.Save(ctx, t))

	found, err := s.repo.FindByID(ctx, t.ID)
	s.Require().NoError(err)
	s.Equal(t.ID, found.ID)
	s.Equal("warung-padang", found.Slug)
	s.Equal(domain.PlanTierGrowth, found.Plan.Tier)
}

func (s *TenantRepoSuite) TestExistsBySlug_ExistingSlug_ReturnsTrue() {
	ctx := context.Background()
	t, _ := domain.NewTenant("Test", "slug-exists", domain.PlanTierFree)
	_, _ = s.pool.Exec(ctx, "SET app.current_tenant_id = $1", t.ID.String())
	s.Require().NoError(s.repo.Save(ctx, t))

	exists, err := s.repo.ExistsBySlug(ctx, "slug-exists")
	s.Require().NoError(err)
	s.True(exists)
}

func (s *TenantRepoSuite) TestFindByID_NotFound_ReturnsErrTenantNotFound() {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, "SET app.current_tenant_id = $1", "00000000-0000-0000-0000-000000000001")
	s.Require().NoError(err)

	_, err = s.repo.FindByID(ctx, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func (s *TenantRepoSuite) TestRLS_DifferentTenant_HidesData() {
	ctx := context.Background()
	tenantA, _ := domain.NewTenant("Tenant A", "tenant-a", domain.PlanTierFree)
	_, _ = s.pool.Exec(ctx, "SET app.current_tenant_id = $1", tenantA.ID.String())
	s.Require().NoError(s.repo.Save(ctx, tenantA))

	// Now query as a different tenant — RLS should hide tenantA's row
	tenantBID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, _ = s.pool.Exec(ctx, "SET app.current_tenant_id = $1", tenantBID)

	_, err := s.repo.FindByID(ctx, tenantA.ID)
	s.ErrorIs(err, domain.ErrTenantNotFound, "RLS must hide other tenants' data")
}
```

- [ ] **Step 3: Run integration test (requires Docker)**

```bash
cd services/tenant && go test -v -race -tags=integration -timeout=120s ./internal/infrastructure/...
```

Expected: all 4 tests PASS. If Docker is not running, skip and run later.

- [ ] **Step 4: Commit**

```bash
git add services/tenant/internal/infrastructure/postgres/tenant_repo_test.go services/tenant/go.mod services/tenant/go.sum
git commit -m "test(tenant): add PostgreSQL integration tests with testcontainers and RLS isolation check"
```

---

### Task 19: gRPC interface layer

**Files:**
- Create: `services/tenant/internal/interfaces/grpc/server.go`
- Create: `services/tenant/internal/interfaces/grpc/tenant_handler.go`

- [ ] **Step 1: Create server.go**

```go
// services/tenant/internal/interfaces/grpc/server.go
package grpc

import (
	"fmt"
	"net"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
	"google.golang.org/grpc"
)

type Server struct {
	grpc *grpc.Server
	port int
}

func NewServer(port int, handler *TenantHandler, verifyFn func(string) (*sharedauth.Claims, error)) *Server {
	opts := middleware.Chain(middleware.Auth(verifyFn))
	srv := grpc.NewServer(opts...)

	// Register the TenantService handler
	// pb.RegisterTenantServiceServer(srv, handler)
	// Uncomment above after buf generate produces the gen/ code

	return &Server{grpc: srv, port: port}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("grpc.Server.Start listen: %w", err)
	}
	return s.grpc.Serve(lis)
}

func (s *Server) GracefulStop() { s.grpc.GracefulStop() }
```

- [ ] **Step 2: Create tenant_handler.go**

```go
// services/tenant/internal/interfaces/grpc/tenant_handler.go
package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sharederrors "github.com/xyn-pos/shared/pkg/errors"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
	"go.opentelemetry.io/otel"

	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"

	// Uncomment when buf generate runs:
	// pb "github.com/xyn-pos/gen/tenant/v1"
)

var tracer = otel.Tracer("tenant-service.grpc")

type TenantHandler struct {
	createTenant  *command.CreateTenantHandler
	createBranch  *command.CreateBranchHandler
	getTenant     *query.GetTenantHandler
	listBranches  *query.ListBranchesHandler
}

func NewTenantHandler(
	createTenant *command.CreateTenantHandler,
	createBranch *command.CreateBranchHandler,
	getTenant *query.GetTenantHandler,
	listBranches *query.ListBranchesHandler,
) *TenantHandler {
	return &TenantHandler{
		createTenant: createTenant,
		createBranch: createBranch,
		getTenant:    getTenant,
		listBranches: listBranches,
	}
}

// CreateTenantRPC is the implementation wired to pb.TenantServiceServer.
// Signature matches the generated interface — rename to CreateTenant after buf gen.
func (h *TenantHandler) CreateTenantRPC(ctx context.Context, idempotencyKey, name, slug string, planTier domain.PlanTier) (*domain.Tenant, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.CreateTenant")
	defer span.End()

	cmd := command.CreateTenantCommand{
		IdempotencyKey: idempotencyKey,
		Name:           name,
		Slug:           slug,
		Plan:           planTier,
	}

	result, err := h.createTenant.Handle(ctx, cmd)
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapToGRPCStatus(err).Err()
	}
	return result, nil
}

// GetTenantRPC is the implementation wired to pb.TenantServiceServer.
func (h *TenantHandler) GetTenantRPC(ctx context.Context, tenantIDStr string) (*domain.Tenant, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.GetTenant")
	defer span.End()

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	result, err := h.getTenant.Handle(ctx, query.GetTenantQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapToGRPCStatus(err).Err()
	}
	return result, nil
}
```

- [ ] **Step 3: Build check**

```bash
cd services/tenant && go build ./internal/interfaces/...
```

- [ ] **Step 4: Commit**

```bash
git add services/tenant/internal/interfaces/
git commit -m "feat(tenant): add gRPC server and handler with OTEL tracing and error mapping"
```

---

### Task 20: provider.go + main.go + Dockerfile

**Files:**
- Create: `services/tenant/provider.go`
- Modify: `services/tenant/cmd/server/main.go`
- Create: `services/tenant/Dockerfile`

- [ ] **Step 1: Create provider.go**

```go
// services/tenant/provider.go
package tenant

import (
	"context"
	"fmt"

	shareddb "github.com/xyn-pos/shared/pkg/database"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	"github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
	grpchandler "github.com/xyn-pos/services/tenant/internal/interfaces/grpc"
)

type Server struct {
	grpc     *grpchandler.Server
	shutdown func()
}

func New(ctx context.Context, cfg *Config) (*Server, error) {
	// Telemetry
	shutdownTelemetry, err := sharedtelemetry.Setup(ctx, sharedtelemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Environment:    cfg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("provider.New telemetry: %w", err)
	}

	// Database
	pool, err := shareddb.NewPool(ctx, shareddb.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		shutdownTelemetry()
		return nil, fmt.Errorf("provider.New database: %w", err)
	}

	// Infrastructure
	repo := postgres.NewTenantRepository(pool)

	// Application
	createTenantH := command.NewCreateTenantHandler(repo, nil)
	createBranchH := command.NewCreateBranchHandler(repo)
	getTenantH := query.NewGetTenantHandler(repo)
	listBranchesH := query.NewListBranchesHandler(repo)

	// Interface
	handler := grpchandler.NewTenantHandler(createTenantH, createBranchH, getTenantH, listBranchesH)
	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, handler, nil) // nil verify = no auth in Phase 3

	return &Server{
		grpc: grpcSrv,
		shutdown: func() {
			pool.Close()
			shutdownTelemetry()
		},
	}, nil
}

func (s *Server) Start() error    { return s.grpc.Start() }
func (s *Server) Stop()           { s.grpc.GracefulStop(); s.shutdown() }
```

- [ ] **Step 2: Replace cmd/server/main.go**

```go
// services/tenant/cmd/server/main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tenant "github.com/xyn-pos/services/tenant"
)

func main() {
	ctx := context.Background()

	cfg, err := tenant.LoadConfig()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	srv, err := tenant.New(ctx, cfg)
	if err != nil {
		slog.Error("provider", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("tenant-service starting", "port", cfg.GRPCPort)

	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("grpc serve", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	srv.Stop()
}
```

- [ ] **Step 3: Create Dockerfile**

```dockerfile
# services/tenant/Dockerfile
FROM golang:1.26.4-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /workspace

# Copy go workspace files
COPY go.work go.work.sum ./
COPY shared/go/go.mod shared/go/go.sum ./shared/go/
COPY gen/go.mod gen/go.sum ./gen/
COPY services/tenant/go.mod services/tenant/go.sum ./services/tenant/

# Download deps
RUN go work sync

# Copy source
COPY shared/go/ ./shared/go/
COPY gen/ ./gen/
COPY services/tenant/ ./services/tenant/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /service \
    ./services/tenant/cmd/server/

# Runtime — distroless, non-root
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
```

- [ ] **Step 4: Build the service**

```bash
cd services/tenant && go build ./cmd/server/ && cd ../..
```

Expected: no errors. Binary created at `services/tenant/cmd/server/server` (or similar).

- [ ] **Step 5: Run all tenant tests**

```bash
go test -v -race ./services/tenant/...
```

Expected: all unit tests PASS (integration tests skipped without `-tags=integration`).

- [ ] **Step 6: Run lint**

```bash
make lint
```

Fix any issues before committing.

- [ ] **Step 7: Commit**

```bash
git add services/tenant/provider.go services/tenant/cmd/server/main.go services/tenant/Dockerfile
git commit -m "feat(tenant): add provider (Manual DI), main.go bootstrap, and multi-stage Dockerfile"
```

---

### Task 21: Final verification and Phase 3 completion commit

- [ ] **Step 1: Run full test suite**

```bash
go test -race ./services/tenant/... ./shared/go/...
```

Expected: all unit tests PASS.

- [ ] **Step 2: Run integration tests (requires Docker)**

```bash
go test -race -tags=integration -timeout=120s ./services/tenant/internal/infrastructure/...
```

Expected: PASS for all 4 testcontainers tests.

- [ ] **Step 3: Run linter**

```bash
make lint
```

Expected: no errors.

- [ ] **Step 4: Build all services (ensure placeholders still compile)**

```bash
make build-all
```

Expected: all 6 services build successfully.

- [ ] **Step 5: Verify proto generation works**

```bash
make proto-gen && make proto-lint
```

Expected: `Generated code in gen/` with no lint errors.

- [ ] **Step 6: Tag Phase 3 complete**

```bash
git add -A
git commit -m "feat(phase3): complete core domain & microservices boilerplate

- Epic 3.2: Common proto definitions (address + tenant service) with buf codegen
- Epic 3.1: 7 shared Go packages (errors, logger, auth, database, telemetry, events, middleware)
- Epic 3.4: PostgreSQL schema + RLS policies via Goose migration
- Epic 3.3: Tenant Service — full DDD template (domain/application/infrastructure/interfaces)
  - Tenant aggregate with plan limit enforcement and domain events
  - Manual DI in provider.go
  - OpenTelemetry on all layers
  - Unit + integration tests with testcontainers

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Definition of Done Checklist

```
[ ] buf generate produces Go code with zero errors
[ ] All shared pkg unit tests pass: go test -race ./shared/go/...
[ ] Tenant service unit tests pass: go test -race ./services/tenant/...
[ ] Integration test passes (testcontainers PostgreSQL 18.4)
[ ] RLS test passes: DifferentTenant returns ErrTenantNotFound
[ ] make lint passes with zero issues
[ ] make build-all — all 6 services compile
[ ] Dockerfile builds: docker build -f services/tenant/Dockerfile .
[ ] OTEL spans visible in Jaeger for a CreateTenant call
```

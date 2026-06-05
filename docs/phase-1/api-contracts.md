# API Contracts — gRPC & REST Standards — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer

---

## 1. Protobuf & Buf Toolchain

### 1.1 Why Buf?

`buf` replaces the raw `protoc` compiler workflow with:
- Consistent lint rules (prevents proto anti-patterns)
- Managed code generation across languages (Go, TypeScript, Dart)
- Breaking change detection (prevents unintentional API breaks)
- BSR (Buf Schema Registry) for schema versioning and dependency management

### 1.2 Workspace Structure

```
proto/
├── buf.yaml                 ← Workspace config
├── buf.gen.yaml             ← Code generation config
├── tenant/
│   └── v1/
│       └── tenant.proto
├── pos/
│   └── v1/
│       └── order.proto
├── payment/
│   └── v1/
│       └── payment.proto
├── inventory/
│   └── v1/
│       └── inventory.proto
└── common/
    └── v1/
        ├── types.proto      ← Shared types (Money, Pagination)
        └── errors.proto     ← Standard error details
```

### 1.3 buf.yaml

```yaml
# proto/buf.yaml
version: v2
modules:
  - path: .
lint:
  use:
    - STANDARD
  except:
    - PACKAGE_VERSION_SUFFIX  # We manage versions in directory structure
breaking:
  use:
    - FILE
```

### 1.4 buf.gen.yaml

```yaml
# proto/buf.gen.yaml
version: v2
plugins:
  # Go gRPC server + client code
  - remote: buf.build/grpc/go
    out: ../services
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false

  # Go Protobuf messages
  - remote: buf.build/protocolbuffers/go
    out: ../services
    opt:
      - paths=source_relative

  # gRPC-Gateway (REST transcoding)
  - remote: buf.build/grpc-ecosystem/gateway
    out: ../services
    opt:
      - paths=source_relative

  # TypeScript (for web BFF)
  - remote: buf.build/bufbuild/es
    out: ../apps/web/src/gen
    opt:
      - target=ts

  # Dart (for Flutter)
  - remote: buf.build/grpc/dart
    out: ../apps/mobile/lib/gen
```

---

## 2. Protobuf Style Guide

### 2.1 File Structure

Every `.proto` file follows this structure:

```protobuf
// 1. File header
syntax = "proto3";

// 2. Package declaration (always versioned)
package xyn.pos.v1;

// 3. Go package option
option go_package = "github.com/xyn/pos-v1/services/pos/internal/interfaces/grpc/pb;pb";

// 4. Imports (stdlib first, then external, then internal)
import "google/protobuf/timestamp.proto";
import "google/protobuf/field_mask.proto";
import "validate/validate.proto";           // buf validate
import "common/v1/types.proto";             // internal shared types

// 5. Service definition(s)
// 6. Request/Response messages
// 7. Data messages (entities, value objects)
// 8. Enum definitions
```

### 2.2 Naming Rules

| Element | Convention | Example |
|---|---|---|
| Package | `snake_case`, versioned | `xyn.pos.v1` |
| Service | `PascalCase` + `Service` | `OrderService` |
| RPC methods | `PascalCase`, CRUD verb + noun | `CreateOrder`, `GetOrder`, `ListOrders` |
| Messages | `PascalCase` | `Order`, `CreateOrderRequest` |
| Request messages | `{Method}Request` | `CreateOrderRequest` |
| Response messages | `{Method}Response` | `CreateOrderResponse` |
| Fields | `snake_case` | `order_id`, `created_at`, `total_amount` |
| Enums | `PascalCase` | `OrderStatus` |
| Enum values | `SCREAMING_SNAKE_CASE` with type prefix | `ORDER_STATUS_PENDING` |

### 2.3 Standard Field Types

```protobuf
// Always use well-known types for timestamps
google.protobuf.Timestamp created_at = 10;

// Use string for UUIDs (not bytes) — human-readable in logs
string order_id = 1;  // UUID v4 as string

// Money is NEVER a float — use our custom Money type
xyn.common.v1.Money total_amount = 5;
```

```protobuf
// common/v1/types.proto
message Money {
  string currency_code = 1;  // ISO 4217: "IDR", "USD"
  int64 amount_minor = 2;    // Amount in minor units (cents, sen)
                              // $12.34 → amount_minor = 1234
                              // Rp 15.000 → amount_minor = 1500000
}

message Pagination {
  int32 page = 1;
  int32 page_size = 2;  // max: 100
}

message PaginationMeta {
  int32 page = 1;
  int32 page_size = 2;
  int32 total_count = 3;
  int32 total_pages = 4;
}
```

**Why integer for money?** Floating-point arithmetic is not exact. `0.1 + 0.2 = 0.30000000000000004` in IEEE 754. Financial calculations must use integer minor units (or `decimal` in application code) and convert only at display time.

### 2.4 Field Numbering Rules

- Fields 1–15: High-frequency fields (single-byte encoding in protobuf)
- Fields 16–2047: Less frequent fields
- Fields 2048+: Reserve for future use
- Never reuse a field number (even after deletion — add to reserved list)
- Always reserve deleted field names and numbers:

```protobuf
message Order {
  reserved 6, 7;                  // was: discount_code, coupon_id (removed in v1.2)
  reserved "discount_code", "coupon_id";
}
```

---

## 3. Service Definitions

### 3.1 Order Service

```protobuf
// pos/v1/order.proto
syntax = "proto3";
package xyn.pos.v1;

import "google/protobuf/timestamp.proto";
import "google/api/annotations.proto";    // gRPC-Gateway REST annotations
import "validate/validate.proto";
import "common/v1/types.proto";

service OrderService {
  // ── Commands (write) ──────────────────────────────────────────
  rpc CreateOrder   (CreateOrderRequest)   returns (CreateOrderResponse);
  rpc AddItem       (AddItemRequest)       returns (AddItemResponse);
  rpc RemoveItem    (RemoveItemRequest)    returns (RemoveItemResponse);
  rpc UpdateItemQty (UpdateItemQtyRequest) returns (UpdateItemQtyResponse);
  rpc ApplyDiscount (ApplyDiscountRequest) returns (ApplyDiscountResponse);
  rpc CancelOrder   (CancelOrderRequest)   returns (CancelOrderResponse);

  // ── Queries (read) ────────────────────────────────────────────
  rpc GetOrder      (GetOrderRequest)      returns (GetOrderResponse);
  rpc ListOrders    (ListOrdersRequest)    returns (ListOrdersResponse);

  // ── Streaming ─────────────────────────────────────────────────
  rpc WatchOrder    (WatchOrderRequest)    returns (stream OrderEvent);
}

// ── Request / Response Messages ──────────────────────────────────

message CreateOrderRequest {
  string idempotency_key = 1 [(validate.rules).string.uuid = true];
  string branch_id       = 2 [(validate.rules).string.uuid = true];
  string table_number    = 3;  // nullable for takeaway
  OrderType order_type   = 4 [(validate.rules).enum.defined_only = true];
}

message CreateOrderResponse {
  string order_id   = 1;
  Order  order      = 2;
}

message AddItemRequest {
  string order_id        = 1 [(validate.rules).string.uuid = true];
  string product_id      = 2 [(validate.rules).string.uuid = true];
  int32  quantity        = 3 [(validate.rules).int32 = {gte: 1, lte: 999}];
  string notes           = 4 [(validate.rules).string.max_len = 200];
  repeated string addon_ids = 5;
}

// ── Domain Messages ───────────────────────────────────────────────

message Order {
  string                     order_id      = 1;
  string                     branch_id     = 2;
  string                     cashier_id    = 3;
  OrderStatus                status        = 4;
  OrderType                  order_type    = 5;
  string                     table_number  = 6;
  repeated OrderItem         items         = 7;
  xyn.common.v1.Money        subtotal      = 8;
  xyn.common.v1.Money        tax_amount    = 9;
  xyn.common.v1.Money        discount      = 10;
  xyn.common.v1.Money        total         = 11;
  google.protobuf.Timestamp  created_at    = 12;
  google.protobuf.Timestamp  updated_at    = 13;
}

message OrderItem {
  string               item_id     = 1;
  string               product_id  = 2;
  string               product_name = 3;
  int32                quantity    = 4;
  xyn.common.v1.Money  unit_price  = 5;
  xyn.common.v1.Money  subtotal    = 6;
  repeated string      addons      = 7;
  string               notes       = 8;
}

// ── Enums ─────────────────────────────────────────────────────────

enum OrderStatus {
  ORDER_STATUS_UNSPECIFIED = 0;
  ORDER_STATUS_DRAFT       = 1;
  ORDER_STATUS_PENDING     = 2;  // Submitted to kitchen
  ORDER_STATUS_PAID        = 3;
  ORDER_STATUS_CANCELLED   = 4;
  ORDER_STATUS_VOID        = 5;  // Reversed after payment
}

enum OrderType {
  ORDER_TYPE_UNSPECIFIED = 0;
  ORDER_TYPE_DINE_IN     = 1;
  ORDER_TYPE_TAKEAWAY    = 2;
  ORDER_TYPE_DELIVERY    = 3;
}
```

### 3.2 Payment Service

```protobuf
// payment/v1/payment.proto
service PaymentService {
  rpc ProcessPayment (ProcessPaymentRequest)  returns (ProcessPaymentResponse);
  rpc SplitPayment   (SplitPaymentRequest)    returns (SplitPaymentResponse);
  rpc VoidPayment    (VoidPaymentRequest)      returns (VoidPaymentResponse);
  rpc RefundPayment  (RefundPaymentRequest)    returns (RefundPaymentResponse);
  rpc GetPayment     (GetPaymentRequest)       returns (GetPaymentResponse);
}

message ProcessPaymentRequest {
  string          idempotency_key = 1 [(validate.rules).string.uuid = true];
  string          order_id        = 2 [(validate.rules).string.uuid = true];
  PaymentMethod   method          = 3;
  xyn.common.v1.Money amount_tendered = 4;  // For cash: amount given by customer
}

message ProcessPaymentResponse {
  string          payment_id      = 1;
  PaymentStatus   status          = 2;
  xyn.common.v1.Money change_amount = 3;  // Cash change
  string          receipt_number  = 4;
  string          gateway_ref     = 5;    // External payment gateway reference
}
```

---

## 4. Idempotency Standard

### 4.1 Which Operations Require Idempotency Keys?

| Operation | Required? | Reason |
|---|---|---|
| CreateOrder | ✅ Yes | Duplicate orders must be prevented |
| ProcessPayment | ✅ Yes | Duplicate charges must be prevented |
| RefundPayment | ✅ Yes | Duplicate refunds must be prevented |
| VoidPayment | ✅ Yes | Financial reversal |
| AddItem | ⚠️ Optional | Items can be added multiple times legitimately |
| GetOrder | ❌ No | Read operation, naturally idempotent |
| ListOrders | ❌ No | Read operation |

### 4.2 Implementation

**Client side:** Generate a UUID v4 at the point of user action. Store it in the outbox with the pending operation. If the request fails, retry with the **same** key.

```typescript
// apps/web/src/services/payment.ts
async function processPayment(orderId: string, method: PaymentMethod): Promise<Payment> {
  const idempotencyKey = crypto.randomUUID(); // Generated ONCE, before the call
  // Store in sessionStorage to survive page refresh during network error
  sessionStorage.setItem(`idem_${orderId}`, idempotencyKey);

  return await paymentClient.processPayment({
    idempotencyKey,
    orderId,
    method,
  });
}
```

**Server side (Go):**

```go
// application/command/process_payment.go
func (h *ProcessPaymentHandler) Handle(ctx context.Context, cmd ProcessPaymentCommand) (*ProcessPaymentResult, error) {
    // Check idempotency store first
    cached, err := h.idempStore.Get(ctx, cmd.IdempotencyKey)
    if err == nil {
        return cached, nil // Return cached result — no side effects
    }
    if !errors.Is(err, idempotency.ErrNotFound) {
        return nil, fmt.Errorf("idempotency check: %w", err)
    }

    // Execute the operation
    result, err := h.executePayment(ctx, cmd)
    if err != nil {
        return nil, err
    }

    // Store result — TTL 24h
    if storeErr := h.idempStore.Set(ctx, cmd.IdempotencyKey, result, 24*time.Hour); storeErr != nil {
        // Log but don't fail — the payment succeeded, the idempotency store is best-effort
        slog.WarnContext(ctx, "failed to store idempotency result", "key", cmd.IdempotencyKey, "err", storeErr)
    }

    return result, nil
}
```

### 4.3 Idempotency Key Format

```
Format: UUID v4 (RFC 4122)
Example: 018e1234-5678-7abc-def0-123456789abc
Length: 36 characters

Redis key: idem:{service}:{key}
Example:   idem:payment:018e1234-5678-7abc-def0-123456789abc
TTL:       24 hours
```

---

## 5. gRPC Error Handling Standard

### 5.1 Error Code Mapping

Use standard gRPC status codes, not custom codes:

| gRPC Status | HTTP | Use Case |
|---|---|---|
| `OK` (0) | 200 | Success |
| `INVALID_ARGUMENT` (3) | 400 | Validation failure (bad input) |
| `NOT_FOUND` (5) | 404 | Resource doesn't exist |
| `ALREADY_EXISTS` (6) | 409 | Duplicate creation attempt |
| `PERMISSION_DENIED` (7) | 403 | Authorized but not permitted |
| `UNAUTHENTICATED` (16) | 401 | Not authenticated |
| `RESOURCE_EXHAUSTED` (8) | 429 | Rate limit exceeded |
| `FAILED_PRECONDITION` (9) | 422 | Business rule violated |
| `INTERNAL` (13) | 500 | Unexpected server error |
| `UNAVAILABLE` (14) | 503 | Service temporarily unavailable |

### 5.2 Structured Error Details

Use `google.rpc.Status` with error details for machine-readable errors:

```go
// shared/go/pkg/errors/grpc.go
func NewValidationError(violations map[string]string) error {
    details := &errdetails.BadRequest{}
    for field, desc := range violations {
        details.FieldViolations = append(details.FieldViolations, &errdetails.BadRequest_FieldViolation{
            Field:       field,
            Description: desc,
        })
    }
    st, _ := status.New(codes.InvalidArgument, "validation failed").WithDetails(details)
    return st.Err()
}

// Usage in handler:
if err := validateOrder(req); err != nil {
    return nil, errors.NewValidationError(map[string]string{
        "quantity": "must be between 1 and 999",
        "product_id": "must be a valid UUID",
    })
}
```

### 5.3 Error Response on REST (gRPC-Gateway)

gRPC-Gateway translates gRPC errors to JSON automatically:

```json
// HTTP 422 response for FAILED_PRECONDITION
{
  "code": 9,
  "message": "Order cannot be paid: insufficient stock for item 'Burger' (requested: 3, available: 1)",
  "details": [
    {
      "@type": "type.googleapis.com/google.rpc.PreconditionFailure",
      "violations": [
        {
          "type": "INSUFFICIENT_STOCK",
          "subject": "product:018e1234-5678-7abc-def0-123456789abc",
          "description": "Requested 3 units but only 1 available"
        }
      ]
    }
  ]
}
```

---

## 6. gRPC-Gateway REST Annotations

For external clients (third-party integrations, webhooks), expose REST endpoints via gRPC-Gateway:

```protobuf
import "google/api/annotations.proto";

service OrderService {
  rpc CreateOrder (CreateOrderRequest) returns (CreateOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders"
      body: "*"
    };
  };

  rpc GetOrder (GetOrderRequest) returns (GetOrderResponse) {
    option (google.api.http) = {
      get: "/v1/orders/{order_id}"
    };
  };

  rpc ListOrders (ListOrdersRequest) returns (ListOrdersResponse) {
    option (google.api.http) = {
      get: "/v1/orders"
    };
  };
}
```

---

## 7. API Versioning Strategy

**Rule:** APIs are versioned via package name, not URL path (for gRPC). URL path versioning applies to REST fallback.

```
gRPC: package xyn.pos.v1  →  xyn.pos.v2 (when breaking changes needed)
REST: /v1/orders           →  /v2/orders

Compatibility guarantee:
- v1 and v2 run simultaneously for 6 months minimum
- v1 EOL announced 3 months before shutdown
- Clients receive deprecation warnings in response headers
```

**Non-breaking changes (allowed without version bump):**
- Adding new optional fields to messages
- Adding new RPC methods to a service
- Adding new enum values (if client handles UNRECOGNIZED)

**Breaking changes (require new version):**
- Removing fields or RPC methods
- Changing field types
- Changing field semantics (meaning of a field changes)
- Removing enum values

---

## 8. Streaming API Contracts

### 8.1 KDS Real-Time Updates (Server Streaming)

```protobuf
service KitchenService {
  // Server streams new/updated orders to KDS display
  rpc WatchKitchenQueue (WatchKitchenQueueRequest) returns (stream KitchenTicketEvent);
}

message WatchKitchenQueueRequest {
  string branch_id   = 1;
  string station_id  = 2;  // e.g., "grill", "salad", "drinks"
}

message KitchenTicketEvent {
  EventType           type   = 1;
  KitchenTicket       ticket = 2;
  google.protobuf.Timestamp event_time = 3;

  enum EventType {
    EVENT_TYPE_UNSPECIFIED = 0;
    EVENT_TYPE_NEW         = 1;  // New order arrived
    EVENT_TYPE_UPDATED     = 2;  // Item added or removed
    EVENT_TYPE_COMPLETED   = 3;  // Order marked done
    EVENT_TYPE_CANCELLED   = 4;
  }
}
```

### 8.2 Offline Sync Protocol (Bidirectional Streaming)

```protobuf
service SyncService {
  rpc SyncStream (stream SyncMessage) returns (stream SyncMessage);
}

message SyncMessage {
  string message_id = 1;  // Client-generated UUID for correlation
  oneof payload {
    SyncHandshake  handshake  = 2;  // First message: sends last checkpoint
    OutboxEvent    event      = 3;  // Client sends pending offline events
    SyncAck        ack        = 4;  // Server acks each event
    SyncConflict   conflict   = 5;  // Server signals a conflict
    ServerPush     push       = 6;  // Server pushes downstream changes
    SyncHeartbeat  heartbeat  = 7;  // Keep-alive ping/pong
  }
}

message SyncHandshake {
  string device_id       = 1;
  string last_checkpoint = 2;  // Last server sequence number seen by this device
  string app_version     = 3;
}

message SyncAck {
  string   message_id = 1;   // Correlates to the OutboxEvent message_id
  AckStatus status    = 2;
  string   server_id  = 3;   // Server-assigned ID for the created resource

  enum AckStatus {
    ACK_STATUS_UNSPECIFIED = 0;
    ACK_STATUS_ACCEPTED    = 1;
    ACK_STATUS_DUPLICATE   = 2;  // Idempotency key already processed
    ACK_STATUS_REJECTED    = 3;  // Validation error
  }
}
```

---

## 9. Authentication in gRPC

Auth tokens are passed as gRPC metadata (equivalent to HTTP headers):

```go
// Client interceptor (Go)
func AuthInterceptor(token string) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply any,
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

// Server interceptor — validates token and injects claims into context
func AuthServerInterceptor(verifier TokenVerifier) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (any, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization token")
        }
        claims, err := verifier.Verify(tokens[0])
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        ctx = auth.WithClaims(ctx, claims)
        return handler(ctx, req)
    }
}
```

---

## 10. Webhook Contracts (External Integrations)

For payment gateway callbacks (Midtrans/Xendit) and third-party integrations:

```
POST /webhooks/payment/{gateway}

Headers:
  X-Webhook-Id: {unique event ID from gateway}
  X-Webhook-Timestamp: {Unix timestamp}
  X-Webhook-Signature: {HMAC-SHA256 of payload}

Body: {gateway-specific JSON}
```

**Webhook processing rules:**
1. Verify signature before processing anything
2. Respond `200 OK` immediately — process async in a worker
3. Store the raw payload before processing (for replay/debugging)
4. The processing is idempotent — check `X-Webhook-Id` against a processed-events store
5. If processing fails, return `500` — the gateway will retry with exponential backoff

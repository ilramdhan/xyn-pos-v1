# Error Handling — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Authoritative Standard

---

## 1. Philosophy

### Errors Are Values, Not Exceptions

Go treats errors as first-class values — they are returned, inspected, and wrapped. This is a feature, not a limitation. It forces you to think about failure paths explicitly rather than hoping a catch block somewhere handles it.

**Three rules:**
1. **Every error must be handled** — no silent drops. `_ = someFunc()` for errors is prohibited.
2. **Every error must carry context** — wrap with `fmt.Errorf("operation: %w", err)` at every layer boundary.
3. **Never expose internal errors to clients** — domain errors are translated to user-safe messages at the interface boundary.

---

## 2. Error Taxonomy

```
┌─────────────────────────────────────────────────────────────────┐
│                      Error Categories                           │
│                                                                 │
│  Domain Errors          Application Errors   Infrastructure     │
│  ─────────────          ──────────────────   ──────────────     │
│  Business rule          Use case failure     DB, network,       │
│  violations             (wraps domain)       external API       │
│                                                                 │
│  ErrOrderNotFound       ErrCreateOrderFailed ErrDBTimeout       │
│  ErrInsufficientStock   ErrCheckoutFailed    ErrKafkaUnreachable│
│  ErrInvalidTransition   ErrPaymentDenied     ErrGatewayTimeout  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.1 Domain Errors (Sentinel Values)

Defined in the domain layer. These are **business rule violations** — the operation was understood but cannot succeed in the current state.

```go
// services/pos/internal/domain/order/errors.go
package order

import "errors"

var (
    // State errors
    ErrOrderNotFound       = errors.New("order not found")
    ErrOrderAlreadyPaid    = errors.New("order is already paid")
    ErrOrderCancelled      = errors.New("order is cancelled")
    ErrOrderVoided         = errors.New("order is voided")
    ErrInvalidStateChange  = errors.New("invalid order state transition")
    ErrOrderEmpty          = errors.New("order has no items")

    // Business rule errors
    ErrInsufficientStock   = errors.New("insufficient stock")
    ErrProductNotFound     = errors.New("product not found")
    ErrDuplicateItem       = errors.New("item already exists in order")
    ErrQuantityExceeded    = errors.New("quantity exceeds maximum allowed")
)
```

**Rules for domain errors:**
- Always defined as `var Err... = errors.New(...)` — never `fmt.Errorf`
- Zero external imports in domain layer — stdlib only
- Short, descriptive, lowercase messages (Go convention)
- Never include dynamic data in the error variable itself — that goes in the wrapping

### 2.2 Rich Errors with `samber/oops`

For errors that need to carry structured context (HTTP status, error code, user-safe message, stack trace):

```go
// services/pos/internal/application/command/checkout.go
import "github.com/samber/oops"

func (h *CheckoutHandler) Handle(ctx context.Context, cmd CheckoutCommand) error {
    stock, err := h.stockRepo.GetStock(ctx, cmd.ProductID)
    if err != nil {
        // Infrastructure error — wrap with oops for structured context
        return oops.
            Code("INSUFFICIENT_STOCK").
            In("CheckoutHandler").
            Tags("order", "inventory").
            With("product_id", cmd.ProductID, "requested", cmd.Quantity).
            Public("Not enough stock to complete this order.").
            Wrap(err)
    }

    if stock.Available < cmd.Quantity {
        // Domain error — create a new oops error with full context
        return oops.
            Code("INSUFFICIENT_STOCK").
            In("CheckoutHandler").
            With("product_id", cmd.ProductID, "available", stock.Available, "requested", cmd.Quantity).
            Public(fmt.Sprintf("Only %d units available for '%s'.", stock.Available, stock.ProductName)).
            Errorf("insufficient stock: have %d, need %d", stock.Available, cmd.Quantity)
    }
    // ...
}
```

**`oops` fields reference:**
```go
oops.Code("MACHINE_READABLE_CODE")     // maps to ErrorResponse.error_code
oops.Public("User-safe message")       // safe to show to end users
oops.In("ServiceName.MethodName")      // component location for debugging
oops.Tags("domain", "subdomain")       // for log filtering
oops.With("key", value, "key2", value2) // structured context for logs
oops.Wrap(err)                          // wraps an existing error
oops.Errorf("format %s", arg)           // creates new error
oops.Wrapf(err, "format %s", arg)       // wraps with message
```

---

## 3. Error Flow: Layer by Layer

```
Client Request
      │
      ▼
interfaces/grpc/handler.go   ← maps errors to gRPC status codes
      │                          never returns raw errors to clients
      ▼
application/command/*.go     ← wraps domain errors with use-case context
      │                          uses oops for rich context
      ▼
domain/*/                    ← returns sentinel errors (ErrXxx)
      │                          zero wrapping here
      ▼
infrastructure/postgres/*.go ← wraps DB errors, maps sql.ErrNoRows
                                 uses fmt.Errorf("repo.Method: %w", err)
```

### 3.1 Infrastructure Layer — Repository

```go
// services/pos/internal/infrastructure/postgres/order_repo.go
func (r *orderRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
    var o orderRow
    err := r.pool.QueryRow(ctx, queryFindByID, id).Scan(&o.id, &o.status /* ... */)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrOrderNotFound // translate DB error to domain error
        }
        // Wrap infrastructure errors with context — never expose raw DB errors
        return nil, fmt.Errorf("orderRepo.FindByID id=%s: %w", id, err)
    }
    return o.toDomain(), nil
}

func (r *orderRepo) Save(ctx context.Context, order *domain.Order) error {
    _, err := r.pool.Exec(ctx, querySave, order.ID, order.Status /* ... */)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) {
            switch pgErr.Code {
            case pgerrcode.UniqueViolation:
                return fmt.Errorf("orderRepo.Save: %w: %w", domain.ErrDuplicateOrder, err)
            case pgerrcode.ForeignKeyViolation:
                return fmt.Errorf("orderRepo.Save: foreign key violation: %w", err)
            }
        }
        return fmt.Errorf("orderRepo.Save id=%s: %w", order.ID, err)
    }
    return nil
}
```

### 3.2 Application Layer — Use Case

```go
// services/pos/internal/application/command/cancel_order.go
func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) error {
    order, err := h.repo.FindByID(ctx, cmd.OrderID)
    if err != nil {
        if errors.Is(err, domain.ErrOrderNotFound) {
            // Re-wrap as oops with user-facing message
            return oops.Code("ORDER_NOT_FOUND").
                Public("The order you're trying to cancel doesn't exist.").
                Wrapf(err, "CancelOrderHandler.Handle order_id=%s", cmd.OrderID)
        }
        return oops.Code("INTERNAL_ERROR").
            In("CancelOrderHandler").
            Wrapf(err, "fetch order for cancellation order_id=%s", cmd.OrderID)
    }

    if err := order.Cancel(cmd.Reason); err != nil {
        // Domain error from aggregate — wrap with context
        return oops.Code("ORDER_INVALID_STATE").
            Public(fmt.Sprintf("Cannot cancel order: %s", err.Error())).
            With("order_id", cmd.OrderID, "current_status", order.Status).
            Wrapf(err, "CancelOrderHandler: cancel order_id=%s", cmd.OrderID)
    }

    if err := h.repo.Save(ctx, order); err != nil {
        return oops.Code("INTERNAL_ERROR").
            In("CancelOrderHandler").
            Wrapf(err, "save cancelled order_id=%s", cmd.OrderID)
    }

    h.events.Publish(ctx, order.PopEvents()...)
    return nil
}
```

### 3.3 Interface Layer — gRPC Handler

```go
// services/pos/internal/interfaces/grpc/order_handler.go
func (h *OrderHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
    cmd := application.CancelOrderCommand{
        OrderID: uuid.MustParse(req.OrderId),
        Reason:  req.Reason,
        UserID:  auth.ClaimsFromContext(ctx).UserID,
    }

    if err := h.svc.CancelOrder(ctx, cmd); err != nil {
        return nil, mapToGRPCError(ctx, err)
    }

    return &pb.CancelOrderResponse{OrderId: req.OrderId}, nil
}

// mapToGRPCError translates application/domain errors to gRPC status errors.
// This is the ONLY place where errors are translated to gRPC codes.
func mapToGRPCError(ctx context.Context, err error) error {
    // Extract oops context if available
    var oopsErr oops.OopsError
    if errors.As(err, &oopsErr) {
        code := oopsErr.Code()
        publicMsg := oopsErr.Public()

        // Log the full internal error with structured context
        slog.ErrorContext(ctx, "request failed",
            slog.String("error_code", code),
            slog.String("error", err.Error()),
            slog.Any("context", oopsErr.Context()),
        )

        // Map error codes to gRPC status codes
        grpcCode := errorCodeToGRPCStatus(code)
        if publicMsg != "" {
            return status.Error(grpcCode, publicMsg)
        }
        return status.Error(grpcCode, defaultMessageFor(grpcCode))
    }

    // Unknown error — log and return generic 500
    slog.ErrorContext(ctx, "unhandled error", "err", err)
    return status.Error(codes.Internal, "An unexpected error occurred")
}

func errorCodeToGRPCStatus(code string) codes.Code {
    m := map[string]codes.Code{
        "ORDER_NOT_FOUND":       codes.NotFound,
        "PRODUCT_NOT_FOUND":     codes.NotFound,
        "ORDER_ALREADY_PAID":    codes.FailedPrecondition,
        "ORDER_INVALID_STATE":   codes.FailedPrecondition,
        "INSUFFICIENT_STOCK":    codes.FailedPrecondition,
        "UNAUTHORIZED":          codes.Unauthenticated,
        "FORBIDDEN":             codes.PermissionDenied,
        "VALIDATION_FAILED":     codes.InvalidArgument,
        "RATE_LIMIT_EXCEEDED":   codes.ResourceExhausted,
        "SERVICE_UNAVAILABLE":   codes.Unavailable,
    }
    if c, ok := m[code]; ok {
        return c
    }
    return codes.Internal
}
```

---

## 4. Panic Recovery

### 4.1 gRPC Recovery Interceptor

```go
// shared/go/pkg/middleware/recovery.go
package middleware

import (
    "context"
    "log/slog"
    "runtime/debug"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// RecoveryInterceptor catches panics in gRPC handlers and converts them
// to INTERNAL status errors. Panics are always logged with stack traces.
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (resp any, err error) {
        defer func() {
            if r := recover(); r != nil {
                stack := debug.Stack()
                slog.ErrorContext(ctx, "panic recovered in gRPC handler",
                    "method", info.FullMethod,
                    "panic", r,
                    "stack", string(stack),
                )
                err = status.Errorf(codes.Internal, "internal server error")
            }
        }()
        return handler(ctx, req)
    }
}

// StreamRecoveryInterceptor handles panics in streaming gRPC handlers.
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
    return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo,
        handler grpc.StreamHandler) (err error) {
        defer func() {
            if r := recover(); r != nil {
                stack := debug.Stack()
                slog.ErrorContext(ss.Context(), "panic recovered in gRPC stream",
                    "method", info.FullMethod,
                    "panic", r,
                    "stack", string(stack),
                )
                err = status.Errorf(codes.Internal, "internal server error")
            }
        }()
        return handler(srv, ss)
    }
}
```

### 4.2 HTTP Recovery Middleware

```go
// shared/go/pkg/middleware/http_recovery.go
func HTTPRecovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rc := recover(); rc != nil {
                stack := debug.Stack()
                slog.ErrorContext(r.Context(), "panic recovered in HTTP handler",
                    "path", r.URL.Path,
                    "panic", rc,
                    "stack", string(stack),
                )
                requestID := requestIDFromContext(r.Context())
                resp := response.InternalError(requestID)
                response.WriteError(w, r, resp)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

---

## 5. Context Deadline & Cancellation

```go
// Services must respect context cancellation at every I/O point.
func (r *orderRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
    // pgx automatically respects context cancellation — no extra code needed
    // But be explicit about the timeout source:
    row, err := r.pool.QueryRow(ctx, query, id)
    if err != nil {
        if errors.Is(ctx.Err(), context.DeadlineExceeded) {
            return nil, fmt.Errorf("orderRepo.FindByID: deadline exceeded: %w", err)
        }
        if errors.Is(ctx.Err(), context.Canceled) {
            return nil, fmt.Errorf("orderRepo.FindByID: request cancelled: %w", err)
        }
        return nil, fmt.Errorf("orderRepo.FindByID: %w", err)
    }
    // ...
}
```

**Timeout configuration per operation type:**
```go
// shared/go/pkg/database/timeouts.go
const (
    TimeoutRead        = 5 * time.Second
    TimeoutWrite       = 10 * time.Second
    TimeoutTransaction = 30 * time.Second
    TimeoutBatch       = 2 * time.Minute
    TimeoutMigration   = 10 * time.Minute
)

// Apply at handler level, not repository level
ctx, cancel := context.WithTimeout(ctx, database.TimeoutWrite)
defer cancel()
```

---

## 6. External Service Error Handling

### 6.1 Payment Gateway Errors

```go
// services/payment/internal/infrastructure/gateway/midtrans.go
type GatewayError struct {
    Code       string
    Message    string
    Retryable  bool
    StatusCode int
}

func (e *GatewayError) Error() string {
    return fmt.Sprintf("gateway error %s: %s", e.Code, e.Message)
}

func (g *MidtransGateway) Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
    resp, err := g.client.Charge(ctx, req)
    if err != nil {
        // Network error — always retryable
        return nil, &GatewayError{
            Code:      "GATEWAY_UNREACHABLE",
            Message:   "Payment gateway is unreachable",
            Retryable: true,
        }
    }

    switch resp.StatusCode {
    case "200", "201":
        return mapToResult(resp), nil
    case "400":
        return nil, &GatewayError{Code: "INVALID_REQUEST", Message: resp.StatusMessage, Retryable: false}
    case "402":
        return nil, &GatewayError{Code: "PAYMENT_DECLINED", Message: "Card declined", Retryable: false}
    case "503":
        return nil, &GatewayError{Code: "GATEWAY_UNAVAILABLE", Message: resp.StatusMessage, Retryable: true}
    default:
        return nil, &GatewayError{Code: "GATEWAY_ERROR", Message: resp.StatusMessage, Retryable: false}
    }
}
```

### 6.2 Retry with Exponential Backoff

```go
// shared/go/pkg/retry/retry.go
type RetryConfig struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts:     3,
    InitialInterval: 100 * time.Millisecond,
    MaxInterval:     5 * time.Second,
    Multiplier:      2.0,
}

func Do(ctx context.Context, cfg RetryConfig, fn func() error) error {
    interval := cfg.InitialInterval
    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        // Check if error is retryable
        var gatewayErr *GatewayError
        if errors.As(err, &gatewayErr) && !gatewayErr.Retryable {
            return err // Non-retryable — fail fast
        }

        if attempt == cfg.MaxAttempts {
            return fmt.Errorf("max retry attempts (%d) exceeded: %w", cfg.MaxAttempts, err)
        }

        slog.WarnContext(ctx, "retrying after error",
            "attempt", attempt,
            "max_attempts", cfg.MaxAttempts,
            "retry_after", interval,
            "err", err,
        )

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(interval):
        }

        interval = min(time.Duration(float64(interval)*cfg.Multiplier), cfg.MaxInterval)
    }
    return nil
}
```

---

## 7. Frontend Error Handling

### 7.1 TypeScript — Global Error Boundary

```typescript
// apps/web/src/components/providers/ErrorBoundary.tsx
'use client';
import { Component, type ReactNode } from 'react';
import { ErrorResponse } from '@/lib/api/response';

interface Props { children: ReactNode; fallback?: ReactNode; }
interface State { error: Error | null; }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error) {
    // Report to observability
    console.error('[ErrorBoundary]', error);
  }

  render() {
    if (this.state.error) {
      return this.props.fallback ?? <DefaultErrorFallback />;
    }
    return this.props.children;
  }
}
```

### 7.2 TanStack Query — Global Error Handler

```typescript
// apps/web/src/lib/query/client.ts
import { QueryClient } from '@tanstack/react-query';
import { ApiError } from '@/lib/api/client';
import { toast } from 'sonner';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        // Don't retry on 4xx client errors
        if (error instanceof ApiError && error.httpStatus < 500) return false;
        return failureCount < 3;
      },
      staleTime: 30_000,
    },
    mutations: {
      onError: (error) => {
        if (error instanceof ApiError) {
          const { response } = error;
          // Show validation errors inline — don't toast
          if (response.error_code === 'VALIDATION_FAILED') return;
          // Show user-safe message for other errors
          toast.error(response.message, {
            description: response.error_code ? `Error: ${response.error_code}` : undefined,
            action: response.doc_url
              ? { label: 'Learn more', onClick: () => window.open(response.doc_url) }
              : undefined,
          });
        }
      },
    },
  },
});
```

### 7.3 Form Validation Error Mapping

```typescript
// apps/web/src/hooks/useFormErrors.ts
import { FieldError } from '@/lib/api/response';
import { UseFormSetError, FieldValues, Path } from 'react-hook-form';

// Maps API FieldError[] to React Hook Form field errors
export function applyServerErrors<T extends FieldValues>(
  errors: FieldError[],
  setError: UseFormSetError<T>,
) {
  for (const err of errors) {
    setError(err.field as Path<T>, {
      type: 'server',
      message: err.message,
    });
  }
}
```

---

## 8. Mobile Error Handling (Flutter)

### 8.1 ApiException

```dart
// apps/mobile/lib/core/api/exceptions.dart
import 'response.dart';

class ApiException implements Exception {
  const ApiException({
    required this.response,
    required this.httpStatus,
  });

  final ErrorResponse response;
  final int httpStatus;

  bool get isNetworkError => httpStatus == 0;
  bool get isClientError => httpStatus >= 400 && httpStatus < 500;
  bool get isServerError => httpStatus >= 500;
  bool get isRetryable => httpStatus == 503 || httpStatus == 504 || httpStatus == 429 || isNetworkError;

  @override
  String toString() => 'ApiException(${response.statusCode}: ${response.message})';
}
```

### 8.2 Riverpod AsyncValue Error Handling

```dart
// Consistent error UI across all Riverpod providers
class AsyncErrorWidget extends StatelessWidget {
  const AsyncErrorWidget({super.key, required this.error, required this.onRetry});

  final Object error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final message = switch (error) {
      ApiException e when e.response.errorCode == 'INSUFFICIENT_STOCK' =>
        'Not enough stock available.',
      ApiException e when e.isNetworkError =>
        'No internet connection.',
      ApiException e when e.isServerError =>
        'Server error. Please try again.',
      ApiException e => e.response.message,
      _ => 'Something went wrong.',
    };

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message),
          if (onRetry != null) ...[
            const SizedBox(height: 16),
            ElevatedButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ],
      ),
    );
  }
}
```

---

## 9. Logging Standards for Errors

```go
// ✅ Correct: structured, contextual, actionable
slog.ErrorContext(ctx, "payment processing failed",
    slog.String("error_code", "PAYMENT_GATEWAY_ERROR"),
    slog.String("order_id", order.ID.String()),
    slog.String("gateway", "midtrans"),
    slog.Int("attempt", attempt),
    slog.String("err", err.Error()),
)

// ❌ Wrong: unstructured, no context
log.Printf("payment failed: %v", err)

// Log levels for errors:
// slog.Warn  — recoverable: cache miss, retry succeeded, degraded mode
// slog.Error — needs attention: operation failed, will affect user
// slog.Error + PagerDuty alert — SLO breach: payment service down, DB unreachable

// Never log:
// - Password, PIN, card numbers
// - Full JWT tokens (log only the last 8 chars for correlation: token[len-8:])
// - PII beyond IDs (email, phone, address in body)
```

---

## 10. Error Handling Checklist

Before every PR, verify:

```
Infrastructure layer:
✅ sql.ErrNoRows (pgx.ErrNoRows) mapped to domain sentinel error
✅ pgconn.PgError unique violation mapped to domain sentinel error
✅ All errors wrapped with fmt.Errorf("repoName.MethodName: %w", err)
✅ Context deadline/cancellation errors handled separately

Application layer:
✅ Domain sentinel errors wrapped with oops.Code() and oops.Public()
✅ oops.With() includes relevant entity IDs for debugging
✅ No raw error strings exposed (use oops.Public for user-facing messages)

Interface layer:
✅ mapToGRPCError used — never return raw errors from handlers
✅ Panic recovery interceptor installed on all services
✅ No stack traces in HTTP/gRPC response bodies

General:
✅ Every error is either handled or returned — no ignored errors
✅ Errors logged at the correct level (warn/error)
✅ Sensitive data never in error messages or logs
```

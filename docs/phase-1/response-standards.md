# Response Standards — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Authoritative Standard | All REST endpoints MUST follow this.

---

## 1. Philosophy

### Why a Standard Response Envelope?

Without a standard, every endpoint returns a different shape. Frontend code becomes `if (res.data) ... else if (res.result) ... else if (res.order)...` — brittle and inconsistent. A standard envelope:

- **Predictable** — clients always know where to find data, errors, metadata
- **Traceable** — every response carries a `request_id` for distributed tracing
- **Typed** — Go generics + TypeScript generics + Dart generics: one pattern, three languages
- **Self-documenting** — `is_success: false` with `error_code: "ORDER_ALREADY_PAID"` is human-readable
- **Machine-readable** — `error_code` is a stable string clients can switch on, unlike `message` which can change

### Scope

This standard applies to **REST endpoints** exposed via gRPC-Gateway and Next.js BFF API routes.  
gRPC internal calls use the native Protobuf + `google.rpc.Status` pattern (see `api-contracts.md`).

---

## 2. Response Shape Taxonomy

### 2.1 Tier 1 — BaseResponse (All responses share this)

```
BaseResponse
├── request_id    string     — OpenTelemetry trace ID (or UUID if no trace)
├── status_code   string     — HTTP status code as string: "200", "201", "400"...
├── is_success    bool       — true when status_code is 2xx
├── message       string     — human-readable, safe to display to end users
└── timestamp     string     — ISO 8601 UTC: "2026-06-05T14:00:00.000Z"
```

### 2.2 Tier 2 — DataResponse[T] (Single entity result)

```
DataResponse[T] extends BaseResponse
└── data  T    — the entity or DTO
```

### 2.3 Tier 3 — ListResponse[T] (Paginated collection)

```
ListResponse[T] extends BaseResponse
├── data  []T
└── meta  PaginationMeta
         ├── page         int
         ├── page_size    int
         ├── total_count  int
         ├── total_pages  int
         ├── has_next     bool
         └── has_prev     bool
```

### 2.4 Tier 4 — ErrorResponse (Failure cases)

```
ErrorResponse extends BaseResponse
├── error_code    string            — stable machine-readable code (e.g. "ORDER_NOT_FOUND")
├── doc_url       string            — link to error catalog entry (nullable)
└── errors        []FieldError      — only for validation failures
                  ├── field    string   — dot-notation path: "items[0].quantity"
                  ├── message  string   — human-readable
                  └── code     string   — "REQUIRED", "MIN_VALUE", "INVALID_FORMAT"
```

---

## 3. Go Implementation

### 3.1 Core Types

```go
// shared/go/pkg/response/response.go
package response

import (
    "time"

    "github.com/google/uuid"
)

// HTTP status code constants — string type to match JSON output directly.
const (
    StatusOK                  = "200"
    StatusCreated             = "201"
    StatusNoContent           = "204"
    StatusBadRequest          = "400"
    StatusUnauthorized        = "401"
    StatusForbidden           = "403"
    StatusNotFound            = "404"
    StatusMethodNotAllowed    = "405"
    StatusConflict            = "409"
    StatusUnprocessableEntity = "422"
    StatusTooManyRequests     = "429"
    StatusInternalError       = "500"
    StatusServiceUnavailable  = "503"
    StatusGatewayTimeout      = "504"
)

// FieldError represents a single validation violation.
type FieldError struct {
    Field   string `json:"field"`             // dot-notation: "items[0].quantity"
    Message string `json:"message"`           // human-readable, safe to display
    Code    string `json:"code"`              // machine-readable: "REQUIRED", "MIN_VALUE"
}

// PaginationMeta holds pagination context for list responses.
type PaginationMeta struct {
    Page       int  `json:"page"`
    PageSize   int  `json:"page_size"`
    TotalCount int  `json:"total_count"`
    TotalPages int  `json:"total_pages"`
    HasNext    bool `json:"has_next"`
    HasPrev    bool `json:"has_prev"`
}

// baseFields are the common fields included in every response.
// Unexported — consumers use the typed response structs below.
type baseFields struct {
    RequestID  string `json:"request_id"`
    StatusCode string `json:"status_code"`
    IsSuccess  bool   `json:"is_success"`
    Message    string `json:"message"`
    Timestamp  string `json:"timestamp"`
}

func newBase(requestID, statusCode, message string) baseFields {
    if requestID == "" {
        requestID = uuid.NewString()
    }
    code2xx := statusCode[0] == '2'
    return baseFields{
        RequestID:  requestID,
        StatusCode: statusCode,
        IsSuccess:  code2xx,
        Message:    message,
        Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
    }
}

// BaseResponse is returned for operations with no data payload (delete, void, logout).
type BaseResponse struct {
    baseFields
}

// DataResponse[T] wraps a single entity result.
type DataResponse[T any] struct {
    baseFields
    Data T `json:"data"`
}

// ListResponse[T] wraps a paginated collection.
type ListResponse[T any] struct {
    baseFields
    Data []T            `json:"data"`
    Meta PaginationMeta `json:"meta"`
}

// ErrorResponse wraps error details for failure cases.
type ErrorResponse struct {
    baseFields
    ErrorCode string       `json:"error_code,omitempty"`
    DocURL    string       `json:"doc_url,omitempty"`
    Errors    []FieldError `json:"errors,omitempty"`
}
```

### 3.2 Constructor Functions

```go
// shared/go/pkg/response/constructors.go
package response

// ── Success Constructors ──────────────────────────────────────────────────────

// OK returns a 200 response with no data payload.
func OK(requestID, message string) BaseResponse {
    return BaseResponse{baseFields: newBase(requestID, StatusOK, message)}
}

// Created returns a 201 response with the created entity.
func Created[T any](requestID, message string, data T) DataResponse[T] {
    return DataResponse[T]{
        baseFields: newBase(requestID, StatusCreated, message),
        Data:       data,
    }
}

// Data returns a 200 response with a single entity.
func Data[T any](requestID, message string, data T) DataResponse[T] {
    return DataResponse[T]{
        baseFields: newBase(requestID, StatusOK, message),
        Data:       data,
    }
}

// List returns a 200 paginated list response.
func List[T any](requestID, message string, data []T, meta PaginationMeta) ListResponse[T] {
    if data == nil {
        data = []T{} // never return null for arrays
    }
    return ListResponse[T]{
        baseFields: newBase(requestID, StatusOK, message),
        Data:       data,
        Meta:       meta,
    }
}

// NoContent returns a 204 response (typically for DELETE).
func NoContent(requestID, message string) BaseResponse {
    return BaseResponse{baseFields: newBase(requestID, StatusNoContent, message)}
}

// ── Error Constructors ────────────────────────────────────────────────────────

// BadRequest returns a 400 error response.
func BadRequest(requestID, message, errorCode string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusBadRequest, message),
        ErrorCode:  errorCode,
        DocURL:     errorDocURL(errorCode),
    }
}

// ValidationFailed returns a 400 response with field-level errors.
func ValidationFailed(requestID string, errs []FieldError) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusBadRequest, "Validation failed"),
        ErrorCode:  "VALIDATION_FAILED",
        Errors:     errs,
    }
}

// Unauthorized returns a 401 error response.
func Unauthorized(requestID, message string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusUnauthorized, message),
        ErrorCode:  "UNAUTHORIZED",
    }
}

// Forbidden returns a 403 error response.
func Forbidden(requestID, message string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusForbidden, message),
        ErrorCode:  "FORBIDDEN",
    }
}

// NotFound returns a 404 error response.
func NotFound(requestID, message, errorCode string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusNotFound, message),
        ErrorCode:  errorCode,
        DocURL:     errorDocURL(errorCode),
    }
}

// Conflict returns a 409 error response (duplicate resource, state conflict).
func Conflict(requestID, message, errorCode string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusConflict, message),
        ErrorCode:  errorCode,
        DocURL:     errorDocURL(errorCode),
    }
}

// UnprocessableEntity returns a 422 for business rule violations.
func UnprocessableEntity(requestID, message, errorCode string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusUnprocessableEntity, message),
        ErrorCode:  errorCode,
        DocURL:     errorDocURL(errorCode),
    }
}

// TooManyRequests returns a 429 rate limit response.
func TooManyRequests(requestID string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusTooManyRequests, "Rate limit exceeded. Please slow down."),
        ErrorCode:  "RATE_LIMIT_EXCEEDED",
    }
}

// InternalError returns a 500 response. Never expose internal error details.
func InternalError(requestID string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusInternalError, "An unexpected error occurred. Our team has been notified."),
        ErrorCode:  "INTERNAL_ERROR",
    }
}

// ServiceUnavailable returns a 503 response.
func ServiceUnavailable(requestID string) ErrorResponse {
    return ErrorResponse{
        baseFields: newBase(requestID, StatusServiceUnavailable, "Service temporarily unavailable. Please retry."),
        ErrorCode:  "SERVICE_UNAVAILABLE",
    }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// NewPaginationMeta constructs PaginationMeta from page params and total count.
func NewPaginationMeta(page, pageSize, totalCount int) PaginationMeta {
    if pageSize <= 0 {
        pageSize = 20
    }
    totalPages := (totalCount + pageSize - 1) / pageSize
    return PaginationMeta{
        Page:       page,
        PageSize:   pageSize,
        TotalCount: totalCount,
        TotalPages: totalPages,
        HasNext:    page < totalPages,
        HasPrev:    page > 1,
    }
}

// errorDocURL returns the documentation URL for a given error code.
// Returns empty string if no doc exists (won't appear in JSON due to omitempty).
func errorDocURL(code string) string {
    base := "https://docs.xyn.app/errors"
    if code == "" {
        return ""
    }
    return base + "#" + code
}
```

### 3.3 HTTP Handler Integration

```go
// shared/go/pkg/response/http.go
package response

import (
    "encoding/json"
    "net/http"
    "strconv"
)

// Write serializes the response and writes it to the http.ResponseWriter.
// It sets Content-Type: application/json and the appropriate HTTP status code.
func Write(w http.ResponseWriter, r *http.Request, statusCode int, body any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("X-Request-ID", requestIDFromContext(r.Context()))
    w.WriteHeader(statusCode)
    _ = json.NewEncoder(w).Encode(body)
}

// WriteOK is a convenience function for 200 responses.
func WriteOK(w http.ResponseWriter, r *http.Request, body any) {
    Write(w, r, http.StatusOK, body)
}

// WriteCreated is a convenience function for 201 responses.
func WriteCreated(w http.ResponseWriter, r *http.Request, body any) {
    Write(w, r, http.StatusCreated, body)
}

// WriteError maps an ErrorResponse to its correct HTTP status code and writes it.
func WriteError(w http.ResponseWriter, r *http.Request, err ErrorResponse) {
    code, _ := strconv.Atoi(err.StatusCode)
    if code == 0 {
        code = http.StatusInternalServerError
    }
    Write(w, r, code, err)
}
```

### 3.4 gRPC-Gateway Error Mapper

```go
// shared/go/pkg/response/grpc_gateway.go
package response

import (
    "context"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// GatewayErrorHandler replaces the default gRPC-Gateway error handler
// so all errors use our standard ErrorResponse format.
func GatewayErrorHandler(ctx context.Context, mux *runtime.ServeMux,
    marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {

    requestID := requestIDFromContext(ctx)
    st := status.Convert(err)

    httpCode := grpcCodeToHTTP(st.Code())
    errCode := grpcCodeToErrorCode(st.Code())

    errResp := ErrorResponse{
        baseFields: newBase(requestID, intToStatusString(httpCode), st.Message()),
        ErrorCode:  errCode,
        DocURL:     errorDocURL(errCode),
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(httpCode)
    _ = marshaler.NewEncoder(w).Encode(errResp)
}

func grpcCodeToHTTP(code codes.Code) int {
    m := map[codes.Code]int{
        codes.OK:                 200,
        codes.InvalidArgument:    400,
        codes.Unauthenticated:    401,
        codes.PermissionDenied:   403,
        codes.NotFound:           404,
        codes.AlreadyExists:      409,
        codes.FailedPrecondition: 422,
        codes.ResourceExhausted:  429,
        codes.Internal:           500,
        codes.Unavailable:        503,
        codes.DeadlineExceeded:   504,
    }
    if code, ok := m[code]; ok {
        return code
    }
    return 500
}

func grpcCodeToErrorCode(code codes.Code) string {
    m := map[codes.Code]string{
        codes.InvalidArgument:    "INVALID_ARGUMENT",
        codes.Unauthenticated:    "UNAUTHORIZED",
        codes.PermissionDenied:   "FORBIDDEN",
        codes.NotFound:           "NOT_FOUND",
        codes.AlreadyExists:      "ALREADY_EXISTS",
        codes.FailedPrecondition: "PRECONDITION_FAILED",
        codes.ResourceExhausted:  "RATE_LIMIT_EXCEEDED",
        codes.Internal:           "INTERNAL_ERROR",
        codes.Unavailable:        "SERVICE_UNAVAILABLE",
    }
    if c, ok := m[code]; ok {
        return c
    }
    return "UNKNOWN_ERROR"
}
```

### 3.5 Request ID Extraction

```go
// shared/go/pkg/response/context.go
package response

import (
    "context"

    "go.opentelemetry.io/otel/trace"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// requestIDFromContext extracts the trace ID from the OpenTelemetry span.
// Falls back to a header-injected request ID, then to a generated UUID.
func requestIDFromContext(ctx context.Context) string {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        return span.SpanContext().TraceID().String()
    }
    if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
        return id
    }
    return "no-trace"
}
```

---

## 4. TypeScript Implementation (Next.js / BFF)

```typescript
// apps/web/src/lib/api/response.ts

export interface PaginationMeta {
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface FieldError {
  field: string;    // dot-notation: "items[0].quantity"
  message: string;
  code: string;     // "REQUIRED" | "MIN_VALUE" | "INVALID_FORMAT"
}

interface BaseFields {
  request_id: string;
  status_code: string;
  is_success: boolean;
  message: string;
  timestamp: string;
}

export type BaseResponse = BaseFields;

export interface DataResponse<T> extends BaseFields {
  data: T;
}

export interface ListResponse<T> extends BaseFields {
  data: T[];
  meta: PaginationMeta;
}

export interface ErrorResponse extends BaseFields {
  error_code?: string;
  doc_url?: string;
  errors?: FieldError[];
}

export type ApiResponse<T> = DataResponse<T> | ListResponse<T> | BaseResponse | ErrorResponse;

// Type guards
export function isErrorResponse(res: unknown): res is ErrorResponse {
  return typeof res === 'object' && res !== null && 'is_success' in res && !(res as BaseFields).is_success;
}

export function isDataResponse<T>(res: ApiResponse<T>): res is DataResponse<T> {
  return 'data' in res && !Array.isArray((res as DataResponse<T>).data);
}

export function isListResponse<T>(res: ApiResponse<T>): res is ListResponse<T> {
  return 'data' in res && Array.isArray((res as ListResponse<T>).data) && 'meta' in res;
}
```

```typescript
// apps/web/src/lib/api/client.ts
// Centralized fetch wrapper that handles the response envelope

import { ApiResponse, ErrorResponse, isErrorResponse } from './response';

export class ApiError extends Error {
  constructor(
    public readonly response: ErrorResponse,
    public readonly httpStatus: number,
  ) {
    super(response.message);
    this.name = 'ApiError';
  }
}

export async function apiFetch<T>(
  url: string,
  options?: RequestInit,
): Promise<ApiResponse<T>> {
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  const body = await res.json() as ApiResponse<T>;

  if (isErrorResponse(body)) {
    throw new ApiError(body, res.status);
  }

  return body;
}

// TanStack Query integration
export function queryFn<T>(url: string) {
  return async () => {
    const res = await apiFetch<T>(url);
    if (isDataResponse(res)) return res.data;
    if (isListResponse(res)) return res; // return full ListResponse for pagination
    return null;
  };
}
```

---

## 5. Dart / Flutter Implementation

```dart
// apps/mobile/lib/core/api/response.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'response.freezed.dart';
part 'response.g.dart';

@freezed
class PaginationMeta with _$PaginationMeta {
  const factory PaginationMeta({
    required int page,
    @JsonKey(name: 'page_size') required int pageSize,
    @JsonKey(name: 'total_count') required int totalCount,
    @JsonKey(name: 'total_pages') required int totalPages,
    @JsonKey(name: 'has_next') required bool hasNext,
    @JsonKey(name: 'has_prev') required bool hasPrev,
  }) = _PaginationMeta;

  factory PaginationMeta.fromJson(Map<String, dynamic> json) =>
      _$PaginationMetaFromJson(json);
}

@freezed
class FieldError with _$FieldError {
  const factory FieldError({
    required String field,
    required String message,
    required String code,
  }) = _FieldError;

  factory FieldError.fromJson(Map<String, dynamic> json) =>
      _$FieldErrorFromJson(json);
}

// Generic DataResponse — Dart 3.x generics
@Freezed(genericArgumentFactories: true)
class DataResponse<T> with _$DataResponse<T> {
  const factory DataResponse({
    @JsonKey(name: 'request_id') required String requestId,
    @JsonKey(name: 'status_code') required String statusCode,
    @JsonKey(name: 'is_success') required bool isSuccess,
    required String message,
    required String timestamp,
    required T data,
  }) = _DataResponse<T>;

  factory DataResponse.fromJson(
    Map<String, dynamic> json,
    T Function(Object?) fromJsonT,
  ) => _$DataResponseFromJson(json, fromJsonT);
}

@Freezed(genericArgumentFactories: true)
class ListResponse<T> with _$ListResponse<T> {
  const factory ListResponse({
    @JsonKey(name: 'request_id') required String requestId,
    @JsonKey(name: 'status_code') required String statusCode,
    @JsonKey(name: 'is_success') required bool isSuccess,
    required String message,
    required String timestamp,
    required List<T> data,
    required PaginationMeta meta,
  }) = _ListResponse<T>;

  factory ListResponse.fromJson(
    Map<String, dynamic> json,
    T Function(Object?) fromJsonT,
  ) => _$ListResponseFromJson(json, fromJsonT);
}

@freezed
class ErrorResponse with _$ErrorResponse {
  const factory ErrorResponse({
    @JsonKey(name: 'request_id') required String requestId,
    @JsonKey(name: 'status_code') required String statusCode,
    @JsonKey(name: 'is_success') required bool isSuccess,
    required String message,
    required String timestamp,
    @JsonKey(name: 'error_code') String? errorCode,
    @JsonKey(name: 'doc_url') String? docUrl,
    List<FieldError>? errors,
  }) = _ErrorResponse;

  factory ErrorResponse.fromJson(Map<String, dynamic> json) =>
      _$ErrorResponseFromJson(json);
}
```

---

## 6. Swagger / OpenAPI Annotations (Go)

Use `swaggo/swag` annotations on gRPC-Gateway HTTP handlers.

```go
// services/pos/internal/interfaces/http/order_handler.go

// CreateOrder creates a new POS order.
//
// @Summary      Create a new order
// @Description  Creates a new draft order for the authenticated cashier's shift.
//               The order starts in DRAFT status and can be modified before checkout.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Idempotency-Key  header    string                true  "UUID v4 idempotency key"
// @Param        request            body      CreateOrderRequest    true  "Order creation payload"
// @Success      201                {object}  response.DataResponse[OrderDTO]
// @Failure      400                {object}  response.ErrorResponse  "Validation failed"
// @Failure      401                {object}  response.ErrorResponse  "Not authenticated"
// @Failure      403                {object}  response.ErrorResponse  "Insufficient permissions"
// @Failure      409                {object}  response.ErrorResponse  "Duplicate idempotency key"
// @Failure      422                {object}  response.ErrorResponse  "Business rule violation"
// @Failure      429                {object}  response.ErrorResponse  "Rate limit exceeded"
// @Failure      500                {object}  response.ErrorResponse  "Internal server error"
// @Router       /v1/orders [post]
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

### Swagger Main Config

```go
// services/pos/cmd/server/docs.go

// @title           xyn-pos API — POS Service
// @version         1.0
// @description     POS Core & Cart bounded context. Handles order lifecycle from draft to paid.
// @termsOfService  https://xyn.app/terms

// @contact.name    xyn Engineering
// @contact.email   engineering@xyn.app

// @license.name    Proprietary
// @license.url     https://xyn.app/license

// @host            api.xyn.app
// @BasePath        /

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 PASETO v4 token. Format: "Bearer <token>"

// @tag.name         Orders
// @tag.description  Order lifecycle management (create, modify, checkout)

// @tag.name         Shifts
// @tag.description  Cashier shift management (open, close, cash count)
```

### Makefile Target

```makefile
# Generate Swagger docs for a service
swagger-gen:
    swag init \
        --dir ./services/pos/cmd/server,./services/pos/internal/interfaces/http \
        --output ./services/pos/docs \
        --generalInfo docs.go \
        --parseDependency \
        --parseInternal

# Serve Swagger UI locally
swagger-ui:
    docker run -p 8080:8080 \
        -e SWAGGER_JSON=/docs/swagger.json \
        -v $(PWD)/services/pos/docs:/docs \
        swaggerapi/swagger-ui
```

---

## 7. Error Code Catalog

All `error_code` values used across the platform. Stable — never rename a code that's been released.

### 7.1 Authentication & Authorization
| Code | HTTP | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Token missing or invalid |
| `TOKEN_EXPIRED` | 401 | PASETO token has expired |
| `FORBIDDEN` | 403 | Authenticated but lacks permission |
| `INSUFFICIENT_ROLE` | 403 | Role too low for this operation |
| `PIN_REQUIRED` | 403 | Sensitive operation requires PIN verification |
| `PIN_INVALID` | 403 | Wrong PIN entered |
| `TENANT_SUSPENDED` | 403 | Tenant account suspended |

### 7.2 Validation
| Code | HTTP | Description |
|---|---|---|
| `VALIDATION_FAILED` | 400 | One or more fields failed validation (see `errors[]`) |
| `INVALID_UUID` | 400 | Field must be a valid UUID v4 |
| `INVALID_CURRENCY` | 400 | Currency code must be ISO 4217 |
| `INVALID_DATE_RANGE` | 400 | Start date must be before end date |
| `IDEMPOTENCY_KEY_MISSING` | 400 | Financial operation requires X-Idempotency-Key header |

### 7.3 Orders
| Code | HTTP | Description |
|---|---|---|
| `ORDER_NOT_FOUND` | 404 | Order does not exist or belongs to another tenant |
| `ORDER_ALREADY_PAID` | 409 | Cannot modify a paid order |
| `ORDER_CANCELLED` | 409 | Cannot modify a cancelled order |
| `ORDER_EMPTY` | 422 | Cannot checkout an empty order |
| `ORDER_INVALID_STATE` | 422 | State transition not allowed |

### 7.4 Payment
| Code | HTTP | Description |
|---|---|---|
| `PAYMENT_NOT_FOUND` | 404 | Payment record not found |
| `PAYMENT_ALREADY_PROCESSED` | 409 | Idempotency key already used for a completed payment |
| `PAYMENT_AMOUNT_MISMATCH` | 422 | Total payments do not equal order total |
| `PAYMENT_GATEWAY_ERROR` | 502 | Upstream payment gateway returned an error |
| `INSUFFICIENT_CASH_TENDERED` | 422 | Cash amount less than order total |
| `VOID_WINDOW_EXPIRED` | 422 | Void window (e.g., same-day) has passed |
| `REFUND_EXCEEDS_ORIGINAL` | 422 | Refund amount exceeds original payment |

### 7.5 Inventory
| Code | HTTP | Description |
|---|---|---|
| `PRODUCT_NOT_FOUND` | 404 | Product does not exist |
| `SKU_ALREADY_EXISTS` | 409 | SKU is already in use by another product |
| `INSUFFICIENT_STOCK` | 422 | Not enough stock to fulfill the request |
| `WAREHOUSE_NOT_FOUND` | 404 | Warehouse does not exist |
| `NEGATIVE_STOCK_NOT_ALLOWED` | 422 | Stock would go below zero (if not backorder-enabled) |

### 7.6 System
| Code | HTTP | Description |
|---|---|---|
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | Service is temporarily down |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `RESOURCE_EXHAUSTED` | 429 | Plan quota exceeded (e.g., max branches) |
| `GATEWAY_TIMEOUT` | 504 | Upstream service did not respond in time |

---

## 8. Usage Examples

### 8.1 GET single resource

```go
// Handler
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    requestID := requestIDFromContext(r.Context())
    orderID := chi.URLParam(r, "order_id")

    order, err := h.svc.GetOrder(r.Context(), orderID)
    if err != nil {
        if errors.Is(err, domain.ErrOrderNotFound) {
            response.WriteError(w, r, response.NotFound(requestID, "Order not found", "ORDER_NOT_FOUND"))
            return
        }
        slog.ErrorContext(r.Context(), "get order failed", "err", err)
        response.WriteError(w, r, response.InternalError(requestID))
        return
    }

    response.WriteOK(w, r, response.Data(requestID, "Order retrieved", toOrderDTO(order)))
}
```

**Response:**
```json
{
  "request_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "status_code": "200",
  "is_success": true,
  "message": "Order retrieved",
  "timestamp": "2026-06-05T14:00:00.000Z",
  "data": {
    "order_id": "018e1234-5678-7abc-def0-123456789abc",
    "status": "pending",
    "total": 22420
  }
}
```

### 8.2 Validation error

```json
{
  "request_id": "abc123",
  "status_code": "400",
  "is_success": false,
  "message": "Validation failed",
  "timestamp": "2026-06-05T14:00:01.000Z",
  "error_code": "VALIDATION_FAILED",
  "errors": [
    { "field": "items[0].quantity", "message": "must be between 1 and 999", "code": "RANGE_VIOLATION" },
    { "field": "branch_id", "message": "must be a valid UUID", "code": "INVALID_UUID" }
  ]
}
```

### 8.3 Business rule error

```json
{
  "request_id": "abc124",
  "status_code": "422",
  "is_success": false,
  "message": "Cannot checkout: insufficient stock for 'Burger' (requested: 3, available: 1)",
  "timestamp": "2026-06-05T14:00:02.000Z",
  "error_code": "INSUFFICIENT_STOCK",
  "doc_url": "https://docs.xyn.app/errors#INSUFFICIENT_STOCK"
}
```

### 8.4 Paginated list

```json
{
  "request_id": "abc125",
  "status_code": "200",
  "is_success": true,
  "message": "Orders retrieved",
  "timestamp": "2026-06-05T14:00:03.000Z",
  "data": [ { "order_id": "..." }, { "order_id": "..." } ],
  "meta": {
    "page": 2,
    "page_size": 20,
    "total_count": 142,
    "total_pages": 8,
    "has_next": true,
    "has_prev": true
  }
}
```

---

## 9. Rules Summary

```
✅ Every REST endpoint MUST use BaseResponse/DataResponse/ListResponse/ErrorResponse
✅ Never return naked JSON objects without the envelope
✅ Always include request_id (from OTEL trace or generated UUID)
✅ Arrays in data field are NEVER null — return [] for empty collections
✅ Money amounts in responses are int64 minor units (cents/sen)
✅ error_code MUST be a stable string constant — never interpolate dynamic values
✅ message is human-readable and may change — clients must NOT switch on message
✅ HTTP status code in body MUST match the actual HTTP status header
✅ Validation errors go in errors[] — never concatenated into message
✅ Never expose internal error details (stack traces, SQL errors) in responses
✅ Every new error_code MUST be added to the catalog in this document
```

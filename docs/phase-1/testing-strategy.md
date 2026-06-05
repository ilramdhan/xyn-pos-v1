# Testing Strategy — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Authoritative Standard  
> Tools: testify 1.11.1 | testcontainers-go 0.42.0 | k6 1.7.1 | golangci-lint 2.12.2

---

## 1. Testing Philosophy

### The Testing Pyramid

```
        ╱╲
       ╱E2E╲          ~5% — Slowest, most expensive
      ╱──────╲         Full user journeys via gRPC + real services
     ╱Integr. ╲
    ╱────────────╲     ~25% — Medium speed
   ╱              ╲    Real DB (testcontainers), real Kafka
  ╱   Unit Tests   ╲
 ╱──────────────────╲  ~70% — Fastest
╱   (Domain logic)  ╲  No I/O, pure functions, < 10ms each
```

**Rule:** Push tests down the pyramid. Every behavior testable at the unit level should be. Integration tests only for things that require real infrastructure. E2E only for critical user journeys.

### Test Behavior, Not Implementation

```go
// ✅ Tests WHAT the system does (behavior)
func TestOrder_Cancel_WhenPending_ShouldTransitionToCancelled(t *testing.T) {
    order := NewTestOrder(WithStatus(StatusPending))
    err := order.Cancel("customer request")
    assert.NoError(t, err)
    assert.Equal(t, StatusCancelled, order.Status)
    assert.NotEmpty(t, order.PopEvents()) // domain event was emitted
}

// ❌ Tests HOW it does it (implementation)
mockRepo.AssertNumberOfCalls(t, "Save", 1)
mockPublisher.AssertCalledWith(t, "OrderCancelled", ...)
```

---

## 2. Unit Tests

### 2.1 Domain Layer Tests (No Mocks)

Domain aggregates are pure functions — no I/O, no dependencies. Test them without mocks.

```go
// services/pos/internal/domain/order/order_test.go
package order_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
    "github.com/xyn/pos-v1/services/pos/internal/domain/order"
)

// Use testify suite for related test groups
type OrderSuite struct {
    suite.Suite
}

func TestOrderSuite(t *testing.T) {
    suite.Run(t, new(OrderSuite))
}

// Table-driven test for state machine transitions
func (s *OrderSuite) TestCancel_StateTransitions() {
    tests := []struct {
        name        string
        fromStatus  order.Status
        expectError error
    }{
        {"from draft",    order.StatusDraft,    nil},
        {"from pending",  order.StatusPending,  nil},
        {"from paid",     order.StatusPaid,     order.ErrOrderAlreadyPaid},
        {"from cancelled",order.StatusCancelled,order.ErrOrderCancelled},
        {"from void",     order.StatusVoided,   order.ErrOrderVoided},
    }

    for _, tc := range tests {
        s.Run(tc.name, func() {
            o := order.NewTestOrder(order.WithStatus(tc.fromStatus))
            err := o.Cancel("test reason")
            if tc.expectError != nil {
                s.ErrorIs(err, tc.expectError)
            } else {
                s.NoError(err)
                s.Equal(order.StatusCancelled, o.Status)
            }
        })
    }
}

func (s *OrderSuite) TestAddItem_ShouldRecalculateTotals() {
    o := order.NewTestOrder()
    item := order.NewTestItem(order.WithUnitPrice(decimal.New(5000, 0)), order.WithQuantity(3))

    err := o.AddItem(item)

    s.Require().NoError(err)
    s.Equal(decimal.New(15000, 0), o.Subtotal)
    s.Equal(1, len(o.Items))
}

func (s *OrderSuite) TestCheckout_EmptyOrder_ShouldFail() {
    o := order.NewTestOrder() // no items
    err := o.PrepareForCheckout()
    s.ErrorIs(err, order.ErrOrderEmpty)
}
```

### 2.2 Test Fixtures / Builders

```go
// services/pos/internal/domain/order/testdata.go
// +build test  ← only compiled during tests
package order

import "github.com/google/uuid"

type TestOrderOption func(*Order)

func WithStatus(s Status) TestOrderOption {
    return func(o *Order) { o.Status = s }
}

func WithTenantID(id uuid.UUID) TestOrderOption {
    return func(o *Order) { o.TenantID = id }
}

func NewTestOrder(opts ...TestOrderOption) *Order {
    o := &Order{
        ID:       uuid.New(),
        TenantID: uuid.New(),
        BranchID: uuid.New(),
        Status:   StatusDraft,
        Items:    []OrderItem{},
    }
    for _, opt := range opts {
        opt(o)
    }
    return o
}
```

### 2.3 Application Layer Tests (With Mocks)

Generate mocks with `mockery v2`:

```bash
# Generate mock for OrderRepository interface
mockery --name=OrderRepository \
    --dir=services/pos/internal/domain/order \
    --output=services/pos/internal/mocks \
    --filename=mock_order_repository.go \
    --with-expecter  # enables type-safe .EXPECT() API
```

```go
// services/pos/internal/application/command/cancel_order_test.go
package command_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/xyn/pos-v1/services/pos/internal/mocks"
    "github.com/xyn/pos-v1/services/pos/internal/domain/order"
    "github.com/xyn/pos-v1/services/pos/internal/application/command"
)

type CancelOrderSuite struct {
    suite.Suite
    repo    *mocks.MockOrderRepository
    handler *command.CancelOrderHandler
}

func (s *CancelOrderSuite) SetupTest() {
    s.repo = mocks.NewMockOrderRepository(s.T())
    s.handler = command.NewCancelOrderHandler(s.repo)
}

func TestCancelOrderSuite(t *testing.T) {
    suite.Run(t, new(CancelOrderSuite))
}

func (s *CancelOrderSuite) TestHandle_OrderNotFound_ShouldReturnError() {
    cmd := command.CancelOrderCommand{OrderID: uuid.New()}

    // Type-safe mock expectation via .EXPECT()
    s.repo.EXPECT().FindByID(mock.Anything, cmd.OrderID).
        Return(nil, order.ErrOrderNotFound).Once()

    err := s.handler.Handle(context.Background(), cmd)

    s.Error(err)
    var oopsErr oops.OopsError
    s.True(errors.As(err, &oopsErr))
    s.Equal("ORDER_NOT_FOUND", oopsErr.Code())
}

func (s *CancelOrderSuite) TestHandle_AlreadyPaid_ShouldReturnDomainError() {
    existingOrder := order.NewTestOrder(order.WithStatus(order.StatusPaid))
    cmd := command.CancelOrderCommand{OrderID: existingOrder.ID}

    s.repo.EXPECT().FindByID(mock.Anything, cmd.OrderID).
        Return(existingOrder, nil).Once()

    err := s.handler.Handle(context.Background(), cmd)

    s.Error(err)
    s.True(errors.Is(err, order.ErrOrderAlreadyPaid))
}
```

### 2.4 Coverage Configuration

```yaml
# .testcoverage.yml
threshold:
  file:    70   # No single file below 70%
  package: 75   # No package below 75%
  total:   80   # Overall project minimum

exclude:
  paths:
    - ".*/mock_.*\\.go"    # Generated mocks
    - ".*/wire_gen\\.go"   # Generated DI
    - ".*/pb/.*\\.go"      # Generated proto
    - ".*/testdata/.*"
    - "cmd/.*"             # Main entrypoints (tested via e2e)
```

---

## 3. Integration Tests

Integration tests use real databases via `testcontainers-go 0.42.0`.

### 3.1 Test Suite Setup

```go
// services/pos/internal/infrastructure/postgres/suite_test.go
package postgres_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

type PostgresSuite struct {
    suite.Suite
    container *postgres.PostgresContainer
    pool      *pgxpool.Pool
}

func (s *PostgresSuite) SetupSuite() {
    ctx := context.Background()

    container, err := postgres.Run(ctx,
        "postgres:18.4-alpine",
        postgres.WithDatabase("xyn_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    s.Require().NoError(err)
    s.container = container

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    s.Require().NoError(err)

    // Run migrations
    db, err := sql.Open("pgx", connStr)
    s.Require().NoError(err)
    s.Require().NoError(goose.Up(db, "../../migrations"))

    s.pool, err = pgxpool.New(ctx, connStr)
    s.Require().NoError(err)
}

func (s *PostgresSuite) TearDownSuite() {
    s.pool.Close()
    _ = s.container.Terminate(context.Background())
}

func (s *PostgresSuite) SetupTest() {
    // Truncate all tables between tests for isolation
    _, err := s.pool.Exec(context.Background(),
        "TRUNCATE orders, order_items, payments RESTART IDENTITY CASCADE")
    s.Require().NoError(err)
}

func TestPostgresSuite(t *testing.T) {
    suite.Run(t, new(PostgresSuite))
}
```

### 3.2 Repository Integration Test

```go
// services/pos/internal/infrastructure/postgres/order_repo_test.go
func (s *PostgresSuite) TestFindByID_Existing_ShouldReturnOrder() {
    ctx := context.Background()
    repo := postgres.NewOrderRepository(s.pool)

    // Arrange — insert test data
    original := order.NewTestOrder(
        order.WithTenantID(testTenantID),
        order.WithStatus(order.StatusPending),
    )
    s.Require().NoError(repo.Save(ctx, original))

    // Set RLS context
    _, err := s.pool.Exec(ctx, "SET app.current_tenant_id = $1", testTenantID.String())
    s.Require().NoError(err)

    // Act
    found, err := repo.FindByID(ctx, original.ID)

    // Assert
    s.Require().NoError(err)
    s.Equal(original.ID, found.ID)
    s.Equal(order.StatusPending, found.Status)
}

func (s *PostgresSuite) TestFindByID_DifferentTenant_ShouldReturnNotFound() {
    ctx := context.Background()
    repo := postgres.NewOrderRepository(s.pool)

    // Insert order for tenant A
    original := order.NewTestOrder(order.WithTenantID(tenantA))
    s.Require().NoError(repo.Save(ctx, original))

    // Query as tenant B — RLS should block access
    _, err := s.pool.Exec(ctx, "SET app.current_tenant_id = $1", tenantB.String())
    s.Require().NoError(err)

    _, err = repo.FindByID(ctx, original.ID)
    s.ErrorIs(err, domain.ErrOrderNotFound) // RLS makes it look like 404, not 403
}
```

### 3.3 Kafka Integration Test

```go
// shared/go/pkg/events/kafka_test.go
func (s *KafkaSuite) TestPublish_ShouldBeConsumedBySubscriber() {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    publisher := kafka.NewPublisher(s.brokers)
    consumer := kafka.NewConsumer(s.brokers, "test-group")

    topic := "test.orders.created"
    payload := []byte(`{"order_id":"test-123"}`)

    received := make(chan []byte, 1)
    go consumer.Subscribe(ctx, topic, func(msg []byte) {
        received <- msg
    })

    time.Sleep(500 * time.Millisecond) // Allow consumer to connect
    s.Require().NoError(publisher.Publish(ctx, topic, payload))

    select {
    case msg := <-received:
        s.Equal(payload, msg)
    case <-ctx.Done():
        s.Fail("message not received within timeout")
    }
}
```

---

## 4. E2E Tests

E2E tests exercise full service stacks via gRPC clients.

```go
// tests/e2e/pos/checkout_flow_test.go
//go:build e2e

package pos_e2e_test

func TestCheckoutFlow_CashPayment_ShouldCompleteSuccessfully(t *testing.T) {
    ctx := context.Background()
    env := e2e.SetupTestEnvironment(t) // starts all services via docker-compose

    // Step 1: Create order
    orderResp, err := env.POSClient.CreateOrder(ctx, &pb.CreateOrderRequest{
        IdempotencyKey: uuid.NewString(),
        BranchId:       env.TestBranchID,
        OrderType:      pb.OrderType_ORDER_TYPE_DINE_IN,
        TableNumber:    "T-01",
    })
    require.NoError(t, err)
    orderID := orderResp.OrderId

    // Step 2: Add items
    _, err = env.POSClient.AddItem(ctx, &pb.AddItemRequest{
        OrderId:   orderID,
        ProductId: env.TestProductID,
        Quantity:  2,
    })
    require.NoError(t, err)

    // Step 3: Process payment
    paymentResp, err := env.PaymentClient.ProcessPayment(ctx, &pb.ProcessPaymentRequest{
        IdempotencyKey:  uuid.NewString(),
        OrderId:         orderID,
        Method:          pb.PaymentMethod_PAYMENT_METHOD_CASH,
        AmountTendered:  &pb.Money{CurrencyCode: "IDR", AmountMinor: 50000},
    })
    require.NoError(t, err)
    assert.Equal(t, pb.PaymentStatus_PAYMENT_STATUS_COMPLETED, paymentResp.Status)

    // Step 4: Verify order is paid
    order, err := env.POSClient.GetOrder(ctx, &pb.GetOrderRequest{OrderId: orderID})
    require.NoError(t, err)
    assert.Equal(t, pb.OrderStatus_ORDER_STATUS_PAID, order.Order.Status)

    // Step 5: Verify stock was deducted (eventual consistency — poll)
    assert.Eventually(t, func() bool {
        stock, _ := env.InventoryClient.GetStock(ctx, &pb.GetStockRequest{ProductId: env.TestProductID})
        return stock.QuantityOnHand == env.InitialStockQty-2
    }, 10*time.Second, 500*time.Millisecond)
}
```

---

## 5. Stress & Load Testing with k6 1.7.1

### 5.1 Test Scenarios

```javascript
// tests/stress/pos_checkout.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const checkoutErrorRate = new Rate('checkout_error_rate');
const checkoutDuration = new Trend('checkout_duration', true);

// Scenario definitions
export const options = {
    scenarios: {
        // Scenario 1: Normal load (baseline)
        normal_load: {
            executor: 'constant-arrival-rate',
            rate: 50,           // 50 requests/second
            timeUnit: '1s',
            duration: '5m',
            preAllocatedVUs: 100,
            maxVUs: 200,
        },
        // Scenario 2: Peak load (lunch rush, payday)
        peak_load: {
            executor: 'ramping-arrival-rate',
            startRate: 50,
            timeUnit: '1s',
            stages: [
                { target: 200, duration: '2m' },  // ramp up
                { target: 200, duration: '5m' },  // sustain peak
                { target: 50,  duration: '2m' },  // ramp down
            ],
            preAllocatedVUs: 300,
            maxVUs: 500,
            startTime: '6m',
        },
        // Scenario 3: Spike test (flash sale / promotion start)
        spike: {
            executor: 'ramping-arrival-rate',
            startRate: 50,
            timeUnit: '1s',
            stages: [
                { target: 500, duration: '30s' }, // sudden spike
                { target: 500, duration: '1m' },  // sustain
                { target: 50,  duration: '30s' }, // recover
            ],
            preAllocatedVUs: 600,
            maxVUs: 1000,
            startTime: '16m',
        },
    },

    // SLO thresholds — test FAILS if violated
    thresholds: {
        'http_req_duration{endpoint:create_order}': ['p(95)<500', 'p(99)<1000'],
        'http_req_duration{endpoint:checkout}':     ['p(95)<800', 'p(99)<1500'],
        'http_req_duration{endpoint:get_order}':    ['p(95)<200', 'p(99)<400'],
        'checkout_error_rate':                      ['rate<0.01'],  // < 1% errors
        'http_req_failed':                          ['rate<0.01'],
    },
};

// Test setup — create products, tenants, etc.
export function setup() {
    const tenant = createTestTenant();
    const product = createTestProduct(tenant.id, { price: 15000, stock: 10000 });
    return { tenantToken: tenant.token, productId: product.id };
}

export default function (data) {
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${data.tenantToken}`,
    };

    // Create order
    const createStart = Date.now();
    const createRes = http.post(`${BASE_URL}/v1/orders`, JSON.stringify({
        idempotency_key: crypto.randomUUID(),
        branch_id: TEST_BRANCH_ID,
        order_type: 'DINE_IN',
        table_number: `T-${Math.floor(Math.random() * 20) + 1}`,
    }), { headers, tags: { endpoint: 'create_order' } });

    check(createRes, {
        'create order: status 201': (r) => r.status === 201,
        'create order: has order_id': (r) => r.json('data.order_id') !== undefined,
    });
    checkoutErrorRate.add(createRes.status !== 201);
    checkoutDuration.add(Date.now() - createStart);

    if (createRes.status !== 201) return;
    const orderId = createRes.json('data.order_id');

    // Add items
    http.post(`${BASE_URL}/v1/orders/${orderId}/items`, JSON.stringify({
        product_id: data.productId,
        quantity: Math.floor(Math.random() * 3) + 1,
    }), { headers });

    // Process payment
    const paymentRes = http.post(`${BASE_URL}/v1/payments`, JSON.stringify({
        idempotency_key: crypto.randomUUID(),
        order_id: orderId,
        method: 'CASH',
        amount_tendered: { currency_code: 'IDR', amount_minor: 100000 },
    }), { headers, tags: { endpoint: 'checkout' } });

    check(paymentRes, {
        'checkout: status 200': (r) => r.status === 200,
        'checkout: payment completed': (r) => r.json('data.status') === 'COMPLETED',
    });
    checkoutErrorRate.add(paymentRes.status !== 200);

    sleep(Math.random() * 2); // think time between transactions
}

export function teardown(data) {
    // Cleanup test data
}
```

### 5.2 Additional Stress Scenarios

```javascript
// tests/stress/inventory_stock_deduction.js
// Tests concurrent stock deductions — should not go negative
export const options = {
    scenarios: {
        concurrent_stock_deduction: {
            executor: 'per-vu-iterations',
            vus: 100,
            iterations: 10,  // 100 VUs × 10 = 1000 total deductions
        },
    },
    thresholds: {
        'negative_stock_detected': ['count==0'],  // MUST be 0
    },
};

// tests/stress/offline_sync.js
// Tests sync service with 1000 devices syncing simultaneously
export const options = {
    scenarios: {
        mass_sync: {
            executor: 'constant-vus',
            vus: 1000,
            duration: '2m',
        },
    },
    thresholds: {
        'http_req_duration{endpoint:sync}': ['p(95)<2000'],
        'sync_conflict_rate': ['rate<0.05'], // < 5% conflicts
    },
};
```

### 5.3 Database Connection Pool Stress

```javascript
// tests/stress/db_pool_exhaustion.js
// Verify the service degrades gracefully when DB pool is exhausted
// Expected: 503 responses, not panics or connection leaks
export const options = {
    scenarios: {
        pool_exhaustion: {
            executor: 'constant-arrival-rate',
            rate: 1000,  // Beyond pool capacity
            timeUnit: '1s',
            duration: '1m',
            preAllocatedVUs: 1200,
            maxVUs: 2000,
        },
    },
    thresholds: {
        'http_req_failed': ['rate<0.50'],  // 50% failure acceptable
        // But must NOT see 500 (panic) — only 429/503 (graceful)
        'http_req_duration': ['p(99)<5000'],  // No extreme latency spikes
    },
};
```

### 5.4 SLO Definitions

| Endpoint | p95 Latency | p99 Latency | Error Rate |
|---|---|---|---|
| `POST /v1/orders` | < 500ms | < 1000ms | < 0.1% |
| `POST /v1/orders/{id}/items` | < 300ms | < 500ms | < 0.1% |
| `POST /v1/payments` | < 800ms | < 1500ms | < 0.5% |
| `GET /v1/orders/{id}` | < 200ms | < 400ms | < 0.1% |
| `GET /v1/orders` (list) | < 300ms | < 600ms | < 0.1% |
| `POST /v1/sync` (mobile) | < 2000ms | < 5000ms | < 1% |
| KDS stream (gRPC) | < 100ms first event | — | < 0.1% |

### 5.5 Running Load Tests

```makefile
# Makefile
STRESS_TARGET ?= http://localhost:8080

# Run all scenarios against local environment
stress-test:
    k6 run tests/stress/pos_checkout.js \
        -e BASE_URL=$(STRESS_TARGET) \
        --out influxdb=http://localhost:8086/k6

# Run specific scenario
stress-test-spike:
    k6 run tests/stress/pos_checkout.js \
        --scenario spike \
        -e BASE_URL=$(STRESS_TARGET)

# Run with HTML report
stress-test-report:
    k6 run tests/stress/pos_checkout.js \
        -e BASE_URL=$(STRESS_TARGET) \
        --out json=results.json
    k6 run --help | grep "web-dashboard"
    # Or: k6 run --web-dashboard tests/stress/pos_checkout.js
```

---

## 6. Test Organization & Naming

```
services/pos/
├── internal/
│   ├── domain/order/
│   │   ├── order.go
│   │   └── order_test.go          ← Unit tests (same package _test)
│   ├── application/command/
│   │   ├── cancel_order.go
│   │   └── cancel_order_test.go   ← Unit tests with mocks
│   └── infrastructure/postgres/
│       ├── order_repo.go
│       └── order_repo_test.go     ← Integration tests (testcontainers)
├── tests/
│   └── e2e/                       ← E2E tests (tag: e2e)
└── tests/stress/                  ← k6 scripts
```

**Test naming convention:**
```
TestFunctionName_Scenario_ExpectedResult

Examples:
TestCancelOrder_WhenPending_ShouldSucceed
TestCancelOrder_WhenPaid_ShouldReturnErrAlreadyPaid
TestFindByID_DifferentTenant_ShouldReturnNotFound
TestCheckoutFlow_CashPayment_ShouldCompleteSuccessfully
```

---

## 7. CI Pipeline Integration

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26.4' }

      - name: Run unit tests
        run: |
          go test -v -race -count=1 \
            -coverprofile=coverage.out \
            -tags=!integration,!e2e \
            ./services/.../internal/domain/...
            ./services/.../internal/application/...

      - name: Check coverage
        run: |
          go-test-coverage --config=.testcoverage.yml
          # Fails if thresholds not met

  integration-tests:
    runs-on: ubuntu-latest
    needs: unit-tests   # Only run if unit tests pass
    services:
      postgres:
        image: postgres:18.4-alpine
        env: { POSTGRES_PASSWORD: test, POSTGRES_DB: xyn_test }
      redis:
        image: redis:8.8-alpine
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26.4' }

      - name: Run integration tests
        run: |
          go test -v -race -count=1 \
            -tags=integration \
            -timeout=5m \
            ./services/.../internal/infrastructure/...

  stress-tests:
    runs-on: ubuntu-latest
    needs: integration-tests
    if: github.ref == 'refs/heads/main' # Only on main branch
    steps:
      - uses: actions/checkout@v4
      - uses: grafana/setup-k6-action@v1
        with: { k6-version: '1.7.1' }

      - name: Start test environment
        run: docker-compose -f docker-compose.test.yml up -d

      - name: Run stress tests
        run: |
          k6 run tests/stress/pos_checkout.js \
            -e BASE_URL=http://localhost:8080 \
            --out json=k6-results.json

      - name: Upload k6 results
        uses: actions/upload-artifact@v4
        with:
          name: k6-results
          path: k6-results.json
```

---

## 8. Test Data Management

### 8.1 Seed Data Strategy

```go
// tests/testdata/seed.go
package testdata

// SeedTenant creates a complete test tenant with branch, cashier, and products.
// Use this for integration and e2e tests that need realistic data.
func SeedTenant(ctx context.Context, pool *pgxpool.Pool) (*TestTenant, error) {
    tenant := &TestTenant{
        ID:    uuid.New(),
        Slug:  fmt.Sprintf("test-%s", uuid.New().String()[:8]),
        Token: generateTestToken(tenant.ID),
    }
    // Insert tenant, branch, user, products, initial stock...
    return tenant, nil
}

// SeedProduct creates a product with controlled initial stock.
func SeedProduct(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, stock int) (*TestProduct, error) {
    // ...
}
```

### 8.2 Golden Files for API Responses

For complex response shapes, use golden files:

```go
// tests/golden/update.go
// Run with: go test ./... -update-golden
var updateGolden = flag.Bool("update-golden", false, "update golden files")

func AssertMatchesGolden(t *testing.T, name string, actual []byte) {
    t.Helper()
    goldenPath := filepath.Join("testdata", "golden", name+".json")

    if *updateGolden {
        os.WriteFile(goldenPath, actual, 0644)
        return
    }

    expected, err := os.ReadFile(goldenPath)
    require.NoError(t, err)
    assert.JSONEq(t, string(expected), string(actual))
}
```

---

## 9. Testing Checklist

```
Before merging a PR:
✅ All unit tests pass (go test ./...)
✅ Race condition detector clean (go test -race ./...)
✅ Coverage thresholds met (per .testcoverage.yml)
✅ No test imports in production code (build tags enforce this)
✅ Integration tests pass for changed infrastructure code
✅ No real credentials or PII in test fixtures
✅ Table-driven tests used for multiple scenarios
✅ New domain error has a corresponding test case

For new endpoints:
✅ Unit test: happy path
✅ Unit test: each error case (not found, invalid state, etc.)
✅ Unit test: validation edge cases
✅ Integration test: real DB interaction
✅ k6 scenario updated if this is a high-traffic endpoint
```

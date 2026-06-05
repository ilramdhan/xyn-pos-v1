# Engineering Rules & Best Practices — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer  
> These rules are non-negotiable. PRs that violate them will be rejected.

---

## 1. Golang Rules

### 1.1 Project Structure (DDD / Clean Architecture)

```
services/{service-name}/
├── cmd/server/main.go          ← ONLY: wire up dependencies, start server
├── internal/
│   ├── domain/                 ← ZERO external imports allowed here
│   ├── application/            ← imports domain only
│   ├── infrastructure/         ← imports domain (implements ports)
│   └── interfaces/             ← imports application only
```

**Enforced rule:** `domain` package must have **zero** non-stdlib imports. Run `govulncheck` and a custom linter check in CI. Any `import` pointing to `infrastructure` or `interfaces` from `domain` or `application` is a blocking failure.

### 1.2 Naming Conventions

Follow Go's own stdlib naming conventions:

| Element | Convention | Example |
|---|---|---|
| Package names | Lowercase, short, no underscores | `order`, `escpos`, `kafka` |
| Interfaces | Noun or `er` suffix | `OrderRepository`, `Printer` |
| Structs | PascalCase, noun | `OrderAggregate`, `PaymentService` |
| Functions | PascalCase (exported), camelCase (unexported) | `CreateOrder`, `validateAmount` |
| Constants | ALL_CAPS (avoid), prefer PascalCase | `StatusPending`, `DefaultTimeout` |
| Error vars | `Err` prefix | `ErrOrderNotFound`, `ErrInsufficientStock` |
| Test files | `_test.go` suffix | `order_test.go` |
| Mock files | `mock_` prefix (generated) | `mock_order_repository.go` |

**Prohibited:**
- `Manager`, `Handler` suffixes on domain objects (use role-specific names: `Cashier`, `Dispatcher`)
- Hungarian notation (`strName`, `intQty`)
- Stuttering: package `order` should not export `OrderOrder`, just `Order`

### 1.3 Error Handling

**Rule: Never swallow errors. Always wrap with context.**

```go
// ✅ Good — wrap with context at every layer boundary
func (r *orderRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
    row := r.db.QueryRowContext(ctx, query, id)
    if err := row.Scan(&o); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrOrderNotFound
        }
        return nil, fmt.Errorf("orderRepo.FindByID: %w", err)
    }
    return &o, nil
}

// ❌ Bad — loses context
if err != nil {
    return err
}

// ❌ Bad — swallows error
if err != nil {
    log.Println(err)
    return nil, nil
}
```

Use `samber/oops` for rich error context (HTTP status codes, user-facing messages, stack traces):

```go
return nil, oops.Code("ORDER_NOT_FOUND").
    Public("Order not found").
    Wrapf(err, "find order %s", id)
```

**Domain errors are value types, not strings:**
```go
// domain/order/errors.go
var (
    ErrOrderNotFound     = errors.New("order not found")
    ErrOrderAlreadyPaid  = errors.New("order is already paid")
    ErrInvalidOrderState = errors.New("invalid order state transition")
)
```

### 1.4 Context Propagation

**Every function that does I/O must accept `context.Context` as its first parameter.**

```go
// ✅ Correct
func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*Order, error)

// ❌ Wrong — no context
func (s *OrderService) CreateOrder(cmd CreateOrderCommand) (*Order, error)
```

Never store context in a struct:
```go
// ❌ Prohibited
type OrderService struct {
    ctx context.Context  // Don't do this
}
```

### 1.5 Concurrency Rules

- Use `sync.WaitGroup` and channels for coordinating goroutines
- Always specify goroutine lifetime: every `go func()` must have a documented exit condition
- Use `errgroup.Group` for parallel operations that must all succeed
- Never close a channel from the receiver side
- Protect shared state with `sync.Mutex`, not `sync/atomic` unless you've profiled that it matters

```go
// ✅ Goroutine with clear lifetime
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

g, ctx := errgroup.WithContext(ctx)
g.Go(func() error {
    return s.inventoryService.DeductStock(ctx, items)
})
g.Go(func() error {
    return s.loyaltyService.AwardPoints(ctx, customerID, amount)
})
if err := g.Wait(); err != nil {
    return fmt.Errorf("post-payment processing: %w", err)
}
```

### 1.6 Logging Rules

Use `log/slog` (stdlib, Go 1.21+) with structured fields. Never use `fmt.Println` or `log.Printf` in production code.

```go
// ✅ Structured logging
slog.InfoContext(ctx, "order created",
    slog.String("order_id", order.ID.String()),
    slog.String("tenant_id", order.TenantID.String()),
    slog.Int("item_count", len(order.Items)),
)

// ❌ Unstructured
log.Printf("Order %s created with %d items", order.ID, len(order.Items))
```

Log levels:
- `Debug`: noisy operational data (only in dev/staging)
- `Info`: significant business events (order created, payment completed)
- `Warn`: recoverable anomalies (cache miss, retry attempt)
- `Error`: failures that need attention but don't crash the service

Never log sensitive data: passwords, PINs, full card numbers, PII beyond tenant/user IDs.

### 1.7 Testing Rules

**Coverage targets (enforced in CI):**
- Domain layer: ≥ 90%
- Application layer: ≥ 80%
- Infrastructure layer: ≥ 60% (integration tests count)
- Overall: ≥ 75%

**Test types and tooling:**

| Type | Tool | Location | Speed |
|---|---|---|---|
| Unit | `testify/suite` + `mockery` | `*_test.go` alongside source | < 10ms/test |
| Integration | `testcontainers-go` | `*_integration_test.go` | < 5s/test |
| E2E | `grpc-testing` + real services | `tests/e2e/` | < 60s/scenario |

**Golden rule: test behavior, not implementation.**

```go
// ✅ Tests the behavior (order can be cancelled when pending)
func (s *OrderSuite) TestCancel_WhenPending_ShouldSucceed() {
    order := NewOrder(...)
    err := order.Cancel("Customer request")
    s.NoError(err)
    s.Equal(StatusCancelled, order.Status)
}

// ❌ Tests the implementation (how many times a method was called)
mockRepo.AssertNumberOfCalls(t, "Save", 1)
```

**Mock generation:** Use `mockery v2` to generate mocks from interfaces. Never write mocks by hand.

```bash
# Generate mocks for all interfaces in domain/order/
mockery --name=OrderRepository --dir=internal/domain/order --output=internal/mocks
```

**Integration tests use real databases via Testcontainers:**

```go
func TestOrderRepository_Integration(t *testing.T) {
    ctx := context.Background()
    container, err := testcontainers.RunContainer(ctx,
        "postgres:16-alpine",
        testcontainers.WithEnv(map[string]string{
            "POSTGRES_PASSWORD": "test",
            "POSTGRES_DB":       "xyn_test",
        }),
    )
    // ... run goose migrations, then test real SQL queries
}
```

### 1.8 Linting Configuration

Required linters (enforced in CI via `golangci-lint`):

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck          # All errors must be handled
    - govet             # Suspicious constructs
    - staticcheck       # Extended static analysis
    - revive            # Style rules
    - gocyclo           # Max cyclomatic complexity: 15
    - dupl              # Duplicate code detection
    - gosec             # Security issues
    - exhaustive        # All switch cases handled
    - godot             # Comments end with period
    - noctx             # HTTP requests must use context
    - bodyclose         # HTTP response bodies must be closed
    - sqlcloserows      # sql.Rows must be closed
    - unparam           # Unused function parameters
    - unused            # Unused code
```

---

## 2. Web Frontend Rules (Next.js / React / TypeScript)

### 2.1 Component Structure

```
apps/web/src/
├── app/                    ← Next.js App Router (pages & layouts)
├── components/
│   ├── ui/                 ← shadcn/ui base components (auto-generated, don't edit)
│   └── features/           ← Feature-specific components
│       └── pos/
│           ├── CartPanel/
│           │   ├── index.tsx        ← Public export only
│           │   ├── CartPanel.tsx    ← Component
│           │   ├── CartPanel.test.tsx
│           │   └── useCart.ts       ← Local hook
├── hooks/                  ← Shared hooks
├── lib/                    ← Utilities, adapters
├── store/                  ← Zustand stores
└── services/               ← TanStack Query + API calls
```

### 2.2 TypeScript Rules

- `strict: true` in `tsconfig.json` — no exceptions
- No `any` type — use `unknown` and narrow with type guards
- No `@ts-ignore` — fix the type issue or use `@ts-expect-error` with a comment explaining why
- All API responses must have a defined type (generate from Protobuf or OpenAPI spec)
- Use `zod` for runtime validation of external data (API responses, form inputs)

### 2.3 State Management Rules

| State Type | Tool | Rule |
|---|---|---|
| Server state (API data) | TanStack Query | Always use `useQuery`/`useMutation`, never manual fetch |
| Global UI state | Zustand | Keep slices small; one store per domain |
| Form state | React Hook Form | Never use controlled inputs for complex forms |
| Component-local state | `useState` | Fine for UI-only state (open/close, hover) |
| URL state | `useSearchParams` | Filters, pagination, selected item |

**Never** put server data in Zustand. TanStack Query is the cache for server data.

### 2.4 Performance Rules

- All pages must pass Core Web Vitals: LCP < 2.5s, FID < 100ms, CLS < 0.1
- Images: always use `next/image` with explicit width/height
- Dynamic imports: use `next/dynamic` for components > 30KB that aren't immediately visible
- Bundle analysis: run `next build --analyze` before every release

---

## 3. Mobile Rules (Flutter / Dart)

### 3.1 Architecture: Feature-First with Riverpod

```
lib/
├── core/                   ← Shared utilities, theme, routing
├── features/
│   ├── auth/
│   │   ├── data/           ← Repository implementations
│   │   ├── domain/         ← Entities, repository interfaces
│   │   └── presentation/   ← Screens, widgets, providers
│   └── pos/
├── shared/
│   ├── widgets/            ← Reusable widgets
│   └── providers/          ← Shared Riverpod providers
└── main.dart
```

### 3.2 Dart / Flutter Rules

- Use `freezed` + `json_serializable` for all data models — no manual `copyWith` or `toJson`
- Use `Drift` for local database — never raw SQLite
- Never call `setState` from outside the widget where it's defined
- All async operations must handle loading and error states via `AsyncValue`
- Use `go_router` for navigation — no `Navigator.push` directly

```dart
// ✅ Riverpod AsyncNotifier pattern
@riverpod
class CartNotifier extends _$CartNotifier {
  @override
  FutureOr<Cart> build() => ref.read(cartRepositoryProvider).getActiveCart();

  Future<void> addItem(Product product) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final cart = await future;
      return cart.addItem(CartItem.fromProduct(product));
    });
  }
}
```

### 3.3 Offline-First Rules

- **All** data displayed in the UI must be readable from the local Drift database
- Network calls are for synchronization, not for rendering
- `AsyncNotifier` should read from local DB first, then trigger background sync
- Never show a blank screen because of a network error — show stale data with an indicator

---

## 4. Security Rules

### 4.1 Authentication & Authorization

- All gRPC endpoints must validate JWT/PASETO via interceptor — no exceptions
- Extract `tenant_id` and `user_id` from the verified token, never from the request body
- RBAC checks happen in the **application layer** (use case), not in the interface layer
- PIN-based operations (opening cash drawer, voiding orders) require a separate PIN verification step

```go
// ✅ Correct — tenant_id from verified token context
func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    claims := auth.ClaimsFromContext(ctx) // extracted by interceptor
    cmd := application.CreateOrderCommand{
        TenantID: claims.TenantID,  // from token, not req
        UserID:   claims.UserID,    // from token, not req
    }
}
```

### 4.2 Input Validation

- Validate all external input at the `interfaces` layer before it enters the application layer
- Use protobuf field validation (`buf validate`) for gRPC endpoints
- Use `zod` for web form validation
- Sanitize all user-generated content before storing (XSS prevention)
- Never construct SQL strings from user input — always use parameterized queries

### 4.3 Secrets Management

- **Zero secrets in source code** — use environment variables
- Development: `.env.local` (git-ignored)
- Staging/Production: HashiCorp Vault or Kubernetes Secrets (sealed with Sealed Secrets)
- Never log secrets, tokens, or credentials
- Rotate secrets on suspicion of compromise immediately

### 4.4 API Security

- Rate limiting at the API Gateway layer (KrakenD)
- Request size limits: 1MB for regular endpoints, 10MB for file uploads
- CORS: whitelist only known origins in production
- HTTPS everywhere — no HTTP in production, even internal services
- SQL injection prevention: 100% parameterized queries, enforced by `gosec` linter

---

## 5. Git Workflow & Branching Strategy

### 5.1 Branch Naming

```
main              ← Protected. Production-ready at all times.
develop           ← Integration branch. Always deployable to staging.
feature/{ticket}-{short-desc}    ← e.g., feature/XYN-42-add-cart-api
fix/{ticket}-{short-desc}        ← e.g., fix/XYN-99-fix-stock-deduction
chore/{short-desc}               ← e.g., chore/update-dependencies
release/{version}                ← e.g., release/1.2.0
hotfix/{ticket}-{short-desc}     ← Emergency fixes to main
```

### 5.2 Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>

<body> (optional)

<footer> (optional, for breaking changes or issue references)
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `ci`

```
feat(pos): add support for split payment

Allows an order to be paid with multiple payment methods (cash + QRIS).
Each payment method creates a separate PaymentEntry record.
Total must equal the order grand total before checkout is allowed.

Closes #42
```

**Rules:**
- Subject line ≤ 72 characters
- Use imperative mood: "add feature" not "added feature"
- Breaking changes must include `BREAKING CHANGE:` in the footer
- Reference tickets with `Closes #N` or `Refs #N`

### 5.3 Pull Request Rules

**Before opening a PR:**
- [ ] All tests pass locally (`go test ./...` or `npm test`)
- [ ] Linting passes (`golangci-lint run` or `npm run lint`)
- [ ] No secrets committed (`git secrets --scan`)
- [ ] PR description explains the **why**, not just the **what**
- [ ] Breaking changes are documented

**PR size limits:**
- Aim for < 400 lines changed per PR
- Large features must be broken into smaller, independently reviewable PRs
- Each PR should be shippable independently (feature flags for incomplete features)

**Review requirements:**
- Minimum 1 approving review for `develop` branch
- Minimum 2 approving reviews for `main` branch
- No self-merges
- CI must pass before merge (tests, lint, build)

### 5.4 Merge Strategy

- `feature → develop`: **Squash merge** (keep history clean)
- `develop → main`: **Merge commit** (preserves the merge point for rollback)
- `hotfix → main`: **Merge commit** + backport to `develop`
- **No force-push** to `main` or `develop`

### 5.5 Release Process

```
1. Create release branch: git checkout -b release/1.2.0 develop
2. Bump version numbers (go.mod, package.json)
3. Update CHANGELOG.md
4. Final QA on staging environment
5. PR: release/1.2.0 → main (merge commit)
6. Tag: git tag -a v1.2.0 -m "Release 1.2.0"
7. Backport: PR release/1.2.0 → develop
8. ArgoCD auto-syncs production from main
```

---

## 6. Infrastructure Rules

### 6.1 Docker

- All services must have a multi-stage Dockerfile (build stage + runtime stage)
- Runtime image: `gcr.io/distroless/static:nonroot` or `alpine:3.x` (not `ubuntu`)
- Never run as root in containers (`USER nonroot:nonroot`)
- Health checks must be defined in every Dockerfile

### 6.2 Kubernetes

- Resource limits (`requests` and `limits`) must be defined for every container
- Liveness and readiness probes required for every service
- No secrets in ConfigMaps — use Kubernetes Secrets + Sealed Secrets
- PodDisruptionBudget for all critical services (minAvailable: 1)

### 6.3 Infrastructure as Code

- All infrastructure changes via Terraform — no manual console changes
- Every Terraform change must have a plan reviewed before apply
- State stored in remote backend (S3-compatible MinIO) with state locking
- Modules for reusable components (e.g., a `microservice` module that creates Deployment + Service + HPA)

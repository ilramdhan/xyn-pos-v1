# Mobile Architecture — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Stack: Flutter 3.44.0 | Dart 3.12 | Riverpod 3.2.1 | Drift 2.33.0

---

## 1. Project Structure

```
apps/mobile/
├── lib/
│   ├── main.dart
│   ├── app.dart                      ← MaterialApp + GoRouter + ProviderScope
│   ├── core/
│   │   ├── config/
│   │   │   └── app_config.dart       ← Env vars via dart-define
│   │   ├── api/
│   │   │   ├── client.dart           ← HTTP/gRPC client setup
│   │   │   ├── response.dart         ← BaseResponse, DataResponse, etc.
│   │   │   └── exceptions.dart       ← ApiException
│   │   ├── database/
│   │   │   └── app_database.dart     ← Drift database
│   │   ├── sync/
│   │   │   ├── sync_engine.dart      ← Outbox processor
│   │   │   ├── conflict_resolver.dart
│   │   │   └── sync_provider.dart
│   │   ├── theme/
│   │   │   ├── app_colors.dart
│   │   │   ├── app_text_styles.dart
│   │   │   └── app_theme.dart
│   │   ├── router/
│   │   │   └── app_router.dart       ← GoRouter configuration
│   │   └── utils/
│   ├── features/
│   │   ├── auth/
│   │   │   ├── data/
│   │   │   │   ├── auth_repository_impl.dart
│   │   │   │   └── datasources/
│   │   │   ├── domain/
│   │   │   │   ├── auth_repository.dart   ← Interface
│   │   │   │   └── entities/
│   │   │   └── presentation/
│   │   │       ├── screens/
│   │   │       ├── widgets/
│   │   │       └── providers/
│   │   │           └── auth_provider.dart
│   │   ├── pos/                       ← POS Core feature
│   │   ├── cart/                      ← Cart management
│   │   ├── payment/                   ← Checkout & payment
│   │   ├── inventory/                 ← Stock lookup
│   │   ├── settings/                  ← Printer, scanner config
│   │   └── sync/                      ← Sync status UI
│   └── shared/
│       ├── widgets/                   ← Reusable UI components
│       └── providers/                 ← Shared Riverpod providers
├── test/
│   ├── unit/
│   ├── widget/
│   └── integration/
├── assets/
├── pubspec.yaml
└── CLAUDE.md
```

---

## 2. Riverpod 3.2.1 — Critical Breaking Changes from v2

> ⚠️ **Riverpod 3.x has significant breaking changes.** Do not copy v2 patterns.

### 2.1 Key Changes in Riverpod 3

| Feature | Riverpod 2.x | Riverpod 3.x |
|---|---|---|
| Code generation | `@riverpod` annotation | `@riverpod` annotation (same, but updated generator) |
| `AsyncNotifier.build` | Returns `FutureOr<T>` | Returns `FutureOr<T>` (unchanged) |
| `Ref` type | `Ref<T>` was generic | `Ref` is now a plain type (no generic) |
| Provider scope | `ProviderScope` | `ProviderScope` (unchanged) |
| `riverpod_lint` | v2 rules | Updated rules for v3 |
| `flutter_riverpod` | 2.x | 3.2.1 |
| `riverpod_annotation` | 2.x | 3.x |
| `riverpod_generator` | 2.x | 3.x |

### 2.2 Standard Provider Patterns

```dart
// pubspec.yaml dependencies:
// flutter_riverpod: ^3.2.1
// riverpod_annotation: ^3.0.0
// riverpod_generator: ^3.0.0  (dev dependency)
// riverpod_lint: ^3.0.0        (dev dependency)

// ── Simple provider (computed value) ─────────────────────────────────
@riverpod
String appVersion(Ref ref) => AppConfig.version;  // Ref, not Ref<T>

// ── Async provider (data fetching) ───────────────────────────────────
@riverpod
Future<List<Product>> products(Ref ref) async {
  final repo = ref.watch(productRepositoryProvider);
  return repo.getAll();
}

// ── AsyncNotifier (stateful + async operations) ───────────────────────
@riverpod
class CartNotifier extends _$CartNotifier {
  @override
  Future<Cart> build() async {
    // Build loads the initial state — called once on first watch
    final db = ref.watch(appDatabaseProvider);
    return db.cartDao.getActiveCart();
  }

  Future<void> addItem(Product product, int quantity) async {
    // Update state optimistically
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final current = await future;
      final updated = current.addItem(CartItem.from(product, quantity));
      // Persist to local DB
      final db = ref.read(appDatabaseProvider);
      await db.cartDao.save(updated);
      // Queue for sync
      ref.read(syncEngineProvider).enqueue(CartItemAddedEvent(updated));
      return updated;
    });
  }

  Future<void> removeItem(String itemId) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final current = await future;
      return current.removeItem(itemId);
    });
  }

  Future<void> clear() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final db = ref.read(appDatabaseProvider);
      await db.cartDao.clearActive();
      return Cart.empty();
    });
  }
}

// ── Notifier (stateful + sync operations) ────────────────────────────
@riverpod
class PrinterSettingsNotifier extends _$PrinterSettingsNotifier {
  @override
  PrinterSettings build() {
    // Load from SharedPreferences synchronously
    return ref.watch(sharedPreferencesProvider).getPrinterSettings();
  }

  void setPrinterType(PrinterType type) {
    state = state.copyWith(printerType: type);
    ref.read(sharedPreferencesProvider).savePrinterSettings(state);
  }
}
```

### 2.3 Provider Dependencies & Lifecycle

```dart
// ── Repository provider (singleton for the app lifetime) ─────────────
@Riverpod(keepAlive: true)
ProductRepository productRepository(Ref ref) {
  final db = ref.watch(appDatabaseProvider);
  final client = ref.watch(apiClientProvider);
  return ProductRepositoryImpl(localDb: db, remoteClient: client);
}

// ── Scoped provider (recreated when a parameter changes) ─────────────
@riverpod
Future<Order> order(Ref ref, String orderId) async {
  final repo = ref.watch(orderRepositoryProvider);
  return repo.getById(orderId);
}

// Usage: ref.watch(orderProvider('018e1234-...'))

// ── Family provider (parameterized) ──────────────────────────────────
@riverpod
Future<Stock> stockByProduct(Ref ref, String productId) async {
  // Auto-disposed when no longer watched
  return ref.watch(inventoryRepositoryProvider).getStock(productId);
}
```

### 2.4 Widget Integration

```dart
// ✅ Correct Riverpod 3 pattern
class ProductGridScreen extends ConsumerWidget {
  const ProductGridScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final productsAsync = ref.watch(productsProvider);

    return productsAsync.when(
      loading: () => const ProductGridSkeleton(),
      error: (error, stack) => AsyncErrorWidget(
        error: error,
        onRetry: () => ref.invalidate(productsProvider),
      ),
      data: (products) => products.isEmpty
          ? const EmptyProductsState()
          : ProductGrid(products: products),
    );
  }
}

// ✅ ConsumerStatefulWidget for complex lifecycle needs
class POSScreen extends ConsumerStatefulWidget {
  const POSScreen({super.key});
  @override
  ConsumerState<POSScreen> createState() => _POSScreenState();
}

class _POSScreenState extends ConsumerState<POSScreen> {
  late final BarcodeListener _barcodeListener;

  @override
  void initState() {
    super.initState();
    _barcodeListener = BarcodeListener(
      onScan: (barcode) => ref.read(cartNotifierProvider.notifier).addByBarcode(barcode),
    );
  }

  @override
  void dispose() {
    _barcodeListener.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) { ... }
}
```

---

## 3. Drift 2.33.0 — Local Database Schema

### 3.1 Database Definition

```dart
// lib/core/database/app_database.dart
import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

part 'app_database.g.dart';

// ── Table Definitions ─────────────────────────────────────────────────

class Products extends Table {
  TextColumn get id => text()();
  TextColumn get tenantId => text().named('tenant_id')();
  TextColumn get name => text()();
  TextColumn get sku => text()();
  TextColumn get categoryId => text().named('category_id').nullable()();
  IntColumn get unitPriceMinor => integer().named('unit_price_minor')();  // int64 minor units
  TextColumn get currencyCode => text().named('currency_code').withDefault(const Constant('IDR'))();
  BoolColumn get trackInventory => boolean().named('track_inventory').withDefault(const Constant(true))();
  BoolColumn get isActive => boolean().named('is_active').withDefault(const Constant(true))();
  TextColumn get imageUrl => text().named('image_url').nullable()();
  DateTimeColumn get updatedAt => dateTime().named('updated_at')();
  DateTimeColumn get syncedAt => dateTime().named('synced_at').nullable()();

  @override
  Set<Column> get primaryKey => {id};

  @override
  List<Index> get indexes => [
    Index('idx_products_tenant_sku', '(tenant_id, sku)', unique: true),
    Index('idx_products_tenant_active', '(tenant_id, is_active)'),
  ];
}

class Orders extends Table {
  TextColumn get id => text()();
  TextColumn get tenantId => text().named('tenant_id')();
  TextColumn get branchId => text().named('branch_id')();
  TextColumn get idempotencyKey => text().named('idempotency_key').unique()();
  TextColumn get status => textEnum<OrderStatus>()();
  TextColumn get tableNumber => text().named('table_number').nullable()();
  IntColumn get subtotalMinor => integer().named('subtotal_minor').withDefault(const Constant(0))();
  IntColumn get taxMinor => integer().named('tax_minor').withDefault(const Constant(0))();
  IntColumn get discountMinor => integer().named('discount_minor').withDefault(const Constant(0))();
  IntColumn get totalMinor => integer().named('total_minor').withDefault(const Constant(0))();
  DateTimeColumn get createdAt => dateTime().named('created_at')();
  DateTimeColumn get updatedAt => dateTime().named('updated_at')();
  DateTimeColumn get syncedAt => dateTime().named('synced_at').nullable()();
  TextColumn get conflictState => text().named('conflict_state').nullable()();

  @override
  Set<Column> get primaryKey => {id};
}

class OrderItems extends Table {
  TextColumn get id => text()();
  TextColumn get orderId => text().named('order_id').references(Orders, #id)();
  TextColumn get productId => text().named('product_id')();
  TextColumn get productName => text().named('product_name')();
  IntColumn get quantity => integer()();
  IntColumn get unitPriceMinor => integer().named('unit_price_minor')();
  IntColumn get subtotalMinor => integer().named('subtotal_minor')();
  TextColumn get notes => text().nullable()();

  @override
  Set<Column> get primaryKey => {id};
}

// Outbox for offline operations — NEVER delete processed entries for audit
class OutboxEvents extends Table {
  IntColumn get id => integer().autoIncrement()();
  TextColumn get eventType => text().named('event_type')();   // 'cart.item_added', 'order.created'
  TextColumn get aggregateId => text().named('aggregate_id')();
  TextColumn get payload => text()();                          // JSON
  TextColumn get idempotencyKey => text().named('idempotency_key').unique()();
  IntColumn get retryCount => integer().named('retry_count').withDefault(const Constant(0))();
  TextColumn get status => textEnum<OutboxStatus>()();
  TextColumn get errorMessage => text().named('error_message').nullable()();
  DateTimeColumn get createdAt => dateTime().named('created_at')();
  DateTimeColumn get processedAt => dateTime().named('processed_at').nullable()();
}

class SyncState extends Table {
  TextColumn get id => text()();           // 'global' singleton or per-entity
  TextColumn get lastCheckpoint => text().named('last_checkpoint')();  // Server sequence number
  DateTimeColumn get lastSyncAt => dateTime().named('last_sync_at').nullable()();
  TextColumn get deviceId => text().named('device_id')();

  @override
  Set<Column> get primaryKey => {id};
}

// ── Database Class ────────────────────────────────────────────────────

@DriftDatabase(tables: [Products, Orders, OrderItems, OutboxEvents, SyncState])
class AppDatabase extends _$AppDatabase {
  AppDatabase() : super(_openConnection());

  @override
  int get schemaVersion => 1;

  @override
  MigrationStrategy get migration => MigrationStrategy(
    onCreate: (m) => m.createAll(),
    onUpgrade: (m, from, to) async {
      // Handle schema migrations here
    },
    beforeOpen: (details) async {
      // Enable WAL mode for better concurrent read performance
      await customStatement('PRAGMA journal_mode = WAL');
      await customStatement('PRAGMA foreign_keys = ON');
    },
  );
}

DatabaseConnection _openConnection() {
  return DatabaseConnection(driftDatabase(name: 'xyn_pos_v1'));
}
```

### 3.2 DAO Pattern

```dart
// lib/features/pos/data/daos/order_dao.dart
part of '../../../core/database/app_database.dart';

extension OrderDao on AppDatabase {
  // Read operations
  Future<Order?> getOrderById(String id) =>
      (select(orders)..where((o) => o.id.equals(id))).getSingleOrNull();

  Stream<List<Order>> watchPendingOrders(String tenantId) =>
      (select(orders)
        ..where((o) => o.tenantId.equals(tenantId) & o.status.isIn(['DRAFT', 'PENDING']))
        ..orderBy([(o) => OrderingTerm.desc(o.createdAt)]))
      .watch();

  // Write operations
  Future<void> upsertOrder(OrdersCompanion order) =>
      into(orders).insertOnConflictUpdate(order);

  Future<void> updateOrderStatus(String id, OrderStatus status) =>
      (update(orders)..where((o) => o.id.equals(id)))
          .write(OrdersCompanion(status: Value(status)));

  // Outbox operations
  Future<List<OutboxEvent>> getPendingOutboxEvents() =>
      (select(outboxEvents)
        ..where((e) => e.status.equals('PENDING'))
        ..orderBy([(e) => OrderingTerm.asc(e.id)])
        ..limit(100))
      .get();

  Future<void> markOutboxEventProcessed(int eventId, String? serverId) =>
      (update(outboxEvents)..where((e) => e.id.equals(eventId)))
          .write(OutboxEventsCompanion(
            status: const Value('PROCESSED'),
            processedAt: Value(DateTime.now()),
          ));

  Future<void> markOutboxEventFailed(int eventId, String error, int retryCount) =>
      (update(outboxEvents)..where((e) => e.id.equals(eventId)))
          .write(OutboxEventsCompanion(
            status: Value(retryCount >= 3 ? 'DEAD' : 'PENDING'),
            errorMessage: Value(error),
            retryCount: Value(retryCount),
          ));
}
```

---

## 4. Offline-First Sync Engine

### 4.1 Outbox Processor

```dart
// lib/core/sync/sync_engine.dart
import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:connectivity_plus/connectivity_plus.dart';

@Riverpod(keepAlive: true)
SyncEngine syncEngine(Ref ref) {
  final db = ref.watch(appDatabaseProvider);
  final client = ref.watch(syncClientProvider);
  final engine = SyncEngine(db: db, client: client);
  ref.onDispose(engine.dispose);
  return engine;
}

class SyncEngine {
  SyncEngine({required this.db, required this.client});

  final AppDatabase db;
  final SyncClient client;

  StreamSubscription? _connectivitySub;
  bool _isSyncing = false;

  void init() {
    // Watch connectivity — sync when online
    _connectivitySub = Connectivity().onConnectivityChanged.listen((result) {
      final isOnline = result != ConnectivityResult.none;
      if (isOnline && !_isSyncing) {
        processPendingOutbox();
      }
    });
  }

  // Enqueue an offline operation (called from UI actions)
  Future<void> enqueue(OutboxEventsCompanion event) async {
    await db.into(db.outboxEvents).insert(event);
  }

  // Process pending outbox (called on connectivity restore or background)
  Future<SyncResult> processPendingOutbox() async {
    if (_isSyncing) return SyncResult.alreadyRunning;
    _isSyncing = true;

    final result = SyncResult();
    try {
      // Pull server changes first (avoid creating conflicts with stale data)
      await _pullServerChanges();

      // Then push local changes
      final events = await db.getPendingOutboxEvents();
      for (final event in events) {
        try {
          final serverResponse = await client.sendEvent(event);

          switch (serverResponse.status) {
            case SyncAckStatus.accepted:
              await db.markOutboxEventProcessed(event.id, serverResponse.serverId);
              result.processed++;

            case SyncAckStatus.duplicate:
              // Server already has this — idempotency worked
              await db.markOutboxEventProcessed(event.id, null);
              result.duplicates++;

            case SyncAckStatus.conflict:
              await _resolveConflict(event, serverResponse.serverVersion);
              result.conflicts++;

            case SyncAckStatus.rejected:
              await db.markOutboxEventFailed(event.id, serverResponse.errorMessage ?? 'rejected', event.retryCount + 1);
              result.failed++;
          }
        } on ApiException catch (e) {
          if (e.isNetworkError || e.isServerError) {
            // Transient error — will retry next sync
            result.retriable++;
            break; // Stop processing this batch — network is unstable
          }
          await db.markOutboxEventFailed(event.id, e.response.message, event.retryCount + 1);
          result.failed++;
        }
      }
    } finally {
      _isSyncing = false;
    }
    return result;
  }

  Future<void> _pullServerChanges() async {
    final syncState = await db.getSyncState();
    final checkpoint = syncState?.lastCheckpoint ?? '0';

    await for (final push in client.streamChanges(fromCheckpoint: checkpoint)) {
      await _applyServerPush(push);
    }
  }

  Future<void> _resolveConflict(OutboxEvent local, dynamic serverVersion) async {
    final resolver = ConflictResolver();
    final resolution = resolver.resolve(local, serverVersion);

    switch (resolution) {
      case ConflictResolution.useServer:
        await _applyServerVersion(serverVersion);
        await db.markOutboxEventProcessed(local.id, null);

      case ConflictResolution.useLocal:
        // Retry with the local version — override server
        await client.sendEventWithOverride(local);
        await db.markOutboxEventProcessed(local.id, null);

      case ConflictResolution.merge:
        // CRDT merge (for stock deltas)
        final merged = _mergeCRDT(local, serverVersion);
        await client.sendEventWithOverride(merged);
        await db.markOutboxEventProcessed(local.id, null);
    }
  }

  void dispose() {
    _connectivitySub?.cancel();
  }
}
```

### 4.2 Conflict Resolution Rules

```dart
// lib/core/sync/conflict_resolver.dart
enum ConflictResolution { useServer, useLocal, merge }

class ConflictResolver {
  ConflictResolution resolve(OutboxEvent local, dynamic serverVersion) {
    return switch (local.eventType) {
      // New offline orders always win — server has no prior version
      'order.created' => ConflictResolution.useLocal,

      // Server wins for paid/voided orders — financial integrity
      'order.updated' when _serverIsFinalized(serverVersion) => ConflictResolution.useServer,

      // Stock deductions use CRDT delta merging
      'inventory.stock_deducted' => ConflictResolution.merge,

      // Payment records — server is always authoritative
      'payment.completed' => ConflictResolution.useServer,
      'payment.voided' => ConflictResolution.useServer,

      // Product/price changes — server is authoritative (admin action)
      'product.updated' => ConflictResolution.useServer,

      // Default: last write wins based on timestamp
      _ => _lastWriteWins(local, serverVersion),
    };
  }

  bool _serverIsFinalized(dynamic version) {
    final status = version?['status'] as String?;
    return status == 'PAID' || status == 'VOIDED' || status == 'CANCELLED';
  }

  ConflictResolution _lastWriteWins(OutboxEvent local, dynamic serverVersion) {
    final serverTime = DateTime.tryParse(serverVersion?['updated_at'] ?? '') ?? DateTime(0);
    final localTime = local.createdAt;
    return localTime.isAfter(serverTime)
        ? ConflictResolution.useLocal
        : ConflictResolution.useServer;
  }
}
```

---

## 5. Background Sync

### 5.1 WorkManager (Android) + BGTaskScheduler (iOS)

```dart
// lib/core/sync/background_sync.dart
import 'package:workmanager/workmanager.dart';

const _syncTaskName = 'xyn_pos_background_sync';

@pragma('vm:entry-point')
void callbackDispatcher() {
  Workmanager().executeTask((taskName, inputData) async {
    // Background isolate — must reinitialize everything
    final container = ProviderContainer();
    try {
      final engine = container.read(syncEngineProvider);
      final result = await engine.processPendingOutbox();
      return result.failed == 0; // true = success, false = retry
    } catch (e) {
      return false; // Trigger retry
    } finally {
      container.dispose();
    }
  });
}

Future<void> registerBackgroundSync() async {
  await Workmanager().initialize(callbackDispatcher, isInDebugMode: false);

  await Workmanager().registerPeriodicTask(
    _syncTaskName,
    _syncTaskName,
    frequency: const Duration(minutes: 15),
    constraints: Constraints(
      networkType: NetworkType.connected,
      requiresBatteryNotLow: false,  // POS devices are usually plugged in
    ),
    backoffPolicy: BackoffPolicy.exponential,
    backoffPolicyDelay: const Duration(minutes: 1),
  );
}
```

---

## 6. State Management Patterns

### 6.1 Optimistic Updates

```dart
// When adding a cart item, update UI instantly before server confirms
Future<void> addItem(Product product) async {
  // 1. Generate IDs and idempotency key locally
  final itemId = const Uuid().v4();
  final idempotencyKey = const Uuid().v4();

  // 2. Update UI optimistically (don't wait for server)
  state = AsyncData(state.value!.addItem(CartItem(
    id: itemId,
    productId: product.id,
    quantity: 1,
    unitPriceMinor: product.unitPriceMinor,
    status: SyncStatus.pending,  // Show pending indicator in UI
  )));

  // 3. Persist to local DB
  await db.orderItemDao.insert(CartItem(...));

  // 4. Enqueue for sync (will be processed when online)
  await syncEngine.enqueue(OutboxEventsCompanion(
    eventType: const Value('cart.item_added'),
    aggregateId: Value(itemId),
    idempotencyKey: Value(idempotencyKey),
    payload: Value(jsonEncode({...})),
    status: const Value('PENDING'),
    createdAt: Value(DateTime.now()),
  ));
}
```

### 6.2 Sync Status Indicators

```dart
// Show per-item sync status in the cart
enum SyncStatus { synced, pending, failed, conflict }

class CartItemTile extends ConsumerWidget {
  const CartItemTile({super.key, required this.item});
  final CartItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ListTile(
      title: Text(item.productName),
      trailing: Row(children: [
        Text(item.formattedSubtotal),
        const SizedBox(width: 8),
        _SyncStatusIcon(status: item.syncStatus),
      ]),
    );
  }
}

class _SyncStatusIcon extends StatelessWidget {
  const _SyncStatusIcon({required this.status});
  final SyncStatus status;

  @override
  Widget build(BuildContext context) => switch (status) {
    SyncStatus.synced   => const Icon(Icons.cloud_done, size: 14, color: Colors.green),
    SyncStatus.pending  => const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2)),
    SyncStatus.failed   => const Icon(Icons.cloud_off, size: 14, color: Colors.red),
    SyncStatus.conflict => const Icon(Icons.warning, size: 14, color: Colors.orange),
  };
}
```

---

## 7. Testing Strategy (Mobile)

```dart
// Widget tests with Riverpod
void main() {
  group('CartNotifier', () {
    test('addItem should optimistically update state', () async {
      final container = ProviderContainer(overrides: [
        appDatabaseProvider.overrideWith((_) => InMemoryDatabase()),
        syncEngineProvider.overrideWith((_) => MockSyncEngine()),
      ]);

      final notifier = container.read(cartNotifierProvider.notifier);
      await notifier.addItem(TestProduct.burger());

      final state = container.read(cartNotifierProvider);
      expect(state.value?.items.length, equals(1));
      expect(state.value?.items.first.productName, equals('Burger'));
    });
  });
}

// Integration tests using real Drift in-memory
InMemoryDatabase createTestDatabase() {
  return AppDatabase(NativeDatabase.memory());
}
```

---

## 8. Performance Rules

```
✅ Use const constructors everywhere possible — reduces widget rebuilds
✅ Use select() on Riverpod to watch only the fields you need
✅ Images: use CachedNetworkImage with disk cache
✅ Product grid: use SliverGrid for large lists (200+ products)
✅ Drift queries: always use streams (watch()) for reactive UI
✅ Never run database queries on the UI thread — Drift handles this but be aware
✅ Use RepaintBoundary around expensive widgets (charts, grids)

❌ Never call ref.read() inside build() — use ref.watch()
❌ Never create providers inside build() — define them at file scope
❌ Never use setState on a ConsumerWidget — use ref.read(provider.notifier)
❌ Never make Drift queries synchronous with .get() in build — use .watch() + AsyncValue
```

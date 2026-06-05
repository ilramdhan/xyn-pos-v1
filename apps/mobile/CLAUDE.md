# Mobile App — AI Assistant Context

> Scope: apps/mobile/  
> Stack: Flutter 3.44.0 | Dart 3.12 | Riverpod 3.2.1 | Drift 2.33.0  
> Parent context: ../../CLAUDE.md (read that first)  
> Deep-dive: docs/phase-1/mobile-architecture.md

---

## Critical: Riverpod 3.x Breaking Changes

Riverpod 3 is NOT backward compatible with v2. Key changes:

```dart
// ✅ Riverpod 3 — Ref is no longer generic
@riverpod
String appVersion(Ref ref) => '1.0.0';  // Ref, not Ref<String>

// ❌ Riverpod 2 (DO NOT USE)
@riverpod
String appVersion(AutoDisposeRef<String> ref) => '1.0.0';

// ✅ Riverpod 3 — AsyncNotifier pattern
@riverpod
class CartNotifier extends _$CartNotifier {
  @override
  Future<Cart> build() async => _loadFromDb();

  Future<void> addItem(Product p) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async { ... });
  }
}
```

---

## Offline-First Laws

These are architectural laws, not guidelines:

```
1. ALL data displayed in UI must read from Drift (local SQLite) first
2. Network calls are for sync, not for rendering
3. Every user action that writes data goes through the Outbox
4. Never show a blank screen due to network error — show stale data + indicator
5. Every write operation generates a UUID v4 idempotency key at creation time
6. Stock deductions use CRDT delta merging — never absolute values
```

---

## Money Handling

```dart
// ✅ Always int (minor units) — never double for money
final int priceMinor = product.unitPriceMinor;  // 15000 = Rp 15.000

// ✅ Display formatting
String formatMoney(int minor, String currency) {
  final amount = minor / 100;  // Only for display
  return NumberFormat.currency(locale: 'id_ID', symbol: 'Rp ').format(amount);
}

// ❌ NEVER
final double price = 15.00;  // Floating point errors in financial calculations
```

---

## State Management Hierarchy

```
Server data (API/sync result):
  → Write to Drift first
  → Riverpod provider reads from Drift via watch() (reactive)
  → UI rebuilds automatically

User action (add to cart, checkout):
  → Write to Drift optimistically
  → Queue in OutboxEvents table
  → Update Riverpod state (triggers rebuild)
  → SyncEngine picks up Outbox when online

UI-only state (modal open, selected tab):
  → useState in StatefulWidget OR simple Notifier
  → Never goes to DB or Outbox
```

---

## Feature Structure Rules

```
features/{name}/
├── data/
│   ├── datasources/
│   │   ├── {name}_local_datasource.dart    ← Drift queries
│   │   └── {name}_remote_datasource.dart   ← API calls
│   └── {name}_repository_impl.dart         ← Combines local + remote
├── domain/
│   ├── entities/
│   │   └── {entity}.dart                   ← Freezed models
│   └── {name}_repository.dart              ← Abstract interface
└── presentation/
    ├── screens/
    ├── widgets/
    └── providers/
        └── {name}_provider.dart            ← @riverpod notifiers
```

---

## Drift Schema Rules

```dart
// ✅ Money: always IntColumn (minor units)
IntColumn get unitPriceMinor => integer().named('unit_price_minor')();

// ✅ IDs: TextColumn (UUID v4 strings)
TextColumn get id => text()();

// ✅ Enums: use textEnum
TextColumn get status => textEnum<OrderStatus>()();

// ✅ Timestamps: DateTimeColumn
DateTimeColumn get createdAt => dateTime().named('created_at')();

// ✅ Enable WAL mode in beforeOpen for performance
await customStatement('PRAGMA journal_mode = WAL');
await customStatement('PRAGMA foreign_keys = ON');
```

---

## AsyncValue Usage Pattern

```dart
// Every data-fetching widget MUST handle all three states
Widget build(BuildContext context, WidgetRef ref) {
  final productsAsync = ref.watch(productsProvider);

  return productsAsync.when(
    loading: () => const ProductsSkeleton(),    // Always show skeleton
    error: (e, _) => AsyncErrorWidget(          // Never blank screen
      error: e,
      onRetry: () => ref.invalidate(productsProvider),
    ),
    data: (products) => products.isEmpty
        ? const EmptyProductsState()            // Always handle empty
        : ProductGrid(products: products),
  );
}
```

---

## Dart 3.12 Features to Use

```dart
// Pattern matching (switch expressions)
final message = switch (error) {
  ApiException e when e.isNetworkError => 'No internet',
  ApiException e => e.response.message,
  _ => 'Unknown error',
};

// Records
typedef ProductWithStock = (Product product, int stockQty);
final (product, stock) = await getProductWithStock(id);

// Sealed classes for domain states
sealed class SyncState {}
class SyncIdle extends SyncState {}
class SyncInProgress extends SyncState { final int total; }
class SyncCompleted extends SyncState { final int processed; }
class SyncFailed extends SyncState { final String error; }
```

---

## Hardware Integration

```dart
// Always use the adapter pattern — UI never knows about USB/Bluetooth
abstract class IReceiptPrinter {
  Future<void> print(List<int> escposBytes);
  Future<bool> isConnected();
}

// In provider:
@Riverpod(keepAlive: true)
IReceiptPrinter receiptPrinter(Ref ref) {
  final settings = ref.watch(printerSettingsProvider);
  return switch (settings.type) {
    PrinterType.bluetooth => BluetoothPrinterAdapter(settings.address),
    PrinterType.usb       => UsbPrinterAdapter(settings.vendorId),
    PrinterType.network   => NetworkPrinterAdapter(settings.host, settings.port),
  };
}
```

---

## Testing

```dart
// Riverpod testing pattern
test('addItem updates cart state', () async {
  final container = ProviderContainer(overrides: [
    appDatabaseProvider.overrideWith((_) => AppDatabase(NativeDatabase.memory())),
    syncEngineProvider.overrideWith((_) => FakeSyncEngine()),
  ]);

  final notifier = container.read(cartNotifierProvider.notifier);
  await notifier.addItem(TestData.burgerProduct);

  final state = await container.read(cartNotifierProvider.future);
  expect(state.items, hasLength(1));
  expect(state.items.first.productName, 'Burger');
});
```

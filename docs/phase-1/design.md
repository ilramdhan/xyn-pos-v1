# UI/UX Design System & Guidelines — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer

---

## 1. Design Principles

These 5 principles guide every UI/UX decision in xyn-pos:

1. **Speed over Completeness** — A cashier processes 200+ transactions per day. Every extra tap/click costs real money. POS screens must be optimized for the fastest possible workflow.
2. **Forgiving** — Hardware fails. Network drops. Inputs get mistyped. The UI must degrade gracefully and prevent data loss.
3. **Glanceable** — A cashier glances at the screen while talking to a customer. Information hierarchy must be clear at a distance.
4. **Consistent** — The same action in two different parts of the app must look and feel the same. Muscle memory is a feature.
5. **Accessible** — WCAG 2.1 AA compliance minimum. High contrast for bright retail environments.

---

## 2. Design System: shadcn/ui Foundation

### 2.1 Why shadcn/ui

shadcn/ui is not a component library you install as a dependency — it's a collection of copy-pasteable, fully-owned components built on Radix UI primitives and styled with Tailwind CSS. This means:

- **Zero vendor lock-in**: you own the component source code
- **Full customization**: modify any component without fighting a library's opinion
- **Accessible by default**: Radix UI primitives handle ARIA, keyboard navigation, and focus management
- **Consistent theming**: a single CSS variable theme drives every component

### 2.2 Color System (CSS Variables)

```css
/* apps/web/src/styles/globals.css */
:root {
  /* Brand */
  --color-brand-50:  #eff6ff;
  --color-brand-500: #3b82f6;
  --color-brand-600: #2563eb;
  --color-brand-900: #1e3a8a;

  /* Semantic (mapped to brand for light mode) */
  --color-primary:        var(--color-brand-600);
  --color-primary-hover:  var(--color-brand-700);
  --color-primary-fg:     #ffffff;

  /* Surface */
  --color-background:     #ffffff;
  --color-surface:        #f8fafc;
  --color-surface-raised: #ffffff;
  --color-border:         #e2e8f0;

  /* Status */
  --color-success:   #16a34a;
  --color-warning:   #d97706;
  --color-danger:    #dc2626;
  --color-info:      #0284c7;

  /* Text */
  --color-text-primary:   #0f172a;
  --color-text-secondary: #475569;
  --color-text-muted:     #94a3b8;
  --color-text-disabled:  #cbd5e1;
}

.dark {
  --color-background:     #0f172a;
  --color-surface:        #1e293b;
  --color-surface-raised: #334155;
  --color-border:         #334155;
  --color-text-primary:   #f8fafc;
  --color-text-secondary: #94a3b8;
}
```

**Rule:** Never use hardcoded hex colors in component code. Always reference CSS variables via Tailwind's `text-primary`, `bg-surface`, etc.

### 2.3 Typography Scale

```css
/* Font: Inter (system fallback: system-ui) */
--font-sans: 'Inter', system-ui, -apple-system, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;

/* Scale */
--text-xs:   0.75rem / 1rem       /* 12px — labels, badges */
--text-sm:   0.875rem / 1.25rem   /* 14px — table cells, helper text */
--text-base: 1rem / 1.5rem        /* 16px — body copy */
--text-lg:   1.125rem / 1.75rem   /* 18px — subheadings */
--text-xl:   1.25rem / 1.75rem    /* 20px — section titles */
--text-2xl:  1.5rem / 2rem        /* 24px — page titles */
--text-3xl:  1.875rem / 2.25rem   /* 30px — display */
```

### 2.4 Spacing System

Use Tailwind's default spacing scale (4px base unit):

```
1 = 4px, 2 = 8px, 3 = 12px, 4 = 16px, 6 = 24px, 8 = 32px, 12 = 48px, 16 = 64px
```

Component internal padding: `p-3` (12px) or `p-4` (16px).  
Section gaps: `gap-4` (16px) or `gap-6` (24px).  
Page margins: `px-6` (24px) desktop, `px-4` (16px) mobile.

### 2.5 Component Catalog

Core components to build or configure from shadcn:

```
Atoms
├── Button (primary, secondary, outline, ghost, destructive)
├── Input (text, number, search)
├── Badge (status variants: paid, pending, cancelled)
├── Avatar
├── Spinner / Skeleton
└── Tooltip

Molecules
├── FormField (label + input + error message)
├── DataTable (with sorting, filtering, pagination)
├── CommandMenu (⌘K global search)
├── DatePicker
├── CurrencyInput (formatted decimal input with locale support)
└── StatusBadge (order status with color coding)

Organisms
├── Sidebar (collapsible, with nav items)
├── TopBar (breadcrumbs, notifications, user menu)
├── OrderCard (POS cart item)
├── ProductGrid (POS product selector)
├── PaymentModal (checkout flow)
├── ReceiptPreview
└── KDSTicket (Kitchen Display card)
```

---

## 3. POS Interface Layout

The POS screen is the most performance-critical screen in the app. Layout is optimized for single-screen operation without scrolling.

### 3.1 Desktop POS Layout (1920×1080)

```
┌──────────────────────────────────────────────────────────────────────────┐
│ TopBar: Branch Name | Cashier | Shift Status | Clock                     │
├──────────────────────────────────────────────┬───────────────────────────┤
│                                              │  Cart Panel               │
│  Product Panel (left 65%)                   │  ┌─────────────────────┐  │
│  ┌─────────────────────────────────────────┐ │  │ Table: T-01         │  │
│  │  🔍 Search / Scan Barcode               │ │  │ Cashier: John       │  │
│  └─────────────────────────────────────────┘ │  ├─────────────────────┤  │
│                                              │  │ Item 1       x2 $8  │  │
│  [Categories row: All | Food | Drinks | ...]│  │ Item 2       x1 $4  │  │
│                                              │  │ ...                 │  │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐  │  ├─────────────────────┤  │
│  │  🍔   │ │  🥤   │ │  🍟   │ │  🍜   │  │  │ Subtotal:    $22.00 │  │
│  │Burger │ │ Cola  │ │ Fries │ │Noodle │  │  │ Tax (11%):    $2.42 │  │
│  │$5.00  │ │$2.00  │ │$3.00  │ │$7.00  │  │  │ Discount:    -$2.00 │  │
│  └───────┘ └───────┘ └───────┘ └───────┘  │  │ TOTAL:       $22.42 │  │
│                                              │  ├─────────────────────┤  │
│  [Product grid continues...]                │  │ [CHECKOUT →]        │  │
│                                              │  └─────────────────────┘  │
└──────────────────────────────────────────────┴───────────────────────────┘
```

### 3.2 Keyboard Shortcuts (POS Critical Path)

| Action | Shortcut | Notes |
|---|---|---|
| Focus search | `/` or `F3` | Start scanning/typing |
| Quick checkout | `F12` | Opens payment modal |
| Add item by barcode | Auto (scanner) | No key press needed |
| Increase qty | `+` when item focused | |
| Decrease qty | `-` when item focused | |
| Remove item | `Del` when item focused | Confirmation for paid items |
| New order / clear | `Ctrl+N` | |
| Open drawer | `Ctrl+D` | Requires cashier PIN |
| Open calculator | `Ctrl+K` | For cash payment change |

### 3.3 Payment Modal Flow

```
Checkout triggered
       │
       ▼
┌──────────────────────────────────────┐
│ Payment Modal                         │
│                                       │
│ Total: $22.42                         │
│                                       │
│ Payment Method:                       │
│ [💵 Cash] [💳 Card] [📱 QRIS] [+Split]│
│                                       │
│ [Cash selected]                       │
│ Amount received: [    $25.00       ] │
│ Change:               $2.58           │
│                                       │
│         [Cancel]  [✓ CONFIRM PAYMENT]│
└──────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ ✅ Payment Successful!                │
│                                       │
│ Order #ORD-2024-001234               │
│ Total: $22.42 | Change: $2.58        │
│                                       │
│ [Print Receipt] [New Order] [Done]   │
└──────────────────────────────────────┘
```

---

## 4. Dashboard Layout (SaaS Admin)

```
┌─────────────────────────────────────────────────────────────────────┐
│  Sidebar (64px collapsed / 240px expanded)                          │
│  ┌──────┐                                                           │
│  │ Logo │                                                           │
│  ├──────┤  ┌────────────────────────────────────────────────────┐  │
│  │ 🏠   │  │  Page Content                                       │  │
│  │ 📊   │  │  ┌──────────────────────────────────────────────┐  │  │
│  │ 🛒   │  │  │  Breadcrumb > Page Title                      │  │  │
│  │ 📦   │  │  └──────────────────────────────────────────────┘  │  │
│  │ 👥   │  │                                                     │  │
│  │ 📈   │  │  [KPI Cards row]                                    │  │
│  │ ⚙️   │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐│  │
│  └──────┘  │  │ Revenue │ │ Orders  │ │ Avg Sale│ │ Stock   ││  │
│            │  │ $12,450 │ │   142   │ │  $87.6  │ │ Alerts 3││  │
│            │  └─────────┘ └─────────┘ └─────────┘ └─────────┘│  │
│            │                                                     │  │
│            │  [Charts / Tables / Feature-specific content]       │  │
│            └────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Mobile Design System (Flutter)

### 5.1 Design Tokens

Use `flutter_gen` to generate Dart constants from design tokens:

```dart
// lib/core/theme/app_colors.dart (generated from design tokens)
abstract class AppColors {
  static const brandPrimary   = Color(0xFF2563EB);
  static const brandPrimaryFg = Color(0xFFFFFFFF);
  static const success        = Color(0xFF16A34A);
  static const warning        = Color(0xFFD97706);
  static const danger         = Color(0xFFDC2626);
  static const textPrimary    = Color(0xFF0F172A);
  static const textSecondary  = Color(0xFF475569);
  static const surface        = Color(0xFFF8FAFC);
  static const border         = Color(0xFFE2E8F0);
}
```

### 5.2 Typography

```dart
abstract class AppTextStyles {
  static const headline1 = TextStyle(
    fontSize: 24, fontWeight: FontWeight.w700, color: AppColors.textPrimary,
  );
  static const headline2 = TextStyle(
    fontSize: 20, fontWeight: FontWeight.w600, color: AppColors.textPrimary,
  );
  static const body = TextStyle(
    fontSize: 16, fontWeight: FontWeight.w400, color: AppColors.textPrimary,
  );
  static const caption = TextStyle(
    fontSize: 12, fontWeight: FontWeight.w400, color: AppColors.textSecondary,
  );
  static const moneyLarge = TextStyle(
    fontSize: 28, fontWeight: FontWeight.w700, color: AppColors.textPrimary,
    fontFeatures: [FontFeature.tabularFigures()],  // Aligned decimal points
  );
}
```

**Tabular figures** (`fontFeature.tabularFigures`) are required for all currency displays so digits align vertically in lists.

### 5.3 Mobile POS Screen Layout

```
┌────────────────────────────┐
│  Status bar (safe area)    │
├────────────────────────────┤
│  Search / Scan Bar    [📷] │
├────────────────────────────┤
│  Category chips (scroll >) │
│  [All] [Food] [Drinks]...  │
├────────────────────────────┤
│  Product grid (2 columns)  │
│  ┌──────────┐┌──────────┐ │
│  │  Image   ││  Image   │ │
│  │  Name    ││  Name    │ │
│  │  $5.00   ││  $3.00   │ │
│  └──────────┘└──────────┘ │
│  [more products...]        │
├────────────────────────────┤
│  Cart Preview (collapsible)│
│  3 items — $22.42          │
│  [View Cart] [CHECKOUT →]  │
└────────────────────────────┘
```

---

## 6. Responsive Breakpoints

| Breakpoint | Width | Target Device |
|---|---|---|
| `xs` | < 480px | Mobile phones |
| `sm` | 480–767px | Large phones, small tablets |
| `md` | 768–1023px | Tablets, small laptops |
| `lg` | 1024–1279px | Laptops, desktop POS terminals |
| `xl` | 1280–1535px | Large desktops |
| `2xl` | ≥ 1536px | Very large displays, TV dashboards |

POS interface is designed for `lg` and above. Dashboard is responsive across all breakpoints.

---

## 7. Accessibility Requirements

- **Color contrast**: All text must meet WCAG AA (4.5:1 for normal text, 3:1 for large text)
- **Focus indicators**: Visible focus ring on all interactive elements (especially important for keyboard-only POS workflows)
- **Screen reader**: All images have `alt` text; icons have `aria-label`; status changes are announced via `aria-live`
- **Touch targets**: Minimum 44×44px for all tappable elements (mobile)
- **Error messages**: Associated with their input via `aria-describedby`; don't rely on color alone

---

## 8. Loading & Error States

Every data-fetching component must handle 3 states:

| State | UI Pattern |
|---|---|
| **Loading** | Skeleton screens (not spinners for content areas) |
| **Error** | Inline error with retry button; never a full-page error for partial failures |
| **Empty** | Illustrated empty state with a clear call-to-action |

```tsx
// ✅ Required pattern for data components
function ProductGrid() {
  const { data, isLoading, isError, refetch } = useProducts();

  if (isLoading) return <ProductGridSkeleton />;
  if (isError) return <ErrorState onRetry={refetch} message="Could not load products" />;
  if (data?.length === 0) return <EmptyState title="No products" action={<AddProductButton />} />;

  return <Grid>{data.map(p => <ProductCard key={p.id} product={p} />)}</Grid>;
}
```

---

## 9. Offline State Indicators (Mobile)

When the device is offline, the mobile POS must:
1. Show a persistent banner: "Offline — changes will sync when connected"
2. Continue to function fully for order creation and cash payment
3. Disable payment methods that require connectivity (QRIS, card payment gateway)
4. Show a sync progress indicator when reconnected ("Syncing 3 orders...")

```dart
class OfflineBanner extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isOnline = ref.watch(connectivityProvider);

    if (isOnline) return const SizedBox.shrink();

    return Container(
      color: AppColors.warning,
      padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 16),
      child: Row(
        children: [
          const Icon(Icons.wifi_off, size: 16, color: Colors.white),
          const SizedBox(width: 8),
          Text('Offline mode — transactions will sync when connected',
              style: AppTextStyles.caption.copyWith(color: Colors.white)),
        ],
      ),
    );
  }
}
```

---

## 10. KDS (Kitchen Display System) Design

The KDS screen displays in a kitchen environment — bright, noisy, viewed at a distance. Design requirements differ from the standard POS UI:

- **Minimum font size**: 18px for item names, 24px for table/order numbers
- **High contrast**: Dark background (#0f172a) with white text — reduces glare
- **Color-coded ticket age**: Green (< 5 min) → Yellow (5–10 min) → Red (> 10 min)
- **Touch targets**: 60×60px minimum (kitchen staff wear gloves)
- **No scrolling**: All active tickets visible simultaneously; paginate if needed

```
┌────────────────────────────────────────────────────────────────┐
│  KDS — Branch: Main Kitchen              2024-01-15  14:23:01  │
├─────────────────┬─────────────────┬─────────────────┬──────────┤
│  TABLE 3  🟢    │  TABLE 7  🟡    │  TABLE 1  🔴    │  TAKE-  │
│  ORD-1234       │  ORD-1235       │  ORD-1230       │  AWAY   │
│  ─────────────  │  ─────────────  │  ─────────────  │  ──────  │
│  2x Burger      │  1x Noodle      │  3x Pizza       │ 1x Wrap  │
│    NO ONION     │  1x Sprite      │    EXTRA CHEESE │          │
│  1x Cola        │                 │  2x Garlic Bread│          │
│                 │                 │                 │          │
│  [✓ DONE]       │  [✓ DONE]       │  [✓ DONE]       │[✓ DONE] │
└─────────────────┴─────────────────┴─────────────────┴──────────┘
```

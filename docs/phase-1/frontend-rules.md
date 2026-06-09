# Frontend Rules — xyn-pos-v1

> Scope: apps/web/ (Next.js 16) | apps/mobile/ (Flutter)  
> Stack: Next.js 16.2.7 | React 19.2 | TypeScript 5.8+ | Tailwind CSS 4.3 | shadcn/ui

---

## 1. Golden Rules (Never Violate)

```
❌ NEVER use `any` type — use `unknown` + type guard, or a proper type
❌ NEVER fetch data in useEffect — use TanStack Query v5
❌ NEVER put server state in Zustand — Zustand is for local/UI state only
❌ NEVER use <img> — use next/image
❌ NEVER hardcode colors, spacing, or font sizes — use Tailwind CSS variables
❌ NEVER create a component without exporting it from index.ts
❌ NEVER write business logic in a component — extract to a hook or service file
❌ NEVER build a new component when a shadcn/ui or shared component exists
❌ NEVER skip loading + error + empty state handling in data-fetching components
❌ NEVER use the pages/ directory — App Router only
❌ NEVER @ts-ignore — use @ts-expect-error with an explanatory comment
❌ NEVER inline event handlers for complex logic — extract named handlers
```

```
✅ ALWAYS use Server Components by default; add 'use client' only when needed
✅ ALWAYS wrap API calls in TanStack Query (useQuery / useMutation)
✅ ALWAYS use React Hook Form for forms with 2+ fields
✅ ALWAYS add a test file next to every component (ComponentName.test.tsx)
✅ ALWAYS handle all 3 data states: isLoading → skeleton, isError → ErrorState, empty → EmptyState
✅ ALWAYS use cn() for conditional className merging (never string concatenation)
✅ ALWAYS add aria-label to icon-only buttons
✅ ALWAYS use the shared formatCurrency() for money display (never raw number formatting)
✅ ALWAYS export from index.ts in feature folders
```

---

## 2. Component Architecture

### 2.1 Three-Layer Component Model

```
apps/web/src/components/
├── ui/           ← shadcn/ui primitives (Button, Input, Card, Dialog, etc.)
│                   Rule: NEVER edit these — re-run `npx shadcn@latest add` to update
│
├── shared/       ← Domain-aware shared components built on top of ui/
│                   Rule: MUST be reusable across at least 2 feature areas
│                   Examples: StatusBadge, CurrencyInput, DataTableWrapper,
│                             StatCard, ErrorState, EmptyState, SkeletonTable
│
└── features/     ← Feature-specific components (composed from ui/ + shared/)
                    Structure: features/{domain}/{ComponentName}/
                    Rule: NOT reusable outside their domain
                    Examples: features/pos/ProductGrid, features/dashboard/SalesSummary
```

### 2.2 Feature Component Folder Structure

Every non-trivial component lives in its own folder:

```
features/pos/ProductGrid/
├── index.ts              ← Re-exports only: export { ProductGrid } from './ProductGrid'
├── ProductGrid.tsx       ← The component
├── ProductGrid.test.tsx  ← Co-located test
└── ProductCard.tsx       ← Sub-component (no own test if simple)
```

Simple, single-file components (< 50 lines, no sub-components) can be a single file without a folder.

### 2.3 'use client' Decision Tree

```
Is the component using any of these?
├── useState / useReducer / useContext     → 'use client'
├── useEffect / useLayoutEffect            → 'use client'
├── Browser APIs (window, navigator, etc.) → 'use client'
├── Event listeners (onClick, onSubmit...) → 'use client' if complex
└── TanStack Query hooks (useQuery, etc.)  → 'use client'

Otherwise → Server Component (no directive needed)
```

---

## 3. DRY & Reusability Rules

### 3.1 The Three-Times Rule

Build something generic only after you've needed it in 3 different places. Not 2. Not 1.

**Wrong:**
```tsx
// dashboard/SalesPage.tsx
function formatRp(amount: number) { return `Rp ${(amount / 100).toLocaleString('id-ID')}` }

// pos/CartPanel.tsx
function formatRupiah(amount: number) { return `Rp ${(amount / 100).toLocaleString('id-ID')}` }
```

**Right:**
```tsx
// lib/format.ts — one canonical function
export function formatCurrency(minorUnits: number, locale = 'id-ID', currency = 'IDR'): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minorUnits / 100);
}
```

### 3.2 Shared Component Catalog

These shared components MUST be used instead of re-implementing:

| Component | Location | Use for |
|---|---|---|
| `StatusBadge` | `shared/StatusBadge` | Order status, payment status, subscription status |
| `CurrencyInput` | `shared/CurrencyInput` | Any money input field |
| `StatCard` | `shared/StatCard` | KPI metric display cards |
| `DataTableWrapper` | `shared/DataTableWrapper` | Any sortable/paginated table |
| `ErrorState` | `shared/ErrorState` | Query error fallback |
| `EmptyState` | `shared/EmptyState` | Empty list/search results |
| `SkeletonCard` | `shared/SkeletonCard` | Card loading state |
| `SkeletonTable` | `shared/SkeletonTable` | Table loading state |
| `ConfirmDialog` | `shared/ConfirmDialog` | Destructive action confirmation |
| `PageHeader` | `layout/PageHeader` | Every page title + breadcrumb |

### 3.3 Props Pattern

```tsx
// ✅ Use interface, not type for component props
interface ProductCardProps {
  product: ProductDTO;
  onAdd: (productId: string) => void;
  className?: string;    // always accept className for composition
}

// ✅ Spread className with cn()
export function ProductCard({ product, onAdd, className }: ProductCardProps) {
  return (
    <Card className={cn("cursor-pointer hover:shadow-md transition-shadow", className)}>
      ...
    </Card>
  );
}
```

---

## 4. State Management Rules

### 4.1 TanStack Query v5 — Server State

```typescript
// services/orders.ts — one file per domain
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export const orderKeys = {
  all: ['orders'] as const,
  list: (filters: OrderFilters) => [...orderKeys.all, 'list', filters] as const,
  detail: (id: string) => [...orderKeys.all, 'detail', id] as const,
};

export function useOrders(filters: OrderFilters) {
  return useQuery({
    queryKey: orderKeys.list(filters),
    queryFn: () => apiFetch<ListResponse<OrderDTO>>('/v1/orders', { params: filters }),
    staleTime: 30_000,
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateOrderRequest) =>
      apiFetch<DataResponse<OrderDTO>>('/v1/orders', { method: 'POST', body: req }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: orderKeys.all }),
  });
}
```

### 4.2 Zustand v5 — Local/UI State

```typescript
// store/pos.store.ts
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface POSState {
  activeOrderId: string | null;
  tableNumber: string | null;
  shiftId: string | null;
  setActiveOrder: (orderId: string, table: string) => void;
  clearOrder: () => void;
}

export const usePOSStore = create<POSState>()(
  devtools(
    (set) => ({
      activeOrderId: null,
      tableNumber: null,
      shiftId: null,
      setActiveOrder: (orderId, table) => set({ activeOrderId: orderId, tableNumber: table }),
      clearOrder: () => set({ activeOrderId: null, tableNumber: null }),
    }),
    { name: 'pos-store' },
  ),
);
```

**Rule:** If data comes from an API, it belongs in TanStack Query — not Zustand.

---

## 5. Form Rules

### 5.1 React Hook Form + Zod

All forms with 2+ fields MUST use React Hook Form + Zod:

```typescript
// schemas/auth.schema.ts
import { z } from 'zod';

export const loginSchema = z.object({
  email: z.string().email('Email tidak valid'),
  password: z.string().min(8, 'Password minimal 8 karakter'),
});

export type LoginFormData = z.infer<typeof loginSchema>;
```

```tsx
// features/auth/LoginForm.tsx
'use client';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

export function LoginForm() {
  const form = useForm<LoginFormData>({ resolver: zodResolver(loginSchema) });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Email</FormLabel>
              <FormControl><Input {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </form>
    </Form>
  );
}
```

---

## 6. Testing Rules

### 6.1 Coverage Targets

| Layer | Tool | Target |
|---|---|---|
| Shared components | Vitest + Testing Library | 80% branch coverage |
| Feature components | Vitest + Testing Library | Happy path + error state |
| Hooks (useQuery wrappers) | Vitest + msw | Mock API, test loading/error/success |
| E2E critical paths | Playwright | POS checkout flow, Login flow |

### 6.2 Component Test Pattern

```tsx
// shared/StatusBadge/StatusBadge.test.tsx
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge', () => {
  it('renders paid status with correct color', () => {
    render(<StatusBadge status="paid" />);
    const badge = screen.getByText('Paid');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-success');
  });

  it('renders all order statuses without crashing', () => {
    const statuses = ['draft', 'pending_payment', 'paid', 'cancelled', 'parked'] as const;
    statuses.forEach(status => {
      const { unmount } = render(<StatusBadge status={status} />);
      unmount();
    });
  });
});
```

### 6.3 Test Utilities

```typescript
// test/utils.tsx — shared test helpers
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';

export function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}
```

---

## 7. Naming Conventions

| Entity | Convention | Example |
|---|---|---|
| Component files | PascalCase | `ProductCard.tsx` |
| Hook files | camelCase, `use` prefix | `useCartItems.ts` |
| Service files | camelCase | `orders.ts` |
| Store files | camelCase, `.store.ts` suffix | `pos.store.ts` |
| Schema files | camelCase, `.schema.ts` suffix | `login.schema.ts` |
| Test files | Same as source + `.test.tsx` | `ProductCard.test.tsx` |
| Route folders | kebab-case | `app/(dashboard)/order-history/` |
| Type DTOs | PascalCase + `DTO` suffix | `OrderDTO`, `TenantDTO` |

---

## 8. Import Rules

```typescript
// ✅ Correct import order (enforced by ESLint)
// 1. React/Next
import { useState } from 'react';
import { useRouter } from 'next/navigation';

// 2. Third-party
import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

// 3. Internal — absolute paths via @/ alias
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/shared/StatusBadge';
import { useOrders } from '@/services/orders';
import { formatCurrency } from '@/lib/format';

// 4. Relative — only for siblings in the same folder
import { ProductCard } from './ProductCard';
```

**Rule:** Never use relative imports that traverse more than one directory level (`../../`).

---

## 9. Responsive Design Rules

```
Mobile-first breakpoints:
- Default (< 640px): Mobile POS, stacked layouts
- sm (640px+):       Tablet landscape
- md (768px+):       Compact sidebar appears
- lg (1024px+):      Full POS two-panel layout, full sidebar
- xl (1280px+):      Wider product grid, more columns
- 2xl (1536px+):     Dashboard TV mode, KDS full-screen

POS terminal: lg+ only (no phone support — hardware terminal)
Dashboard: All breakpoints (managers use mobile too)
KDS: lg+ only (kitchen display, always big screen)
```

```tsx
// ✅ Mobile-first Tailwind classes
<div className="flex flex-col lg:flex-row gap-4">
  <main className="flex-1 min-w-0">...</main>
  <aside className="w-full lg:w-80 shrink-0">...</aside>
</div>
```

---

## 10. Performance Rules

```tsx
// ✅ Dynamic import for heavy components
const PaymentModal = dynamic(() => import('@/components/features/pos/PaymentModal'), {
  loading: () => <Skeleton className="h-96 w-full" />,
});

// ✅ Memoize stable callbacks in event-heavy components
const handleAddItem = useCallback((productId: string) => {
  addItemMutation.mutate({ orderId, productId });
}, [orderId, addItemMutation]);

// ✅ Memoize expensive renders
const sortedItems = useMemo(() => [...items].sort((a, b) => a.name.localeCompare(b.name)), [items]);
```

---

## 11. Money Display Rules

All money values from the backend are **int64 minor units (sen)**. Display rules:

```typescript
// lib/format.ts
export function formatCurrency(sen: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(sen / 100);
}
// formatCurrency(150000) → "Rp 1.500"
// formatCurrency(1234567) → "Rp 12.346"

// ❌ Never:
<span>{item.price / 100}</span>         // raw number, no formatting
<span>Rp {item.price}</span>            // wrong unit + no formatting

// ✅ Always:
<span>{formatCurrency(item.price)}</span>
```

---

## 12. CI Requirements

Every PR touching `apps/web/` must pass:
1. `pnpm typecheck` — zero TypeScript errors
2. `pnpm lint` — zero ESLint errors  
3. `pnpm test` — all Vitest tests green
4. `pnpm build` — production build succeeds

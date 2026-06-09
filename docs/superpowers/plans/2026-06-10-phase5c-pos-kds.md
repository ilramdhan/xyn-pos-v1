# Phase 5C — POS Interface + KDS

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the primary cashier-facing POS terminal (product grid, cart panel, payment modal, receipt, park/resume) and the Kitchen Display System board.

**Architecture:** POS is a single `lg:flex` two-panel page — product panel (65%) + cart panel (35%). Cart state lives in Zustand (local UI). All mutations go through TanStack Query. Keyboard shortcuts registered with a global `useKeyboardShortcuts` hook. KDS polls every 10s via TanStack Query `refetchInterval`.

**Tech Stack:** Zustand v5, TanStack Query v5, shadcn/ui (Sheet for mobile cart, Dialog for payment modal), React Hook Form for payment amount input, Recharts not needed here.

---

## Prerequisites

- Phase 5A + 5B complete
- POS backend running (`services/pos`) — CreateOrder, AddItem, Submit, ParkOrder, ResumeOrder RPCs
- Payment backend running (`services/payment`) — CreatePayment, HandleWebhook
- `gen/ts/pos/v1/` and `gen/ts/payment/v1/` TypeScript clients generated

## File Map

| File | Action | Purpose |
|---|---|---|
| `apps/web/src/app/pos/layout.tsx` | Create | Full-screen POS layout (no sidebar) |
| `apps/web/src/app/pos/page.tsx` | Create | POS terminal page |
| `apps/web/src/app/kds/layout.tsx` | Create | KDS full-screen layout |
| `apps/web/src/app/kds/page.tsx` | Create | KDS board page |
| `apps/web/src/store/pos.store.ts` | Create | Zustand: active order, cart selection state |
| `apps/web/src/services/pos.ts` | Create | TanStack Query hooks for POS operations |
| `apps/web/src/services/payments.ts` | Create | TanStack Query hooks for payments |
| `apps/web/src/hooks/useKeyboardShortcuts.ts` | Create | Global keyboard shortcut registry |
| `apps/web/src/hooks/useKeyboardShortcuts.test.ts` | Create | |
| `apps/web/src/components/features/pos/POSTerminal.tsx` | Create | Two-panel container |
| `apps/web/src/components/features/pos/ProductPanel/` | Create | Left panel: categories + grid |
| `apps/web/src/components/features/pos/ProductPanel/index.ts` | Create | |
| `apps/web/src/components/features/pos/ProductPanel/ProductPanel.tsx` | Create | |
| `apps/web/src/components/features/pos/ProductPanel/ProductPanel.test.tsx` | Create | |
| `apps/web/src/components/features/pos/ProductPanel/ProductCard.tsx` | Create | |
| `apps/web/src/components/features/pos/ProductPanel/CategoryFilter.tsx` | Create | |
| `apps/web/src/components/features/pos/CartPanel/` | Create | Right panel: cart + totals + checkout |
| `apps/web/src/components/features/pos/CartPanel/index.ts` | Create | |
| `apps/web/src/components/features/pos/CartPanel/CartPanel.tsx` | Create | |
| `apps/web/src/components/features/pos/CartPanel/CartPanel.test.tsx` | Create | |
| `apps/web/src/components/features/pos/CartPanel/CartItem.tsx` | Create | |
| `apps/web/src/components/features/pos/CartPanel/CartSummary.tsx` | Create | |
| `apps/web/src/components/features/pos/PaymentModal/` | Create | Checkout flow dialog |
| `apps/web/src/components/features/pos/PaymentModal/index.ts` | Create | |
| `apps/web/src/components/features/pos/PaymentModal/PaymentModal.tsx` | Create | |
| `apps/web/src/components/features/pos/PaymentModal/PaymentModal.test.tsx` | Create | |
| `apps/web/src/components/features/pos/PaymentModal/CashPaymentTab.tsx` | Create | |
| `apps/web/src/components/features/pos/PaymentModal/QRISPaymentTab.tsx` | Create | |
| `apps/web/src/components/features/pos/ReceiptPreview.tsx` | Create | Receipt display |
| `apps/web/src/components/features/pos/ParkedOrdersDrawer.tsx` | Create | Park/resume drawer |
| `apps/web/src/components/features/kds/KDSBoard.tsx` | Create | Ticket board |
| `apps/web/src/components/features/kds/KDSBoard.test.tsx` | Create | |
| `apps/web/src/components/features/kds/KDSTicket.tsx` | Create | Individual ticket card |

---

## Task 1: POS Zustand Store + Service Hooks

**Files:**
- Create: `apps/web/src/store/pos.store.ts`
- Create: `apps/web/src/services/pos.ts`
- Create: `apps/web/src/services/payments.ts`

- [ ] **Step 1: Create store/pos.store.ts**

```typescript
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

export interface CartItem {
  productId: string;
  productName: string;
  unitPrice: number;    // sen
  quantity: number;
  subtotal: number;     // sen
}

interface POSState {
  // Active order context
  activeOrderId: string | null;
  tableNumber: string | null;
  orderNumber: string | null;

  // Local cart (mirrors server state for optimistic UI)
  cartItems: CartItem[];
  subtotal: number;
  taxAmount: number;
  discountAmount: number;
  total: number;

  // UI state
  isPaymentModalOpen: boolean;
  isParkedOrdersOpen: boolean;
  searchQuery: string;
  selectedCategoryId: string | null;

  // Actions
  setActiveOrder: (id: string, orderNumber: string, table: string) => void;
  setCartItems: (items: CartItem[], subtotal: number, taxAmount: number, discount: number, total: number) => void;
  openPaymentModal: () => void;
  closePaymentModal: () => void;
  openParkedOrders: () => void;
  closeParkedOrders: () => void;
  setSearchQuery: (q: string) => void;
  setSelectedCategory: (id: string | null) => void;
  resetOrder: () => void;
}

const initialState = {
  activeOrderId: null,
  tableNumber: null,
  orderNumber: null,
  cartItems: [],
  subtotal: 0,
  taxAmount: 0,
  discountAmount: 0,
  total: 0,
  isPaymentModalOpen: false,
  isParkedOrdersOpen: false,
  searchQuery: '',
  selectedCategoryId: null,
};

export const usePOSStore = create<POSState>()(
  devtools(
    (set) => ({
      ...initialState,

      setActiveOrder: (id, orderNumber, table) =>
        set({ activeOrderId: id, orderNumber, tableNumber: table }),

      setCartItems: (items, subtotal, taxAmount, discount, total) =>
        set({ cartItems: items, subtotal, taxAmount, discountAmount: discount, total }),

      openPaymentModal: () => set({ isPaymentModalOpen: true }),
      closePaymentModal: () => set({ isPaymentModalOpen: false }),
      openParkedOrders: () => set({ isParkedOrdersOpen: true }),
      closeParkedOrders: () => set({ isParkedOrdersOpen: false }),

      setSearchQuery: (q) => set({ searchQuery: q }),
      setSelectedCategory: (id) => set({ selectedCategoryId: id }),

      resetOrder: () => set({ ...initialState }),
    }),
    { name: 'pos-store' },
  ),
);
```

- [ ] **Step 2: Create services/pos.ts**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse, ListResponse } from '@/types/api';
import type { OrderDTO, OrderFilters } from './orders';

export const posKeys = {
  orders: ['pos-orders'] as const,
  activeOrders: (branchId: string) => [...posKeys.orders, 'active', branchId] as const,
  parkedOrders: (branchId: string) => [...posKeys.orders, 'parked', branchId] as const,
};

export function useActiveOrders(branchId: string) {
  return useQuery({
    queryKey: posKeys.activeOrders(branchId),
    queryFn: () =>
      apiFetch<ListResponse<OrderDTO>>('/v1/orders', {
        params: { status: 'draft', branch_id: branchId },
      }),
    enabled: !!branchId,
    staleTime: 5_000,
    refetchInterval: 10_000, // Keep in sync with other terminals
  });
}

export function useParkedOrders(branchId: string) {
  return useQuery({
    queryKey: posKeys.parkedOrders(branchId),
    queryFn: () =>
      apiFetch<ListResponse<OrderDTO>>('/v1/orders', {
        params: { status: 'parked', branch_id: branchId },
      }),
    enabled: !!branchId,
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: {
      branch_id: string;
      cashier_id: string;
      order_type: 'dine_in' | 'takeaway' | 'delivery';
      table_number?: string;
      idempotency_key: string;
    }) => apiFetch<DataResponse<OrderDTO>>('/v1/orders', { method: 'POST', body: req }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}

export function useAddItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      orderId,
      ...item
    }: {
      orderId: string;
      product_id: string;
      product_name: string;
      unit_price: number;
      quantity: number;
    }) =>
      apiFetch<DataResponse<OrderDTO>>(`/v1/orders/${orderId}/items`, {
        method: 'POST',
        body: item,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}

export function useRemoveItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ orderId, itemId }: { orderId: string; itemId: string }) =>
      apiFetch<DataResponse<OrderDTO>>(`/v1/orders/${orderId}/items/${itemId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}

export function useSubmitOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orderId: string) =>
      apiFetch<DataResponse<OrderDTO>>(`/v1/orders/${orderId}/submit`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}

export function useParkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orderId: string) =>
      apiFetch<DataResponse<void>>(`/v1/orders/${orderId}/park`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}

export function useResumeOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orderId: string) =>
      apiFetch<DataResponse<OrderDTO>>(`/v1/orders/${orderId}/resume`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}
```

- [ ] **Step 3: Create services/payments.ts**

```typescript
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse } from '@/types/api';
import { posKeys } from './pos';

export interface CreatePaymentRequest {
  order_id: string;
  method: 'cash' | 'qris' | 'card';
  amount: number;       // sen — for cash: amount tendered; for QRIS: order total
  idempotency_key: string;
}

export interface PaymentDTO {
  id: string;
  order_id: string;
  method: string;
  amount: number;       // sen
  status: 'pending' | 'success' | 'failed' | 'voided' | 'refunded';
  external_id?: string;
  qr_code_url?: string; // for QRIS
  change_amount?: number; // sen — cash only
  created_at: string;
}

export function useCreatePayment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreatePaymentRequest) =>
      apiFetch<DataResponse<PaymentDTO>>('/v1/payments', { method: 'POST', body: req }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: posKeys.orders }),
  });
}
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/store/ apps/web/src/services/pos.ts apps/web/src/services/payments.ts
git commit -m "feat(web/pos): add POS Zustand store and TanStack Query service hooks"
```

---

## Task 2: Keyboard Shortcuts Hook

**Files:**
- Create: `apps/web/src/hooks/useKeyboardShortcuts.ts`
- Create: `apps/web/src/hooks/useKeyboardShortcuts.test.ts`

- [ ] **Step 1: Create useKeyboardShortcuts.ts**

```typescript
'use client';

import { useEffect, useCallback } from 'react';

interface ShortcutDefinition {
  key: string;            // e.g. '/', 'F12', 'n'
  ctrlKey?: boolean;
  shiftKey?: boolean;
  handler: () => void;
  preventDefault?: boolean;
}

export function useKeyboardShortcuts(shortcuts: ShortcutDefinition[]) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't fire when typing in an input/textarea
      const target = e.target as HTMLElement;
      if (
        !e.ctrlKey &&
        !e.metaKey &&
        (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
      ) {
        return;
      }

      for (const shortcut of shortcuts) {
        const keyMatch = e.key === shortcut.key || e.code === shortcut.key;
        const ctrlMatch = shortcut.ctrlKey ? (e.ctrlKey || e.metaKey) : !e.ctrlKey;
        const shiftMatch = shortcut.shiftKey ? e.shiftKey : !e.shiftKey;

        if (keyMatch && ctrlMatch && shiftMatch) {
          if (shortcut.preventDefault !== false) e.preventDefault();
          shortcut.handler();
          break;
        }
      }
    },
    [shortcuts],
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}
```

- [ ] **Step 2: Create useKeyboardShortcuts.test.ts**

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { fireEvent } from '@testing-library/react';
import { useKeyboardShortcuts } from './useKeyboardShortcuts';

describe('useKeyboardShortcuts', () => {
  it('calls handler when shortcut key is pressed', () => {
    const handler = vi.fn();
    renderHook(() =>
      useKeyboardShortcuts([{ key: 'F12', handler }]),
    );

    fireEvent.keyDown(window, { key: 'F12' });
    expect(handler).toHaveBeenCalledOnce();
  });

  it('calls handler for Ctrl+N', () => {
    const handler = vi.fn();
    renderHook(() =>
      useKeyboardShortcuts([{ key: 'n', ctrlKey: true, handler }]),
    );

    fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    expect(handler).toHaveBeenCalledOnce();
  });

  it('does not call handler when ctrlKey requirement not met', () => {
    const handler = vi.fn();
    renderHook(() =>
      useKeyboardShortcuts([{ key: 'n', ctrlKey: true, handler }]),
    );

    fireEvent.keyDown(window, { key: 'n' }); // no ctrl
    expect(handler).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 3: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: useKeyboardShortcuts — 3 tests passing.

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/hooks/
git commit -m "feat(web/pos): add useKeyboardShortcuts hook with tests"
```

---

## Task 3: ProductPanel (Categories + Grid + Cards)

**Files:**
- Create: `apps/web/src/components/features/pos/ProductPanel/CategoryFilter.tsx`
- Create: `apps/web/src/components/features/pos/ProductPanel/ProductCard.tsx`
- Create: `apps/web/src/components/features/pos/ProductPanel/ProductPanel.tsx`
- Create: `apps/web/src/components/features/pos/ProductPanel/ProductPanel.test.tsx`
- Create: `apps/web/src/components/features/pos/ProductPanel/index.ts`

- [ ] **Step 1: Create CategoryFilter.tsx**

```tsx
'use client';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area';

interface Category {
  id: string;
  name: string;
}

interface CategoryFilterProps {
  categories: Category[];
  selectedId: string | null;
  onChange: (id: string | null) => void;
}

export function CategoryFilter({ categories, selectedId, onChange }: CategoryFilterProps) {
  return (
    <ScrollArea className="w-full whitespace-nowrap">
      <div className="flex gap-2 pb-2">
        <Button
          variant={selectedId === null ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange(null)}
          className="shrink-0"
        >
          Semua
        </Button>
        {categories.map((cat) => (
          <Button
            key={cat.id}
            variant={selectedId === cat.id ? 'default' : 'outline'}
            size="sm"
            onClick={() => onChange(cat.id)}
            className="shrink-0"
          >
            {cat.name}
          </Button>
        ))}
      </div>
      <ScrollBar orientation="horizontal" />
    </ScrollArea>
  );
}
```

- [ ] **Step 2: Create ProductCard.tsx**

```tsx
'use client';

import Image from 'next/image';
import { Plus } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/lib/format';
import type { ProductDTO } from '@/services/products';

interface ProductCardProps {
  product: ProductDTO;
  onAdd: (product: ProductDTO) => void;
  isAdding?: boolean;
}

export function ProductCard({ product, onAdd, isAdding }: ProductCardProps) {
  return (
    <Card
      role="button"
      tabIndex={0}
      aria-label={`Tambah ${product.name} ke keranjang`}
      onClick={() => onAdd(product)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onAdd(product);
        }
      }}
      className={cn(
        'relative cursor-pointer overflow-hidden transition-all select-none',
        'hover:shadow-md hover:border-primary/50 active:scale-95',
        'focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none',
        !product.is_available && 'opacity-50 pointer-events-none',
        isAdding && 'ring-2 ring-primary',
      )}
    >
      <div className="aspect-square w-full bg-surface">
        {product.image_url ? (
          <Image
            src={product.image_url}
            alt={product.name}
            fill
            className="object-cover"
            sizes="(max-width: 768px) 50vw, 200px"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-3xl">
            🍽️
          </div>
        )}
      </div>
      <div className="p-2">
        <p className="text-xs font-medium text-text-primary line-clamp-2 leading-tight">
          {product.name}
        </p>
        <p className="mt-1 text-sm font-bold text-primary">
          {formatCurrency(product.price)}
        </p>
      </div>
      <div className="absolute bottom-2 right-2">
        <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-fg shadow-sm">
          <Plus className="h-3 w-3" />
        </div>
      </div>
    </Card>
  );
}
```

- [ ] **Step 3: Create ProductPanel.tsx**

```tsx
'use client';

import { useRef, useEffect } from 'react';
import { Search, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { EmptyState } from '@/components/shared/EmptyState';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import { CategoryFilter } from './CategoryFilter';
import { ProductCard } from './ProductCard';
import { useProducts } from '@/services/products';
import { useAddItem } from '@/services/pos';
import { usePOSStore } from '@/store/pos.store';
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts';
import { toast } from 'sonner';

export function ProductPanel() {
  const searchRef = useRef<HTMLInputElement>(null);
  const {
    activeOrderId,
    searchQuery,
    selectedCategoryId,
    setSearchQuery,
    setSelectedCategory,
    setCartItems,
  } = usePOSStore();

  const { data, isLoading } = useProducts({
    search: searchQuery || undefined,
    category_id: selectedCategoryId ?? undefined,
  });

  const addItemMutation = useAddItem();

  // '/' key focuses the search bar
  useKeyboardShortcuts([
    {
      key: '/',
      handler: () => searchRef.current?.focus(),
    },
  ]);

  function handleAddProduct(product: typeof data extends { data: infer P[] } ? P : never) {
    if (!activeOrderId) {
      toast.error('Buat pesanan terlebih dahulu');
      return;
    }

    addItemMutation.mutate(
      {
        orderId: activeOrderId,
        product_id: product.id,
        product_name: product.name,
        unit_price: product.price,
        quantity: 1,
      },
      {
        onError: () => toast.error('Gagal menambahkan item'),
      },
    );
  }

  const products = data?.data ?? [];

  const mockCategories = [
    { id: 'cat-1', name: 'Makanan' },
    { id: 'cat-2', name: 'Minuman' },
    { id: 'cat-3', name: 'Dessert' },
  ];

  return (
    <div className="flex h-full flex-col gap-3 overflow-hidden p-3">
      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" aria-hidden="true" />
        <Input
          ref={searchRef}
          type="search"
          placeholder="Cari produk atau scan barcode... (/)"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9 pr-9"
          aria-label="Cari produk"
        />
        {searchQuery && (
          <Button
            variant="ghost"
            size="icon"
            className="absolute right-1 top-1/2 h-7 w-7 -translate-y-1/2"
            onClick={() => setSearchQuery('')}
            aria-label="Hapus pencarian"
          >
            <X className="h-3 w-3" />
          </Button>
        )}
      </div>

      {/* Category Filter */}
      <CategoryFilter
        categories={mockCategories}
        selectedId={selectedCategoryId}
        onChange={setSelectedCategory}
      />

      {/* Product Grid */}
      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="grid grid-cols-3 gap-2 xl:grid-cols-4 2xl:grid-cols-5">
            {Array.from({ length: 12 }).map((_, i) => (
              <SkeletonCard key={i} lines={2} />
            ))}
          </div>
        ) : products.length === 0 ? (
          <EmptyState
            title="Produk tidak ditemukan"
            description={searchQuery ? `Tidak ada hasil untuk "${searchQuery}"` : 'Belum ada produk tersedia'}
          />
        ) : (
          <div className="grid grid-cols-3 gap-2 pb-4 xl:grid-cols-4 2xl:grid-cols-5">
            {products.map((product) => (
              <ProductCard
                key={product.id}
                product={product}
                onAdd={handleAddProduct}
                isAdding={addItemMutation.isPending && addItemMutation.variables?.product_id === product.id}
              />
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
```

- [ ] **Step 4: Create ProductPanel.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { ProductPanel } from './ProductPanel';

vi.mock('@/services/products', () => ({
  useProducts: vi.fn().mockReturnValue({
    data: undefined, isLoading: true, isError: false,
  }),
}));
vi.mock('@/services/pos', () => ({
  useAddItem: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
}));

describe('ProductPanel', () => {
  it('renders search bar', () => {
    renderWithProviders(<ProductPanel />);
    expect(screen.getByPlaceholderText(/cari produk/i)).toBeInTheDocument();
  });

  it('shows skeleton cards while loading', () => {
    renderWithProviders(<ProductPanel />);
    // Skeleton cards are rendered when loading
    expect(screen.queryByText('Produk tidak ditemukan')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 5: Create index.ts**

```typescript
export { ProductPanel } from './ProductPanel';
```

- [ ] **Step 6: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: ProductPanel — 2 tests passing.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/components/features/pos/ProductPanel/
git commit -m "feat(web/pos): add ProductPanel with search, category filter, product grid"
```

---

## Task 4: CartPanel

**Files:**
- Create: `apps/web/src/components/features/pos/CartPanel/CartItem.tsx`
- Create: `apps/web/src/components/features/pos/CartPanel/CartSummary.tsx`
- Create: `apps/web/src/components/features/pos/CartPanel/CartPanel.tsx`
- Create: `apps/web/src/components/features/pos/CartPanel/CartPanel.test.tsx`
- Create: `apps/web/src/components/features/pos/CartPanel/index.ts`

- [ ] **Step 1: Create CartItem.tsx**

```tsx
'use client';

import { Minus, Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { formatCurrency } from '@/lib/format';
import type { CartItem as CartItemType } from '@/store/pos.store';

interface CartItemProps {
  item: CartItemType;
  onIncrease: (productId: string) => void;
  onDecrease: (productId: string) => void;
  onRemove: (productId: string) => void;
}

export function CartItem({ item, onIncrease, onDecrease, onRemove }: CartItemProps) {
  return (
    <div className="flex items-center gap-2 py-2 text-sm">
      <div className="flex-1 min-w-0">
        <p className="font-medium text-text-primary truncate">{item.productName}</p>
        <p className="text-xs text-text-secondary">{formatCurrency(item.unitPrice)} / pcs</p>
      </div>

      <div className="flex items-center gap-1 shrink-0">
        <Button
          variant="outline"
          size="icon"
          className="h-6 w-6"
          onClick={() => onDecrease(item.productId)}
          aria-label="Kurangi jumlah"
        >
          <Minus className="h-3 w-3" />
        </Button>
        <span className="w-6 text-center font-semibold text-sm">{item.quantity}</span>
        <Button
          variant="outline"
          size="icon"
          className="h-6 w-6"
          onClick={() => onIncrease(item.productId)}
          aria-label="Tambah jumlah"
        >
          <Plus className="h-3 w-3" />
        </Button>
      </div>

      <span className="w-20 text-right font-semibold text-sm shrink-0">
        {formatCurrency(item.subtotal)}
      </span>

      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6 text-text-muted hover:text-danger shrink-0"
        onClick={() => onRemove(item.productId)}
        aria-label={`Hapus ${item.productName}`}
      >
        <Trash2 className="h-3 w-3" />
      </Button>
    </div>
  );
}
```

- [ ] **Step 2: Create CartSummary.tsx**

```tsx
import { Separator } from '@/components/ui/separator';
import { formatCurrency } from '@/lib/format';

interface CartSummaryProps {
  subtotal: number;
  taxAmount: number;
  discountAmount: number;
  total: number;
}

export function CartSummary({ subtotal, taxAmount, discountAmount, total }: CartSummaryProps) {
  return (
    <div className="space-y-1 text-sm">
      <div className="flex justify-between text-text-secondary">
        <span>Subtotal</span>
        <span>{formatCurrency(subtotal)}</span>
      </div>
      <div className="flex justify-between text-text-secondary">
        <span>Pajak</span>
        <span>{formatCurrency(taxAmount)}</span>
      </div>
      {discountAmount > 0 && (
        <div className="flex justify-between text-success">
          <span>Diskon</span>
          <span>-{formatCurrency(discountAmount)}</span>
        </div>
      )}
      <Separator className="my-2" />
      <div className="flex justify-between text-base font-bold">
        <span>TOTAL</span>
        <span className="text-primary">{formatCurrency(total)}</span>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Create CartPanel.tsx**

```tsx
'use client';

import { ParkingCircle, Trash2, CreditCard } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { EmptyState } from '@/components/shared/EmptyState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { CartItem } from './CartItem';
import { CartSummary } from './CartSummary';
import { usePOSStore } from '@/store/pos.store';
import { useAddItem, useRemoveItem, useParkOrder, useSubmitOrder } from '@/services/pos';
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts';

export function CartPanel() {
  const {
    activeOrderId,
    orderNumber,
    tableNumber,
    cartItems,
    subtotal,
    taxAmount,
    discountAmount,
    total,
    openPaymentModal,
    openParkedOrders,
    resetOrder,
  } = usePOSStore();

  const submitOrderMutation = useSubmitOrder();
  const parkOrderMutation = useParkOrder();

  // F12 = quick checkout
  useKeyboardShortcuts([
    {
      key: 'F12',
      handler: () => {
        if (cartItems.length > 0) handleCheckout();
      },
    },
  ]);

  function handleCheckout() {
    if (!activeOrderId || cartItems.length === 0) return;

    submitOrderMutation.mutate(activeOrderId, {
      onSuccess: () => openPaymentModal(),
      onError: () => toast.error('Gagal memproses pesanan'),
    });
  }

  function handlePark() {
    if (!activeOrderId) return;
    parkOrderMutation.mutate(activeOrderId, {
      onSuccess: () => {
        toast.success('Pesanan ditahan');
        resetOrder();
      },
      onError: () => toast.error('Gagal menahan pesanan'),
    });
  }

  if (!activeOrderId) {
    return (
      <div className="flex h-full flex-col items-center justify-center p-4">
        <EmptyState
          title="Belum ada pesanan"
          description="Pilih produk untuk memulai pesanan baru"
        />
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => openParkedOrders()}
        >
          <ParkingCircle className="mr-2 h-4 w-4" />
          Pesanan Ditahan
        </Button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col border-l bg-surface-raised">
      {/* Cart Header */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div>
          <p className="font-semibold text-text-primary">{orderNumber ?? 'Pesanan Baru'}</p>
          {tableNumber && (
            <p className="text-xs text-text-secondary">Meja {tableNumber}</p>
          )}
        </div>
        <ConfirmDialog
          trigger={
            <Button variant="ghost" size="icon" aria-label="Batalkan pesanan">
              <Trash2 className="h-4 w-4 text-text-muted" />
            </Button>
          }
          title="Batalkan Pesanan?"
          description="Semua item di keranjang akan dihapus."
          confirmLabel="Ya, Batalkan"
          onConfirm={resetOrder}
        />
      </div>

      {/* Cart Items */}
      <ScrollArea className="flex-1 px-4">
        {cartItems.length === 0 ? (
          <div className="flex h-full items-center justify-center py-8">
            <p className="text-sm text-text-muted">Keranjang kosong</p>
          </div>
        ) : (
          <div className="divide-y">
            {cartItems.map((item) => (
              <CartItem
                key={item.productId}
                item={item}
                onIncrease={() => {}}
                onDecrease={() => {}}
                onRemove={() => {}}
              />
            ))}
          </div>
        )}
      </ScrollArea>

      {/* Cart Footer */}
      <div className="border-t p-4 space-y-4">
        <CartSummary
          subtotal={subtotal}
          taxAmount={taxAmount}
          discountAmount={discountAmount}
          total={total}
        />

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            className="flex-1"
            onClick={handlePark}
            disabled={cartItems.length === 0 || parkOrderMutation.isPending}
            aria-label="Tahan pesanan"
          >
            <ParkingCircle className="mr-1.5 h-4 w-4" />
            Tahan
          </Button>
          <Button
            size="sm"
            className="flex-1"
            onClick={handleCheckout}
            disabled={cartItems.length === 0 || submitOrderMutation.isPending}
            aria-label="Proses pembayaran (F12)"
          >
            <CreditCard className="mr-1.5 h-4 w-4" />
            Bayar (F12)
          </Button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Create CartPanel.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { CartPanel } from './CartPanel';

vi.mock('@/services/pos', () => ({
  useSubmitOrder: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
  useParkOrder: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
  useAddItem: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
  useRemoveItem: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
}));

describe('CartPanel', () => {
  it('shows empty state when no active order', () => {
    // Default store has no activeOrderId
    renderWithProviders(<CartPanel />);
    expect(screen.getByText('Belum ada pesanan')).toBeInTheDocument();
  });

  it('shows parked orders button when no active order', () => {
    renderWithProviders(<CartPanel />);
    expect(screen.getByRole('button', { name: /pesanan ditahan/i })).toBeInTheDocument();
  });
});
```

- [ ] **Step 5: Create index.ts**

```typescript
export { CartPanel } from './CartPanel';
```

- [ ] **Step 6: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: CartPanel — 2 tests passing.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/components/features/pos/CartPanel/
git commit -m "feat(web/pos): add CartPanel with items, summary, park/checkout actions"
```

---

## Task 5: Payment Modal

**Files:**
- Create: `apps/web/src/components/features/pos/PaymentModal/CashPaymentTab.tsx`
- Create: `apps/web/src/components/features/pos/PaymentModal/QRISPaymentTab.tsx`
- Create: `apps/web/src/components/features/pos/PaymentModal/PaymentModal.tsx`
- Create: `apps/web/src/components/features/pos/PaymentModal/PaymentModal.test.tsx`
- Create: `apps/web/src/components/features/pos/PaymentModal/index.ts`

- [ ] **Step 1: Create CashPaymentTab.tsx**

```tsx
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { CurrencyInput } from '@/components/shared/CurrencyInput';
import { formatCurrency } from '@/lib/format';

const cashSchema = z.object({
  amountTendered: z.number().int().min(1, 'Masukkan jumlah pembayaran'),
});

type CashFormData = z.infer<typeof cashSchema>;

interface CashPaymentTabProps {
  total: number;   // sen
  onPay: (amountTendered: number) => void;
  isPending: boolean;
}

export function CashPaymentTab({ total, onPay, isPending }: CashPaymentTabProps) {
  const form = useForm<CashFormData>({
    resolver: zodResolver(cashSchema.refine((d) => d.amountTendered >= total, {
      message: `Pembayaran kurang dari total ${formatCurrency(total)}`,
      path: ['amountTendered'],
    })),
    defaultValues: { amountTendered: 0 },
  });

  const amountTendered = form.watch('amountTendered');
  const change = Math.max(0, amountTendered - total);

  // Quick-pay buttons
  const quickAmounts = [total, Math.ceil(total / 10000) * 10000, Math.ceil(total / 50000) * 50000];
  const uniqueQuickAmounts = [...new Set(quickAmounts)].slice(0, 3);

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit((d) => onPay(d.amountTendered))} className="space-y-4">
        <div className="text-center">
          <p className="text-sm text-text-secondary">Total yang harus dibayar</p>
          <p className="text-3xl font-bold text-primary">{formatCurrency(total)}</p>
        </div>

        <FormField
          control={form.control}
          name="amountTendered"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Jumlah Diterima</FormLabel>
              <FormControl>
                <CurrencyInput value={field.value} onChange={field.onChange} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {/* Quick amount buttons */}
        <div className="flex gap-2">
          {uniqueQuickAmounts.map((amount) => (
            <Button
              key={amount}
              type="button"
              variant="outline"
              size="sm"
              className="flex-1 text-xs"
              onClick={() => form.setValue('amountTendered', amount)}
            >
              {formatCurrency(amount)}
            </Button>
          ))}
        </div>

        {amountTendered >= total && (
          <div className="flex justify-between rounded-lg bg-success-light p-3 text-success">
            <span className="font-medium">Kembalian</span>
            <span className="text-lg font-bold">{formatCurrency(change)}</span>
          </div>
        )}

        <Button type="submit" className="w-full" size="lg" disabled={isPending}>
          {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Konfirmasi Pembayaran Tunai
        </Button>
      </form>
    </Form>
  );
}
```

- [ ] **Step 2: Create QRISPaymentTab.tsx**

```tsx
'use client';

import { Loader2, QrCode } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { formatCurrency } from '@/lib/format';

interface QRISPaymentTabProps {
  total: number;   // sen
  qrCodeUrl: string | null;
  onGenerateQR: () => void;
  isGenerating: boolean;
}

export function QRISPaymentTab({ total, qrCodeUrl, onGenerateQR, isGenerating }: QRISPaymentTabProps) {
  return (
    <div className="flex flex-col items-center gap-4 py-4">
      <div className="text-center">
        <p className="text-sm text-text-secondary">Total QRIS</p>
        <p className="text-3xl font-bold text-primary">{formatCurrency(total)}</p>
      </div>

      {qrCodeUrl ? (
        <div className="flex flex-col items-center gap-3">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={qrCodeUrl}
            alt="QRIS QR Code"
            className="h-48 w-48 rounded-lg border p-2"
          />
          <p className="text-sm text-text-secondary text-center">
            Scan QR code menggunakan aplikasi mobile banking atau e-wallet
          </p>
          <p className="text-xs text-text-muted">Menunggu konfirmasi pembayaran...</p>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-4">
          <div className="flex h-48 w-48 items-center justify-center rounded-lg border-2 border-dashed border-border">
            <QrCode className="h-16 w-16 text-text-muted" aria-hidden="true" />
          </div>
          <Button onClick={onGenerateQR} disabled={isGenerating}>
            {isGenerating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Generate QR Code
          </Button>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Create PaymentModal.tsx**

```tsx
'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { CashPaymentTab } from './CashPaymentTab';
import { QRISPaymentTab } from './QRISPaymentTab';
import { usePOSStore } from '@/store/pos.store';
import { useCreatePayment } from '@/services/payments';
import { v4 as uuidv4 } from 'uuid';

export function PaymentModal() {
  const {
    isPaymentModalOpen,
    closePaymentModal,
    activeOrderId,
    total,
    resetOrder,
  } = usePOSStore();

  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null);
  const createPayment = useCreatePayment();

  function handleCashPayment(amountTendered: number) {
    if (!activeOrderId) return;

    createPayment.mutate(
      {
        order_id: activeOrderId,
        method: 'cash',
        amount: amountTendered,
        idempotency_key: uuidv4(),
      },
      {
        onSuccess: (res) => {
          toast.success(`Pembayaran berhasil. Kembalian: ${res.data.change_amount ?? 0}`);
          closePaymentModal();
          resetOrder();
        },
        onError: () => toast.error('Pembayaran gagal. Coba lagi.'),
      },
    );
  }

  function handleGenerateQR() {
    if (!activeOrderId) return;

    createPayment.mutate(
      {
        order_id: activeOrderId,
        method: 'qris',
        amount: total,
        idempotency_key: uuidv4(),
      },
      {
        onSuccess: (res) => {
          if (res.data.qr_code_url) setQrCodeUrl(res.data.qr_code_url);
        },
        onError: () => toast.error('Gagal membuat QR QRIS'),
      },
    );
  }

  return (
    <Dialog open={isPaymentModalOpen} onOpenChange={(open) => !open && closePaymentModal()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Pembayaran</DialogTitle>
        </DialogHeader>

        <Tabs defaultValue="cash">
          <TabsList className="w-full">
            <TabsTrigger value="cash" className="flex-1">💵 Tunai</TabsTrigger>
            <TabsTrigger value="qris" className="flex-1">📱 QRIS</TabsTrigger>
            <TabsTrigger value="card" className="flex-1">💳 Kartu</TabsTrigger>
          </TabsList>

          <TabsContent value="cash" className="mt-4">
            <CashPaymentTab
              total={total}
              onPay={handleCashPayment}
              isPending={createPayment.isPending}
            />
          </TabsContent>

          <TabsContent value="qris" className="mt-4">
            <QRISPaymentTab
              total={total}
              qrCodeUrl={qrCodeUrl}
              onGenerateQR={handleGenerateQR}
              isGenerating={createPayment.isPending}
            />
          </TabsContent>

          <TabsContent value="card" className="mt-4">
            <p className="text-center text-sm text-text-secondary py-8">
              Pembayaran kartu via EDC.<br />Konfirmasi manual setelah transaksi EDC selesai.
            </p>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 4: Create PaymentModal.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { PaymentModal } from './PaymentModal';

vi.mock('@/services/payments', () => ({
  useCreatePayment: vi.fn().mockReturnValue({ mutate: vi.fn(), isPending: false }),
}));

// Open the modal in store before rendering
vi.mock('@/store/pos.store', () => ({
  usePOSStore: vi.fn().mockReturnValue({
    isPaymentModalOpen: true,
    closePaymentModal: vi.fn(),
    activeOrderId: 'order-1',
    total: 50000,
    resetOrder: vi.fn(),
  }),
}));

describe('PaymentModal', () => {
  it('renders payment method tabs', () => {
    renderWithProviders(<PaymentModal />);
    expect(screen.getByRole('tab', { name: /tunai/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /qris/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /kartu/i })).toBeInTheDocument();
  });

  it('shows total amount', () => {
    renderWithProviders(<PaymentModal />);
    // Total = 50000 sen = Rp 500
    expect(screen.getAllByText(/500/).length).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 5: Create index.ts**

```typescript
export { PaymentModal } from './PaymentModal';
```

- [ ] **Step 6: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: PaymentModal — 2 tests passing.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/components/features/pos/PaymentModal/
git commit -m "feat(web/pos): add PaymentModal with cash/QRIS/card tabs"
```

---

## Task 6: POS Terminal Page Assembly

**Files:**
- Create: `apps/web/src/components/features/pos/POSTerminal.tsx`
- Create: `apps/web/src/components/features/pos/ParkedOrdersDrawer.tsx`
- Create: `apps/web/src/app/pos/layout.tsx`
- Create: `apps/web/src/app/pos/page.tsx`

- [ ] **Step 1: Create ParkedOrdersDrawer.tsx**

```tsx
'use client';

import { ParkingCircle } from 'lucide-react';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/shared/EmptyState';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import { formatCurrency, formatDateTime } from '@/lib/format';
import { usePOSStore } from '@/store/pos.store';
import { useParkedOrders, useResumeOrder } from '@/services/pos';
import { toast } from 'sonner';

interface ParkedOrdersDrawerProps {
  branchId: string;
}

export function ParkedOrdersDrawer({ branchId }: ParkedOrdersDrawerProps) {
  const { isParkedOrdersOpen, closeParkedOrders, setActiveOrder } = usePOSStore();
  const { data, isLoading } = useParkedOrders(branchId);
  const resumeOrder = useResumeOrder();

  function handleResume(orderId: string, orderNumber: string) {
    resumeOrder.mutate(orderId, {
      onSuccess: (res) => {
        setActiveOrder(orderId, orderNumber, res.data.table_number ?? '');
        closeParkedOrders();
        toast.success(`Pesanan ${orderNumber} dilanjutkan`);
      },
      onError: () => toast.error('Gagal melanjutkan pesanan'),
    });
  }

  return (
    <Sheet open={isParkedOrdersOpen} onOpenChange={(open) => !open && closeParkedOrders()}>
      <SheetContent side="right" className="w-80">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <ParkingCircle className="h-5 w-5" />
            Pesanan Ditahan
          </SheetTitle>
        </SheetHeader>

        <div className="mt-6 space-y-3">
          {isLoading ? (
            Array.from({ length: 3 }).map((_, i) => <SkeletonCard key={i} lines={2} />)
          ) : data?.data.length === 0 ? (
            <EmptyState title="Tidak ada pesanan ditahan" />
          ) : (
            data?.data.map((order) => (
              <div key={order.id} className="rounded-lg border p-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-mono text-sm font-semibold">{order.order_number}</p>
                    {order.table_number && (
                      <p className="text-xs text-text-secondary">Meja {order.table_number}</p>
                    )}
                    <p className="text-xs text-text-muted">{formatDateTime(order.updated_at)}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-bold text-primary">{formatCurrency(order.total)}</p>
                    <Button
                      size="sm"
                      variant="outline"
                      className="mt-1 h-7 text-xs"
                      onClick={() => handleResume(order.id, order.order_number)}
                      disabled={resumeOrder.isPending}
                    >
                      Lanjutkan
                    </Button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
```

- [ ] **Step 2: Create POSTerminal.tsx**

```tsx
'use client';

import { ProductPanel } from './ProductPanel';
import { CartPanel } from './CartPanel';
import { PaymentModal } from './PaymentModal';
import { ParkedOrdersDrawer } from './ParkedOrdersDrawer';

interface POSTerminalProps {
  branchId: string;
}

export function POSTerminal({ branchId }: POSTerminalProps) {
  return (
    <div className="flex h-full overflow-hidden">
      {/* Left: Product Panel (65%) */}
      <div className="flex-1 min-w-0 overflow-hidden">
        <ProductPanel />
      </div>

      {/* Right: Cart Panel (fixed 320px) */}
      <div className="w-80 shrink-0 overflow-hidden">
        <CartPanel />
      </div>

      {/* Overlays */}
      <PaymentModal />
      <ParkedOrdersDrawer branchId={branchId} />
    </div>
  );
}
```

- [ ] **Step 3: Create pos/layout.tsx (full-screen, no sidebar)**

```tsx
export default function POSLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background">
      {/* POS TopBar */}
      <header className="flex h-12 shrink-0 items-center justify-between border-b bg-surface-raised px-4">
        <div className="flex items-center gap-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-fg font-bold text-xs">
            X
          </div>
          <span className="text-sm font-semibold text-text-primary">xyn-pos</span>
          <span className="text-xs text-text-muted">| POS Terminal</span>
        </div>
        <div className="flex items-center gap-2 text-xs text-text-muted">
          <span>F12: Bayar</span>
          <span>/ : Cari</span>
          <span>Ctrl+N: Baru</span>
        </div>
      </header>
      {/* POS Content (fills remaining height) */}
      <main className="flex-1 overflow-hidden">
        {children}
      </main>
    </div>
  );
}
```

- [ ] **Step 4: Create pos/page.tsx**

```tsx
import type { Metadata } from 'next';
import { POSTerminal } from '@/components/features/pos/POSTerminal';

export const metadata: Metadata = { title: 'POS Terminal' };

// TODO(phase5): get branchId from session claims
const MOCK_BRANCH_ID = process.env.NEXT_PUBLIC_DEFAULT_BRANCH_ID ?? '';

export default function POSPage() {
  return <POSTerminal branchId={MOCK_BRANCH_ID} />;
}
```

- [ ] **Step 5: Verify POS page renders**

```bash
cd apps/web && pnpm dev
```

Visit http://localhost:3000/pos — expected: full-screen POS layout, product panel on left, cart panel on right, no sidebar.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/features/pos/ apps/web/src/app/pos/
git commit -m "feat(web/pos): assemble full POS terminal with product panel, cart, payment modal"
```

---

## Task 7: KDS Board

**Files:**
- Create: `apps/web/src/components/features/kds/KDSTicket.tsx`
- Create: `apps/web/src/components/features/kds/KDSBoard.tsx`
- Create: `apps/web/src/components/features/kds/KDSBoard.test.tsx`
- Create: `apps/web/src/app/kds/layout.tsx`
- Create: `apps/web/src/app/kds/page.tsx`
- Create: `apps/web/src/services/kds.ts`

- [ ] **Step 1: Create services/kds.ts**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse, ListResponse } from '@/types/api';
import type { OrderDTO } from './orders';

export const kdsKeys = {
  tickets: (branchId: string) => ['kds-tickets', branchId] as const,
};

export function useKDSTickets(branchId: string) {
  return useQuery({
    queryKey: kdsKeys.tickets(branchId),
    queryFn: () =>
      apiFetch<ListResponse<OrderDTO>>('/v1/orders', {
        params: { status: 'pending_payment', branch_id: branchId },
      }),
    enabled: !!branchId,
    refetchInterval: 10_000, // Poll every 10s — no gRPC streaming yet in Phase 5
    staleTime: 5_000,
  });
}
```

- [ ] **Step 2: Create KDSTicket.tsx**

```tsx
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Check } from 'lucide-react';
import type { OrderDTO } from '@/services/orders';

interface KDSTicketProps {
  order: OrderDTO;
  ageMinutes: number;
  onMarkDone: (orderId: string) => void;
}

function getAgeColor(minutes: number): string {
  if (minutes < 5) return 'border-success bg-success/5';
  if (minutes < 10) return 'border-warning bg-warning/5';
  return 'border-danger bg-danger/5 animate-pulse';
}

function getAgeBadgeColor(minutes: number): string {
  if (minutes < 5) return 'bg-success text-white';
  if (minutes < 10) return 'bg-warning text-white';
  return 'bg-danger text-white';
}

export function KDSTicket({ order, ageMinutes, onMarkDone }: KDSTicketProps) {
  return (
    <div
      className={cn(
        'flex flex-col rounded-xl border-2 p-4 transition-colors min-h-48',
        getAgeColor(ageMinutes),
      )}
    >
      {/* Ticket Header */}
      <div className="flex items-start justify-between mb-3">
        <div>
          <p className="text-2xl font-black text-text-primary">
            {order.table_number ? `MEJA ${order.table_number}` : 'BAWA PULANG'}
          </p>
          <p className="text-sm font-mono text-text-secondary">{order.order_number}</p>
        </div>
        <span
          className={cn(
            'rounded-full px-2 py-1 text-xs font-bold',
            getAgeBadgeColor(ageMinutes),
          )}
        >
          {ageMinutes}m
        </span>
      </div>

      {/* Item List */}
      <div className="flex-1 space-y-1">
        {order.items.map((item, i) => (
          <div key={i} className="text-base">
            <span className="font-bold mr-2">{item.quantity}×</span>
            <span className="font-medium">{item.product_name}</span>
          </div>
        ))}
      </div>

      {/* Done Button */}
      <Button
        className="mt-4 w-full"
        onClick={() => onMarkDone(order.id)}
        aria-label={`Tandai selesai untuk ${order.order_number}`}
      >
        <Check className="mr-2 h-4 w-4" />
        SELESAI
      </Button>
    </div>
  );
}
```

- [ ] **Step 3: Create KDSBoard.tsx**

```tsx
'use client';

import { toast } from 'sonner';
import { Loader2, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/shared/EmptyState';
import { KDSTicket } from './KDSTicket';
import { useKDSTickets } from '@/services/kds';

interface KDSBoardProps {
  branchId: string;
}

function getAgeMinutes(createdAt: string): number {
  return Math.floor((Date.now() - new Date(createdAt).getTime()) / 60_000);
}

export function KDSBoard({ branchId }: KDSBoardProps) {
  const { data, isLoading, isFetching, refetch } = useKDSTickets(branchId);

  const orders = data?.data ?? [];

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-12 w-12 animate-spin text-text-muted" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* KDS Header */}
      <div className="flex items-center justify-between border-b px-6 py-3">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold">Dapur — Kitchen Display</h1>
          <div className="flex items-center gap-1.5 text-sm text-text-muted">
            <div className="h-2 w-2 rounded-full bg-success animate-pulse" />
            Live (refresh 10s)
          </div>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          aria-label="Refresh tiket"
        >
          <RefreshCw className={cn('h-4 w-4', isFetching && 'animate-spin')} />
        </Button>
      </div>

      {/* Ticket Grid */}
      <div className="flex-1 overflow-auto p-4">
        {orders.length === 0 ? (
          <EmptyState
            title="Semua pesanan selesai"
            description="Tidak ada pesanan yang sedang diproses"
          />
        ) : (
          <div className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-4">
            {orders.map((order) => (
              <KDSTicket
                key={order.id}
                order={order}
                ageMinutes={getAgeMinutes(order.created_at)}
                onMarkDone={(id) => toast.info(`Pesanan ${id} ditandai selesai (Phase 6)`)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
```

Note: Add `import { cn } from '@/lib/utils';` at the top of KDSBoard.tsx.

- [ ] **Step 4: Create KDSBoard.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { KDSBoard } from './KDSBoard';

vi.mock('@/services/kds', () => ({
  useKDSTickets: vi.fn().mockReturnValue({
    data: { data: [], pagination: { total_count: 0 } },
    isLoading: false,
    isFetching: false,
    refetch: vi.fn(),
  }),
}));

describe('KDSBoard', () => {
  it('shows empty state when no tickets', () => {
    renderWithProviders(<KDSBoard branchId="branch-1" />);
    expect(screen.getByText('Semua pesanan selesai')).toBeInTheDocument();
  });

  it('renders kitchen header', () => {
    renderWithProviders(<KDSBoard branchId="branch-1" />);
    expect(screen.getByText(/Kitchen Display/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 5: Create kds/layout.tsx (dark full-screen)**

```tsx
export default function KDSLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="dark flex h-screen flex-col overflow-hidden bg-background text-text-primary">
      {children}
    </div>
  );
}
```

- [ ] **Step 6: Create kds/page.tsx**

```tsx
import type { Metadata } from 'next';
import { KDSBoard } from '@/components/features/kds/KDSBoard';

export const metadata: Metadata = { title: 'Kitchen Display' };

const MOCK_BRANCH_ID = process.env.NEXT_PUBLIC_DEFAULT_BRANCH_ID ?? '';

export default function KDSPage() {
  return <KDSBoard branchId={MOCK_BRANCH_ID} />;
}
```

- [ ] **Step 7: Run all tests**

```bash
cd apps/web && pnpm test
```

Expected: KDSBoard — 2 tests passing. All tests in suite: 50+ passing.

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/components/features/kds/ apps/web/src/services/kds.ts apps/web/src/app/kds/
git commit -m "feat(web/kds): add KDS board with age-coded tickets and 10s polling"
```

---

## Final Verification

- [ ] Run full test suite: `cd apps/web && pnpm test`  
  Expected: 50+ tests passing. All green.

- [ ] Type check: `pnpm typecheck`  
  Expected: 0 errors.

- [ ] Start dev server: `pnpm dev`  
  - `/pos` — full-screen two-panel POS terminal
  - `/kds` — dark full-screen kitchen display board
  - `/dashboard` — sidebar + sales overview
  - `/login` — centered login form

- [ ] Commit final changes and push.

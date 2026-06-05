# Web App — AI Assistant Context

> Scope: apps/web/  
> Stack: Next.js 16.2.7 | React 19.2 | TypeScript 5.8+ | Tailwind CSS 4.3 | shadcn/ui (June 2026)  
> Parent context: ../../CLAUDE.md (read that first)

---

## Project Structure

```
apps/web/src/
├── app/                      ← Next.js 16 App Router (file-system routing)
│   ├── (auth)/               ← Auth layout group
│   ├── (dashboard)/          ← Admin dashboard layout group
│   ├── pos/                  ← POS terminal page
│   └── kds/                  ← Kitchen Display System page
├── components/
│   ├── ui/                   ← shadcn/ui generated components (don't edit, re-run CLI)
│   └── features/             ← Feature-specific components
│       ├── pos/
│       │   ├── ProductGrid/
│       │   │   ├── index.ts       ← Re-exports only
│       │   │   ├── ProductGrid.tsx
│       │   │   ├── ProductGrid.test.tsx
│       │   │   └── ProductCard.tsx
│       │   └── CartPanel/
│       └── payment/
├── hooks/                    ← Shared hooks
├── lib/
│   ├── api/                  ← apiFetch, response types, ApiError
│   ├── hardware/             ← Printer, barcode adapters
│   └── utils.ts
├── services/                 ← TanStack Query hooks (one file per domain)
│   ├── orders.ts             ← useOrders, useCreateOrder, useCheckout
│   └── products.ts
├── store/                    ← Zustand stores (one per domain concern)
│   └── pos.store.ts          ← Active order, selected table, cashier shift
└── gen/                      ← Generated TypeScript from proto (never edit)
```

---

## Critical Rules

```
✅ Server Components by default — add 'use client' only when you need:
   - useState / useReducer
   - useEffect
   - Browser APIs (window, document)
   - Event listeners

✅ All API responses wrapped in DataResponse<T> / ListResponse<T> (see response-standards.md)
✅ TanStack Query v5 for ALL server state — never fetch in useEffect
✅ Zustand v5 for LOCAL/UI state only (open/close, selected tab, shift state)
✅ React Hook Form for ALL forms — never controlled inputs for complex forms
✅ next/image for ALL images — never <img> tag
✅ Error boundary wrapping all feature areas

❌ Never put server data in Zustand
❌ Never fetch in useEffect — use TanStack Query
❌ Never use the pages/ directory (App Router only)
❌ Never use any as a type — use unknown + type guard
❌ Never @ts-ignore — fix the type or use @ts-expect-error with comment
❌ Never hardcode colors — use Tailwind CSS variables
```

---

## TanStack Query v5 Patterns

```typescript
// ✅ Query with error handling
function useOrder(orderId: string) {
  return useQuery({
    queryKey: ['orders', orderId],
    queryFn: () => apiFetch<OrderDTO>(`/v1/orders/${orderId}`),
    enabled: !!orderId,
    staleTime: 30_000,
  });
}

// ✅ Mutation with optimistic update
function useAddCartItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (item: AddItemRequest) => apiFetch<OrderDTO>('/v1/orders/.../items', {
      method: 'POST',
      body: JSON.stringify(item),
    }),
    onMutate: async (newItem) => {
      // Cancel outgoing refetches, snapshot, optimistic update
      await queryClient.cancelQueries({ queryKey: ['cart'] });
      const previous = queryClient.getQueryData(['cart']);
      queryClient.setQueryData(['cart'], (old) => addItemOptimistically(old, newItem));
      return { previous };
    },
    onError: (_, __, context) => {
      queryClient.setQueryData(['cart'], context?.previous); // Rollback
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ['cart'] }),
  });
}
```

---

## Zustand v5 Store Pattern

```typescript
// store/pos.store.ts
// Zustand v5: no more subscribeWithSelector middleware needed — use slice pattern
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface POSState {
  activeOrderId: string | null;
  tableNumber: string | null;
  shiftId: string | null;
  setActiveOrder: (orderId: string, table: string) => void;
  openShift: (shiftId: string) => void;
  reset: () => void;
}

export const usePOSStore = create<POSState>()(
  devtools(
    (set) => ({
      activeOrderId: null,
      tableNumber: null,
      shiftId: null,
      setActiveOrder: (orderId, table) => set({ activeOrderId: orderId, tableNumber: table }),
      openShift: (shiftId) => set({ shiftId }),
      reset: () => set({ activeOrderId: null, tableNumber: null }),
    }),
    { name: 'pos-store' },
  ),
);
```

---

## Response Handling Pattern

```typescript
// Every component that fetches data MUST handle all 3 states
function OrderDetails({ orderId }: { orderId: string }) {
  const { data, isLoading, isError, error, refetch } = useOrder(orderId);

  if (isLoading) return <OrderDetailsSkeleton />;
  if (isError) {
    const apiError = error as ApiError;
    return <ErrorState message={apiError.response.message} onRetry={refetch} />;
  }
  if (!isDataResponse(data)) return null;

  return <OrderCard order={data.data} />;
}
```

---

## Tailwind CSS 4.3 Notes

Tailwind 4 uses CSS-first configuration — no `tailwind.config.js`:

```css
/* src/styles/globals.css */
@import "tailwindcss";

/* Define custom tokens as CSS variables */
@theme {
  --color-brand: oklch(55% 0.2 250);
  --font-sans: 'Inter', system-ui, sans-serif;
}
```

```typescript
// shadcn/ui components use cn() utility — always use it
import { cn } from "@/lib/utils";
<div className={cn("base-classes", condition && "conditional-class", className)} />
```

---

## shadcn/ui Rules

```bash
# Add new components via CLI — never copy-paste from docs
npx shadcn@latest add button
npx shadcn@latest add dialog
npx shadcn@latest add data-table

# After adding: component lives in src/components/ui/
# You OWN the code — customize freely
# But: don't modify files that will be regenerated (check shadcn docs)
```

---

## gRPC-Web Client

```typescript
// src/lib/api/grpc-client.ts
// Generated TypeScript clients from buf generate
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { OrderService } from '@/gen/pos/v1/order_connect';

const transport = createConnectTransport({
  baseUrl: process.env.NEXT_PUBLIC_API_URL!,
  credentials: 'include',  // For cookie-based auth
});

export const orderClient = createClient(OrderService, transport);
```

---

## Testing (Vitest)

```typescript
// Component tests
import { render, screen } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import userEvent from '@testing-library/user-event';

test('CartPanel shows empty state when no items', () => {
  render(
    <QueryClientProvider client={testQueryClient}>
      <CartPanel />
    </QueryClientProvider>
  );
  expect(screen.getByText('Cart is empty')).toBeInTheDocument();
});
```

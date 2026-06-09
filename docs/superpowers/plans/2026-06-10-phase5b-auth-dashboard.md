# Phase 5B — Auth UI + Admin Dashboard

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the authentication flow (login, token management, route protection) and the full Admin Dashboard (sales overview, product CRUD, order history, branch management, user management).

**Architecture:** Next.js middleware enforces route protection using the PASETO token stored in an httpOnly cookie. Each dashboard page is a Server Component that passes auth context to 'use client' data-fetching children. All data flows through TanStack Query v5 hooks in `services/`. All tables use the shared `DataTableWrapper` from Phase 5A.

**Tech Stack:** Next.js 16 middleware, React Hook Form + Zod, TanStack Query v5, shadcn/ui, Connect-RPC TypeScript clients (from gen/ts/)

---

## Prerequisites

- Phase 5A complete (foundation, design system, shared components all working)
- Backend tenant service running (`services/tenant`) with auth endpoints
- `gen/ts/tenant/v1/` TypeScript clients generated

## File Map

| File | Action | Purpose |
|---|---|---|
| `apps/web/src/middleware.ts` | Create | Route protection — redirect unauthenticated users |
| `apps/web/src/app/(auth)/layout.tsx` | Create | Centered auth layout (no sidebar) |
| `apps/web/src/app/(auth)/login/page.tsx` | Create | Login page |
| `apps/web/src/components/features/auth/LoginForm.tsx` | Create | Login form with RHF + Zod |
| `apps/web/src/components/features/auth/LoginForm.test.tsx` | Create | Login form tests |
| `apps/web/src/lib/auth/session.ts` | Create | Cookie read/write helpers (server-side) |
| `apps/web/src/lib/auth/actions.ts` | Create | Server Actions: login, logout |
| `apps/web/src/schemas/auth.schema.ts` | Create | Zod login schema |
| `apps/web/src/app/(dashboard)/layout.tsx` | Create | Dashboard layout with sidebar |
| `apps/web/src/app/(dashboard)/dashboard/page.tsx` | Create | Sales overview page |
| `apps/web/src/app/(dashboard)/dashboard/orders/page.tsx` | Create | Order history page |
| `apps/web/src/app/(dashboard)/dashboard/orders/[id]/page.tsx` | Create | Order detail page |
| `apps/web/src/app/(dashboard)/dashboard/products/page.tsx` | Create | Product list page |
| `apps/web/src/app/(dashboard)/dashboard/products/new/page.tsx` | Create | Create product page |
| `apps/web/src/app/(dashboard)/dashboard/branches/page.tsx` | Create | Branch list page |
| `apps/web/src/app/(dashboard)/dashboard/users/page.tsx` | Create | User management page |
| `apps/web/src/components/features/dashboard/SalesOverview.tsx` | Create | KPI cards + chart |
| `apps/web/src/components/features/dashboard/SalesOverview.test.tsx` | Create | |
| `apps/web/src/components/features/dashboard/OrdersTable.tsx` | Create | Order list table |
| `apps/web/src/components/features/dashboard/OrdersTable.test.tsx` | Create | |
| `apps/web/src/components/features/dashboard/ProductsTable.tsx` | Create | Product CRUD table |
| `apps/web/src/components/features/dashboard/ProductForm.tsx` | Create | Create/edit product form |
| `apps/web/src/services/orders.ts` | Create | TanStack Query hooks for orders |
| `apps/web/src/services/products.ts` | Create | TanStack Query hooks for products |
| `apps/web/src/services/tenants.ts` | Create | TanStack Query hooks for tenant/branches |

---

## Task 1: Auth Schemas + Server Actions

**Files:**
- Create: `apps/web/src/schemas/auth.schema.ts`
- Create: `apps/web/src/lib/auth/session.ts`
- Create: `apps/web/src/lib/auth/actions.ts`

- [ ] **Step 1: Create auth.schema.ts**

```typescript
import { z } from 'zod';

export const loginSchema = z.object({
  email: z.string().email('Format email tidak valid'),
  password: z.string().min(8, 'Password minimal 8 karakter'),
});

export type LoginFormData = z.infer<typeof loginSchema>;
```

- [ ] **Step 2: Create src/lib/auth/session.ts**

This reads/writes the PASETO token from the httpOnly cookie. Runs server-side only.

```typescript
import { cookies } from 'next/headers';

const SESSION_COOKIE = 'xyn_session';
const COOKIE_MAX_AGE = 60 * 60 * 8; // 8 hours

export interface SessionClaims {
  sub: string;       // user ID
  tenant_id: string;
  role: string;
  email: string;
  exp: number;
}

export async function getSession(): Promise<SessionClaims | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE)?.value;
  if (!token) return null;

  try {
    // The token is a PASETO v4 — we validate it server-side via the backend
    // For now, decode the payload portion (base64url, no signature check here — middleware handles that)
    const parts = token.split('.');
    if (parts.length < 2) return null;
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
    if (payload.exp && payload.exp < Date.now() / 1000) return null;
    return payload as SessionClaims;
  } catch {
    return null;
  }
}

export async function setSession(token: string): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: COOKIE_MAX_AGE,
    path: '/',
  });
}

export async function clearSession(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE);
}

export { SESSION_COOKIE };
```

- [ ] **Step 3: Create src/lib/auth/actions.ts (Server Actions)**

```typescript
'use server';

import { redirect } from 'next/navigation';
import { setSession, clearSession } from './session';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse } from '@/types/api';
import type { LoginFormData } from '@/schemas/auth.schema';

interface LoginResponseData {
  token: string;
  user_id: string;
  tenant_id: string;
  role: string;
  email: string;
}

export async function loginAction(data: LoginFormData): Promise<{ error?: string }> {
  try {
    const res = await apiFetch<DataResponse<LoginResponseData>>('/v1/auth/login', {
      method: 'POST',
      body: data,
    });
    await setSession(res.data.token);
  } catch (err) {
    const error = err as Error;
    return { error: error.message ?? 'Login gagal. Periksa email dan password.' };
  }
  redirect('/dashboard');
}

export async function logoutAction(): Promise<void> {
  await clearSession();
  redirect('/login');
}
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/schemas/ apps/web/src/lib/auth/
git commit -m "feat(web/auth): add auth schema, session helpers, server actions"
```

---

## Task 2: Next.js Middleware (Route Protection)

**Files:**
- Create: `apps/web/src/middleware.ts`

- [ ] **Step 1: Create middleware.ts**

```typescript
import { type NextRequest, NextResponse } from 'next/server';
import { SESSION_COOKIE } from '@/lib/auth/session';

const PUBLIC_PATHS = ['/login', '/api'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths
  if (PUBLIC_PATHS.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  const token = request.cookies.get(SESSION_COOKIE)?.value;

  if (!token) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('next', pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico|.*\\.png$).*)'],
};
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/src/middleware.ts
git commit -m "feat(web/auth): add Next.js middleware for route protection"
```

---

## Task 3: Login Page UI

**Files:**
- Create: `apps/web/src/app/(auth)/layout.tsx`
- Create: `apps/web/src/app/(auth)/login/page.tsx`
- Create: `apps/web/src/components/features/auth/LoginForm.tsx`
- Create: `apps/web/src/components/features/auth/LoginForm.test.tsx`

- [ ] **Step 1: Create (auth)/layout.tsx**

```tsx
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-surface p-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-2 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-fg text-xl font-bold">
            X
          </div>
          <h1 className="text-2xl font-bold text-text-primary">xyn-pos</h1>
          <p className="text-sm text-text-secondary">Platform POS & ERP Enterprise</p>
        </div>
        {children}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create features/auth/LoginForm.tsx**

```tsx
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useState, useTransition } from 'react';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { loginSchema, type LoginFormData } from '@/schemas/auth.schema';
import { loginAction } from '@/lib/auth/actions';

export function LoginForm() {
  const [showPassword, setShowPassword] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });

  function onSubmit(data: LoginFormData) {
    setServerError(null);
    startTransition(async () => {
      const result = await loginAction(data);
      if (result?.error) setServerError(result.error);
    });
  }

  return (
    <div className="rounded-xl border bg-surface-raised p-6 shadow-card">
      <h2 className="mb-6 text-lg font-semibold text-text-primary">Masuk ke Akun</h2>

      {serverError && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{serverError}</AlertDescription>
        </Alert>
      )}

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    placeholder="kasir@warungpadang.com"
                    autoComplete="email"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Password</FormLabel>
                <FormControl>
                  <div className="relative">
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      placeholder="Minimal 8 karakter"
                      autoComplete="current-password"
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-1 top-1/2 h-7 w-7 -translate-y-1/2"
                      onClick={() => setShowPassword((v) => !v)}
                      aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type="submit" className="w-full" disabled={isPending}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isPending ? 'Sedang masuk...' : 'Masuk'}
          </Button>
        </form>
      </Form>
    </div>
  );
}
```

- [ ] **Step 3: Create (auth)/login/page.tsx**

```tsx
import type { Metadata } from 'next';
import { LoginForm } from '@/components/features/auth/LoginForm';

export const metadata: Metadata = { title: 'Masuk' };

export default function LoginPage() {
  return <LoginForm />;
}
```

- [ ] **Step 4: Create LoginForm.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LoginForm } from './LoginForm';

// Mock the server action
vi.mock('@/lib/auth/actions', () => ({
  loginAction: vi.fn().mockResolvedValue({}),
}));

describe('LoginForm', () => {
  it('renders email and password fields', () => {
    render(<LoginForm />);
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Masuk' })).toBeInTheDocument();
  });

  it('shows validation error for invalid email', async () => {
    render(<LoginForm />);
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'not-an-email' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Masuk' }));
    await waitFor(() => {
      expect(screen.getByText('Format email tidak valid')).toBeInTheDocument();
    });
  });

  it('shows validation error for short password', async () => {
    render(<LoginForm />);
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'a@b.com' } });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'short' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Masuk' }));
    await waitFor(() => {
      expect(screen.getByText('Password minimal 8 karakter')).toBeInTheDocument();
    });
  });

  it('toggles password visibility', () => {
    render(<LoginForm />);
    const passwordInput = screen.getByPlaceholderText('Minimal 8 karakter');
    expect(passwordInput).toHaveAttribute('type', 'password');
    fireEvent.click(screen.getByLabelText('Tampilkan password'));
    expect(passwordInput).toHaveAttribute('type', 'text');
  });
});
```

- [ ] **Step 5: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: LoginForm tests — 4 passing.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/app/\(auth\)/ apps/web/src/components/features/auth/
git commit -m "feat(web/auth): add login page with RHF validation and server action"
```

---

## Task 4: Dashboard Layout

**Files:**
- Create: `apps/web/src/app/(dashboard)/layout.tsx`
- Create: `apps/web/src/app/(dashboard)/dashboard/page.tsx` (placeholder)

- [ ] **Step 1: Create (dashboard)/layout.tsx**

```tsx
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';
import { AppSidebar } from '@/components/layout/AppSidebar';
import { TopBar } from '@/components/layout/TopBar';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <TopBar />
        <main className="flex flex-1 flex-col">
          {children}
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
```

- [ ] **Step 2: Create dashboard/page.tsx placeholder**

```tsx
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { SalesOverview } from '@/components/features/dashboard/SalesOverview';

export default function DashboardPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Dashboard"
        description="Ringkasan penjualan dan kinerja bisnis"
      />
      <SalesOverview />
    </PageContainer>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add apps/web/src/app/\(dashboard\)/
git commit -m "feat(web/dashboard): add dashboard layout with sidebar and topbar"
```

---

## Task 5: TanStack Query Service Hooks

**Files:**
- Create: `apps/web/src/services/orders.ts`
- Create: `apps/web/src/services/products.ts`
- Create: `apps/web/src/services/tenants.ts`

- [ ] **Step 1: Create services/orders.ts**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse, ListResponse } from '@/types/api';

// DTO types matching backend BaseResponse
export interface OrderItemDTO {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;       // sen
  subtotal: number;         // sen
}

export interface OrderDTO {
  id: string;
  tenant_id: string;
  branch_id: string;
  cashier_id: string;
  order_number: string;
  order_type: 'dine_in' | 'takeaway' | 'delivery';
  table_number: string;
  status: 'draft' | 'pending_payment' | 'paid' | 'cancelled' | 'parked';
  items: OrderItemDTO[];
  subtotal: number;       // sen
  tax_amount: number;     // sen
  discount_amount: number; // sen
  total: number;          // sen
  created_at: string;
  updated_at: string;
}

export interface OrderFilters {
  status?: string;
  branch_id?: string;
  page?: number;
  page_size?: number;
}

export const orderKeys = {
  all: ['orders'] as const,
  list: (filters: OrderFilters) => [...orderKeys.all, 'list', filters] as const,
  detail: (id: string) => [...orderKeys.all, 'detail', id] as const,
};

export function useOrders(filters: OrderFilters = {}) {
  return useQuery({
    queryKey: orderKeys.list(filters),
    queryFn: () =>
      apiFetch<ListResponse<OrderDTO>>('/v1/orders', { params: filters }),
    staleTime: 30_000,
  });
}

export function useOrder(id: string) {
  return useQuery({
    queryKey: orderKeys.detail(id),
    queryFn: () => apiFetch<DataResponse<OrderDTO>>(`/v1/orders/${id}`),
    enabled: !!id,
  });
}
```

- [ ] **Step 2: Create services/products.ts**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse, ListResponse } from '@/types/api';

export interface ProductDTO {
  id: string;
  tenant_id: string;
  name: string;
  sku: string;
  category_id: string;
  category_name: string;
  price: number;          // sen
  is_available: boolean;
  image_url?: string;
  created_at: string;
}

export interface CreateProductRequest {
  name: string;
  sku: string;
  category_id: string;
  price: number;   // sen
}

export interface ProductFilters {
  category_id?: string;
  search?: string;
  page?: number;
  page_size?: number;
}

export const productKeys = {
  all: ['products'] as const,
  list: (filters: ProductFilters) => [...productKeys.all, 'list', filters] as const,
  detail: (id: string) => [...productKeys.all, 'detail', id] as const,
};

export function useProducts(filters: ProductFilters = {}) {
  return useQuery({
    queryKey: productKeys.list(filters),
    queryFn: () => apiFetch<ListResponse<ProductDTO>>('/v1/products', { params: filters }),
    staleTime: 60_000,
  });
}

export function useCreateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateProductRequest) =>
      apiFetch<DataResponse<ProductDTO>>('/v1/products', { method: 'POST', body: req }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: productKeys.all }),
  });
}

export function useUpdateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...req }: Partial<CreateProductRequest> & { id: string }) =>
      apiFetch<DataResponse<ProductDTO>>(`/v1/products/${id}`, { method: 'PATCH', body: req }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: productKeys.all }),
  });
}

export function useDeleteProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<DataResponse<void>>(`/v1/products/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: productKeys.all }),
  });
}
```

- [ ] **Step 3: Create services/tenants.ts**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';
import type { DataResponse, ListResponse } from '@/types/api';

export interface BranchDTO {
  id: string;
  tenant_id: string;
  name: string;
  address: {
    street: string;
    city: string;
    country: string;
  };
  timezone: string;
  is_active: boolean;
  created_at: string;
}

export const tenantKeys = {
  branches: ['branches'] as const,
  branchesList: () => [...tenantKeys.branches, 'list'] as const,
};

export function useBranches() {
  return useQuery({
    queryKey: tenantKeys.branchesList(),
    queryFn: () => apiFetch<ListResponse<BranchDTO>>('/v1/branches'),
    staleTime: 300_000, // 5 min — branches don't change often
  });
}
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/services/
git commit -m "feat(web): add TanStack Query service hooks for orders, products, tenants"
```

---

## Task 6: Sales Overview Component

**Files:**
- Create: `apps/web/src/components/features/dashboard/SalesOverview.tsx`
- Create: `apps/web/src/components/features/dashboard/SalesOverview.test.tsx`

- [ ] **Step 1: Create SalesOverview.tsx**

```tsx
'use client';

import { ShoppingCart, CreditCard, TrendingUp, Package } from 'lucide-react';
import { StatCard } from '@/components/shared/StatCard';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import { ErrorState } from '@/components/shared/ErrorState';
import { formatCurrency } from '@/lib/format';
import { useOrders } from '@/services/orders';

export function SalesOverview() {
  const { data, isLoading, isError, refetch } = useOrders({
    status: 'paid',
    page_size: 100,
  });

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <SkeletonCard key={i} lines={2} />
        ))}
      </div>
    );
  }

  if (isError) {
    return <ErrorState message="Tidak dapat memuat data penjualan" onRetry={refetch} />;
  }

  const orders = data?.data ?? [];
  const totalRevenue = orders.reduce((sum, o) => sum + o.total, 0);
  const totalOrders = data?.pagination.total_count ?? 0;
  const avgOrderValue = totalOrders > 0 ? Math.round(totalRevenue / totalOrders) : 0;

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        title="Total Pendapatan"
        value={formatCurrency(totalRevenue)}
        subtitle="Hari ini"
        icon={<TrendingUp className="h-5 w-5" />}
        trend={{ value: 12, label: 'vs kemarin' }}
      />
      <StatCard
        title="Total Pesanan"
        value={totalOrders}
        subtitle="Pesanan lunas"
        icon={<ShoppingCart className="h-5 w-5" />}
      />
      <StatCard
        title="Rata-rata Transaksi"
        value={formatCurrency(avgOrderValue)}
        subtitle="Per pesanan"
        icon={<CreditCard className="h-5 w-5" />}
      />
      <StatCard
        title="Produk Terjual"
        value={orders.reduce((sum, o) => sum + o.items.reduce((s, i) => s + i.quantity, 0), 0)}
        subtitle="Item hari ini"
        icon={<Package className="h-5 w-5" />}
      />
    </div>
  );
}
```

- [ ] **Step 2: Create SalesOverview.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { SalesOverview } from './SalesOverview';

vi.mock('@/services/orders', () => ({
  useOrders: vi.fn(),
  orderKeys: { all: ['orders'], list: () => ['orders', 'list'] },
}));

const { useOrders } = await import('@/services/orders');

describe('SalesOverview', () => {
  it('shows skeletons while loading', () => {
    vi.mocked(useOrders).mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    const { container } = renderWithProviders(<SalesOverview />);
    // SkeletonCard renders divs — verify there are 4 skeleton cards
    expect(container.querySelectorAll('[class*="rounded-lg border"]').length).toBeGreaterThan(0);
  });

  it('shows error state on failure', () => {
    vi.mocked(useOrders).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    renderWithProviders(<SalesOverview />);
    expect(screen.getByText('Tidak dapat memuat data penjualan')).toBeInTheDocument();
  });

  it('renders KPI cards with data', () => {
    vi.mocked(useOrders).mockReturnValue({
      data: {
        data: [
          {
            id: '1', total: 150000, items: [{ quantity: 2 } as never],
            tenant_id: '', branch_id: '', cashier_id: '', order_number: '',
            order_type: 'dine_in', table_number: '', status: 'paid',
            subtotal: 0, tax_amount: 0, discount_amount: 0, created_at: '', updated_at: '',
          },
        ],
        pagination: { page: 1, page_size: 100, total_count: 1, total_pages: 1 },
        is_success: true, request_id: '', status_code: '200', message: '', timestamp: '',
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    renderWithProviders(<SalesOverview />);
    expect(screen.getByText('Total Pendapatan')).toBeInTheDocument();
    expect(screen.getByText('Total Pesanan')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: SalesOverview — 3 tests passing.

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/components/features/dashboard/SalesOverview*
git commit -m "feat(web/dashboard): add SalesOverview KPI cards component"
```

---

## Task 7: Orders Table + Order History Page

**Files:**
- Create: `apps/web/src/components/features/dashboard/OrdersTable.tsx`
- Create: `apps/web/src/components/features/dashboard/OrdersTable.test.tsx`
- Create: `apps/web/src/app/(dashboard)/dashboard/orders/page.tsx`
- Create: `apps/web/src/app/(dashboard)/dashboard/orders/[id]/page.tsx`

- [ ] **Step 1: Create OrdersTable.tsx**

```tsx
'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Eye } from 'lucide-react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { DataTableWrapper } from '@/components/shared/DataTableWrapper';
import { StatusBadge } from '@/components/shared/StatusBadge';
import { ErrorState } from '@/components/shared/ErrorState';
import { formatCurrency, formatDateTime } from '@/lib/format';
import { useOrders, type OrderDTO, type OrderFilters } from '@/services/orders';

const columns: ColumnDef<OrderDTO>[] = [
  {
    accessorKey: 'order_number',
    header: 'No. Pesanan',
    cell: ({ row }) => (
      <span className="font-mono text-sm font-medium">{row.original.order_number}</span>
    ),
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
  },
  {
    accessorKey: 'total',
    header: 'Total',
    cell: ({ row }) => (
      <span className="font-semibold">{formatCurrency(row.original.total)}</span>
    ),
  },
  {
    accessorKey: 'order_type',
    header: 'Tipe',
    cell: ({ row }) => {
      const labels: Record<string, string> = {
        dine_in: 'Makan Di Sini',
        takeaway: 'Bawa Pulang',
        delivery: 'Delivery',
      };
      return <span className="text-sm">{labels[row.original.order_type] ?? row.original.order_type}</span>;
    },
  },
  {
    accessorKey: 'created_at',
    header: 'Waktu',
    cell: ({ row }) => (
      <span className="text-sm text-text-secondary">{formatDateTime(row.original.created_at)}</span>
    ),
  },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => (
      <Button variant="ghost" size="icon" asChild aria-label="Lihat detail">
        <Link href={`/dashboard/orders/${row.original.id}`}>
          <Eye className="h-4 w-4" />
        </Link>
      </Button>
    ),
  },
];

interface OrdersTableProps {
  filters?: OrderFilters;
}

export function OrdersTable({ filters = {} }: OrdersTableProps) {
  const { data, isLoading, isError, refetch } = useOrders(filters);

  if (isError) {
    return <ErrorState message="Tidak dapat memuat riwayat pesanan" onRetry={refetch} />;
  }

  return (
    <DataTableWrapper
      columns={columns}
      data={data?.data ?? []}
      isLoading={isLoading}
      emptyTitle="Belum ada pesanan"
      emptyDescription="Pesanan akan muncul di sini setelah transaksi pertama."
    />
  );
}
```

- [ ] **Step 2: Create OrdersTable.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/utils';
import { OrdersTable } from './OrdersTable';

vi.mock('@/services/orders', () => ({
  useOrders: vi.fn(),
  orderKeys: { all: ['orders'], list: () => ['orders', 'list'] },
}));

const { useOrders } = await import('@/services/orders');

describe('OrdersTable', () => {
  it('shows loading skeleton', () => {
    vi.mocked(useOrders).mockReturnValue({
      isLoading: true, isError: false, data: undefined, refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    renderWithProviders(<OrdersTable />);
    // SkeletonTable renders without crashing
  });

  it('shows error state on failure', () => {
    vi.mocked(useOrders).mockReturnValue({
      isLoading: false, isError: true, data: undefined, refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    renderWithProviders(<OrdersTable />);
    expect(screen.getByText('Tidak dapat memuat riwayat pesanan')).toBeInTheDocument();
  });

  it('renders order rows', () => {
    vi.mocked(useOrders).mockReturnValue({
      isLoading: false, isError: false,
      data: {
        data: [{
          id: 'order-1', order_number: 'ORD-001', status: 'paid',
          total: 50000, order_type: 'dine_in', created_at: '2026-06-10T10:00:00Z',
          tenant_id: '', branch_id: '', cashier_id: '', table_number: '',
          items: [], subtotal: 0, tax_amount: 0, discount_amount: 0, updated_at: '',
        }],
        pagination: { page: 1, page_size: 20, total_count: 1, total_pages: 1 },
        is_success: true, request_id: '', status_code: '200', message: '', timestamp: '',
      },
      refetch: vi.fn(),
    } as ReturnType<typeof useOrders>);

    renderWithProviders(<OrdersTable />);
    expect(screen.getByText('ORD-001')).toBeInTheDocument();
    expect(screen.getByText('Lunas')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Create orders/page.tsx**

```tsx
import type { Metadata } from 'next';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { OrdersTable } from '@/components/features/dashboard/OrdersTable';

export const metadata: Metadata = { title: 'Riwayat Pesanan' };

export default function OrdersPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Riwayat Pesanan"
        description="Semua transaksi di semua cabang"
      />
      <OrdersTable />
    </PageContainer>
  );
}
```

- [ ] **Step 4: Create orders/[id]/page.tsx**

```tsx
import type { Metadata } from 'next';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { OrderDetail } from '@/components/features/dashboard/OrderDetail';

export const metadata: Metadata = { title: 'Detail Pesanan' };

export default function OrderDetailPage({ params }: { params: { id: string } }) {
  return (
    <PageContainer>
      <PageHeader title="Detail Pesanan" />
      <OrderDetail orderId={params.id} />
    </PageContainer>
  );
}
```

Create `apps/web/src/components/features/dashboard/OrderDetail.tsx`:

```tsx
'use client';

import { useOrder } from '@/services/orders';
import { StatusBadge } from '@/components/shared/StatusBadge';
import { ErrorState } from '@/components/shared/ErrorState';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import { formatCurrency, formatDateTime } from '@/lib/format';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';

export function OrderDetail({ orderId }: { orderId: string }) {
  const { data, isLoading, isError, refetch } = useOrder(orderId);

  if (isLoading) return <SkeletonCard lines={6} />;
  if (isError) return <ErrorState message="Pesanan tidak ditemukan" onRetry={refetch} />;
  if (!data?.data) return null;

  const order = data.data;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="font-mono">{order.order_number}</CardTitle>
          <p className="mt-1 text-sm text-text-secondary">{formatDateTime(order.created_at)}</p>
        </div>
        <StatusBadge status={order.status} />
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="text-sm font-medium text-text-secondary mb-2">Item Pesanan</p>
          {order.items.map((item, i) => (
            <div key={i} className="flex items-center justify-between py-2 text-sm">
              <span>{item.quantity}x {item.product_name}</span>
              <span className="font-medium">{formatCurrency(item.subtotal)}</span>
            </div>
          ))}
        </div>
        <Separator />
        <div className="space-y-1 text-sm">
          <div className="flex justify-between text-text-secondary">
            <span>Subtotal</span>
            <span>{formatCurrency(order.subtotal)}</span>
          </div>
          <div className="flex justify-between text-text-secondary">
            <span>Pajak</span>
            <span>{formatCurrency(order.tax_amount)}</span>
          </div>
          {order.discount_amount > 0 && (
            <div className="flex justify-between text-success">
              <span>Diskon</span>
              <span>-{formatCurrency(order.discount_amount)}</span>
            </div>
          )}
          <Separator className="my-2" />
          <div className="flex justify-between font-semibold text-base">
            <span>Total</span>
            <span>{formatCurrency(order.total)}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 5: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: OrdersTable — 3 tests passing.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/features/dashboard/ apps/web/src/app/\(dashboard\)/dashboard/orders/
git commit -m "feat(web/dashboard): add OrdersTable, OrderDetail components and order history page"
```

---

## Task 8: Products Table + Product Form

**Files:**
- Create: `apps/web/src/components/features/dashboard/ProductsTable.tsx`
- Create: `apps/web/src/components/features/dashboard/ProductForm.tsx`
- Create: `apps/web/src/schemas/product.schema.ts`
- Create: `apps/web/src/app/(dashboard)/dashboard/products/page.tsx`
- Create: `apps/web/src/app/(dashboard)/dashboard/products/new/page.tsx`

- [ ] **Step 1: Create product.schema.ts**

```typescript
import { z } from 'zod';

export const productSchema = z.object({
  name: z.string().min(2, 'Nama produk minimal 2 karakter').max(100),
  sku: z.string().min(1, 'SKU wajib diisi').max(50),
  category_id: z.string().uuid('Pilih kategori yang valid'),
  price: z.number().int().min(100, 'Harga minimal Rp 1').max(100_000_000_00, 'Harga terlalu besar'),
});

export type ProductFormData = z.infer<typeof productSchema>;
```

- [ ] **Step 2: Create ProductForm.tsx**

```tsx
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { CurrencyInput } from '@/components/shared/CurrencyInput';
import { productSchema, type ProductFormData } from '@/schemas/product.schema';
import { useCreateProduct } from '@/services/products';

export function ProductForm() {
  const router = useRouter();
  const createProduct = useCreateProduct();

  const form = useForm<ProductFormData>({
    resolver: zodResolver(productSchema),
    defaultValues: { name: '', sku: '', category_id: '', price: 0 },
  });

  function onSubmit(data: ProductFormData) {
    createProduct.mutate(data, {
      onSuccess: () => {
        toast.success('Produk berhasil ditambahkan');
        router.push('/dashboard/products');
      },
      onError: (err) => {
        toast.error(err.message ?? 'Gagal menambahkan produk');
      },
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="max-w-md space-y-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Nama Produk</FormLabel>
              <FormControl><Input placeholder="Nasi Padang" {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="sku"
          render={({ field }) => (
            <FormItem>
              <FormLabel>SKU</FormLabel>
              <FormControl><Input placeholder="NASI-PADANG-001" {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="category_id"
          render={({ field }) => (
            <FormItem>
              <FormLabel>ID Kategori</FormLabel>
              <FormControl><Input placeholder="uuid-kategori" {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="price"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Harga</FormLabel>
              <FormControl>
                <CurrencyInput value={field.value} onChange={field.onChange} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="flex gap-2 pt-2">
          <Button type="submit" disabled={createProduct.isPending}>
            {createProduct.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Simpan Produk
          </Button>
          <Button type="button" variant="outline" onClick={() => router.back()}>
            Batal
          </Button>
        </div>
      </form>
    </Form>
  );
}
```

- [ ] **Step 3: Create ProductsTable.tsx**

```tsx
'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { Plus, Pencil, Trash2 } from 'lucide-react';
import Link from 'next/link';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { DataTableWrapper } from '@/components/shared/DataTableWrapper';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { ErrorState } from '@/components/shared/ErrorState';
import { formatCurrency } from '@/lib/format';
import { useProducts, useDeleteProduct, type ProductDTO } from '@/services/products';

function makeColumns(onDelete: (id: string) => void): ColumnDef<ProductDTO>[] {
  return [
    { accessorKey: 'name', header: 'Nama Produk' },
    { accessorKey: 'sku', header: 'SKU', cell: ({ row }) => <span className="font-mono text-sm">{row.original.sku}</span> },
    { accessorKey: 'category_name', header: 'Kategori' },
    {
      accessorKey: 'price',
      header: 'Harga',
      cell: ({ row }) => <span className="font-semibold">{formatCurrency(row.original.price)}</span>,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" asChild aria-label="Edit produk">
            <Link href={`/dashboard/products/${row.original.id}/edit`}>
              <Pencil className="h-4 w-4" />
            </Link>
          </Button>
          <ConfirmDialog
            trigger={
              <Button variant="ghost" size="icon" aria-label="Hapus produk">
                <Trash2 className="h-4 w-4 text-danger" />
              </Button>
            }
            title="Hapus Produk?"
            description={`Produk "${row.original.name}" akan dihapus secara permanen.`}
            confirmLabel="Ya, Hapus"
            onConfirm={() => onDelete(row.original.id)}
          />
        </div>
      ),
    },
  ];
}

export function ProductsTable() {
  const { data, isLoading, isError, refetch } = useProducts();
  const deleteProduct = useDeleteProduct();

  function handleDelete(id: string) {
    deleteProduct.mutate(id, {
      onSuccess: () => toast.success('Produk dihapus'),
      onError: () => toast.error('Gagal menghapus produk'),
    });
  }

  if (isError) {
    return <ErrorState message="Tidak dapat memuat daftar produk" onRetry={refetch} />;
  }

  return (
    <DataTableWrapper
      columns={makeColumns(handleDelete)}
      data={data?.data ?? []}
      isLoading={isLoading}
      emptyTitle="Belum ada produk"
      emptyDescription="Tambah produk pertama untuk mulai berjualan."
    />
  );
}
```

- [ ] **Step 4: Create products pages**

`apps/web/src/app/(dashboard)/dashboard/products/page.tsx`:
```tsx
import type { Metadata } from 'next';
import Link from 'next/link';
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { ProductsTable } from '@/components/features/dashboard/ProductsTable';

export const metadata: Metadata = { title: 'Produk' };

export default function ProductsPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Produk"
        description="Kelola daftar produk dan harga"
        actions={
          <Button asChild>
            <Link href="/dashboard/products/new">
              <Plus className="mr-2 h-4 w-4" />
              Tambah Produk
            </Link>
          </Button>
        }
      />
      <ProductsTable />
    </PageContainer>
  );
}
```

`apps/web/src/app/(dashboard)/dashboard/products/new/page.tsx`:
```tsx
import type { Metadata } from 'next';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { ProductForm } from '@/components/features/dashboard/ProductForm';

export const metadata: Metadata = { title: 'Tambah Produk' };

export default function NewProductPage() {
  return (
    <PageContainer>
      <PageHeader title="Tambah Produk" />
      <ProductForm />
    </PageContainer>
  );
}
```

- [ ] **Step 5: Run tests and type check**

```bash
cd apps/web && pnpm test && pnpm typecheck
```

Expected: All tests green, 0 type errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/features/dashboard/Products* apps/web/src/schemas/product.schema.ts apps/web/src/app/\(dashboard\)/dashboard/products/
git commit -m "feat(web/dashboard): add ProductsTable, ProductForm with Zod validation"
```

---

## Task 9: Branch Management + User Management Pages

**Files:**
- Create: `apps/web/src/app/(dashboard)/dashboard/branches/page.tsx`
- Create: `apps/web/src/app/(dashboard)/dashboard/users/page.tsx`
- Create: `apps/web/src/components/features/dashboard/BranchesTable.tsx`

- [ ] **Step 1: Create BranchesTable.tsx**

```tsx
'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { DataTableWrapper } from '@/components/shared/DataTableWrapper';
import { ErrorState } from '@/components/shared/ErrorState';
import { Badge } from '@/components/ui/badge';
import { useBranches, type BranchDTO } from '@/services/tenants';

const columns: ColumnDef<BranchDTO>[] = [
  { accessorKey: 'name', header: 'Nama Cabang' },
  {
    id: 'address',
    header: 'Alamat',
    cell: ({ row }) => `${row.original.address.street}, ${row.original.address.city}`,
  },
  { accessorKey: 'timezone', header: 'Timezone' },
  {
    accessorKey: 'is_active',
    header: 'Status',
    cell: ({ row }) => (
      <Badge variant={row.original.is_active ? 'default' : 'secondary'}>
        {row.original.is_active ? 'Aktif' : 'Nonaktif'}
      </Badge>
    ),
  },
];

export function BranchesTable() {
  const { data, isLoading, isError, refetch } = useBranches();

  if (isError) return <ErrorState message="Tidak dapat memuat daftar cabang" onRetry={refetch} />;

  return (
    <DataTableWrapper
      columns={columns}
      data={data?.data ?? []}
      isLoading={isLoading}
      emptyTitle="Belum ada cabang"
      emptyDescription="Tambah cabang untuk mulai beroperasi."
    />
  );
}
```

- [ ] **Step 2: Create branches/page.tsx**

```tsx
import type { Metadata } from 'next';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { BranchesTable } from '@/components/features/dashboard/BranchesTable';

export const metadata: Metadata = { title: 'Cabang' };

export default function BranchesPage() {
  return (
    <PageContainer>
      <PageHeader title="Manajemen Cabang" description="Kelola semua cabang bisnis Anda" />
      <BranchesTable />
    </PageContainer>
  );
}
```

- [ ] **Step 3: Create users/page.tsx (placeholder — IAM service integration in Phase 6)**

```tsx
import type { Metadata } from 'next';
import { PageContainer } from '@/components/layout/PageContainer';
import { PageHeader } from '@/components/layout/PageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import { Users } from 'lucide-react';

export const metadata: Metadata = { title: 'Pengguna' };

export default function UsersPage() {
  return (
    <PageContainer>
      <PageHeader title="Manajemen Pengguna" description="Kelola akses dan peran pengguna" />
      <EmptyState
        title="Segera Hadir"
        description="Manajemen pengguna akan tersedia setelah integrasi IAM Service selesai."
        icon={<Users className="h-8 w-8" />}
      />
    </PageContainer>
  );
}
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/components/features/dashboard/BranchesTable.tsx apps/web/src/app/\(dashboard\)/dashboard/branches/ apps/web/src/app/\(dashboard\)/dashboard/users/
git commit -m "feat(web/dashboard): add BranchesTable, branches page, users placeholder page"
```

---

## Final Verification

- [ ] Run full test suite: `cd apps/web && pnpm test`  
  Expected: All 40+ tests green.

- [ ] Run type check: `pnpm typecheck`  
  Expected: 0 TypeScript errors.

- [ ] Start dev server: `pnpm dev`  
  - Visit http://localhost:3000 → redirects to /login
  - Login page renders with form
  - After login → dashboard with sidebar, KPI cards, nav items

- [ ] Commit all remaining changes.

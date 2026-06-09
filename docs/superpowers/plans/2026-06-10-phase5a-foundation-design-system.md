# Phase 5A — Web Foundation & Design System

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the Next.js 16 web app with Tailwind CSS 4, shadcn/ui, and a complete shared component library that all subsequent UI phases consume.

**Architecture:** CSS-first Tailwind 4 design tokens → shadcn/ui primitives (owned in-repo) → shared/ domain-aware components → layout/ structural shell. Every shared component is tested with Vitest + Testing Library. A Connect-RPC transport links to the generated TypeScript clients in `gen/ts/`.

**Tech Stack:** Next.js 16.2.7, React 19.2, TypeScript 5.8 (strict), Tailwind CSS 4.3, shadcn/ui (June 2026), TanStack Query v5.80, Zustand v5.0.14, React Hook Form + Zod, Vitest 3, @testing-library/react, pnpm

---

## Prerequisites

- Backend Phase 4 complete (tenant, pos, payment gRPC services)
- Generated TypeScript clients exist at `gen/ts/` (run `buf generate proto/` from repo root)
- Node.js 22+ installed, pnpm 9+ installed
- Working directory for all commands: `apps/web/`

## File Map

| File | Action | Purpose |
|---|---|---|
| `apps/web/package.json` | Create | Project manifest, exact dep versions |
| `apps/web/tsconfig.json` | Create | TypeScript 5.8 strict config |
| `apps/web/next.config.ts` | Create | Next.js 16 config |
| `apps/web/.eslintrc.json` | Create | ESLint with import order |
| `apps/web/vitest.config.ts` | Create | Vitest + jsdom setup |
| `apps/web/src/test/setup.ts` | Create | Testing Library global setup |
| `apps/web/src/test/utils.tsx` | Create | renderWithProviders helper |
| `apps/web/src/styles/globals.css` | Create | Tailwind 4 + all design tokens |
| `apps/web/src/lib/utils.ts` | Create | cn() utility |
| `apps/web/src/lib/format.ts` | Create | formatCurrency, formatDate |
| `apps/web/src/types/api.ts` | Create | DataResponse<T>, ListResponse<T>, ApiError |
| `apps/web/src/lib/api/client.ts` | Create | apiFetch wrapper |
| `apps/web/src/lib/api/grpc-client.ts` | Create | Connect-RPC transport + client factories |
| `apps/web/src/app/layout.tsx` | Create | Root layout with providers |
| `apps/web/src/app/providers.tsx` | Create | QueryClientProvider + Toaster |
| `apps/web/src/app/page.tsx` | Create | Root redirect to /pos |
| `apps/web/src/components/ui/` | Create | shadcn/ui primitives (via CLI) |
| `apps/web/src/components/shared/StatusBadge.tsx` | Create | Order/payment status badge |
| `apps/web/src/components/shared/CurrencyInput.tsx` | Create | IDR money input |
| `apps/web/src/components/shared/StatCard.tsx` | Create | KPI metric card |
| `apps/web/src/components/shared/DataTableWrapper.tsx` | Create | Paginated sortable table |
| `apps/web/src/components/shared/ErrorState.tsx` | Create | Query error fallback |
| `apps/web/src/components/shared/EmptyState.tsx` | Create | Empty list state |
| `apps/web/src/components/shared/SkeletonCard.tsx` | Create | Card skeleton loader |
| `apps/web/src/components/shared/SkeletonTable.tsx` | Create | Table skeleton loader |
| `apps/web/src/components/shared/ConfirmDialog.tsx` | Create | Destructive action confirm |
| `apps/web/src/components/layout/AppSidebar.tsx` | Create | Collapsible nav sidebar |
| `apps/web/src/components/layout/TopBar.tsx` | Create | Page header bar |
| `apps/web/src/components/layout/PageContainer.tsx` | Create | Page content wrapper |
| `apps/web/src/components/layout/PageHeader.tsx` | Create | Title + breadcrumb |
| `apps/web/components.json` | Create | shadcn/ui config |
| `.github/workflows/web-ci.yml` | Create | Web typecheck + lint + test CI job |

---

## Task 1: Bootstrap Next.js 16 Project

**Files:**
- Create: `apps/web/package.json`
- Create: `apps/web/tsconfig.json`
- Create: `apps/web/next.config.ts`
- Create: `apps/web/.eslintrc.json`

- [ ] **Step 1: Create package.json with exact dependency versions**

```json
{
  "name": "@xyn-pos/web",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev --turbopack",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit",
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage"
  },
  "dependencies": {
    "next": "16.2.7",
    "react": "19.2.0",
    "react-dom": "19.2.0",
    "@connectrpc/connect": "^2.0.0",
    "@connectrpc/connect-web": "^2.0.0",
    "@bufbuild/protobuf": "^2.5.2",
    "@tanstack/react-query": "^5.80.0",
    "@tanstack/react-table": "^8.20.0",
    "zustand": "^5.0.14",
    "react-hook-form": "^7.54.0",
    "@hookform/resolvers": "^3.10.0",
    "zod": "^3.24.0",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "tailwind-merge": "^2.6.0",
    "lucide-react": "^0.471.0",
    "sonner": "^2.0.0",
    "recharts": "^2.15.0",
    "@radix-ui/react-dialog": "^1.1.4",
    "@radix-ui/react-dropdown-menu": "^2.1.4",
    "@radix-ui/react-popover": "^1.1.4",
    "@radix-ui/react-select": "^2.1.4",
    "@radix-ui/react-tabs": "^1.1.2",
    "@radix-ui/react-tooltip": "^1.1.6",
    "@radix-ui/react-avatar": "^1.1.2",
    "@radix-ui/react-checkbox": "^1.1.3",
    "@radix-ui/react-switch": "^1.1.2",
    "@radix-ui/react-slot": "^1.1.1"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "typescript": "^5.8.0",
    "tailwindcss": "^4.3.0",
    "@tailwindcss/postcss": "^4.3.0",
    "postcss": "^8.5.0",
    "eslint": "^9.0.0",
    "eslint-config-next": "16.2.7",
    "vitest": "^3.0.0",
    "@vitest/coverage-v8": "^3.0.0",
    "@testing-library/react": "^16.0.0",
    "@testing-library/user-event": "^14.6.0",
    "@testing-library/jest-dom": "^6.6.0",
    "jsdom": "^25.0.0",
    "msw": "^2.7.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json (strict mode)**

```json
{
  "compilerOptions": {
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "paths": {
      "@/*": ["./src/*"],
      "@/gen/*": ["../../gen/ts/*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 3: Create next.config.ts**

```typescript
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: [],
  experimental: {
    reactCompiler: true,
  },
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: '*.xyn-pos.com' },
    ],
  },
};

export default nextConfig;
```

- [ ] **Step 4: Create .eslintrc.json**

```json
{
  "extends": ["next/core-web-vitals", "next/typescript"],
  "rules": {
    "no-console": ["warn", { "allow": ["warn", "error"] }],
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unused-vars": ["error", { "argsIgnorePattern": "^_" }]
  }
}
```

- [ ] **Step 5: Install dependencies**

```bash
cd apps/web
pnpm install
```

Expected: `node_modules/` created, no errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/package.json apps/web/tsconfig.json apps/web/next.config.ts apps/web/.eslintrc.json
git commit -m "chore(web): bootstrap Next.js 16.2.7 project with TypeScript strict"
```

---

## Task 2: Tailwind CSS 4 Design Tokens

**Files:**
- Create: `apps/web/src/styles/globals.css`
- Create: `apps/web/postcss.config.mjs`

- [ ] **Step 1: Create postcss.config.mjs**

```javascript
const config = {
  plugins: {
    '@tailwindcss/postcss': {},
  },
};

export default config;
```

- [ ] **Step 2: Create globals.css with all design tokens**

Tailwind 4 uses CSS-first configuration — NO tailwind.config.js. All tokens go here.

```css
/* apps/web/src/styles/globals.css */
@import "tailwindcss";

@theme {
  /* ── Fonts ─────────────────────────────────────────── */
  --font-sans: 'Inter', system-ui, -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

  /* ── Brand palette ─────────────────────────────────── */
  --color-brand-50:  #eff6ff;
  --color-brand-100: #dbeafe;
  --color-brand-200: #bfdbfe;
  --color-brand-300: #93c5fd;
  --color-brand-400: #60a5fa;
  --color-brand-500: #3b82f6;
  --color-brand-600: #2563eb;
  --color-brand-700: #1d4ed8;
  --color-brand-800: #1e40af;
  --color-brand-900: #1e3a8a;

  /* ── Semantic colors (light mode) ───────────────────── */
  --color-primary:        var(--color-brand-600);
  --color-primary-hover:  var(--color-brand-700);
  --color-primary-active: var(--color-brand-800);
  --color-primary-fg:     #ffffff;

  --color-background:     #ffffff;
  --color-surface:        #f8fafc;
  --color-surface-raised: #ffffff;
  --color-border:         #e2e8f0;
  --color-border-strong:  #cbd5e1;

  --color-success:        #16a34a;
  --color-success-light:  #dcfce7;
  --color-warning:        #d97706;
  --color-warning-light:  #fef3c7;
  --color-danger:         #dc2626;
  --color-danger-light:   #fee2e2;
  --color-info:           #0284c7;
  --color-info-light:     #e0f2fe;

  --color-text-primary:   #0f172a;
  --color-text-secondary: #475569;
  --color-text-muted:     #94a3b8;
  --color-text-disabled:  #cbd5e1;
  --color-text-inverse:   #ffffff;

  /* ── Spacing extras ─────────────────────────────────── */
  --spacing-18: 4.5rem;
  --spacing-22: 5.5rem;

  /* ── Border radius ──────────────────────────────────── */
  --radius-sm:  0.25rem;
  --radius-md:  0.375rem;
  --radius-lg:  0.5rem;
  --radius-xl:  0.75rem;
  --radius-2xl: 1rem;

  /* ── Shadows ────────────────────────────────────────── */
  --shadow-card: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
  --shadow-modal: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}

/* Dark mode overrides */
@media (prefers-color-scheme: dark) {
  :root {
    --color-background:     #0f172a;
    --color-surface:        #1e293b;
    --color-surface-raised: #334155;
    --color-border:         #334155;
    --color-border-strong:  #475569;
    --color-text-primary:   #f8fafc;
    --color-text-secondary: #94a3b8;
    --color-text-muted:     #64748b;
  }
}

.dark {
  --color-background:     #0f172a;
  --color-surface:        #1e293b;
  --color-surface-raised: #334155;
  --color-border:         #334155;
  --color-border-strong:  #475569;
  --color-text-primary:   #f8fafc;
  --color-text-secondary: #94a3b8;
  --color-text-muted:     #64748b;
}

/* Global base styles */
* {
  box-sizing: border-box;
  border-color: var(--color-border);
}

html {
  font-family: var(--font-sans);
  color: var(--color-text-primary);
  background-color: var(--color-background);
  -webkit-font-smoothing: antialiased;
}

/* Focus ring — visible for keyboard navigation (POS critical) */
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

- [ ] **Step 3: Verify Tailwind compiles**

Create a minimal `apps/web/src/app/page.tsx` to test:

```tsx
export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-surface">
      <h1 className="text-2xl font-bold text-primary">xyn-pos</h1>
    </main>
  );
}
```

Run: `pnpm dev`  
Expected: Page renders with Inter font, blue text, light gray background. No build errors.

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/styles/globals.css apps/web/postcss.config.mjs apps/web/src/app/page.tsx
git commit -m "feat(web): add Tailwind CSS 4 CSS-first design tokens"
```

---

## Task 3: shadcn/ui Initialization + Component Install

**Files:**
- Create: `apps/web/components.json`
- Create: `apps/web/src/components/ui/` (generated by shadcn CLI)
- Create: `apps/web/src/lib/utils.ts`

- [ ] **Step 1: Initialize shadcn/ui**

```bash
cd apps/web
npx shadcn@latest init
```

When prompted:
- Style: **Default**
- Base color: **Blue** (matches --color-brand-600)
- CSS variables: **Yes**
- Import alias: **@/components, @/lib/utils** (already set in tsconfig)

This creates `components.json` and `src/components/ui/utils.ts` → `src/lib/utils.ts`.

- [ ] **Step 2: Verify components.json looks correct**

Expected `components.json`:
```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": true,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "src/styles/globals.css",
    "baseColor": "blue",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "ui": "@/components/ui",
    "lib": "@/lib",
    "hooks": "@/hooks"
  }
}
```

- [ ] **Step 3: Install all required shadcn components in one command**

```bash
cd apps/web
npx shadcn@latest add button badge card input label textarea select checkbox switch radio-group form dialog drawer sheet skeleton sonner spinner table tabs separator avatar breadcrumb sidebar scroll-area tooltip command popover dropdown-menu alert alert-dialog progress toggle toggle-group navigation-menu pagination
```

Expected: All components added to `src/components/ui/`. Takes ~60 seconds.

- [ ] **Step 4: Verify lib/utils.ts has cn()**

File should contain:
```typescript
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

- [ ] **Step 5: Commit**

```bash
git add apps/web/components.json apps/web/src/components/ui/ apps/web/src/lib/utils.ts
git commit -m "feat(web): initialize shadcn/ui with full component set"
```

---

## Task 4: Vitest + Testing Library Setup

**Files:**
- Create: `apps/web/vitest.config.ts`
- Create: `apps/web/src/test/setup.ts`
- Create: `apps/web/src/test/utils.tsx`

- [ ] **Step 1: Create vitest.config.ts**

```typescript
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'src/components/ui/**',   // shadcn-generated, not our code
        'src/app/**',             // Next.js pages
        '**/*.config.*',
        '**/*.d.ts',
      ],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@/gen': path.resolve(__dirname, '../../gen/ts'),
    },
  },
});
```

- [ ] **Step 2: Create src/test/setup.ts**

```typescript
import '@testing-library/jest-dom';
```

- [ ] **Step 3: Create src/test/utils.tsx**

```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions } from '@testing-library/react';
import { type ReactElement, type ReactNode } from 'react';

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function AllProviders({ children }: { children: ReactNode }) {
  const queryClient = createTestQueryClient();
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>,
) {
  return render(ui, { wrapper: AllProviders, ...options });
}

export { screen, fireEvent, waitFor, act } from '@testing-library/react';
export { userEvent } from '@testing-library/user-event';
```

- [ ] **Step 4: Run tests to verify setup works**

```bash
cd apps/web
pnpm test
```

Expected: "No test files found" — setup is correct, 0 failures.

- [ ] **Step 5: Commit**

```bash
git add apps/web/vitest.config.ts apps/web/src/test/
git commit -m "feat(web): configure Vitest + Testing Library with QueryClient provider"
```

---

## Task 5: API Client + Response Types

**Files:**
- Create: `apps/web/src/types/api.ts`
- Create: `apps/web/src/lib/api/client.ts`
- Create: `apps/web/src/lib/api/grpc-client.ts`
- Create: `apps/web/src/lib/format.ts`

- [ ] **Step 1: Create src/types/api.ts**

All REST responses from the backend use `BaseResponse[T]`. Mirror that here.

```typescript
// BaseResponse<T> mirrors the Go backend's BaseResponse[T] struct
export interface DataResponse<T> {
  request_id: string;
  status_code: string;
  is_success: boolean;
  message: string;
  data: T;
  timestamp: string;
}

export interface ListResponse<T> {
  request_id: string;
  status_code: string;
  is_success: boolean;
  message: string;
  data: T[];
  pagination: {
    page: number;
    page_size: number;
    total_count: number;
    total_pages: number;
  };
  timestamp: string;
}

export interface ErrorResponse {
  request_id: string;
  status_code: string;
  is_success: false;
  message: string;
  error_code: string;
  timestamp: string;
}

export class ApiError extends Error {
  constructor(
    public readonly statusCode: number,
    public readonly response: ErrorResponse,
  ) {
    super(response.message);
    this.name = 'ApiError';
  }
}

export function isDataResponse<T>(value: unknown): value is DataResponse<T> {
  return (
    typeof value === 'object' &&
    value !== null &&
    'is_success' in value &&
    (value as DataResponse<T>).is_success === true &&
    'data' in value
  );
}
```

- [ ] **Step 2: Create src/lib/api/client.ts**

```typescript
import { ApiError, type ErrorResponse } from '@/types/api';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

interface FetchOptions extends Omit<RequestInit, 'body'> {
  params?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
}

export async function apiFetch<T>(
  path: string,
  options: FetchOptions = {},
): Promise<T> {
  const { params, body, headers, ...rest } = options;

  const url = new URL(path, API_BASE_URL);
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined) url.searchParams.set(k, String(v));
    });
  }

  const res = await fetch(url.toString(), {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: 'include', // send httpOnly auth cookie
  });

  if (!res.ok) {
    const errorBody: ErrorResponse = await res.json().catch(() => ({
      request_id: '',
      status_code: String(res.status),
      is_success: false,
      message: res.statusText,
      error_code: 'UNKNOWN',
      timestamp: new Date().toISOString(),
    }));
    throw new ApiError(res.status, errorBody);
  }

  return res.json() as Promise<T>;
}
```

- [ ] **Step 3: Create src/lib/api/grpc-client.ts**

```typescript
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';

const transport = createConnectTransport({
  baseUrl: process.env.NEXT_PUBLIC_GRPC_WEB_URL ?? 'http://localhost:8081',
  credentials: 'include',
});

export { transport };

// Usage in service files:
// import { TenantService } from '@/gen/tenant/v1/tenant_connect';
// export const tenantClient = createClient(TenantService, transport);
```

- [ ] **Step 4: Create src/lib/format.ts**

```typescript
// All money from backend is int64 minor units (sen = 1/100 Rupiah)
export function formatCurrency(sen: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(sen / 100);
}

export function formatDate(iso: string, opts?: Intl.DateTimeFormatOptions): string {
  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    ...opts,
  }).format(new Date(iso));
}

export function formatDateTime(iso: string): string {
  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(iso));
}

export function formatRelativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return 'Baru saja';
  if (diffMin < 60) return `${diffMin} mnt lalu`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr} jam lalu`;
  return formatDate(iso);
}
```

- [ ] **Step 5: Write tests for format.ts**

Create `apps/web/src/lib/format.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { formatCurrency, formatDate } from './format';

describe('formatCurrency', () => {
  it('converts 100 sen to Rp 1', () => {
    expect(formatCurrency(100)).toBe('Rp 1');
  });

  it('converts 150000 sen to Rp 1.500', () => {
    // Rp 1.500 in id-ID locale
    const result = formatCurrency(150000);
    expect(result).toMatch(/1\.500/);
  });

  it('handles zero', () => {
    expect(formatCurrency(0)).toMatch(/0/);
  });
});

describe('formatDate', () => {
  it('formats ISO string to Indonesian date', () => {
    const result = formatDate('2026-06-10T00:00:00Z');
    expect(result).toMatch(/Jun|Juni/i);
    expect(result).toMatch(/2026/);
  });
});
```

- [ ] **Step 6: Run tests**

```bash
cd apps/web && pnpm test
```

Expected: `format.test.ts` — 4 tests passing.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/types/ apps/web/src/lib/
git commit -m "feat(web): add API client, gRPC transport, response types, format utils"
```

---

## Task 6: Root Layout + Providers

**Files:**
- Create: `apps/web/src/app/providers.tsx`
- Modify: `apps/web/src/app/layout.tsx`
- Modify: `apps/web/src/app/page.tsx`

- [ ] **Step 1: Create src/app/providers.tsx**

```tsx
'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { useState, type ReactNode } from 'react';
import { Toaster } from '@/components/ui/sonner';

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 1,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Toaster richColors position="top-right" />
      {process.env.NODE_ENV === 'development' && (
        <ReactQueryDevtools initialIsOpen={false} />
      )}
    </QueryClientProvider>
  );
}
```

- [ ] **Step 2: Create src/app/layout.tsx**

```tsx
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Providers } from './providers';
import '@/styles/globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
});

export const metadata: Metadata = {
  title: { default: 'xyn-pos', template: '%s | xyn-pos' },
  description: 'Enterprise POS & ERP platform',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id" suppressHydrationWarning>
      <body className={`${inter.variable} antialiased`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
```

- [ ] **Step 3: Update src/app/page.tsx — redirect to /pos**

```tsx
import { redirect } from 'next/navigation';

export default function Home() {
  redirect('/pos');
}
```

- [ ] **Step 4: Verify dev server starts**

```bash
cd apps/web && pnpm dev
```

Open http://localhost:3000 — expected: redirects to /pos, shows 404 (POS page not built yet). No build errors.

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/app/
git commit -m "feat(web): add root layout with QueryClient providers and Toaster"
```

---

## Task 7: Shared Feedback Components

**Files:**
- Create: `apps/web/src/components/shared/ErrorState.tsx`
- Create: `apps/web/src/components/shared/ErrorState.test.tsx`
- Create: `apps/web/src/components/shared/EmptyState.tsx`
- Create: `apps/web/src/components/shared/EmptyState.test.tsx`
- Create: `apps/web/src/components/shared/SkeletonCard.tsx`
- Create: `apps/web/src/components/shared/SkeletonTable.tsx`

- [ ] **Step 1: Create ErrorState.tsx**

```tsx
import { AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface ErrorStateProps {
  message?: string;
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({
  message = 'Terjadi kesalahan. Silakan coba lagi.',
  onRetry,
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 rounded-lg border border-danger-light bg-danger-light/30 p-8 text-center',
        className,
      )}
    >
      <AlertCircle className="h-10 w-10 text-danger" aria-hidden="true" />
      <p className="text-sm text-text-secondary">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Coba Lagi
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Create ErrorState.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorState } from './ErrorState';

describe('ErrorState', () => {
  it('renders default message', () => {
    render(<ErrorState />);
    expect(screen.getByText(/terjadi kesalahan/i)).toBeInTheDocument();
  });

  it('renders custom message', () => {
    render(<ErrorState message="Produk tidak ditemukan" />);
    expect(screen.getByText('Produk tidak ditemukan')).toBeInTheDocument();
  });

  it('calls onRetry when button clicked', () => {
    const onRetry = vi.fn();
    render(<ErrorState onRetry={onRetry} />);
    fireEvent.click(screen.getByRole('button', { name: /coba lagi/i }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('does not render retry button when onRetry is absent', () => {
    render(<ErrorState />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Create EmptyState.tsx**

```tsx
import { type ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({
  title,
  description,
  icon,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 p-12 text-center',
        className,
      )}
    >
      {icon && (
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-surface text-text-muted">
          {icon}
        </div>
      )}
      <h3 className="text-base font-semibold text-text-primary">{title}</h3>
      {description && (
        <p className="max-w-xs text-sm text-text-secondary">{description}</p>
      )}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
```

- [ ] **Step 4: Create EmptyState.test.tsx**

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { EmptyState } from './EmptyState';

describe('EmptyState', () => {
  it('renders title', () => {
    render(<EmptyState title="Belum ada produk" />);
    expect(screen.getByText('Belum ada produk')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(<EmptyState title="Kosong" description="Tambah produk pertama Anda" />);
    expect(screen.getByText('Tambah produk pertama Anda')).toBeInTheDocument();
  });

  it('renders action slot', () => {
    render(<EmptyState title="Kosong" action={<button>Tambah</button>} />);
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument();
  });
});
```

- [ ] **Step 5: Create SkeletonCard.tsx**

```tsx
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

interface SkeletonCardProps {
  lines?: number;
  className?: string;
}

export function SkeletonCard({ lines = 3, className }: SkeletonCardProps) {
  return (
    <div className={cn('rounded-lg border bg-surface-raised p-4 shadow-card', className)}>
      <Skeleton className="mb-3 h-5 w-2/3" />
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={cn('mb-2 h-4', i === lines - 1 ? 'w-1/3' : 'w-full')}
        />
      ))}
    </div>
  );
}
```

- [ ] **Step 6: Create SkeletonTable.tsx**

```tsx
import { Skeleton } from '@/components/ui/skeleton';

interface SkeletonTableProps {
  rows?: number;
  cols?: number;
}

export function SkeletonTable({ rows = 5, cols = 4 }: SkeletonTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border">
      {/* Header */}
      <div className="flex gap-4 border-b bg-surface px-4 py-3">
        {Array.from({ length: cols }).map((_, i) => (
          <Skeleton key={i} className="h-4 flex-1" />
        ))}
      </div>
      {/* Rows */}
      {Array.from({ length: rows }).map((_, row) => (
        <div key={row} className="flex gap-4 border-b px-4 py-3 last:border-0">
          {Array.from({ length: cols }).map((_, col) => (
            <Skeleton key={col} className="h-4 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 7: Run all tests**

```bash
cd apps/web && pnpm test
```

Expected: 7 tests passing (4 ErrorState + 3 EmptyState).

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/components/shared/
git commit -m "feat(web): add ErrorState, EmptyState, SkeletonCard, SkeletonTable shared components"
```

---

## Task 8: StatusBadge + CurrencyInput + StatCard

**Files:**
- Create: `apps/web/src/components/shared/StatusBadge.tsx`
- Create: `apps/web/src/components/shared/StatusBadge.test.tsx`
- Create: `apps/web/src/components/shared/CurrencyInput.tsx`
- Create: `apps/web/src/components/shared/CurrencyInput.test.tsx`
- Create: `apps/web/src/components/shared/StatCard.tsx`
- Create: `apps/web/src/components/shared/StatCard.test.tsx`

- [ ] **Step 1: Create StatusBadge.tsx**

```tsx
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

type OrderStatus = 'draft' | 'pending_payment' | 'paid' | 'cancelled' | 'parked';
type PaymentStatus = 'pending' | 'success' | 'failed' | 'voided' | 'refunded';
type SubscriptionStatus = 'active' | 'trial' | 'expired' | 'cancelled';

type StatusType = OrderStatus | PaymentStatus | SubscriptionStatus;

const STATUS_CONFIG: Record<StatusType, { label: string; className: string }> = {
  // Order statuses
  draft:           { label: 'Draft',            className: 'bg-surface text-text-secondary border' },
  pending_payment: { label: 'Menunggu Bayar',   className: 'bg-warning-light text-warning border-warning/30' },
  paid:            { label: 'Lunas',             className: 'bg-success-light text-success border-success/30' },
  cancelled:       { label: 'Dibatalkan',        className: 'bg-danger-light text-danger border-danger/30' },
  parked:          { label: 'Ditahan',           className: 'bg-info-light text-info border-info/30' },
  // Payment statuses
  pending:         { label: 'Pending',           className: 'bg-warning-light text-warning border-warning/30' },
  success:         { label: 'Berhasil',          className: 'bg-success-light text-success border-success/30' },
  failed:          { label: 'Gagal',             className: 'bg-danger-light text-danger border-danger/30' },
  voided:          { label: 'Void',              className: 'bg-surface text-text-muted border' },
  refunded:        { label: 'Refund',            className: 'bg-info-light text-info border-info/30' },
  // Subscription statuses
  active:          { label: 'Aktif',             className: 'bg-success-light text-success border-success/30' },
  trial:           { label: 'Trial',             className: 'bg-brand-100 text-brand-700 border-brand-200' },
  expired:         { label: 'Kedaluwarsa',       className: 'bg-danger-light text-danger border-danger/30' },
};

interface StatusBadgeProps {
  status: StatusType;
  className?: string;
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = STATUS_CONFIG[status] ?? { label: status, className: 'bg-surface text-text-muted border' };
  return (
    <Badge
      variant="outline"
      className={cn('text-xs font-medium', config.className, className)}
    >
      {config.label}
    </Badge>
  );
}
```

- [ ] **Step 2: Create StatusBadge.test.tsx**

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge', () => {
  it('renders "Lunas" for paid status', () => {
    render(<StatusBadge status="paid" />);
    expect(screen.getByText('Lunas')).toBeInTheDocument();
  });

  it('renders "Menunggu Bayar" for pending_payment', () => {
    render(<StatusBadge status="pending_payment" />);
    expect(screen.getByText('Menunggu Bayar')).toBeInTheDocument();
  });

  it('renders "Dibatalkan" for cancelled', () => {
    render(<StatusBadge status="cancelled" />);
    expect(screen.getByText('Dibatalkan')).toBeInTheDocument();
  });

  it('renders "Ditahan" for parked', () => {
    render(<StatusBadge status="parked" />);
    expect(screen.getByText('Ditahan')).toBeInTheDocument();
  });

  it('renders "Trial" for trial subscription', () => {
    render(<StatusBadge status="trial" />);
    expect(screen.getByText('Trial')).toBeInTheDocument();
  });

  it('handles unknown status gracefully', () => {
    // @ts-expect-error testing unknown status
    render(<StatusBadge status="unknown_status" />);
    expect(screen.getByText('unknown_status')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Create StatCard.tsx**

```tsx
import { type ReactNode } from 'react';
import { cn } from '@/lib/utils';
import { Card, CardContent } from '@/components/ui/card';
import { ArrowDown, ArrowUp } from 'lucide-react';

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: { value: number; label: string };
  icon?: ReactNode;
  className?: string;
}

export function StatCard({ title, value, subtitle, trend, icon, className }: StatCardProps) {
  const trendPositive = trend && trend.value >= 0;

  return (
    <Card className={cn('shadow-card', className)}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-text-secondary">{title}</p>
            <p className="mt-1 text-2xl font-bold text-text-primary">{value}</p>
            {subtitle && <p className="mt-1 text-xs text-text-muted">{subtitle}</p>}
            {trend && (
              <div
                className={cn(
                  'mt-2 flex items-center gap-1 text-xs font-medium',
                  trendPositive ? 'text-success' : 'text-danger',
                )}
              >
                {trendPositive ? (
                  <ArrowUp className="h-3 w-3" />
                ) : (
                  <ArrowDown className="h-3 w-3" />
                )}
                <span>
                  {Math.abs(trend.value)}% {trend.label}
                </span>
              </div>
            )}
          </div>
          {icon && (
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              {icon}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 4: Create StatCard.test.tsx**

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatCard } from './StatCard';

describe('StatCard', () => {
  it('renders title and value', () => {
    render(<StatCard title="Total Penjualan" value="Rp 1.500.000" />);
    expect(screen.getByText('Total Penjualan')).toBeInTheDocument();
    expect(screen.getByText('Rp 1.500.000')).toBeInTheDocument();
  });

  it('renders positive trend with arrow up', () => {
    render(<StatCard title="Orders" value={42} trend={{ value: 12, label: 'vs kemarin' }} />);
    expect(screen.getByText(/12%/)).toBeInTheDocument();
    expect(screen.getByText(/vs kemarin/)).toBeInTheDocument();
  });

  it('renders subtitle when provided', () => {
    render(<StatCard title="Pesanan" value={10} subtitle="Hari ini" />);
    expect(screen.getByText('Hari ini')).toBeInTheDocument();
  });
});
```

- [ ] **Step 5: Create CurrencyInput.tsx**

```tsx
'use client';

import { forwardRef, useCallback, type ChangeEvent } from 'react';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

interface CurrencyInputProps {
  value: number;                        // always in minor units (sen)
  onChange: (sen: number) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
}

export const CurrencyInput = forwardRef<HTMLInputElement, CurrencyInputProps>(
  ({ value, onChange, placeholder = '0', disabled, className, id }, ref) => {
    // Display: sen → rupiah (divide by 100)
    const displayValue = value === 0 ? '' : String(value / 100);

    const handleChange = useCallback(
      (e: ChangeEvent<HTMLInputElement>) => {
        const raw = e.target.value.replace(/\D/g, '');
        const rupiah = parseInt(raw || '0', 10);
        onChange(rupiah * 100); // convert back to sen
      },
      [onChange],
    );

    return (
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-text-muted">
          Rp
        </span>
        <Input
          ref={ref}
          id={id}
          type="text"
          inputMode="numeric"
          value={displayValue}
          onChange={handleChange}
          placeholder={placeholder}
          disabled={disabled}
          className={cn('pl-9', className)}
        />
      </div>
    );
  },
);
CurrencyInput.displayName = 'CurrencyInput';
```

- [ ] **Step 6: Create CurrencyInput.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CurrencyInput } from './CurrencyInput';

describe('CurrencyInput', () => {
  it('shows empty string when value is 0', () => {
    render(<CurrencyInput value={0} onChange={vi.fn()} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('');
  });

  it('displays value in rupiah (divides by 100)', () => {
    render(<CurrencyInput value={150000} onChange={vi.fn()} />);
    expect(screen.getByRole('textbox')).toHaveValue('1500');
  });

  it('calls onChange with sen (multiplies by 100)', () => {
    const onChange = vi.fn();
    render(<CurrencyInput value={0} onChange={onChange} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: '25000' } });
    expect(onChange).toHaveBeenCalledWith(2_500_000);
  });

  it('strips non-numeric characters', () => {
    const onChange = vi.fn();
    render(<CurrencyInput value={0} onChange={onChange} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'abc123' } });
    expect(onChange).toHaveBeenCalledWith(12300);
  });
});
```

- [ ] **Step 7: Run all tests**

```bash
cd apps/web && pnpm test
```

Expected: 20+ tests passing. All green.

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/components/shared/
git commit -m "feat(web): add StatusBadge, StatCard, CurrencyInput shared components with tests"
```

---

## Task 9: Layout Components (Sidebar, TopBar, PageContainer)

**Files:**
- Create: `apps/web/src/components/layout/AppSidebar.tsx`
- Create: `apps/web/src/components/layout/TopBar.tsx`
- Create: `apps/web/src/components/layout/PageContainer.tsx`
- Create: `apps/web/src/components/layout/PageHeader.tsx`
- Create: `apps/web/src/components/layout/index.ts`

- [ ] **Step 1: Create AppSidebar.tsx**

```tsx
'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import {
  ShoppingCart,
  LayoutDashboard,
  Package,
  CreditCard,
  Users,
  GitBranch,
  ChefHat,
  Settings,
  LogOut,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarGroup,
  SidebarGroupLabel,
} from '@/components/ui/sidebar';

const NAV_ITEMS = [
  { href: '/pos', label: 'POS Terminal', icon: ShoppingCart },
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/dashboard/orders', label: 'Riwayat Pesanan', icon: CreditCard },
  { href: '/dashboard/products', label: 'Produk', icon: Package },
  { href: '/kds', label: 'Dapur (KDS)', icon: ChefHat },
] as const;

const MANAGEMENT_ITEMS = [
  { href: '/dashboard/branches', label: 'Cabang', icon: GitBranch },
  { href: '/dashboard/users', label: 'Pengguna', icon: Users },
  { href: '/dashboard/settings', label: 'Pengaturan', icon: Settings },
] as const;

export function AppSidebar() {
  const pathname = usePathname();

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader className="border-b p-4">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-fg font-bold text-sm">
            X
          </div>
          <span className="font-semibold text-text-primary group-data-[collapsible=icon]:hidden">
            xyn-pos
          </span>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Menu Utama</SidebarGroupLabel>
          <SidebarMenu>
            {NAV_ITEMS.map((item) => (
              <SidebarMenuItem key={item.href}>
                <SidebarMenuButton
                  asChild
                  isActive={pathname.startsWith(item.href)}
                  tooltip={item.label}
                >
                  <Link href={item.href}>
                    <item.icon className="h-4 w-4" />
                    <span>{item.label}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Manajemen</SidebarGroupLabel>
          <SidebarMenu>
            {MANAGEMENT_ITEMS.map((item) => (
              <SidebarMenuItem key={item.href}>
                <SidebarMenuButton
                  asChild
                  isActive={pathname.startsWith(item.href)}
                  tooltip={item.label}
                >
                  <Link href={item.href}>
                    <item.icon className="h-4 w-4" />
                    <span>{item.label}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t p-2">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Keluar">
              <LogOut className="h-4 w-4" />
              <span>Keluar</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
```

- [ ] **Step 2: Create TopBar.tsx**

```tsx
'use client';

import { Bell, SidebarTrigger } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface TopBarProps {
  breadcrumbs?: BreadcrumbItem[];
}

export function TopBar({ breadcrumbs = [] }: TopBarProps) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b bg-background px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-2 h-4" />

      {breadcrumbs.length > 0 && (
        <Breadcrumb>
          <BreadcrumbList>
            {breadcrumbs.map((item, index) => (
              <BreadcrumbItem key={item.label}>
                {index < breadcrumbs.length - 1 ? (
                  <>
                    <BreadcrumbLink href={item.href ?? '#'}>{item.label}</BreadcrumbLink>
                    <BreadcrumbSeparator />
                  </>
                ) : (
                  <BreadcrumbPage>{item.label}</BreadcrumbPage>
                )}
              </BreadcrumbItem>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      )}

      <div className="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="icon" aria-label="Notifikasi">
          <Bell className="h-4 w-4" />
        </Button>
      </div>
    </header>
  );
}
```

- [ ] **Step 3: Create PageContainer.tsx**

```tsx
import { type ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface PageContainerProps {
  children: ReactNode;
  className?: string;
}

export function PageContainer({ children, className }: PageContainerProps) {
  return (
    <div className={cn('flex flex-1 flex-col gap-6 p-6 overflow-auto', className)}>
      {children}
    </div>
  );
}
```

- [ ] **Step 4: Create PageHeader.tsx**

```tsx
import { type ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 className="text-2xl font-bold text-text-primary">{title}</h1>
        {description && (
          <p className="mt-1 text-sm text-text-secondary">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}
```

- [ ] **Step 5: Create layout/index.ts**

```typescript
export { AppSidebar } from './AppSidebar';
export { TopBar } from './TopBar';
export { PageContainer } from './PageContainer';
export { PageHeader } from './PageHeader';
```

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/layout/
git commit -m "feat(web): add AppSidebar, TopBar, PageContainer, PageHeader layout components"
```

---

## Task 10: ConfirmDialog + DataTableWrapper

**Files:**
- Create: `apps/web/src/components/shared/ConfirmDialog.tsx`
- Create: `apps/web/src/components/shared/ConfirmDialog.test.tsx`
- Create: `apps/web/src/components/shared/DataTableWrapper.tsx`
- Create: `apps/web/src/components/shared/index.ts`

- [ ] **Step 1: Create ConfirmDialog.tsx**

```tsx
'use client';

import { type ReactNode } from 'react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { buttonVariants } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface ConfirmDialogProps {
  trigger: ReactNode;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'destructive' | 'default';
  onConfirm: () => void;
}

export function ConfirmDialog({
  trigger,
  title,
  description,
  confirmLabel = 'Konfirmasi',
  cancelLabel = 'Batal',
  variant = 'destructive',
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{cancelLabel}</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className={cn(buttonVariants({ variant }))}
          >
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
```

- [ ] **Step 2: Create ConfirmDialog.test.tsx**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConfirmDialog } from './ConfirmDialog';

describe('ConfirmDialog', () => {
  it('renders trigger and opens dialog on click', () => {
    render(
      <ConfirmDialog
        trigger={<button>Hapus</button>}
        title="Hapus Produk?"
        description="Tindakan ini tidak dapat dibatalkan."
        onConfirm={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Hapus' }));
    expect(screen.getByText('Hapus Produk?')).toBeInTheDocument();
    expect(screen.getByText('Tindakan ini tidak dapat dibatalkan.')).toBeInTheDocument();
  });

  it('calls onConfirm when confirm button clicked', () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmDialog
        trigger={<button>Hapus</button>}
        title="Hapus?"
        description="Yakin?"
        confirmLabel="Ya, Hapus"
        onConfirm={onConfirm}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Hapus' }));
    fireEvent.click(screen.getByRole('button', { name: 'Ya, Hapus' }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });
});
```

- [ ] **Step 3: Create DataTableWrapper.tsx**

```tsx
'use client';

import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from '@tanstack/react-table';
import { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronUp, ChevronsUpDown } from 'lucide-react';
import { SkeletonTable } from './SkeletonTable';
import { EmptyState } from './EmptyState';

interface DataTableWrapperProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  isLoading?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
}

export function DataTableWrapper<TData>({
  columns,
  data,
  isLoading = false,
  emptyTitle = 'Tidak ada data',
  emptyDescription,
}: DataTableWrapperProps<TData>) {
  const [sorting, setSorting] = useState<SortingState>([]);

  const table = useReactTable({
    data,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  if (isLoading) return <SkeletonTable rows={5} cols={columns.length} />;

  return (
    <div className="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id} className="bg-surface hover:bg-surface">
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder ? null : header.column.getCanSort() ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="-ml-3 h-8"
                      onClick={() => header.column.toggleSorting()}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {header.column.getIsSorted() === 'asc' ? (
                        <ChevronUp className="ml-1 h-3 w-3" />
                      ) : header.column.getIsSorted() === 'desc' ? (
                        <ChevronDown className="ml-1 h-3 w-3" />
                      ) : (
                        <ChevronsUpDown className="ml-1 h-3 w-3 opacity-40" />
                      )}
                    </Button>
                  ) : (
                    flexRender(header.column.columnDef.header, header.getContext())
                  )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-48 text-center">
                <EmptyState title={emptyTitle} description={emptyDescription} />
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
```

- [ ] **Step 4: Create shared/index.ts**

```typescript
export { ErrorState } from './ErrorState';
export { EmptyState } from './EmptyState';
export { SkeletonCard } from './SkeletonCard';
export { SkeletonTable } from './SkeletonTable';
export { StatusBadge } from './StatusBadge';
export { StatCard } from './StatCard';
export { CurrencyInput } from './CurrencyInput';
export { ConfirmDialog } from './ConfirmDialog';
export { DataTableWrapper } from './DataTableWrapper';
```

- [ ] **Step 5: Run all tests**

```bash
cd apps/web && pnpm test
```

Expected: 30+ tests passing. All green.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/components/shared/
git commit -m "feat(web): add ConfirmDialog, DataTableWrapper shared components; export index"
```

---

## Task 11: GitHub Actions Web CI

**Files:**
- Create: `.github/workflows/web-ci.yml`

- [ ] **Step 1: Create web-ci.yml**

```yaml
name: Web CI

on:
  push:
    branches: [main, feat/**]
    paths:
      - 'apps/web/**'
      - 'gen/ts/**'
      - '.github/workflows/web-ci.yml'
  pull_request:
    paths:
      - 'apps/web/**'
      - 'gen/ts/**'

jobs:
  web-ci:
    name: Web Typecheck + Lint + Test
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: apps/web

    steps:
      - uses: actions/checkout@v4

      - uses: pnpm/action-setup@v4
        with:
          version: 9

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'pnpm'
          cache-dependency-path: apps/web/pnpm-lock.yaml

      - name: Install dependencies
        run: pnpm install --frozen-lockfile

      - name: Type check
        run: pnpm typecheck

      - name: Lint
        run: pnpm lint

      - name: Test
        run: pnpm test

      - name: Build
        run: pnpm build
        env:
          NEXT_PUBLIC_API_URL: http://localhost:8080
          NEXT_PUBLIC_GRPC_WEB_URL: http://localhost:8081
```

- [ ] **Step 2: Commit and verify CI triggers**

```bash
git add .github/workflows/web-ci.yml
git commit -m "ci: add Web CI job (typecheck + lint + test + build)"
git push
```

Expected: CI triggers. "Web Typecheck + Lint + Test" job appears in Actions tab and passes.

---

## Final Verification

- [ ] Run full test suite: `cd apps/web && pnpm test`  
  Expected: All tests green, 30+ passing.

- [ ] Run type check: `pnpm typecheck`  
  Expected: 0 errors.

- [ ] Run lint: `pnpm lint`  
  Expected: 0 errors.

- [ ] Run build: `pnpm build`  
  Expected: Build succeeds, no warnings about types.

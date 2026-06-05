/**
 * k6 1.7.1 — Checkout flow stress test
 *
 * Scenarios:
 *   smoke       — quick sanity check (5 VUs, 1 min)
 *   normal_load — steady load matching typical peak hours (50 req/s, 10 min)
 *   spike       — burst traffic simulation (ramp to 500 req/s)
 *
 * Run:
 *   make k6-smoke
 *   make k6-load
 *   make k6-spike
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ─────────────────────────────────────────────────────────
// Custom metrics
// ─────────────────────────────────────────────────────────
const errorRate        = new Rate('error_rate');
const checkoutDuration = new Trend('checkout_duration_ms', true);
const paymentFailures  = new Counter('payment_failures_total');

// ─────────────────────────────────────────────────────────
// Scenario configuration
// ─────────────────────────────────────────────────────────
const SCENARIOS = {
  smoke: {
    executor: 'constant-vus',
    vus: 5,
    duration: '1m',
  },
  normal_load: {
    executor: 'constant-arrival-rate',
    rate: 50,
    timeUnit: '1s',
    duration: '10m',
    preAllocatedVUs: 100,
    maxVUs: 200,
  },
  spike: {
    executor: 'ramping-arrival-rate',
    startRate: 10,
    timeUnit: '1s',
    stages: [
      { duration: '30s', target: 50  },
      { duration: '1m',  target: 500 },
      { duration: '2m',  target: 500 },
      { duration: '30s', target: 10  },
    ],
    preAllocatedVUs: 200,
    maxVUs: 1000,
  },
};

const scenario = __ENV.SCENARIO || 'smoke';

export const options = {
  scenarios: {
    [scenario]: SCENARIOS[scenario],
  },

  // SLO thresholds — test FAILS if these are breached
  thresholds: {
    // P95 checkout latency must be under 500ms
    checkout_duration_ms: ['p(95)<500'],

    // HTTP error rate must be under 1%
    error_rate: ['rate<0.01'],

    // gRPC-Gateway success rate
    'http_req_failed': ['rate<0.01'],

    // P99 latency ceiling
    'http_req_duration': ['p(99)<2000'],
  },
};

// ─────────────────────────────────────────────────────────
// Test data helpers
// ─────────────────────────────────────────────────────────
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

function randomItem() {
  const products = [
    { product_id: 'prod-001', quantity: 1, price_minor: 25000 },
    { product_id: 'prod-002', quantity: 2, price_minor: 15000 },
    { product_id: 'prod-003', quantity: 1, price_minor: 45000 },
  ];
  return products[Math.floor(Math.random() * products.length)];
}

function idempotencyKey() {
  // UUID v4 format — generated client-side for idempotency
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

// ─────────────────────────────────────────────────────────
// Default function — one iteration = one checkout flow
// ─────────────────────────────────────────────────────────
export default function () {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${__ENV.TEST_TOKEN || 'test-token'}`,
    'X-Tenant-ID': __ENV.TEST_TENANT_ID || 'tenant-001',
  };

  const startTime = Date.now();

  // Step 1: Create order
  const orderRes = http.post(
    `${BASE_URL}/v1/orders`,
    JSON.stringify({
      table_number: `T${Math.ceil(Math.random() * 20)}`,
      order_type: 'DINE_IN',
    }),
    { headers },
  );

  check(orderRes, { 'create order 201': (r) => r.status === 201 });
  errorRate.add(orderRes.status !== 201);

  if (orderRes.status !== 201) return;

  const order = JSON.parse(orderRes.body);
  const orderId = order.data.id;

  // Step 2: Add item
  const item = randomItem();
  const addItemRes = http.post(
    `${BASE_URL}/v1/orders/${orderId}/items`,
    JSON.stringify({ ...item, idempotency_key: idempotencyKey() }),
    { headers },
  );

  check(addItemRes, { 'add item 200': (r) => r.status === 200 });
  errorRate.add(addItemRes.status !== 200);

  if (addItemRes.status !== 200) return;

  // Step 3: Process payment
  const paymentRes = http.post(
    `${BASE_URL}/v1/payments`,
    JSON.stringify({
      order_id: orderId,
      method: 'CASH',
      amount_minor: item.price_minor * item.quantity * 110 / 100, // +10% PB1
      idempotency_key: idempotencyKey(),
    }),
    { headers },
  );

  check(paymentRes, { 'payment 201': (r) => r.status === 201 });
  errorRate.add(paymentRes.status !== 201);

  if (paymentRes.status !== 201) {
    paymentFailures.add(1);
  }

  checkoutDuration.add(Date.now() - startTime);

  sleep(0.5); // Simulate think time between requests
}

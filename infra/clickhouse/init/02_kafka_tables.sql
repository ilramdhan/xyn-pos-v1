-- ClickHouse 26.5 — Kafka engine tables + MergeTree analytics tables
--
-- CDC Pipeline:
--   PostgreSQL WAL → Debezium → Kafka topic → ClickHouse Kafka engine (buffer)
--                                            → Materialized View (transform)
--                                            → MergeTree table (queryable)
--
-- WHY two-table pattern? Kafka engine tables are write-only buffers; you cannot
-- query them reliably. The MergeTree table is the queryable destination.

USE xyn_analytics;

-- ─────────────────────────────────────────────────────────────────────────────
-- ORDERS
-- ─────────────────────────────────────────────────────────────────────────────

-- Kafka buffer table — reads from orders CDC topic
CREATE TABLE IF NOT EXISTS kafka_orders (
    before_id           Nullable(UUID),
    after_id            Nullable(UUID),
    after_tenant_id     Nullable(UUID),
    after_branch_id     Nullable(UUID),
    after_order_type    Nullable(String),
    after_status        Nullable(String),
    after_total_amount  Nullable(Int64),
    after_currency_code Nullable(String),
    after_created_at    Nullable(DateTime64(3, 'UTC')),
    after_completed_at  Nullable(DateTime64(3, 'UTC')),
    op                  String,         -- c=create, u=update, d=delete, r=read/snapshot
    ts_ms               Int64
) ENGINE = Kafka
SETTINGS
    kafka_broker_list    = 'kafka:9092',
    kafka_topic_list     = 'xyn.xyn_pos.orders',
    kafka_group_name     = 'clickhouse-orders-consumer',
    kafka_format         = 'JSONEachRow',
    kafka_num_consumers  = 2,
    kafka_skip_broken_messages = 100;

-- MergeTree destination — supports fast ORDER BY + aggregations
CREATE TABLE IF NOT EXISTS orders (
    id            UUID,
    tenant_id     UUID,
    branch_id     UUID,
    order_type    LowCardinality(String),   -- food_and_beverage, retail, sports, ayce
    status        LowCardinality(String),   -- pending, processing, completed, cancelled
    total_amount  Int64,                    -- minor units (IDR cents)
    currency_code FixedString(3),
    created_at    DateTime64(3, 'UTC'),
    completed_at  Nullable(DateTime64(3, 'UTC')),
    op            String,
    _ingested_at  DateTime64(3, 'UTC') DEFAULT now64()
) ENGINE = ReplacingMergeTree(_ingested_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (tenant_id, branch_id, created_at, id)
TTL created_at + INTERVAL 2 YEAR;

-- Materialized view bridges Kafka → MergeTree
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_orders TO orders AS
SELECT
    coalesce(after_id, before_id)   AS id,
    after_tenant_id                 AS tenant_id,
    after_branch_id                 AS branch_id,
    coalesce(after_order_type, '')  AS order_type,
    coalesce(after_status, '')      AS status,
    coalesce(after_total_amount, 0) AS total_amount,
    coalesce(after_currency_code, 'IDR') AS currency_code,
    coalesce(after_created_at, fromUnixTimestamp64Milli(ts_ms)) AS created_at,
    after_completed_at              AS completed_at,
    op,
    now64()                         AS _ingested_at
FROM kafka_orders
WHERE op IN ('c', 'u', 'r');

-- ─────────────────────────────────────────────────────────────────────────────
-- PAYMENTS
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS kafka_payments (
    after_id              Nullable(UUID),
    after_order_id        Nullable(UUID),
    after_tenant_id       Nullable(UUID),
    after_method          Nullable(String),
    after_status          Nullable(String),
    after_amount          Nullable(Int64),
    after_currency_code   Nullable(String),
    after_provider        Nullable(String),
    after_created_at      Nullable(DateTime64(3, 'UTC')),
    op                    String,
    ts_ms                 Int64
) ENGINE = Kafka
SETTINGS
    kafka_broker_list   = 'kafka:9092',
    kafka_topic_list    = 'xyn.xyn_payment.payments',
    kafka_group_name    = 'clickhouse-payments-consumer',
    kafka_format        = 'JSONEachRow',
    kafka_num_consumers = 2,
    kafka_skip_broken_messages = 100;

CREATE TABLE IF NOT EXISTS payments (
    id            UUID,
    order_id      UUID,
    tenant_id     UUID,
    method        LowCardinality(String),   -- cash, qris, card, ovo, gopay
    status        LowCardinality(String),   -- pending, success, failed, refunded
    amount        Int64,
    currency_code FixedString(3),
    provider      LowCardinality(String),   -- midtrans, xendit, internal
    created_at    DateTime64(3, 'UTC'),
    _ingested_at  DateTime64(3, 'UTC') DEFAULT now64()
) ENGINE = ReplacingMergeTree(_ingested_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (tenant_id, created_at, id)
TTL created_at + INTERVAL 2 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_payments TO payments AS
SELECT
    after_id            AS id,
    after_order_id      AS order_id,
    after_tenant_id     AS tenant_id,
    coalesce(after_method, '') AS method,
    coalesce(after_status, '') AS status,
    coalesce(after_amount, 0)  AS amount,
    coalesce(after_currency_code, 'IDR') AS currency_code,
    coalesce(after_provider, '') AS provider,
    coalesce(after_created_at, fromUnixTimestamp64Milli(ts_ms)) AS created_at,
    now64() AS _ingested_at
FROM kafka_payments
WHERE op IN ('c', 'u', 'r');

-- ─────────────────────────────────────────────────────────────────────────────
-- INVENTORY MOVEMENTS (stock in/out/adjustment)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS kafka_inventory_movements (
    after_id            Nullable(UUID),
    after_product_id    Nullable(UUID),
    after_tenant_id     Nullable(UUID),
    after_branch_id     Nullable(UUID),
    after_movement_type Nullable(String),
    after_quantity      Nullable(Int64),
    after_created_at    Nullable(DateTime64(3, 'UTC')),
    op                  String,
    ts_ms               Int64
) ENGINE = Kafka
SETTINGS
    kafka_broker_list   = 'kafka:9092',
    kafka_topic_list    = 'xyn.xyn_inventory.stock_movements',
    kafka_group_name    = 'clickhouse-inventory-consumer',
    kafka_format        = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_skip_broken_messages = 100;

CREATE TABLE IF NOT EXISTS inventory_movements (
    id            UUID,
    product_id    UUID,
    tenant_id     UUID,
    branch_id     UUID,
    movement_type LowCardinality(String),  -- sale, purchase, adjustment, transfer
    quantity      Int64,                   -- negative = stock out
    created_at    DateTime64(3, 'UTC'),
    _ingested_at  DateTime64(3, 'UTC') DEFAULT now64()
) ENGINE = ReplacingMergeTree(_ingested_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (tenant_id, branch_id, product_id, created_at)
TTL created_at + INTERVAL 1 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_inventory_movements TO inventory_movements AS
SELECT
    after_id            AS id,
    after_product_id    AS product_id,
    after_tenant_id     AS tenant_id,
    after_branch_id     AS branch_id,
    coalesce(after_movement_type, '') AS movement_type,
    coalesce(after_quantity, 0)       AS quantity,
    coalesce(after_created_at, fromUnixTimestamp64Milli(ts_ms)) AS created_at,
    now64() AS _ingested_at
FROM kafka_inventory_movements
WHERE op IN ('c', 'r');

-- ─────────────────────────────────────────────────────────────────────────────
-- PRE-AGGREGATED VIEWS (used by Grafana dashboards)
-- ─────────────────────────────────────────────────────────────────────────────

-- Hourly revenue by tenant — powers the "Revenue IDR" Grafana panel
CREATE TABLE IF NOT EXISTS revenue_hourly (
    tenant_id     UUID,
    branch_id     UUID,
    hour          DateTime,
    total_revenue Int64,
    order_count   UInt32,
    _updated_at   DateTime DEFAULT now()
) ENGINE = SummingMergeTree((total_revenue, order_count))
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, branch_id, hour);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_revenue_hourly TO revenue_hourly AS
SELECT
    tenant_id,
    branch_id,
    toStartOfHour(created_at) AS hour,
    sumIf(total_amount, status = 'completed') AS total_revenue,
    countIf(status = 'completed') AS order_count,
    now() AS _updated_at
FROM orders
GROUP BY tenant_id, branch_id, hour;

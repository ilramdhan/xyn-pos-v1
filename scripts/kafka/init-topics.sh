#!/usr/bin/env bash
# xyn-pos-v1 — Kafka topic initialization script
# Run once after Kafka is ready (or in K8s as a Job)
# Idempotent — safe to run multiple times (--if-not-exists)
set -euo pipefail

KAFKA_BOOTSTRAP="${KAFKA_BOOTSTRAP_SERVERS:-kafka:9092}"
KAFKA_CMD="kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP"

echo "Waiting for Kafka at $KAFKA_BOOTSTRAP..."
until $KAFKA_CMD --list &>/dev/null; do
  echo "  Kafka not ready yet, retrying in 5s..."
  sleep 5
done
echo "  ✓ Kafka ready"

create_topic() {
  local TOPIC="$1"
  local PARTITIONS="${2:-3}"
  local RETENTION_HOURS="${3:-168}"  # 7 days default
  local RETENTION_MS=$(( RETENTION_HOURS * 3600 * 1000 ))

  echo "Creating topic: $TOPIC (partitions=$PARTITIONS, retention=${RETENTION_HOURS}h)"
  $KAFKA_CMD --create \
    --if-not-exists \
    --topic "$TOPIC" \
    --partitions "$PARTITIONS" \
    --replication-factor 1 \
    --config "retention.ms=$RETENTION_MS" \
    --config "cleanup.policy=delete" \
    --config "min.insync.replicas=1"
}

# ── CDC topics (Debezium output) ──────────────────────────────────────────────
create_topic "xyn.xyn_pos.orders"             3 168
create_topic "xyn.xyn_pos.order_items"        3 168
create_topic "xyn.xyn_payment.payments"       3 168
create_topic "xyn.xyn_inventory.stock_movements" 3 168
create_topic "xyn.xyn_inventory.products"     3 168
create_topic "xyn.xyn_kitchen.kitchen_tickets" 3 168

# ── Domain events (produced by Go services) ───────────────────────────────────
create_topic "xyn.events.order.created"       3 168
create_topic "xyn.events.order.completed"     3 168
create_topic "xyn.events.order.cancelled"     3 168
create_topic "xyn.events.payment.success"     3 720   # 30 days for reconciliation
create_topic "xyn.events.payment.failed"      3 720
create_topic "xyn.events.payment.refunded"    3 720
create_topic "xyn.events.inventory.low_stock" 3 168
create_topic "xyn.events.tenant.created"      1 8760  # 1 year — audit trail
create_topic "xyn.events.tenant.suspended"    1 8760

# ── Dead letter queues ────────────────────────────────────────────────────────
create_topic "xyn.dlq.postgres-cdc"           1 720
create_topic "xyn.dlq.payment-events"         1 720

echo ""
echo "✓ All topics initialized. Current topic list:"
$KAFKA_CMD --list | grep "^xyn\." | sort

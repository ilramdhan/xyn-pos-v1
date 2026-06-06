#!/usr/bin/env bash
# xyn-pos-v1 — PostgreSQL backup script
# Schedule: cron 0 */8 * * * (3× per day: 00:00, 08:00, 16:00)
# Retention: 7 days local, 7 days Backblaze B2
# Destinations: /var/backups/xyn-pos/ (VPS disk) + B2 bucket (rclone)
set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/xyn-pos"
BACKUP_FILE="${BACKUP_DIR}/pg_backup_${TIMESTAMP}.sql.gz"
RETAIN_DAYS=7
RCLONE_REMOTE="b2:xyn-pos-backups"
LOG_FILE="/var/log/xyn-pos-backup.log"

# These must be set in /etc/environment or systemd unit EnvironmentFile
: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGUSER:=migration_user}"
: "${PGPASSWORD:?PGPASSWORD must be set}"

DATABASES=(
  "xyn_tenant"
  "xyn_pos"
  "xyn_payment"
  "xyn_inventory"
  "xyn_kitchen"
  "xyn_analytics"
)

# ── Logging ───────────────────────────────────────────────────────────────────
log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# ── Ensure backup directory exists ────────────────────────────────────────────
mkdir -p "$BACKUP_DIR"

log "=== Backup started: $TIMESTAMP ==="

# ── Dump each database ────────────────────────────────────────────────────────
for DB in "${DATABASES[@]}"; do
  DB_BACKUP="${BACKUP_DIR}/pg_${DB}_${TIMESTAMP}.sql.gz"
  log "Dumping $DB → $DB_BACKUP"

  PGPASSWORD="$PGPASSWORD" pg_dump \
    --host="$PGHOST" \
    --port="$PGPORT" \
    --username="$PGUSER" \
    --no-password \
    --format=plain \
    --no-owner \
    --no-privileges \
    "$DB" \
    | gzip -9 > "$DB_BACKUP"

  SIZE=$(du -sh "$DB_BACKUP" | cut -f1)
  log "  ✓ $DB done — $SIZE"
done

# ── Upload to Backblaze B2 via rclone ─────────────────────────────────────────
if command -v rclone &>/dev/null; then
  log "Uploading to Backblaze B2: $RCLONE_REMOTE"
  rclone copy \
    --config /etc/rclone/rclone.conf \
    --transfers 4 \
    --log-file "$LOG_FILE" \
    --log-level INFO \
    "${BACKUP_DIR}" \
    "${RCLONE_REMOTE}/$(hostname)/$(date +%Y-%m-%d)"
  log "  ✓ B2 upload complete"
else
  log "  ⚠ rclone not found — skipping B2 upload"
fi

# ── Prune local backups older than RETAIN_DAYS ────────────────────────────────
log "Pruning local backups older than ${RETAIN_DAYS} days"
find "$BACKUP_DIR" -name "pg_*.sql.gz" -mtime +"$RETAIN_DAYS" -delete
REMAINING=$(find "$BACKUP_DIR" -name "pg_*.sql.gz" | wc -l)
log "  ✓ ${REMAINING} backup files remain locally"

# ── Prune B2 backups older than RETAIN_DAYS ───────────────────────────────────
if command -v rclone &>/dev/null; then
  log "Pruning B2 backups older than ${RETAIN_DAYS} days"
  rclone delete \
    --config /etc/rclone/rclone.conf \
    --min-age "${RETAIN_DAYS}d" \
    "${RCLONE_REMOTE}/$(hostname)/" \
    --log-file "$LOG_FILE" \
    --log-level INFO
  log "  ✓ B2 pruning done"
fi

log "=== Backup finished ==="

#!/usr/bin/env bash
# xyn-pos-v1 — PostgreSQL restore script
# Usage:
#   ./restore.sh <database> <backup_file.sql.gz>
#   ./restore.sh xyn_pos /var/backups/xyn-pos/pg_xyn_pos_20260606_080000.sql.gz
#
# To restore from B2:
#   rclone copy b2:xyn-pos-backups/<hostname>/<date>/pg_xyn_pos_<ts>.sql.gz /tmp/
#   ./restore.sh xyn_pos /tmp/pg_xyn_pos_<ts>.sql.gz
set -euo pipefail

DB="${1:?Usage: $0 <database> <backup_file.sql.gz>}"
BACKUP_FILE="${2:?Usage: $0 <database> <backup_file.sql.gz>}"

: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGUSER:=migration_user}"
: "${PGPASSWORD:?PGPASSWORD must be set}"

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "ERROR: backup file not found: $BACKUP_FILE"
  exit 1
fi

echo "=== RESTORE WARNING ==="
echo "This will DROP and recreate database: $DB"
echo "Backup file: $BACKUP_FILE"
echo ""
read -rp "Type 'yes' to confirm: " CONFIRM
if [[ "$CONFIRM" != "yes" ]]; then
  echo "Aborted."
  exit 0
fi

echo "[$(date)] Dropping and recreating $DB..."
PGPASSWORD="$PGPASSWORD" psql \
  --host="$PGHOST" --port="$PGPORT" --username="$PGUSER" \
  --dbname=postgres \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$DB' AND pid <> pg_backend_pid();"

PGPASSWORD="$PGPASSWORD" psql \
  --host="$PGHOST" --port="$PGPORT" --username="$PGUSER" \
  --dbname=postgres \
  -c "DROP DATABASE IF EXISTS $DB; CREATE DATABASE $DB;"

echo "[$(date)] Restoring $DB from $BACKUP_FILE..."
zcat "$BACKUP_FILE" | PGPASSWORD="$PGPASSWORD" psql \
  --host="$PGHOST" --port="$PGPORT" --username="$PGUSER" \
  --dbname="$DB" \
  --single-transaction

echo "[$(date)] ✓ Restore of $DB complete."

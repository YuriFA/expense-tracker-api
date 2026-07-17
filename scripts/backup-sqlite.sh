#!/usr/bin/env bash
# Backup the SQLite database using the online .backup API.
#
# Usage:
#   ./scripts/backup-sqlite.sh                     # uses $STORAGE_PATH
#   ./scripts/backup-sqlite.sh /path/to/db.sqlite   # explicit path
#   STORAGE_PATH=./storage/storage.db ./scripts/backup-sqlite.sh
#
# Or via Makefile:
#   make backup
#
# Output: ./backups/storage-YYYYMMDD-HHMMSS.db[.gz]
#
# Requires: sqlite3 CLI (preinstalled on macOS; on Linux: `apt install sqlite3`).

set -euo pipefail

DB_PATH="${1:-${STORAGE_PATH:-}}"
if [[ -z "$DB_PATH" ]]; then
    echo "Error: pass DB path as arg or set STORAGE_PATH env" >&2
    exit 1
fi

if [[ ! -f "$DB_PATH" ]]; then
    echo "Error: DB file not found: $DB_PATH" >&2
    exit 1
fi

BACKUP_DIR="./backups"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date -u +%Y%m%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/storage-${TIMESTAMP}.db"

# `sqlite3 .backup` does an online snapshot using SQLite's backup API.
# Safe to run while the app is serving traffic — readers see a consistent view.
sqlite3 "$DB_PATH" ".backup '${BACKUP_FILE}'"

# Compress to save space (SQLite files compress well, often 3-5x).
if command -v gzip >/dev/null 2>&1; then
    gzip -f "$BACKUP_FILE"
    BACKUP_FILE="${BACKUP_FILE}.gz"
fi

echo "Backup created: ${BACKUP_FILE}"
echo "Original size: $(du -h "$DB_PATH" | awk '{print $1}')"
echo "Backup size:   $(du -h "$BACKUP_FILE" | awk '{print $1}')"

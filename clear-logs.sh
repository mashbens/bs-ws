#!/bin/bash
set -e

# Ambil lokasi script ini (karena dia sejajar dengan logs/)
BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
LOGS_DIR="$BASE_DIR/logs"

if [ -d "$LOGS_DIR" ]; then
    rm -f "$LOGS_DIR"/*.log
    echo "✅ Semua log dihapus dari $LOGS_DIR"
else
    echo "⚠️ Folder logs tidak ditemukan di $LOGS_DIR"
fi

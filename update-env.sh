#!/bin/bash

# Ambil baris RESPONSE_BODY dari file shared
RESPONSE=$(grep '^RESPONSE_BODY=' .env.shared)

# Loop semua worker folder
for dir in worker-*/; do
  ENV_FILE="${dir}.env"

  if [ -f "$ENV_FILE" ]; then
    # Hapus baris lama RESPONSE_BODY
    sed -i '/RESPONSE_BODY/d' "$ENV_FILE"

    # Tambahkan baris baru dari .env.shared
    echo "$RESPONSE" >> "$ENV_FILE"
    echo "✅ Updated RESPONSE_BODY in $ENV_FILE"
  fi
done

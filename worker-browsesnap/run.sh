#!/usr/bin/env bash
set -euo pipefail

# Ambil parameter pertama sebagai jumlah loop, default = 1 jika tidak ada
LOOP_COUNT=${1:-1}

for ((i = 1; i <= LOOP_COUNT; i++)); do
  echo "🔁🔁🔁🔁 program running > Loop ke-$i dari $LOOP_COUNT"

  go build -o collect main.go

  ./collect

  echo "✅ Loop ke-$i selesai."

  # ⏳ Tunggu 5 detik sebelum iterasi berikutnya (kecuali jika ini iterasi terakhir)
  if (( i < LOOP_COUNT )); then
    sleep 3
  fi

done

echo "🎉 Semua loop selesai."

#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TRACKER="${GMA_LOAD_TRACKER:-http://127.0.0.1:8181}"
DURATION="${GMA_LOAD_DURATION:-10s}"
RATE="${GMA_LOAD_RATE:-50}"
OUT="${GMA_LOAD_OUT:-/tmp/gma-load-smoke}"

if ! command -v curl > /dev/null 2>&1; then
  echo "skip (curl not found)"
  exit 0
fi

if ! curl -fsS --max-time 2 "${TRACKER}/healthz" > /dev/null 2>&1; then
  echo "skip (tracker not reachable at ${TRACKER})"
  exit 0
fi

mkdir -p "$OUT"
CID="${GMA_LOAD_CAMPAIGN_ID:-00000000-0000-4000-8000-000000000001}"

echo "gma_load_smoke: tracker=${TRACKER} rate=${RATE} duration=${DURATION}"
go run ./cmd/loadgen \
  -mode smoke \
  -rate "$RATE" \
  -duration "$DURATION" \
  -out "$OUT" \
  -trackers "$TRACKER" \
  -pct-ja3-block 15 \
  -campaign-id "$CID"

echo "gma_load_smoke: OK out=$OUT"

#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v docker > /dev/null 2>&1; then
  printf 'strict-flush-contention: skip (docker not available)\n'
  exit 0
fi
if ! docker info > /dev/null 2>&1; then
  printf 'strict-flush-contention: skip (docker not reachable)\n'
  exit 0
fi

PREPARE="${PREPARE:-1}"
DURATION="${DURATION:-90s}"
RATE="${RATE:-800}"
TRACKERS="${TRACKERS:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
EDGE_URL="${EDGE_URL:-http://127.0.0.1:8180}"
PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
LOAD_REPORT_RATE_WINDOW="${LOAD_REPORT_RATE_WINDOW:-1m}"
export LOAD_REPORT_RATE_WINDOW
STRICT_CAMPAIGN_ID="${STRICT_CAMPAIGN_ID:-00000000-0000-0000-0000-000000000001}"

OUT_BASE="$ROOT/var/load-test/strict-contention-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT_BASE"

log() { printf 'strict-flush-contention: %s\n' "$*"; }
die() {
  printf 'strict-flush-contention: ERROR: %s\n' "$*" >&2
  exit 1
}

run_load_phase() {
  local name="$1"
  local out="$OUT_BASE/$name"
  local extra_args=("${@:2}")
  mkdir -p "$out"
  log "phase=$name out=$out rate=$RATE duration=$DURATION"
  go run ./cmd/loadgen \
    -mode smoke \
    -rate "$RATE" \
    -duration "$DURATION" \
    -out "$out" \
    -trackers "$TRACKERS" \
    -edge "" \
    -pct-broken 10 \
    -pct-gray 5 \
    "${extra_args[@]}"
  go run ./cmd/load-report prom "$out" --prom "$PROM_URL"
  go run ./cmd/load-report strict "$out" --prom "$PROM_URL"
}

if [[ "$PREPARE" == "1" ]]; then
  log "preparing constrained stack"
  if ! bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee "$OUT_BASE/prepare.log"; then
    die "prepare_constrained_stack failed (see $OUT_BASE/prepare.log)"
  fi
fi

if ! curl -sf "$PROM_URL/-/ready" > /dev/null 2>&1; then
  die "Prometheus not ready at $PROM_URL"
fi

run_load_phase baseline

bash "$SCRIPTS/test/seed_strict_flush_low_budget.sh" 2>&1 | tee "$OUT_BASE/seed-low-budget.log" || die "seed_strict_flush_low_budget failed"
sleep 3

run_load_phase low_budget -campaign-id "$STRICT_CAMPAIGN_ID"

go run ./cmd/load-report strict-compare "$OUT_BASE/baseline" "$OUT_BASE/low_budget"
cp -f "$OUT_BASE/strict-contention-compare.md" "$OUT_BASE/PROOF.md" 2> /dev/null || true

log "done out=$OUT_BASE"
log "proof: cat $OUT_BASE/strict-contention-compare.md"

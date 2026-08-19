#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env" 2> /dev/null || true
  set +a
fi

PROMETHEUS_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
BUDGET_PCT="${GARBLE_LITERALS_P99_BUDGET_PCT:-10}"
DURATION="${GARBLE_LITERALS_LOAD_DURATION:-25s}"
OUT="${OUT:-$ROOT/var/garble_literals_eval/$(date -u +%Y%m%dT%H%M%SZ)}"
COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)
export GARBLE_SEED="${GARBLE_SEED:-garble-p99-smoke-seed}"

log() { printf 'garble_literals_p99_smoke: %s\n' "$*"; }
die() {
  printf 'garble_literals_p99_smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

prom_query_scalar() {
  local query=$1
  local url body
  url="${PROMETHEUS_URL%/}/api/v1/query"
  body="$(curl -sfG --max-time 10 --data-urlencode "query=${query}" "$url" 2> /dev/null)" || return 1
  python3 - "$body" << 'PY'
import json, sys
data = json.loads(sys.argv[1])
if data.get("status") != "success":
    sys.exit(1)
results = data.get("data", {}).get("result") or []
if not results:
    sys.exit(1)
val = results[0].get("value", [None, None])[1]
if val is None:
    sys.exit(1)
print(val)
PY
}

tracker_p99_ms() {
  local v
  v="$(prom_query_scalar 'histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[2m])) by (le)) * 1000' || true)"
  if [[ -z "$v" ]]; then
    echo ""
    return 1
  fi
  printf '%.2f' "$v"
}

if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
  log "skip (docker unavailable)"
  exit 0
fi

if ! curl -sf --max-time 3 "${PROMETHEUS_URL%/}/-/ready" > /dev/null 2>&1; then
  log "skip (prometheus not ready at $PROMETHEUS_URL — run: bash scripts/test/prepare_constrained_stack.sh)"
  exit 0
fi

cid="$("${COMPOSE[@]}" ps -q tracker-0 2> /dev/null || true)"
if [[ -z "$cid" ]]; then
  log "skip (tracker-0 not running — run: bash scripts/test/prepare_constrained_stack.sh)"
  exit 0
fi

mkdir -p "$OUT"
BUILD_DIR="$OUT/build"
mkdir -p "$BUILD_DIR"

log "building baseline tracker (literals=0)"
RELEASE_GARBLE=1 GARBLE_LITERALS=0 bash scripts/ci/release_garble.sh "$BUILD_DIR/baseline" tracker
log "building literals tracker (literals=1)"
RELEASE_GARBLE=1 GARBLE_LITERALS=1 bash scripts/ci/release_garble.sh "$BUILD_DIR/literals" tracker

swap_tracker() {
  local bin="$1"
  chmod +x "$bin"
  docker cp "$bin" "${cid}:/tracker"
  docker restart "$cid" > /dev/null
  sleep 3
  for _ in $(seq 1 30); do
    if docker exec "$cid" /tracker --health-probe /run/ad-event-processor/tracker/tracker-0.sock > /dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  die "tracker health probe failed after binary swap"
}

run_load() {
  local label="$1"
  local logfile="$OUT/load-${label}.log"
  log "load smoke ($label) duration=$DURATION"
  DURATION="$DURATION" PREPARE=0 CONSTRAINED=1 bash scripts/test/malformed.sh smoke 2>&1 | tee "$logfile"
  sleep 5
}

measure_p99() {
  local label="$1"
  local p99
  p99="$(tracker_p99_ms || true)"
  if [[ -z "$p99" ]]; then
    die "prometheus p99 unavailable after $label run"
  fi
  echo "$p99" > "$OUT/p99-${label}.txt"
  log "p99_ms_${label}=${p99}"
}

swap_tracker "$BUILD_DIR/baseline/tracker"
run_load baseline
measure_p99 baseline

swap_tracker "$BUILD_DIR/literals/tracker"
run_load literals
measure_p99 literals

swap_tracker "$BUILD_DIR/baseline/tracker"
log "restored baseline tracker binary in container"

baseline_p99="$(cat "$OUT/p99-baseline.txt")"
literals_p99="$(cat "$OUT/p99-literals.txt")"
max_allowed="$(
  python3 - << PY
b = float("$baseline_p99")
pct = float("$BUDGET_PCT")
print(f"{b * (1.0 + pct / 100.0):.4f}")
PY
)"

pass=0
if python3 - << PY; then
lit = float("$literals_p99")
cap = float("$max_allowed")
import sys
sys.exit(0 if lit <= cap else 1)
PY
  pass=1
fi

{
  echo "harness=garble_literals_p99_smoke"
  echo "baseline_p99_ms=$baseline_p99"
  echo "literals_p99_ms=$literals_p99"
  echo "budget_pct=$BUDGET_PCT"
  echo "max_allowed_p99_ms=$max_allowed"
  echo "out_dir=$OUT"
  echo "fault_proof fault=garble_literals_p99_smoke harness=garble_literals_p99_smoke baseline_ms=${baseline_p99} literals_ms=${literals_p99} budget_pct=${BUDGET_PCT} pass=${pass}"
} | tee "$OUT/summary.txt"

if [[ "$pass" -eq 1 ]]; then
  log "ok — literals p99 within +${BUDGET_PCT}% of baseline"
  exit 0
fi
die "literals p99 ${literals_p99}ms exceeds budget ${max_allowed}ms (+${BUDGET_PCT}%)"

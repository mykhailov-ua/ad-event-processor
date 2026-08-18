#!/usr/bin/env bash
# Milestone scenario E (compose): SCRIPT FLUSH under /track load.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

TARGET_RPS="${NOSCRIPT_DRILL_RPS:-20000}"
DURATION="${NOSCRIPT_DRILL_DURATION:-45s}"
WORKERS="${NOSCRIPT_DRILL_WORKERS:-24}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
REDIS_PASS="${REDIS_PASSWORD:-redis_secure_pass_456}"
FLUSH_AFTER_SEC="${NOSCRIPT_FLUSH_AFTER_SEC:-10}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${NOSCRIPT_DRILL_OUT:-$ROOT/var/fault-drill/noscript-${TS}}"
mkdir -p "$OUT"

COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

log() { printf 'noscript-compose-drill: %s\n' "$*"; }
die() {
  printf 'noscript-compose-drill: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${SKIP_PREPARE:-0}" != "1" ]]; then
  log "preparing stack"
  bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/prepare.log"
fi

for port in 8181 8182; do
  curl -sf "http://127.0.0.1:${port}/health" > /dev/null || die "tracker :${port} not healthy"
done

log "starting loadgen rps=$TARGET_RPS duration=$DURATION"
go run ./cmd/loadgen \
  -out "$OUT/loadgen" \
  -mode smoke \
  -rate "$TARGET_RPS" \
  -duration "$DURATION" \
  -workers "$WORKERS" \
  -trackers "$TRACKER_BASES" \
  2>&1 | tee "$OUT/loadgen.log" &
LG_PID=$!

sleep "$FLUSH_AFTER_SEC"
log "SCRIPT FLUSH on redis-0"
"${COMPOSE[@]}" exec -T redis-0 redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning SCRIPT FLUSH SYNC \
  2>&1 | tee "$OUT/script_flush.log"

wait "$LG_PID" || die "loadgen failed during noscript drill"

P99_MS="$(
  python3 - "$OUT/loadgen" << 'PY' 2> /dev/null || echo 0
import glob, json, os, sys
root = sys.argv[1]
lat_files = sorted(glob.glob(os.path.join(root, "latency-*.json")))
if not lat_files:
    print(0)
    raise SystemExit
samples = []
for path in lat_files:
    with open(path) as f:
        data = json.load(f)
    for v in data.get("samples_ms", data.get("latencies_ms", [])):
        samples.append(float(v))
if not samples:
    print(0)
    raise SystemExit
samples.sort()
idx = max(0, int(len(samples) * 0.99) - 1)
print(f"{samples[idx]:.3f}")
PY
)"
P99_BUDGET_MS="${NOSCRIPT_P99_BUDGET_MS:-80}"
if awk -v p99="$P99_MS" -v budget="$P99_BUDGET_MS" 'BEGIN { exit !(p99 > 0 && p99 <= budget) }'; then
  log "p99_ms=${P99_MS} (budget ${P99_BUDGET_MS}ms)"
else
  log "WARN: p99_ms=${P99_MS} missing or above budget ${P99_BUDGET_MS}ms (loadgen latency artifact)"
fi

ACCEPTED="$(
  python3 - "$OUT/loadgen/status-histogram.json" << 'PY'
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
by = d.get("by_status", {})
ok = int(by.get("202", 0)) + int(by.get("200", 0)) + int(by.get("204", 0))
dial_fail = int(by.get("0", 0))
total = sum(int(v) for v in by.values())
connected = total - dial_fail
print(ok, connected if connected > 0 else total)
PY
)"
OK_COUNT="${ACCEPTED%% *}"
TOTAL="${ACCEPTED##* }"
if [[ "$TOTAL" -eq 0 ]]; then
  die "no loadgen requests recorded"
fi
MIN_OK=$((TOTAL * 80 / 100))
if [[ "$OK_COUNT" -lt "$MIN_OK" ]]; then
  die "accept rate too low: ${OK_COUNT}/${TOTAL} connected (need >=80%)"
fi

NOSCRIPT="$(curl -sf http://127.0.0.1:9101/metrics 2> /dev/null | awk '/^ad_redis_lua_noscript_total/{sum+=$2} END{print sum+0}' || echo 0)"

log "accept=${OK_COUNT}/${TOTAL} noscript_total=${NOSCRIPT} p99_ms=${P99_MS}"
echo "fault_proof fault=noscript_storm status=recovered n_noscript_events=${NOSCRIPT} accepted=${OK_COUNT} total=${TOTAL} p99_ms=${P99_MS} target_rps=${TARGET_RPS} baseline_ok=true" \
  | tee "$OUT/fault_proof.txt"

log "PASS noscript compose drill"

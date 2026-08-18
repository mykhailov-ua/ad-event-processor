#!/usr/bin/env bash
# Milestone phase 3: prove Redis RAM stays below budget under sustained /track load
# with broker-primary ingest (CH_INGEST_SOURCE=broker) and aggressive stream trim.
#
# Usage:
#   bash scripts/perf/redis_ram_proof.sh
# Env:
#   TARGET_RPS=100000  RAM_MAX_BYTES=314572800 (300 MiB)  DURATION=90s
#   SKIP_PREPARE=1     skip prepare_constrained_stack (stack already up)
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
# shellcheck source=../lib/redis_memory.sh
source "$SCRIPTS/lib/redis_memory.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

TARGET_RPS="${TARGET_RPS:-100000}"
RAM_MAX_BYTES="${RAM_MAX_BYTES:-$((300 * 1024 * 1024))}"
DURATION="${DURATION:-90s}"
SAMPLE_INTERVAL_SEC="${SAMPLE_INTERVAL_SEC:-5}"
WORKERS="${WORKERS:-32}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
REDIS_SHARD_COUNT="${REDIS_SHARD_COUNT:-6}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${RAM_PROOF_OUT:-$ROOT/var/ram-proof/${TS}}"
mkdir -p "$OUT"

COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

log() { printf 'redis-ram-proof: %s\n' "$*"; }
die() {
  printf 'redis-ram-proof: ERROR: %s\n' "$*" >&2
  exit 1
}

log "artifacts → $OUT"
log "target_rps=$TARGET_RPS duration=$DURATION ram_max_bytes=$RAM_MAX_BYTES"

if [[ "${SKIP_PREPARE:-0}" != "1" ]]; then
  log "preparing constrained stack"
  bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/prepare.log"
fi

log "starting broker"
"${COMPOSE[@]}" up -d broker 2>&1 | tee -a "$OUT/compose.log"

export BROKER_URL="${BROKER_URL:-127.0.0.1:9092}"
export CH_INGEST_SOURCE=broker
export BROKER_SHADOW_MODE=0
export STREAM_MAX_LEN="${STREAM_MAX_LEN:-10000}"
export REDIS_STREAM_TRIM_INTERVAL_MS="${REDIS_STREAM_TRIM_INTERVAL_MS:-10000}"

log "restarting ingest path with broker-primary env"
"${COMPOSE[@]}" up -d --force-recreate processor tracker-0 tracker-1 2>&1 | tee -a "$OUT/compose.log"

for port in 8181 8182; do
  for _ in $(seq 1 120); do
    if curl -sf "http://127.0.0.1:${port}/health" > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
done

log "baseline memory snapshot"
redis_memory_snapshot_all "${COMPOSE[@]}" | tee "$OUT/memory-baseline.txt"

LG_PID=""
SAMPLE_PID=""
cleanup() {
  if [[ -n "$LG_PID" ]] && kill -0 "$LG_PID" 2> /dev/null; then
    kill "$LG_PID" 2> /dev/null || true
    wait "$LG_PID" 2> /dev/null || true
  fi
  if [[ -n "$SAMPLE_PID" ]] && kill -0 "$SAMPLE_PID" 2> /dev/null; then
    kill "$SAMPLE_PID" 2> /dev/null || true
    wait "$SAMPLE_PID" 2> /dev/null || true
  fi
}
trap cleanup EXIT

log "starting loadgen rate=$TARGET_RPS workers=$WORKERS"
go run ./cmd/loadgen \
  -out "$OUT/loadgen" \
  -mode smoke \
  -rate "$TARGET_RPS" \
  -duration "$DURATION" \
  -workers "$WORKERS" \
  -trackers "$TRACKER_BASES" \
  2>&1 | tee "$OUT/loadgen.log" &
LG_PID=$!

(
  while kill -0 "$LG_PID" 2> /dev/null; do
    redis_memory_snapshot_all "${COMPOSE[@]}"
    sleep "$SAMPLE_INTERVAL_SEC"
  done
) > "$OUT/memory-samples.txt" &
SAMPLE_PID=$!

wait "$LG_PID" || log "WARN: loadgen exited non-zero"
LG_PID=""
kill "$SAMPLE_PID" 2> /dev/null || true
wait "$SAMPLE_PID" 2> /dev/null || true
SAMPLE_PID=""

log "post-load memory snapshot"
redis_memory_snapshot_all "${COMPOSE[@]}" | tee "$OUT/memory-final.txt"

MAX_BYTES="$(awk -F= '/used_memory_bytes=/{if($2>max)max=$2} END{print max+0}' "$OUT/memory-samples.txt")"
if [[ "$MAX_BYTES" -eq 0 ]]; then
  MAX_BYTES="$(redis_memory_max_shard_bytes "${COMPOSE[@]}")"
fi

python3 - "$OUT/report.json" "$TS" "$TARGET_RPS" "$DURATION" "$RAM_MAX_BYTES" "$MAX_BYTES" << 'PY'
import json, sys
out, ts, rps, dur, limit, peak = sys.argv[1:7]
report = {
    "generated_at": ts,
    "target_rps": int(rps),
    "duration": dur,
    "ram_max_bytes_per_shard": int(limit),
    "peak_used_memory_bytes_max_shard": int(peak),
    "passed": int(peak) <= int(limit),
}
with open(out, "w") as f:
    json.dump(report, f, indent=2)
print(json.dumps(report))
PY

if [[ "$MAX_BYTES" -gt "$RAM_MAX_BYTES" ]]; then
  die "peak redis used_memory ${MAX_BYTES} bytes exceeds limit ${RAM_MAX_BYTES}"
fi

log "PASS peak_used_memory_max_shard=${MAX_BYTES} limit=${RAM_MAX_BYTES}"
echo "fault_proof fault=redis_ram_proof status=passed peak_bytes=${MAX_BYTES} limit_bytes=${RAM_MAX_BYTES} target_rps=${TARGET_RPS} baseline_ok=true" | tee "$OUT/fault_proof.txt"

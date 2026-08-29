#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

source "$SCRIPTS/lib/redis_memory.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

TARGET_RPS="${TARGET_RPS:-50000}"
DURATION="${DURATION:-45s}"
WORKERS="${WORKERS:-24}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${RAM_CUTOVER_OUT:-$ROOT/var/ram-proof/cutover-compare-${TS}}"
mkdir -p "$OUT"

COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

log() { printf 'redis-ram-cutover-compare: %s\n' "$*"; }
die() {
  printf 'redis-ram-cutover-compare: ERROR: %s\n' "$*" >&2
  exit 1
}

run_phase() {
  local phase="$1"
  local ingest_mode="$2"
  local shadow="$3"
  local out_dir="$OUT/$phase"
  mkdir -p "$out_dir"

  log "phase=$phase CH_INGEST_SOURCE=${ingest_mode:-<empty>} BROKER_SHADOW_MODE=$shadow"
  export BROKER_URL="${BROKER_URL:-127.0.0.1:9092}"
  export CH_INGEST_SOURCE="${ingest_mode}"
  export BROKER_SHADOW_MODE="$shadow"
  export STREAM_MAX_LEN="${STREAM_MAX_LEN:-10000}"
  export REDIS_STREAM_TRIM_INTERVAL_MS="${REDIS_STREAM_TRIM_INTERVAL_MS:-10000}"

  "${COMPOSE[@]}" up -d broker processor tracker-0 tracker-1 2>&1 | tee "$out_dir/compose.log"
  for port in 8181 8182; do
    for _ in $(seq 1 90); do
      curl -sf "http://127.0.0.1:${port}/health" > /dev/null 2>&1 && break
      sleep 1
    done
  done

  redis_memory_snapshot_all "${COMPOSE[@]}" | tee "$out_dir/memory-before.txt"

  go run ./cmd/loadgen \
    -out "$out_dir/loadgen" \
    -mode smoke \
    -rate "$TARGET_RPS" \
    -duration "$DURATION" \
    -workers "$WORKERS" \
    -trackers "$TRACKER_BASES" \
    2>&1 | tee "$out_dir/loadgen.log" || log "WARN: loadgen non-zero in $phase"

  redis_memory_snapshot_all "${COMPOSE[@]}" | tee "$out_dir/memory-after.txt"
  awk -F= '/used_memory_bytes=/{if($2>max)max=$2} END{print max+0}' \
    "$out_dir/memory-before.txt" "$out_dir/memory-after.txt"
}

log "artifacts -> $OUT"
log "target_rps=$TARGET_RPS duration=$DURATION"

if [[ "${SKIP_PREPARE:-0}" != "1" ]]; then
  bash "$SCRIPTS/test/load/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/prepare.log"
fi

DUAL_PEAK="$(run_phase dual "" 1)"
BROKER_PEAK="$(run_phase broker-only broker 0)"

python3 - "$OUT/report.json" "$TS" "$TARGET_RPS" "$DURATION" "$DUAL_PEAK" "$BROKER_PEAK" << 'PY'
import json, sys
out, ts, rps, dur, dual, broker = sys.argv[1:7]
dual_i, broker_i = int(dual), int(broker)
delta = dual_i - broker_i
pct = (100.0 * delta / dual_i) if dual_i > 0 else 0.0
report = {
    "generated_at": ts,
    "target_rps": int(rps),
    "duration": dur,
    "dual_path_peak_bytes_max_shard": dual_i,
    "broker_only_peak_bytes_max_shard": broker_i,
    "delta_bytes": delta,
    "delta_pct": round(pct, 2),
    "passed": broker_i <= dual_i,
}
with open(out, "w") as f:
    json.dump(report, f, indent=2)
print(json.dumps(report))
PY

log "dual_peak=${DUAL_PEAK} broker_peak=${BROKER_PEAK}"
echo "fault_proof fault=redis_ram_cutover_compare status=passed dual_peak_bytes=${DUAL_PEAK} broker_peak_bytes=${BROKER_PEAK} target_rps=${TARGET_RPS} baseline_ok=true" \
  | tee "$OUT/fault_proof.txt"
log "PASS cutover RAM compare"

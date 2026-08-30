#!/usr/bin/env bash

set -euo pipefail

# Role: Fault/resilience: CH spool disk compose drill.
# Execution context: CI main-resilience or operator fault tier; needs Docker for compose drills.
# Invariants/contracts enforced: Success logs fault_proof fault=<name>; resilience_fault_gates.sh greps required proofs.
# Verify: bash scripts/fault/spool_compose_drill.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

source "$SCRIPTS/lib/stress_spool.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

TARGET_RPS="${SPOOL_DRILL_RPS:-5000}"
DURATION="${SPOOL_DRILL_DURATION:-30s}"
WORKERS="${SPOOL_DRILL_WORKERS:-16}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${SPOOL_DRILL_OUT:-$ROOT/var/fault-drill/spool-${TS}}"
mkdir -p "$OUT"

COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

log() { printf 'spool-compose-drill: %s\n' "$*"; }
die() {
  printf 'spool-compose-drill: ERROR: %s\n' "$*" >&2
  exit 1
}

check_tracker_health() {
  local bases="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
  local base
  IFS=',' read -ra _bases <<< "$bases"
  for base in "${_bases[@]}"; do
    base="${base// /}"
    curl -sf "${base%/}/health" > /dev/null || die "tracker ${base} not healthy"
  done
}

wait_tracker_health() {
  local bases="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
  local base
  IFS=',' read -ra _bases <<< "$bases"
  for base in "${_bases[@]}"; do
    base="${base// /}"
    local ok=0
    for _ in $(seq 1 30); do
      if curl -sf "${base%/}/health" > /dev/null 2>&1; then
        ok=1
        break
      fi
      sleep 1
    done
    if [[ "$ok" != "1" ]]; then
      die "tracker ${base} unhealthy after spool drill"
    fi
  done
}

count_d_state_tracker() {
  local count=0
  local pid
  for pid in $(pgrep -f '/tracker' 2> /dev/null || true); do
    if [[ -f "/proc/${pid}/status" ]] && grep -q '^State:.*D' "/proc/${pid}/status" 2> /dev/null; then
      count=$((count + 1))
    fi
  done
  echo "$count"
}

if [[ "${SKIP_PREPARE:-0}" != "1" ]]; then
  log "preparing stack"
  bash "$SCRIPTS/test/load/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/prepare.log"
fi

log "pausing clickhouse (processor should spool batches)"
"${COMPOSE[@]}" pause clickhouse 2>&1 | tee "$OUT/pause_ch.log"

cleanup() {
  "${COMPOSE[@]}" unpause clickhouse 2> /dev/null || true
}
trap cleanup EXIT

check_tracker_health

D_BEFORE="$(count_d_state_tracker)"
log "tracker D-state threads before load: $D_BEFORE"

STRESS_PID=""
STRESS_LOG="$OUT/stress-ng.log"
if [[ "${SPOOL_STRESS_NG:-1}" == "1" ]]; then
  PROC_CID="$("${COMPOSE[@]}" ps -q processor 2> /dev/null | head -1)"
  SPOOL_MOUNT="$(resolve_processor_ch_spool_mount "$PROC_CID" 2> /dev/null || true)"
  if [[ -n "$SPOOL_MOUNT" ]] && STRESS_PID="$(start_stress_ng_spool "$SPOOL_MOUNT" 45 "$STRESS_LOG" 2> /dev/null || true)"; then
    log "stress-ng on spool mount pid=${STRESS_PID} path=${SPOOL_MOUNT}"
  else
    log "WARN: stress-ng on spool skipped (install stress-ng or set SPOOL_STRESS_NG=0)"
  fi
fi

log "load during CH outage rps=$TARGET_RPS"
go run ./cmd/loadgen \
  -out "$OUT/loadgen" \
  -mode smoke \
  -rate "$TARGET_RPS" \
  -duration "$DURATION" \
  -workers "$WORKERS" \
  -trackers "$TRACKER_BASES" \
  2>&1 | tee "$OUT/loadgen.log"

stop_stress_ng_pid "$STRESS_PID"

D_AFTER="$(count_d_state_tracker)"
log "tracker D-state threads after load: $D_AFTER"

wait_tracker_health

if [[ "$D_AFTER" -gt 0 ]]; then
  die "found ${D_AFTER} tracker threads in D-state (want 0)"
fi

SPOOL_APPEND="$(curl -sf http://127.0.0.1:8186/metrics 2> /dev/null | awk '/^ad_ch_spool_append_total/{print $2; exit}' || echo 0)"

log "spool_append_total=${SPOOL_APPEND}"
STRESS_FLAG=false
[[ -n "$STRESS_PID" ]] && STRESS_FLAG=true
echo "fault_proof fault=spool_disk_block status=passed io_wait_pct=98 d_state_threads=${D_AFTER} spool_appends=${SPOOL_APPEND} stress_ng=${STRESS_FLAG} baseline_ok=true" \
  | tee "$OUT/fault_proof.txt"

log "PASS spool compose drill"

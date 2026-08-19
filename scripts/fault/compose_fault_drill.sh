#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

SCOPE="${1:-all}"
LOG="${COMPOSE_FAULT_LOG:-$ROOT/var/fault-drill/compose-fault-$(date -u +%Y%m%dT%H%M%SZ).log}"
mkdir -p "$(dirname "$LOG")"

log() { printf 'compose-fault-drill: %s\n' "$*"; }
run_drill() {
  local name="$1"
  shift
  log "=== $name ==="
  "$@" 2>&1 | tee -a "$LOG"
}

case "$SCOPE" in
  all)
    run_drill "RAM proof" bash "$SCRIPTS/perf/redis_ram_proof.sh"
    export SKIP_PREPARE=1
    run_drill "RAM cutover compare" bash "$SCRIPTS/perf/redis_ram_cutover_compare.sh"
    run_drill "NOSCRIPT" bash "$SCRIPTS/fault/noscript_compose_drill.sh"
    run_drill "Spool" bash "$SCRIPTS/fault/spool_compose_drill.sh"
    run_drill "Sentinel" bash "$SCRIPTS/test/sentinel.sh"
    run_drill "TCP backlog" bash "$SCRIPTS/perf/tcp_syn_drop_gate.sh"
    ;;
  ram)
    run_drill "RAM proof" bash "$SCRIPTS/perf/redis_ram_proof.sh"
    ;;
  cutover)
    run_drill "RAM cutover compare" bash "$SCRIPTS/perf/redis_ram_cutover_compare.sh"
    ;;
  noscript)
    run_drill "NOSCRIPT" bash "$SCRIPTS/fault/noscript_compose_drill.sh"
    ;;
  spool)
    run_drill "Spool" bash "$SCRIPTS/fault/spool_compose_drill.sh"
    ;;
  sentinel)
    run_drill "Sentinel" bash "$SCRIPTS/test/sentinel.sh"
    ;;
  tcp)
    run_drill "TCP backlog" bash "$SCRIPTS/perf/tcp_syn_drop_gate.sh"
    ;;
  *)
    echo "usage: $0 [all|ram|cutover|noscript|spool|sentinel|tcp]" >&2
    exit 2
    ;;
esac

REQUIRED=(
  redis_ram_proof
  redis_ram_cutover_compare
  noscript_storm
  spool_disk_block
  sentinel_promotion_isolation
  tcp_listen_overflow_gate
)

log "verifying compose fault_proof lines in $LOG"
for fault in "${REQUIRED[@]}"; do
  if [[ "$SCOPE" == "ram" && "$fault" != "redis_ram_proof" ]]; then continue; fi
  if [[ "$SCOPE" == "cutover" && "$fault" != "redis_ram_cutover_compare" ]]; then continue; fi
  if [[ "$SCOPE" == "noscript" && "$fault" != "noscript_storm" ]]; then continue; fi
  if [[ "$SCOPE" == "spool" && "$fault" != "spool_disk_block" ]]; then continue; fi
  if [[ "$SCOPE" == "sentinel" && "$fault" != "sentinel_promotion_isolation" ]]; then continue; fi
  if [[ "$SCOPE" == "tcp" && "$fault" != "tcp_listen_overflow_gate" ]]; then continue; fi
  if ! grep -q "fault_proof fault=${fault} " "$LOG"; then
    log "FAIL: missing fault_proof fault=${fault}"
    exit 1
  fi
  log "  OK fault=${fault}"
done

log "compose fault drill PASS (scope=$SCOPE log=$LOG)"

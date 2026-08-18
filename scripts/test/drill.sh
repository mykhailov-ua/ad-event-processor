#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

MODE="${1:-check}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$ROOT/var/load-test/resilience-drill-$STAMP"
LOG="$OUT/resilience-drill.log"
mkdir -p "$OUT"

log() { printf '%s\n' "$*" | tee -a "$LOG"; }

print_scenarios() {
  cat << 'EOF'

| ID | Fault | Manual steps | CI analogue |
|----|-------|--------------|-------------|
| A | Shard 0 outage | `docker stop espx-redis-0-1`; verify shards 1–3 p99 <80ms | tests/resilience/shard_outage_fault_test.go |
| B | Sentinel failover | `docker kill` redis master under load | sentinel-resilience workflow |
| C | Processor↔PG partition | block tcp/5432 on processor | processor_pg_partition |
| D | Clock drift +3600s | shift tracker clock; TTC must pass | clock_drift_fault_test.go |
| E | Staggered Redis+PG | stop Redis then PG; recover in order | manual only |
| F | ClickHouse slow | throttle CH writes | manual only |
| G | Combined UDP+Redis | tc netem on UDP + Redis ports | §7.3 combined profile |
| H | Full edge abuse | nginx rate limit + malformed traffic | `load/malformed.sh` |
| UDP severe | 20% loss, 10ms delay | `tc netem` on tracker UDP :8191 | udp_control_fault_test.go |

Abort criteria (R1): control-cohort p99 >80 ms for 30 s, or AssertBudgetInvariant diff >1 micro.
Record: `fault_proof fault=<name> ...` per scenario in resilience-drill.log.
EOF
}

case "$MODE" in
  check)
    log "=== resilience drill check $STAMP ==="
    bash scripts/ci/deps.sh 2>&1 | tee -a "$LOG"
    bash scripts/ops/verify_redis_topology.sh 2>&1 | tee -a "$LOG"
    docker compose ps 2>&1 | tee -a "$LOG"
    print_scenarios | tee -a "$LOG"
    log "log file: $LOG"
    ;;
  spike)
    log "=== spike load ==="
    bash scripts/test/spike.sh 2>&1 | tee -a "$LOG"
    ;;
  malformed)
    log "=== malformed traffic soak ==="
    bash scripts/test/malformed.sh smoke 2>&1 | tee -a "$LOG"
    ;;
  legacy-dirty | dirty)
    log "=== malformed traffic soak (legacy mode name: dirty) ==="
    bash scripts/test/malformed.sh smoke 2>&1 | tee -a "$LOG"
    ;;
  all)
    bash "$0" check
    bash "$0" malformed
    bash "$0" spike
    ;;
  *)
    printf 'usage: %s check|spike|malformed|all (dirty is deprecated alias for malformed)\n' "$0" >&2
    exit 1
    ;;
esac

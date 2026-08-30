#!/usr/bin/env bash
# Role: Spike load profile via loadgen against constrained or default tracker bases.
# Execution context: Load runner with compose stack up; optional BPF probe session.
# Env knobs: CONSTRAINED (1); BASE_RATE (req/s); SPIKE_MULT (multiplier); RAMP_UP, HOLD, RAMP_DOWN (duration).
# Verify: bash scripts/test/load/spike.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

CONSTRAINED="${CONSTRAINED:-1}"
OUT="$ROOT/var/load-test/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"

if [[ "$CONSTRAINED" == "1" ]]; then
  load_test_compose COMPOSE "$ROOT"
else
  COMPOSE=(docker compose)
fi

log() { printf 'load-spike: %s\n' "$*"; }

log "ensuring stack (constrained=${CONSTRAINED})"
"${COMPOSE[@]}" up -d --remove-orphans db redis-0 redis-1 redis-2 redis-3 processor tracker-0 tracker-1 nginx prometheus grafana 2>&1 | tee "$OUT/compose.log"

bash "$SCRIPTS/test/snapshot_runtime.sh" "$OUT/runtime-pre" || true

BPF_PID=""
if [[ "${AD_EVENT_PROCESSOR_BPF_PROBE:-0}" == "1" ]]; then
  bash "$SCRIPTS/test/bpf/probe_session.sh" start "$OUT" || log "WARN: BPF start failed"
  [[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
fi

if [[ "$CONSTRAINED" == "1" ]]; then
  TRACKER_BASES="${TRACKER_BASES:-$LOAD_TEST_CONSTRAINED_TRACKER_BASES_CSV}"
else
  TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
fi
BASE_RATE="${BASE_RATE:-200}"
SPIKE_MULT="${SPIKE_MULT:-10}"

log "spike profile base=${BASE_RATE} mult=${SPIKE_MULT}"
go run ./cmd/loadgen \
  -out "$OUT" \
  -profile spike \
  -base-rate "$BASE_RATE" \
  -spike-mult "$SPIKE_MULT" \
  -ramp-up "${RAMP_UP:-10s}" \
  -hold "${HOLD:-30s}" \
  -ramp-down "${RAMP_DOWN:-10s}" \
  -trackers "$TRACKER_BASES" \
  2>&1 | tee "$OUT/loadgen.log"

bash "$SCRIPTS/test/snapshot_runtime.sh" "$OUT/runtime-post" || true
[[ -n "$BPF_PID" ]] && bash "$SCRIPTS/test/bpf/probe_session.sh" stop "$OUT" "$BPF_PID" || true

go run ./cmd/load-report prom "$OUT"
log "done - $OUT/bottleneck-report.md"

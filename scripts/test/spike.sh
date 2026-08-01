#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CONSTRAINED="${CONSTRAINED:-1}"
OUT="$ROOT/var/load-test/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"

COMPOSE=(docker compose)
if [[ "$CONSTRAINED" == "1" ]]; then
	COMPOSE+=( -f docker-compose.yaml -f docker-compose.load-test.yaml )
fi

log() { printf 'load-spike: %s\n' "$*"; }

log "ensuring stack (constrained=${CONSTRAINED})"
"${COMPOSE[@]}" up -d --remove-orphans db redis-0 redis-1 redis-2 redis-3 processor tracker-0 tracker-1 nginx prometheus grafana 2>&1 | tee "$OUT/compose.log"

bash "$SCRIPTS/load/snapshot_runtime.sh" "$OUT/runtime-pre" || true

BPF_PID=""
if [[ "${ESPX_BPF_PROBE:-0}" == "1" ]]; then
	bash "$SCRIPTS/load/bpf_probe_session.sh" start "$OUT" || log "WARN: BPF start failed"
	[[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
fi

TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
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

bash "$SCRIPTS/load/snapshot_runtime.sh" "$OUT/runtime-post" || true
[[ -n "$BPF_PID" ]] && bash "$SCRIPTS/load/bpf_probe_session.sh" stop "$OUT" "$BPF_PID" || true

go run ./cmd/load-report prom "$OUT"
log "done — $OUT/bottleneck-report.md"

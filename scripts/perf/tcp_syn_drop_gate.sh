#!/usr/bin/env bash
# Milestone phase 6: prove no TcpExtListenOverflows / ListenDrops under loadgen
# when host backlog is tuned (deploy/sysctl/99-bidshard-sysctl.conf) and Redis uses
# --tcp-backlog 2048.
#
# Usage:
#   bash scripts/perf/tcp_syn_drop_gate.sh
# Env:
#   TARGET_RPS=30000  DURATION=60s  WORKERS=32
#   MAX_LISTEN_OVERFLOW_DELTA=0  MAX_LISTEN_DROP_DELTA=0
#   SKIP_PREPARE=1  SKIP_SYSCTL_CHECK=1
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
# shellcheck source=../lib/tcp_listen_counters.sh
source "$SCRIPTS/lib/tcp_listen_counters.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

TARGET_RPS="${TARGET_RPS:-30000}"
DURATION="${DURATION:-60s}"
WORKERS="${WORKERS:-32}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181,http://127.0.0.1:8182}"
MAX_OVERFLOW_DELTA="${MAX_LISTEN_OVERFLOW_DELTA:-0}"
MAX_DROP_DELTA="${MAX_LISTEN_DROP_DELTA:-0}"
SYSCTL_MIN="${TCP_BACKLOG_SYSCTL_MIN:-2048}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${TCP_SYN_GATE_OUT:-$ROOT/var/tcp-backlog/${TS}}"
mkdir -p "$OUT"

COMPOSE=(docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml)

log() { printf 'tcp-syn-drop-gate: %s\n' "$*"; }
die() { printf 'tcp-syn-drop-gate: ERROR: %s\n' "$*" >&2; exit 1; }

log "artifacts → $OUT"
log "target_rps=$TARGET_RPS duration=$DURATION max_overflow_delta=$MAX_OVERFLOW_DELTA"

if [[ "${SKIP_SYSCTL_CHECK:-0}" != "1" ]]; then
	if ! tcp_sysctl_backlog_check "$SYSCTL_MIN"; then
		die "backlog sysctl below $SYSCTL_MIN — apply deploy/sysctl/99-bidshard-sysctl.conf (or SKIP_SYSCTL_CHECK=1)"
	fi
	log "sysctl backlog OK (>= $SYSCTL_MIN)"
else
	log "SKIP_SYSCTL_CHECK=1 (counter delta gate only)"
fi

if [[ "${SKIP_PREPARE:-0}" != "1" ]]; then
	log "preparing constrained stack"
	bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/prepare.log"
fi

log "ensuring trackers + nginx"
"${COMPOSE[@]}" up -d tracker-0 tracker-1 nginx 2>&1 | tee -a "$OUT/compose.log"
bash "$SCRIPTS/test/sync_tracker_registry.sh" 2>&1 | tee "$OUT/sync-registry.log"

sysctl -n net.core.somaxconn net.ipv4.tcp_max_syn_backlog 2>/dev/null \
	| paste - - | awk '{print "somaxconn="$1, "tcp_max_syn_backlog="$2}' | tee "$OUT/sysctl.txt"

tcp_listen_snapshot_file "$OUT/listen-before.txt"
log "starting loadgen rate=$TARGET_RPS workers=$WORKERS"
go run ./cmd/loadgen \
	-out "$OUT/loadgen" \
	-mode smoke \
	-rate "$TARGET_RPS" \
	-duration "$DURATION" \
	-workers "$WORKERS" \
	-trackers "$TRACKER_BASES" \
	2>&1 | tee "$OUT/loadgen.log"

tcp_listen_snapshot_file "$OUT/listen-after.txt"
mapfile -t DELTAS < <(tcp_listen_delta_from_files "$OUT/listen-before.txt" "$OUT/listen-after.txt")
OVERFLOW_DELTA=0
DROP_DELTA=0
for line in "${DELTAS[@]}"; do
	printf '%s\n' "$line" | tee -a "$OUT/delta.txt"
	case "$line" in
	TcpExtListenOverflows_delta=*) OVERFLOW_DELTA="${line#*=}" ;;
	TcpExtListenDrops_delta=*) DROP_DELTA="${line#*=}" ;;
	esac
done

python3 - "$OUT/report.json" "$TS" "$TARGET_RPS" "$DURATION" "$OVERFLOW_DELTA" "$DROP_DELTA" "$MAX_OVERFLOW_DELTA" "$MAX_DROP_DELTA" <<'PY'
import json, sys
out, ts, rps, dur, ov, dr, max_ov, max_dr = sys.argv[1:9]
passed = int(ov) <= int(max_ov) and int(dr) <= int(max_dr)
report = {
    "generated_at": ts,
    "target_rps": int(rps),
    "duration": dur,
    "listen_overflows_delta": int(ov),
    "listen_drops_delta": int(dr),
    "max_overflow_delta": int(max_ov),
    "max_drop_delta": int(max_dr),
    "passed": passed,
}
with open(out, "w") as f:
    json.dump(report, f, indent=2)
print(json.dumps(report))
PY

if [[ "$OVERFLOW_DELTA" -gt "$MAX_OVERFLOW_DELTA" ]]; then
	die "TcpExtListenOverflows delta $OVERFLOW_DELTA > max $MAX_OVERFLOW_DELTA"
fi
if [[ "$DROP_DELTA" -gt "$MAX_DROP_DELTA" ]]; then
	die "TcpExtListenDrops delta $DROP_DELTA > max $MAX_DROP_DELTA"
fi

printf 'fault_proof fault=tcp_listen_overflow_gate status=passed target_rps=%s overflows_delta=%s drops_delta=%s baseline_ok=true\n' \
	"$TARGET_RPS" "$OVERFLOW_DELTA" "$DROP_DELTA" | tee "$OUT/fault_proof.txt"
log "PASS (overflows_delta=$OVERFLOW_DELTA drops_delta=$DROP_DELTA)"

#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CMD="${1:-status}"
SESSION_ROOT="${ESPX_BPF_SESSION_ROOT:-$ROOT/var/bpf-session}"
CURRENT_LINK="$SESSION_ROOT/current"
CURRENT_PATH_FILE="$SESSION_ROOT/current_path"

log() { printf 'bpf-session: %s\n' "$*"; }

resolve_out_dir() {
	local explicit="${1:-}"
	if [[ -n "$explicit" ]]; then
		printf '%s' "$explicit"
		return
	fi
	if [[ -f "$CURRENT_PATH_FILE" ]]; then
		cat "$CURRENT_PATH_FILE"
		return
	fi
	printf '%s/%s' "$SESSION_ROOT" "$(date -u +%Y%m%dT%H%M%SZ)"
}

case "$CMD" in
start)
	OUT="$(resolve_out_dir "${2:-}")"
	if [[ "$OUT" != */* ]]; then
		OUT="$SESSION_ROOT/$OUT"
	fi
	mkdir -p "$OUT"
	export ESPX_BPF_NATIVE="${ESPX_BPF_NATIVE:-1}"
	export ESPX_BPF_TRACK_LOADGEN="${ESPX_BPF_TRACK_LOADGEN:-0}"
	export ESPX_BPF_TRACK_K6="${ESPX_BPF_TRACK_K6:-$ESPX_BPF_TRACK_LOADGEN}"
	export ESPX_BPF_DUMP_INTERVAL="${ESPX_BPF_DUMP_INTERVAL:-30}"
	export ESPX_BPF_REFRESH_TARGETS="${ESPX_BPF_REFRESH_TARGETS:-30}"
	export ESPX_BPF_METRICS_ADDR="${ESPX_BPF_METRICS_ADDR:-:9464}"
	bash "$SCRIPTS/load/bpf_probe_session.sh" start "$OUT"
	mkdir -p "$SESSION_ROOT"
	ln -sfn "$(basename "$OUT")" "$CURRENT_LINK"
	printf '%s\n' "$OUT" >"$CURRENT_PATH_FILE"
	log "session started: $OUT"
	log "metrics: ${ESPX_BPF_METRICS_ADDR} (/metrics)"
	log "stop: bash scripts/dev/bpf_session.sh stop"
	;;
stop)
	OUT="$(resolve_out_dir "${2:-}")"
	PID=""
	if [[ -f "$OUT/bpf/collector.pid" ]]; then
		PID="$(cat "$OUT/bpf/collector.pid")"
	fi
	bash "$SCRIPTS/load/bpf_probe_session.sh" stop "$OUT" "$PID"
	log "session stopped: $OUT"
	;;
status)
	if [[ -f "$CURRENT_PATH_FILE" ]]; then
		OUT="$(cat "$CURRENT_PATH_FILE")"
		log "current session: $OUT"
		if [[ -f "$OUT/bpf/collector.pid" ]]; then
			log "collector pid: $(cat "$OUT/bpf/collector.pid")"
		else
			log "collector: not running"
		fi
	else
		log "no active session (run: bpf_session.sh start)"
	fi
	;;
report)
	OUT="$(resolve_out_dir "${2:-}")"
	go run ./cmd/load-report bpf "$OUT"
	log "report: $OUT/bpf-report.md"
	;;
*)
	printf 'usage: %s start|stop|status|report [session_dir]\n' "$0" >&2
	exit 2
	;;
esac

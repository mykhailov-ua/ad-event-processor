#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CMD="${1:?start|stop}"
OUT_DIR="${2:?output directory required}"
BPF_DIR="$OUT_DIR/bpf"
TARGETS_JSON="$BPF_DIR/targets.json"
PID_FILE="$BPF_DIR/collector.pid"
LOG_FILE="$BPF_DIR/collector.log"

log() { printf 'bpf-probe-session: %s\n' "$*"; }

build_collector() {
	if [[ -x "$ROOT/bin/bpf-collector" && "${BPF_FORCE_REBUILD:-0}" != "1" ]]; then
		return 0
	fi
	mkdir -p "$ROOT/bin"
	local go_bin
	go_bin="$(command -v go 2>/dev/null || true)"
	[[ -z "$go_bin" && -x /usr/local/go/bin/go ]] && go_bin=/usr/local/go/bin/go
	if [[ -z "$go_bin" ]]; then
		log "ERROR: go not found"
		exit 1
	fi
	"$go_bin" build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector
}

sudo_collector() {
	local pass="${ESPX_BPF_SUDO_PASS:-}"
	local collector_quoted
	collector_quoted="$(printf '%q ' "${COLLECTOR_CMD[@]}")"
	local launch_inner="ulimit -l unlimited 2>/dev/null || true; nohup ${collector_quoted} >>$(printf '%q' "$LOG_FILE") 2>&1 & echo \$! > $(printf '%q' "$PID_FILE")"
	if [[ -n "$pass" ]]; then
		printf '%s\n' "$pass" | sudo -S env PATH="${PATH:-/usr/local/go/bin:/usr/bin:/bin}" ESPX_REPO_ROOT="$ROOT" \
			bash -c "$launch_inner" 2>/dev/null
	elif sudo -n true 2>/dev/null; then
		sudo -n env PATH="${PATH:-/usr/local/go/bin:/usr/bin:/bin}" ESPX_REPO_ROOT="$ROOT" \
			bash -c "$launch_inner"
	else
		return 1
	fi
}

kill_collector() {
	local pid=$1
	local pass="${ESPX_BPF_SUDO_PASS:-}"
	if [[ -z "$pid" ]]; then
		return 0
	fi
	log "stopping collector pid=$pid"
	if [[ -n "$pass" ]]; then
		printf '%s\n' "$pass" | sudo -S kill -TERM "$pid" 2>/dev/null || true
	else
		kill -TERM "$pid" 2>/dev/null || true
	fi
	for _ in $(seq 1 50); do
		if [[ -n "$pass" ]]; then
			printf '%s\n' "$pass" | sudo -S kill -0 "$pid" 2>/dev/null || return 0
		else
			kill -0 "$pid" 2>/dev/null || return 0
		fi
		sleep 0.2
	done
	if [[ -n "$pass" ]]; then
		printf '%s\n' "$pass" | sudo -S kill -KILL "$pid" 2>/dev/null || true
	else
		kill -KILL "$pid" 2>/dev/null || true
	fi
}

case "$CMD" in
start)
	mkdir -p "$BPF_DIR"
	if ! bash "$SCRIPTS/load/bpf_requirements.sh"; then
		log "preflight failed — set ESPX_BPF_PROBE=0 to skip"
		exit 1
	fi
	bash "$SCRIPTS/load/bpf_build.sh"
	bash "$SCRIPTS/load/bpf_resolve_targets.sh" "$TARGETS_JSON" "${ESPX_BPF_TARGETS:-tracker,nginx,redis,processor}"

	build_collector

	SAMPLE="${ESPX_BPF_SAMPLE_RATE:-1}"
	SLOW_US="${ESPX_BPF_SLOW_US:-10000}"
	BPF_OBJ="${ESPX_BPF_OBJECT:-$ROOT/deploy/dev/bpf/loadtest_probe.o}"
	DISCOVER_LOADGEN=0
	if [[ "${ESPX_BPF_TRACK_LOADGEN:-${ESPX_BPF_TRACK_K6:-1}}" == "1" ]]; then
		DISCOVER_LOADGEN=1
	fi
	DUMP_INTERVAL="${ESPX_BPF_DUMP_INTERVAL:-0}"
	REFRESH_TARGETS="${ESPX_BPF_REFRESH_TARGETS:-0}"
	METRICS_ADDR="${ESPX_BPF_METRICS_ADDR:-}"
	LOADGEN_COMM="${ESPX_BPF_LOADGEN_COMM:-loadgen}"

	ESPX_REPO_ROOT="$ROOT" \
		ulimit -l unlimited 2>/dev/null || true

	COLLECTOR_CMD=(
		"$ROOT/bin/bpf-collector"
		-session-dir "$BPF_DIR"
		-bpf-object "$BPF_OBJ"
		-sample-rate "$SAMPLE"
		-slow-us "$SLOW_US"
		-discover-loadgen="$DISCOVER_LOADGEN"
		-loadgen-comms "$LOADGEN_COMM"
	)
	if [[ "$DUMP_INTERVAL" != "0" ]]; then
		COLLECTOR_CMD+=(-dump-interval "${DUMP_INTERVAL}s")
	fi
	if [[ "$REFRESH_TARGETS" != "0" ]]; then
		COLLECTOR_CMD+=(-refresh-targets "${REFRESH_TARGETS}s")
	fi
	if [[ -n "$METRICS_ADDR" ]]; then
		COLLECTOR_CMD+=(-metrics-addr "$METRICS_ADDR")
	fi
	if [[ -n "${ESPX_BPF_TRACKER_BINARY:-}" ]]; then
		COLLECTOR_CMD+=(-tracker-binary "$ESPX_BPF_TRACKER_BINARY")
	fi

	launch_bg() {
		nohup "$@" >"$LOG_FILE" 2>&1 &
		echo $! >"$PID_FILE"
	}

	if [[ "$(id -u)" == "0" ]]; then
		launch_bg env ESPX_REPO_ROOT="$ROOT" "${COLLECTOR_CMD[@]}"
	elif sudo_collector; then
		:
	elif command -v sudo >/dev/null 2>&1; then
		log "WARN: sudo password required — set ESPX_BPF_SUDO_PASS or run with NOPASSWD"
		launch_bg env ESPX_REPO_ROOT="$ROOT" "${COLLECTOR_CMD[@]}"
	else
		launch_bg env ESPX_REPO_ROOT="$ROOT" "${COLLECTOR_CMD[@]}"
	fi
	log "started collector pid=$(cat "$PID_FILE") log=$LOG_FILE"
	;;
stop)
	COL_PID="${3:-}"
	if [[ -z "$COL_PID" && -f "$PID_FILE" ]]; then
		COL_PID="$(cat "$PID_FILE")"
	fi
	kill_collector "$COL_PID"
	if [[ -f "$LOG_FILE" ]] && grep -q "memlock rlimit" "$LOG_FILE" 2>/dev/null; then
		log "WARN: collector failed memlock — re-run load test with: sudo ESPX_BPF_PROBE=1 ..."
	fi
	if [[ -f "$BPF_DIR/maps/summary.json" ]]; then
		log "maps dumped"
	else
		log "WARN: no maps/summary.json (collector may have failed — see $LOG_FILE)"
	fi
	go run ./cmd/load-report bpf "$OUT_DIR" || true
	;;
*)
	printf 'usage: %s start|stop <out_dir> [collector_pid]\n' "$0" >&2
	exit 1
	;;
esac

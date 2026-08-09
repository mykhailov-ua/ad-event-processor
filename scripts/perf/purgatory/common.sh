#!/usr/bin/env bash
# Shared constants and helpers for the Hardware & OS Purgatory torture suite.
# Sourced by 01–04; do not execute directly.
#
# Layout (cgroup v2 unified hierarchy assumed):
#   /sys/fs/cgroup/purgatory          — isolated slice for the SUT
#   ${STATE_DIR}/                    — sysctl/tc/pid snapshots for 04_cleanup.sh
#   ${RUN_DIR}/                      — per-run metrics (cpu.stat, wrk/k6 logs)

set -euo pipefail

PURGATORY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Prefer repo root when present; fall back to script-relative paths for standalone use.
if [[ -f "${PURGATORY_DIR}/../../lib/paths.sh" ]]; then
	# shellcheck source=/dev/null
	source "${PURGATORY_DIR}/../../lib/paths.sh"
else
	ROOT="$(cd "${PURGATORY_DIR}/../../.." && pwd)"
	SCRIPTS="${ROOT}/scripts"
fi

# --- Tunables (override via environment) ------------------------------------
CGROUP_NAME="${CGROUP_NAME:-purgatory}"
CGROUP_ROOT="${CGROUP_ROOT:-/sys/fs/cgroup}"
CGROUP_PATH="${CGROUP_PATH:-${CGROUP_ROOT}/${CGROUP_NAME}}"

# Pin SUT to a single physical/logical CPU. Cache polluter runs on another core
# that still shares the same L3 domain when possible (same package).
SUT_CPU="${SUT_CPU:-0}"
POLLUTE_CPU="${POLLUTE_CPU:-1}"

# Exactly 0.5 vCPU: 50 ms quota / 100 ms period.
CPU_MAX="${CPU_MAX:-50000 100000}"
# 64 MiB hard cap, no swap — any unexpected heap growth → OOM kill.
MEMORY_MAX_BYTES="${MEMORY_MAX_BYTES:-67108864}"
MEMORY_SWAP_MAX="${MEMORY_SWAP_MAX:-0}"

IFACE="${IFACE:-lo}"
NETEM_DELAY="${NETEM_DELAY:-5ms}"
NETEM_LOSS="${NETEM_LOSS:-1%}"
NETEM_DUPLICATE="${NETEM_DUPLICATE:-1%}"

STATE_DIR="${STATE_DIR:-/var/tmp/espx-purgatory}"
RUN_DIR="${RUN_DIR:-${ROOT}/var/purgatory}"
STRESS_PID_FILE="${STATE_DIR}/stress-ng.pid"
SUT_PID_FILE="${STATE_DIR}/sut.pid"
SYSCTL_BACKUP="${STATE_DIR}/sysctl.backup"
TC_BACKUP="${STATE_DIR}/tc.backup"
CGROUP_MARKER="${STATE_DIR}/cgroup.created"

# Load target defaults (tracker ingest).
TARGET_BIN="${TARGET_BIN:-${ROOT}/bin/tracker}"
TARGET_HOST="${TARGET_HOST:-127.0.0.1}"
TARGET_PORT="${TARGET_PORT:-8181}"
TARGET_URL="${TARGET_URL:-http://${TARGET_HOST}:${TARGET_PORT}/track}"
BENCH_DURATION="${BENCH_DURATION:-60s}"
BENCH_CONNECTIONS="${BENCH_CONNECTIONS:-10000}"
BENCH_THREADS="${BENCH_THREADS:-4}"

# Campaign UUID matching load-test seed (prepare_constrained_stack.sh).
TRACK_CAMPAIGN_ID="${TRACK_CAMPAIGN_ID:-00000000-0000-0000-0000-000000000001}"

log()  { printf 'purgatory: %s\n' "$*"; }
warn() { printf 'purgatory: WARN: %s\n' "$*" >&2; }
die()  { printf 'purgatory: ERROR: %s\n' "$*" >&2; exit 1; }

require_root() {
	[[ "$(id -u)" -eq 0 ]] || die "must run as root (sudo $0)"
}

require_cmd() {
	local c
	for c in "$@"; do
		command -v "$c" >/dev/null 2>&1 || die "required command not found: $c"
	done
}

have_cmd() {
	command -v "$1" >/dev/null 2>&1
}

ensure_state_dir() {
	mkdir -p "$STATE_DIR" "$RUN_DIR"
	chmod 700 "$STATE_DIR" 2>/dev/null || true
}

# Abort unless we are on a unified cgroup v2 mount.
assert_cgroup_v2() {
	local t
	t="$(stat -fc %T "${CGROUP_ROOT}" 2>/dev/null || true)"
	[[ "$t" == "cgroup2fs" ]] || die \
		"expected cgroup v2 at ${CGROUP_ROOT} (got '${t:-missing}'); mount -t cgroup2 cgroup2 ${CGROUP_ROOT}"
}

# Enable cpu / cpuset / memory in the root subtree so child cgroups can use them.
enable_controllers() {
	local ctl="${CGROUP_ROOT}/cgroup.subtree_control"
	[[ -f "$ctl" ]] || die "missing ${ctl}"
	# Idempotent: echo may fail if already enabled or contested by systemd — retry individually.
	local c
	for c in cpu cpuset memory; do
		if grep -qw "$c" "$ctl" 2>/dev/null; then
			continue
		fi
		if ! echo "+${c}" >"$ctl" 2>/dev/null; then
			warn "failed to enable +${c} on ${ctl} (may already be delegated)"
		fi
	done
}

# Persist sysctl key=value for later restore. Skips unknown keys.
sysctl_backup_one() {
	local key=$1
	local cur
	cur="$(sysctl -n "$key" 2>/dev/null || true)"
	[[ -n "$cur" ]] || { warn "sysctl key unavailable: $key"; return 0; }
	# Escape spaces in multi-value keys (tcp_rmem / tcp_wmem).
	printf '%s=%s\n' "$key" "$cur" >>"$SYSCTL_BACKUP"
}

sysctl_set() {
	local key=$1
	local val=$2
	sysctl -w "${key}=${val}" >/dev/null
}

# Attach an existing PID into the purgatory cgroup (cgroup v2 native path).
cgroup_attach_pid() {
	local pid=$1
	[[ -d "$CGROUP_PATH" ]] || die "cgroup missing: ${CGROUP_PATH} (run 01_setup_purgatory_cgroup.sh first)"
	[[ -d "/proc/${pid}" ]] || die "PID ${pid} is not running"
	echo "$pid" >"${CGROUP_PATH}/cgroup.procs"
}

# Prefer writing cgroup.procs; fall back to cgexec if caller wraps a command.
# Usage: cgroup_exec -- command args...
cgroup_exec() {
	if [[ "${1:-}" == "--" ]]; then
		shift
	fi
	[[ $# -ge 1 ]] || die "cgroup_exec: empty command"
	if have_cmd cgexec && [[ -x "$(command -v cgexec)" ]]; then
		# cgexec is cgroup-v1 oriented; on v2 it often fails — try then fall through.
		if cgexec -g "cpu,cpuset,memory:${CGROUP_NAME}" "$@" 2>/dev/null; then
			return 0
		fi
		warn "cgexec failed on cgroup v2; using cgroup.procs spawn"
	fi
	# Spawn in a subshell, move into cgroup immediately, then exec.
	(
		echo $$ >"${CGROUP_PATH}/cgroup.procs"
		exec "$@"
	)
}

read_cpu_stat() {
	local f="${CGROUP_PATH}/cpu.stat"
	[[ -f "$f" ]] || die "missing ${f}"
	cat "$f"
}

read_memory_events() {
	local f="${CGROUP_PATH}/memory.events"
	[[ -f "$f" ]] || { echo "oom 0"; echo "oom_kill 0"; return 0; }
	cat "$f"
}

oom_kill_count() {
	awk '/^oom_kill / { print $2; found=1 } END { if (!found) print 0 }' \
		<(read_memory_events)
}

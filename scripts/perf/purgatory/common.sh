#!/usr/bin/env bash

set -euo pipefail

PURGATORY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f "${PURGATORY_DIR}/../../lib/paths.sh" ]]; then

  source "${PURGATORY_DIR}/../../lib/paths.sh"
else
  ROOT="$(cd "${PURGATORY_DIR}/../../.." && pwd)"
  SCRIPTS="${ROOT}/scripts"
fi

CGROUP_NAME="${CGROUP_NAME:-purgatory}"
CGROUP_ROOT="${CGROUP_ROOT:-/sys/fs/cgroup}"
CGROUP_PATH="${CGROUP_PATH:-${CGROUP_ROOT}/${CGROUP_NAME}}"

SUT_CPU="${SUT_CPU:-0}"
POLLUTE_CPU="${POLLUTE_CPU:-1}"

CPU_MAX="${CPU_MAX:-50000 100000}"

MEMORY_MAX_BYTES="${MEMORY_MAX_BYTES:-67108864}"
MEMORY_SWAP_MAX="${MEMORY_SWAP_MAX:-0}"

IFACE="${IFACE:-lo}"
NETEM_DELAY="${NETEM_DELAY:-5ms}"
NETEM_LOSS="${NETEM_LOSS:-1%}"
NETEM_DUPLICATE="${NETEM_DUPLICATE:-1%}"

STATE_DIR="${STATE_DIR:-/var/tmp/ad-event-processor-purgatory}"
RUN_DIR="${RUN_DIR:-${ROOT}/var/purgatory}"
STRESS_PID_FILE="${STATE_DIR}/stress-ng.pid"
SUT_PID_FILE="${STATE_DIR}/sut.pid"
SYSCTL_BACKUP="${STATE_DIR}/sysctl.backup"
TC_BACKUP="${STATE_DIR}/tc.backup"
CGROUP_MARKER="${STATE_DIR}/cgroup.created"

TARGET_BIN="${TARGET_BIN:-${ROOT}/bin/tracker}"
TARGET_HOST="${TARGET_HOST:-127.0.0.1}"
TARGET_PORT="${TARGET_PORT:-8181}"
TARGET_URL="${TARGET_URL:-http://${TARGET_HOST}:${TARGET_PORT}/track}"
BENCH_DURATION="${BENCH_DURATION:-60s}"
BENCH_CONNECTIONS="${BENCH_CONNECTIONS:-10000}"
BENCH_THREADS="${BENCH_THREADS:-4}"

TRACK_CAMPAIGN_ID="${TRACK_CAMPAIGN_ID:-00000000-0000-0000-0000-000000000001}"

log() { printf 'purgatory: %s\n' "$*"; }
warn() { printf 'purgatory: WARN: %s\n' "$*" >&2; }
die() {
  printf 'purgatory: ERROR: %s\n' "$*" >&2
  exit 1
}

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "must run as root (sudo $0)"
}

require_cmd() {
  local c
  for c in "$@"; do
    command -v "$c" > /dev/null 2>&1 || die "required command not found: $c"
  done
}

have_cmd() {
  command -v "$1" > /dev/null 2>&1
}

ensure_state_dir() {
  mkdir -p "$STATE_DIR" "$RUN_DIR"
  chmod 700 "$STATE_DIR" 2> /dev/null || true
}

assert_cgroup_v2() {
  local t
  t="$(stat -fc %T "${CGROUP_ROOT}" 2> /dev/null || true)"
  [[ "$t" == "cgroup2fs" ]] || die \
    "expected cgroup v2 at ${CGROUP_ROOT} (got '${t:-missing}'); mount -t cgroup2 cgroup2 ${CGROUP_ROOT}"
}

enable_controllers() {
  local ctl="${CGROUP_ROOT}/cgroup.subtree_control"
  [[ -f "$ctl" ]] || die "missing ${ctl}"

  local c
  for c in cpu cpuset memory; do
    if grep -qw "$c" "$ctl" 2> /dev/null; then
      continue
    fi
    if ! echo "+${c}" > "$ctl" 2> /dev/null; then
      warn "failed to enable +${c} on ${ctl} (may already be delegated)"
    fi
  done
}

sysctl_backup_one() {
  local key=$1
  local cur
  cur="$(sysctl -n "$key" 2> /dev/null || true)"
  [[ -n "$cur" ]] || {
    warn "sysctl key unavailable: $key"
    return 0
  }

  printf '%s=%s\n' "$key" "$cur" >> "$SYSCTL_BACKUP"
}

sysctl_set() {
  local key=$1
  local val=$2
  sysctl -w "${key}=${val}" > /dev/null
}

cgroup_attach_pid() {
  local pid=$1
  [[ -d "$CGROUP_PATH" ]] || die "cgroup missing: ${CGROUP_PATH} (run setup_purgatory_cgroup.sh first)"
  [[ -d "/proc/${pid}" ]] || die "PID ${pid} is not running"
  echo "$pid" > "${CGROUP_PATH}/cgroup.procs"
}

cgroup_exec() {
  if [[ "${1:-}" == "--" ]]; then
    shift
  fi
  [[ $# -gt 0 ]] || die "cgroup_exec: command is required"
  if have_cmd cgexec && [[ -x "$(command -v cgexec)" ]]; then

    if cgexec -g "cpu,cpuset,memory:${CGROUP_NAME}" "$@" 2> /dev/null; then
      return 0
    fi
    warn "cgexec failed on cgroup v2; using cgroup.procs spawn"
  fi

  (
    echo $$ > "${CGROUP_PATH}/cgroup.procs"
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
  [[ -f "$f" ]] || {
    echo "oom 0"
    echo "oom_kill 0"
    return 0
  }
  cat "$f"
}

oom_kill_count() {
  awk '/^oom_kill / { print $2; found=1 } END { if (!found) print 0 }' \
    <(read_memory_events)
}

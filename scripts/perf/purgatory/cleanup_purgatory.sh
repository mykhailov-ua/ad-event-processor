#!/usr/bin/env bash
# Role: Restore tc qdisc, sysctl, and stop stress-ng after purgatory torture session.
# Execution context: Linux root; reads state from purgatory state dir.
# Env knobs: STRESS_PID_FILE; TC_BACKUP (from common.sh).
# Verify: sudo bash scripts/perf/purgatory/cleanup_purgatory.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd tc sysctl
ensure_state_dir

if [[ -f "$STRESS_PID_FILE" ]]; then
  spid="$(cat "$STRESS_PID_FILE" 2> /dev/null || true)"
  if [[ -n "$spid" ]] && kill -0 "$spid" 2> /dev/null; then
    log "stopping stress-ng pid=${spid}"
    kill -TERM "$spid" 2> /dev/null || true
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      kill -0 "$spid" 2> /dev/null || break
      sleep 0.2
    done
    kill -KILL "$spid" 2> /dev/null || true
  fi
  rm -f "$STRESS_PID_FILE"
fi

pkill -f 'stress-ng --cache' 2> /dev/null || true

if [[ -f "$SUT_PID_FILE" ]]; then
  spid="$(cat "$SUT_PID_FILE" 2> /dev/null || true)"
  if [[ -n "$spid" ]] && kill -0 "$spid" 2> /dev/null; then
    log "stopping SUT pid=${spid}"
    kill -TERM "$spid" 2> /dev/null || true
    sleep 0.5
    kill -KILL "$spid" 2> /dev/null || true
  fi
  rm -f "$SUT_PID_FILE"
fi

if [[ -d "$CGROUP_PATH" ]]; then
  log "moving leftover procs out of ${CGROUP_PATH}"
  while read -r pid; do
    [[ -n "$pid" ]] || continue

    echo "$pid" > "${CGROUP_ROOT}/cgroup.procs" 2> /dev/null \
      || echo "$pid" > "${CGROUP_ROOT}/init.scope/cgroup.procs" 2> /dev/null \
      || warn "could not migrate pid ${pid}"
  done < <(cat "${CGROUP_PATH}/cgroup.procs" 2> /dev/null || true)
fi

log "restoring qdisc on ${IFACE}"

tc qdisc del dev "$IFACE" root 2> /dev/null || true

if [[ -f "$TC_BACKUP" ]] && grep -q 'fq_codel' "$TC_BACKUP" 2> /dev/null; then
  tc qdisc add dev "$IFACE" root fq_codel 2> /dev/null || true
elif [[ -f "$TC_BACKUP" ]] && grep -q 'noqueue' "$TC_BACKUP" 2> /dev/null; then
  tc qdisc add dev "$IFACE" root noqueue 2> /dev/null || true
else

  tc qdisc add dev "$IFACE" root noqueue 2> /dev/null \
    || tc qdisc add dev "$IFACE" root fq_codel 2> /dev/null \
    || warn "could not reinstall default qdisc on ${IFACE}"
fi
tc qdisc show dev "$IFACE" | sed 's/^/  /' || true
rm -f "$TC_BACKUP"

if [[ "${KEEP_SYSCTL:-0}" == "1" ]]; then
  warn "KEEP_SYSCTL=1 - leaving degraded sysctl in place"
elif [[ -f "$SYSCTL_BACKUP" ]]; then
  log "restoring sysctl from ${SYSCTL_BACKUP}"
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" ]] && continue
    [[ "$line" == \#* ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    if ! sysctl -w "${key}=${val}" > /dev/null 2>&1; then
      warn "failed to restore ${key}=${val}"
    else
      log "  restored ${key}=${val}"
    fi
  done < "$SYSCTL_BACKUP"
  rm -f "$SYSCTL_BACKUP"
else
  warn "no sysctl backup at ${SYSCTL_BACKUP}; skipping restore"
fi

if [[ -d "$CGROUP_PATH" ]]; then
  log "removing cgroup ${CGROUP_PATH}"

  echo "max" > "${CGROUP_PATH}/memory.max" 2> /dev/null || true
  echo "max" > "${CGROUP_PATH}/cpu.max" 2> /dev/null || true
  if rmdir "$CGROUP_PATH" 2> /dev/null; then
    log "cgroup removed"
  else
    warn "rmdir ${CGROUP_PATH} failed - procs still attached?"
    cat "${CGROUP_PATH}/cgroup.procs" 2> /dev/null | sed 's/^/  pid /' || true
  fi
fi
rm -f "$CGROUP_MARKER" "${STATE_DIR}/cgroup.snapshot"

log "cleanup complete"
log "run artifacts (if any) retained under ${RUN_DIR}/"

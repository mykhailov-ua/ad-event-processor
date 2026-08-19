#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd stat awk mkdir tee
assert_cgroup_v2
ensure_state_dir

log "enabling cpu/cpuset/memory controllers on ${CGROUP_ROOT}"
enable_controllers

if [[ -d "$CGROUP_PATH" ]]; then

  procs="$(cat "${CGROUP_PATH}/cgroup.procs" 2> /dev/null || true)"
  if [[ -n "$procs" ]]; then
    die "cgroup ${CGROUP_PATH} still has PIDs: ${procs}. Run 04_cleanup.sh or kill them first."
  fi
  log "reusing existing cgroup ${CGROUP_PATH}"
else
  log "creating ${CGROUP_PATH}"
  mkdir -p "$CGROUP_PATH"
  touch "$CGROUP_MARKER"
fi

if [[ -f "${CGROUP_PATH}/cgroup.subtree_control" ]]; then
  echo "+cpu +cpuset +memory" > "${CGROUP_PATH}/cgroup.subtree_control" 2> /dev/null || true
fi

log "cpuset: pinning to CPU ${SUT_CPU}"

if [[ -f "${CGROUP_PATH}/cpuset.mems" ]]; then
  if [[ -z "$(cat "${CGROUP_PATH}/cpuset.mems" 2> /dev/null || true)" ]]; then
    echo "0" > "${CGROUP_PATH}/cpuset.mems"
  fi
fi
echo "${SUT_CPU}" > "${CGROUP_PATH}/cpuset.cpus"

log "cpu.max = ${CPU_MAX}  (0.5 vCPU)"
echo "${CPU_MAX}" > "${CGROUP_PATH}/cpu.max"

log "memory.max = ${MEMORY_MAX_BYTES} bytes (64 MiB), memory.swap.max = ${MEMORY_SWAP_MAX}"
echo "${MEMORY_MAX_BYTES}" > "${CGROUP_PATH}/memory.max"
echo "${MEMORY_SWAP_MAX}" > "${CGROUP_PATH}/memory.swap.max"

if [[ -f "${CGROUP_PATH}/memory.high" ]]; then
  echo "max" > "${CGROUP_PATH}/memory.high"
fi

{
  echo "created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "path=${CGROUP_PATH}"
  echo "cpuset.cpus=$(cat "${CGROUP_PATH}/cpuset.cpus")"
  echo "cpu.max=$(cat "${CGROUP_PATH}/cpu.max")"
  echo "memory.max=$(cat "${CGROUP_PATH}/memory.max")"
  echo "memory.swap.max=$(cat "${CGROUP_PATH}/memory.swap.max")"
} | tee "${STATE_DIR}/cgroup.snapshot"

log "verify:"
log "  cpuset.cpus     = $(cat "${CGROUP_PATH}/cpuset.cpus")"
log "  cpu.max         = $(cat "${CGROUP_PATH}/cpu.max")"
log "  memory.max      = $(cat "${CGROUP_PATH}/memory.max")"
log "  memory.swap.max = $(cat "${CGROUP_PATH}/memory.swap.max")"
log "cgroup ready. Next: 02_degrade_network_and_cache.sh"

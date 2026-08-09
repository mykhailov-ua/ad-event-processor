#!/usr/bin/env bash
# 01_setup_purgatory_cgroup.sh
#
# Create an isolated cgroup v2 slice that starves the SUT:
#   - cpuset.cpus  = single core (default 0)
#   - cpu.max      = 50000 100000  → exactly 0.5 vCPU
#   - memory.max   = 64 MiB
#   - memory.swap.max = 0          → OOM-kill on any unplanned heap growth
#
# Usage (root):
#   sudo bash scripts/perf/purgatory/01_setup_purgatory_cgroup.sh
#   SUT_CPU=2 MEMORY_MAX_BYTES=67108864 sudo -E bash .../01_setup_purgatory_cgroup.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd stat awk mkdir tee
assert_cgroup_v2
ensure_state_dir

log "enabling cpu/cpuset/memory controllers on ${CGROUP_ROOT}"
enable_controllers

if [[ -d "$CGROUP_PATH" ]]; then
	# Empty procs before reconfigure; refuse if something foreign is still attached.
	procs="$(cat "${CGROUP_PATH}/cgroup.procs" 2>/dev/null || true)"
	if [[ -n "$procs" ]]; then
		die "cgroup ${CGROUP_PATH} still has PIDs: ${procs}. Run 04_cleanup.sh or kill them first."
	fi
	log "reusing existing cgroup ${CGROUP_PATH}"
else
	log "creating ${CGROUP_PATH}"
	mkdir -p "$CGROUP_PATH"
	touch "$CGROUP_MARKER"
fi

# Child must also advertise controllers if we ever nest further.
if [[ -f "${CGROUP_PATH}/cgroup.subtree_control" ]]; then
	echo "+cpu +cpuset +memory" >"${CGROUP_PATH}/cgroup.subtree_control" 2>/dev/null || true
fi

log "cpuset: pinning to CPU ${SUT_CPU}"
# mems must be set before some kernels accept cpus; use node 0 if present.
if [[ -f "${CGROUP_PATH}/cpuset.mems" ]]; then
	if [[ -z "$(cat "${CGROUP_PATH}/cpuset.mems" 2>/dev/null || true)" ]]; then
		echo "0" >"${CGROUP_PATH}/cpuset.mems"
	fi
fi
echo "${SUT_CPU}" >"${CGROUP_PATH}/cpuset.cpus"

log "cpu.max = ${CPU_MAX}  (0.5 vCPU)"
echo "${CPU_MAX}" >"${CGROUP_PATH}/cpu.max"

log "memory.max = ${MEMORY_MAX_BYTES} bytes (64 MiB), memory.swap.max = ${MEMORY_SWAP_MAX}"
echo "${MEMORY_MAX_BYTES}" >"${CGROUP_PATH}/memory.max"
echo "${MEMORY_SWAP_MAX}" >"${CGROUP_PATH}/memory.swap.max"

# Optional: disable memory.high soft throttle so pressure hits max → OOM faster.
if [[ -f "${CGROUP_PATH}/memory.high" ]]; then
	echo "max" >"${CGROUP_PATH}/memory.high"
fi

# Snapshot for operators / cleanup verification.
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

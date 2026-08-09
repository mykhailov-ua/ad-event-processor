#!/usr/bin/env bash
# 02_degrade_network_and_cache.sh
#
# Degrade the host network stack and pollute the shared L3 cache:
#   1) tc netem on loopback — 5 ms delay, 1% loss, 1% duplicate
#   2) shrink socket buffers + listen backlog (SYN / accept bottleneck)
#   3) background stress-ng L3 cache thrash on POLLUTE_CPU (default 1)
#
# Original sysctl and qdisc are snapshotted under STATE_DIR for 04_cleanup.sh.
#
# Usage (root):
#   sudo bash scripts/perf/purgatory/02_degrade_network_and_cache.sh
#   IFACE=lo POLLUTE_CPU=1 sudo -E bash .../02_degrade_network_and_cache.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd tc sysctl
ensure_state_dir

# stress-ng is mandatory for the cache-pollution phase.
if ! have_cmd stress-ng; then
	die "stress-ng not found. Install: apt-get install -y stress-ng  |  dnf install -y stress-ng"
fi

# ---------------------------------------------------------------------------
# 1) tc netem on IFACE (default: lo)
# ---------------------------------------------------------------------------
log "snapshotting current qdisc on ${IFACE}"
: >"$TC_BACKUP"
tc qdisc show dev "$IFACE" >"$TC_BACKUP" || true

# Replace root qdisc. If netem already present, replace in place.
if tc qdisc show dev "$IFACE" | grep -q 'qdisc netem'; then
	log "replacing existing netem on ${IFACE}"
	tc qdisc change dev "$IFACE" root netem \
		delay "$NETEM_DELAY" loss "$NETEM_LOSS" duplicate "$NETEM_DUPLICATE"
else
	# If a non-default qdisc exists, delete it first (loopback usually has fq_codel/noqueue).
	if tc qdisc show dev "$IFACE" | grep -Eq 'qdisc (fq_codel|cake|htb|prio|pfifo_fast|noqueue)'; then
		log "deleting current root qdisc on ${IFACE}"
		tc qdisc del dev "$IFACE" root 2>/dev/null || true
	fi
	log "adding netem: delay=${NETEM_DELAY} loss=${NETEM_LOSS} duplicate=${NETEM_DUPLICATE}"
	tc qdisc add dev "$IFACE" root netem \
		delay "$NETEM_DELAY" loss "$NETEM_LOSS" duplicate "$NETEM_DUPLICATE"
fi

tc qdisc show dev "$IFACE" | sed 's/^/  /'

# ---------------------------------------------------------------------------
# 2) Kernel socket buffer + backlog starvation
# ---------------------------------------------------------------------------
log "backing up and applying degraded sysctl knobs → ${SYSCTL_BACKUP}"
: >"$SYSCTL_BACKUP"

# Keys we will mutate (order preserved for readability).
SYSCTL_KEYS=(
	net.core.rmem_max
	net.core.wmem_max
	net.ipv4.tcp_rmem
	net.ipv4.tcp_wmem
	net.ipv4.tcp_max_syn_backlog
	net.core.somaxconn
)

for k in "${SYSCTL_KEYS[@]}"; do
	sysctl_backup_one "$k"
done

# Cap absolute socket buffer ceilings at 64 KiB.
sysctl_set net.core.rmem_max 65536
sysctl_set net.core.wmem_max 65536

# tcp_{r,w}mem = "min default max" — clamp max to rmem_max/wmem_max, keep small default.
# These are intentionally hostile: tiny receive/send windows under 10k keep-alives.
sysctl_set net.ipv4.tcp_rmem "4096 87380 65536"
sysctl_set net.ipv4.tcp_wmem "4096 16384 65536"

# SYN backlog + listen backlog: simulate flood / accept-queue choke.
sysctl_set net.ipv4.tcp_max_syn_backlog 128
sysctl_set net.core.somaxconn 128

log "active degraded sysctl:"
for k in "${SYSCTL_KEYS[@]}"; do
	log "  ${k} = $(sysctl -n "$k" 2>/dev/null || echo '?')"
done

# ---------------------------------------------------------------------------
# 3) L3 cache pollution (stress-ng)
# ---------------------------------------------------------------------------
if [[ -f "$STRESS_PID_FILE" ]]; then
	old="$(cat "$STRESS_PID_FILE" 2>/dev/null || true)"
	if [[ -n "$old" ]] && kill -0 "$old" 2>/dev/null; then
		die "stress-ng already running (pid ${old}); run 04_cleanup.sh first"
	fi
	rm -f "$STRESS_PID_FILE"
fi

# Run on a sibling core sharing L3 with SUT_CPU so tracker I-cache/D-cache lines
# are continuously evicted into DRAM. --cache-level 3 targets LLC when supported.
STRESS_ARGS=(
	--cache 1
	--cache-level 3
	--aggressive
	--timeout 0             # run until killed by cleanup
	--metrics-brief
)

log "starting stress-ng cache polluter on CPU ${POLLUTE_CPU}"
# taskset pins the polluter; fallback to bare stress-ng if taskset missing.
if have_cmd taskset; then
	taskset -c "$POLLUTE_CPU" stress-ng "${STRESS_ARGS[@]}" \
		>"${STATE_DIR}/stress-ng.log" 2>&1 &
else
	warn "taskset not found; stress-ng will float across CPUs"
	stress-ng "${STRESS_ARGS[@]}" >"${STATE_DIR}/stress-ng.log" 2>&1 &
fi
STRESS_PID=$!
echo "$STRESS_PID" >"$STRESS_PID_FILE"
sleep 0.5
if ! kill -0 "$STRESS_PID" 2>/dev/null; then
	# Older stress-ng may reject --cache-ways / --cache-level. Retry minimal set.
	warn "stress-ng exited; retrying with minimal --cache flags"
	MIN_ARGS=(--cache 1 --timeout 0)
	if have_cmd taskset; then
		taskset -c "$POLLUTE_CPU" stress-ng "${MIN_ARGS[@]}" \
			>"${STATE_DIR}/stress-ng.log" 2>&1 &
	else
		stress-ng "${MIN_ARGS[@]}" >"${STATE_DIR}/stress-ng.log" 2>&1 &
	fi
	STRESS_PID=$!
	echo "$STRESS_PID" >"$STRESS_PID_FILE"
	sleep 0.5
	kill -0 "$STRESS_PID" 2>/dev/null || die "stress-ng failed to start; see ${STATE_DIR}/stress-ng.log"
fi

log "stress-ng pid=${STRESS_PID} log=${STATE_DIR}/stress-ng.log"
log "network + cache degradation armed. Next: 03_run_torture_benchmark.sh"

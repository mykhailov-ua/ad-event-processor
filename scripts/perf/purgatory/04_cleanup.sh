#!/usr/bin/env bash
# 04_cleanup.sh
#
# Tear down purgatory torture state:
#   - stop stress-ng cache polluter
#   - restore tc qdisc on IFACE
#   - restore sysctl knobs from STATE_DIR backup
#   - kill SUT started by 03 (optional)
#   - remove /sys/fs/cgroup/purgatory
#
# Usage (root):
#   sudo bash scripts/perf/purgatory/04_cleanup.sh
#   KEEP_SYSCTL=1 sudo -E bash .../04_cleanup.sh   # leave degraded sysctl in place

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd tc sysctl
ensure_state_dir

# ---------------------------------------------------------------------------
# 1) Stop stress-ng
# ---------------------------------------------------------------------------
if [[ -f "$STRESS_PID_FILE" ]]; then
	spid="$(cat "$STRESS_PID_FILE" 2>/dev/null || true)"
	if [[ -n "$spid" ]] && kill -0 "$spid" 2>/dev/null; then
		log "stopping stress-ng pid=${spid}"
		kill -TERM "$spid" 2>/dev/null || true
		for _ in 1 2 3 4 5 6 7 8 9 10; do
			kill -0 "$spid" 2>/dev/null || break
			sleep 0.2
		done
		kill -KILL "$spid" 2>/dev/null || true
	fi
	rm -f "$STRESS_PID_FILE"
fi
# Belt-and-suspenders: any leftover stress-ng from this suite.
pkill -f 'stress-ng --cache' 2>/dev/null || true

# ---------------------------------------------------------------------------
# 2) Stop SUT we started (best-effort)
# ---------------------------------------------------------------------------
if [[ -f "$SUT_PID_FILE" ]]; then
	spid="$(cat "$SUT_PID_FILE" 2>/dev/null || true)"
	if [[ -n "$spid" ]] && kill -0 "$spid" 2>/dev/null; then
		log "stopping SUT pid=${spid}"
		kill -TERM "$spid" 2>/dev/null || true
		sleep 0.5
		kill -KILL "$spid" 2>/dev/null || true
	fi
	rm -f "$SUT_PID_FILE"
fi

# Evict any remaining tasks from the cgroup so rmdir succeeds.
if [[ -d "$CGROUP_PATH" ]]; then
	log "moving leftover procs out of ${CGROUP_PATH}"
	while read -r pid; do
		[[ -n "$pid" ]] || continue
		# Move to root cgroup (or parent).
		echo "$pid" >"${CGROUP_ROOT}/cgroup.procs" 2>/dev/null \
			|| echo "$pid" >"${CGROUP_ROOT}/init.scope/cgroup.procs" 2>/dev/null \
			|| warn "could not migrate pid ${pid}"
	done < <(cat "${CGROUP_PATH}/cgroup.procs" 2>/dev/null || true)
fi

# ---------------------------------------------------------------------------
# 3) Restore tc on IFACE
# ---------------------------------------------------------------------------
log "restoring qdisc on ${IFACE}"
# Drop netem / any custom root we installed.
tc qdisc del dev "$IFACE" root 2>/dev/null || true
# Re-apply a sane default. Modern kernels use fq_codel or noqueue on lo.
if [[ -f "$TC_BACKUP" ]] && grep -q 'fq_codel' "$TC_BACKUP" 2>/dev/null; then
	tc qdisc add dev "$IFACE" root fq_codel 2>/dev/null || true
elif [[ -f "$TC_BACKUP" ]] && grep -q 'noqueue' "$TC_BACKUP" 2>/dev/null; then
	tc qdisc add dev "$IFACE" root noqueue 2>/dev/null || true
else
	# Best-effort default for loopback.
	tc qdisc add dev "$IFACE" root noqueue 2>/dev/null \
		|| tc qdisc add dev "$IFACE" root fq_codel 2>/dev/null \
		|| warn "could not reinstall default qdisc on ${IFACE}"
fi
tc qdisc show dev "$IFACE" | sed 's/^/  /' || true
rm -f "$TC_BACKUP"

# ---------------------------------------------------------------------------
# 4) Restore sysctl
# ---------------------------------------------------------------------------
if [[ "${KEEP_SYSCTL:-0}" == "1" ]]; then
	warn "KEEP_SYSCTL=1 — leaving degraded sysctl in place"
elif [[ -f "$SYSCTL_BACKUP" ]]; then
	log "restoring sysctl from ${SYSCTL_BACKUP}"
	while IFS= read -r line || [[ -n "$line" ]]; do
		[[ -z "$line" ]] && continue
		[[ "$line" == \#* ]] && continue
		key="${line%%=*}"
		val="${line#*=}"
		if ! sysctl -w "${key}=${val}" >/dev/null 2>&1; then
			warn "failed to restore ${key}=${val}"
		else
			log "  restored ${key}=${val}"
		fi
	done <"$SYSCTL_BACKUP"
	rm -f "$SYSCTL_BACKUP"
else
	warn "no sysctl backup at ${SYSCTL_BACKUP}; skipping restore"
fi

# ---------------------------------------------------------------------------
# 5) Remove cgroup
# ---------------------------------------------------------------------------
if [[ -d "$CGROUP_PATH" ]]; then
	log "removing cgroup ${CGROUP_PATH}"
	# Reset limits so kernel allows rmdir cleanly on some versions.
	echo "max" >"${CGROUP_PATH}/memory.max" 2>/dev/null || true
	echo "max" >"${CGROUP_PATH}/cpu.max" 2>/dev/null || true
	if rmdir "$CGROUP_PATH" 2>/dev/null; then
		log "cgroup removed"
	else
		warn "rmdir ${CGROUP_PATH} failed — procs still attached?"
		cat "${CGROUP_PATH}/cgroup.procs" 2>/dev/null | sed 's/^/  pid /' || true
	fi
fi
rm -f "$CGROUP_MARKER" "${STATE_DIR}/cgroup.snapshot"

log "cleanup complete"
log "run artifacts (if any) retained under ${RUN_DIR}/"

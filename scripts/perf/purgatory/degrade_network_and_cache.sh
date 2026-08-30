#!/usr/bin/env bash
# Role: Apply tc netem and cache pollution via stress-ng for degraded-network torture runs.
# Execution context: Linux root on ingress interface; backs up qdisc to state dir.
# Env knobs: IFACE (from common.sh); netem delay/loss from degrade script env.
# Verify: sudo bash scripts/perf/purgatory/degrade_network_and_cache.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/common.sh"

require_root
require_cmd tc sysctl
ensure_state_dir

if ! have_cmd stress-ng; then
  die "stress-ng not found. Install: apt-get install -y stress-ng  |  dnf install -y stress-ng"
fi

log "snapshotting current qdisc on ${IFACE}"
: > "$TC_BACKUP"
tc qdisc show dev "$IFACE" > "$TC_BACKUP" || true

if tc qdisc show dev "$IFACE" | grep -q 'qdisc netem'; then
  log "replacing existing netem on ${IFACE}"
  tc qdisc change dev "$IFACE" root netem \
    delay "$NETEM_DELAY" loss "$NETEM_LOSS" duplicate "$NETEM_DUPLICATE"
else

  if tc qdisc show dev "$IFACE" | grep -Eq 'qdisc (fq_codel|cake|htb|prio|pfifo_fast|noqueue)'; then
    log "deleting current root qdisc on ${IFACE}"
    tc qdisc del dev "$IFACE" root 2> /dev/null || true
  fi
  log "adding netem: delay=${NETEM_DELAY} loss=${NETEM_LOSS} duplicate=${NETEM_DUPLICATE}"
  tc qdisc add dev "$IFACE" root netem \
    delay "$NETEM_DELAY" loss "$NETEM_LOSS" duplicate "$NETEM_DUPLICATE"
fi

tc qdisc show dev "$IFACE" | sed 's/^/  /'

log "backing up and applying degraded sysctl knobs -> ${SYSCTL_BACKUP}"
: > "$SYSCTL_BACKUP"

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

sysctl_set net.core.rmem_max 65536
sysctl_set net.core.wmem_max 65536

sysctl_set net.ipv4.tcp_rmem "4096 87380 65536"
sysctl_set net.ipv4.tcp_wmem "4096 16384 65536"

sysctl_set net.ipv4.tcp_max_syn_backlog 128
sysctl_set net.core.somaxconn 128

log "active degraded sysctl:"
for k in "${SYSCTL_KEYS[@]}"; do
  log "  ${k} = $(sysctl -n "$k" 2> /dev/null || echo '?')"
done

if [[ -f "$STRESS_PID_FILE" ]]; then
  old="$(cat "$STRESS_PID_FILE" 2> /dev/null || true)"
  if [[ -n "$old" ]] && kill -0 "$old" 2> /dev/null; then
    die "stress-ng already running (pid ${old}); run cleanup_purgatory.sh first"
  fi
  rm -f "$STRESS_PID_FILE"
fi

STRESS_ARGS=(
  --cache 1
  --cache-level 3
  --aggressive
  --timeout 0
  --metrics-brief
)

log "starting stress-ng cache polluter on CPU ${POLLUTE_CPU}"

if have_cmd taskset; then
  taskset -c "$POLLUTE_CPU" stress-ng "${STRESS_ARGS[@]}" \
    > "${STATE_DIR}/stress-ng.log" 2>&1 &
else
  warn "taskset not found; stress-ng will float across CPUs"
  stress-ng "${STRESS_ARGS[@]}" > "${STATE_DIR}/stress-ng.log" 2>&1 &
fi
STRESS_PID=$!
echo "$STRESS_PID" > "$STRESS_PID_FILE"
sleep 0.5
if ! kill -0 "$STRESS_PID" 2> /dev/null; then

  warn "stress-ng exited; retrying with minimal --cache flags"
  MIN_ARGS=(--cache 1 --timeout 0)
  if have_cmd taskset; then
    taskset -c "$POLLUTE_CPU" stress-ng "${MIN_ARGS[@]}" \
      > "${STATE_DIR}/stress-ng.log" 2>&1 &
  else
    stress-ng "${MIN_ARGS[@]}" > "${STATE_DIR}/stress-ng.log" 2>&1 &
  fi
  STRESS_PID=$!
  echo "$STRESS_PID" > "$STRESS_PID_FILE"
  sleep 0.5
  kill -0 "$STRESS_PID" 2> /dev/null || die "stress-ng failed to start; see ${STATE_DIR}/stress-ng.log"
fi

log "stress-ng pid=${STRESS_PID} log=${STATE_DIR}/stress-ng.log"
log "network + cache degradation armed. Next: run_torture_benchmark.sh"

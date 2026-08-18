#!/usr/bin/env bash
# CPU isolation status / verify for compose cpuset profile and native systemd drop-ins.
#
# Usage:
#   bash scripts/ops/cpu_isolation.sh status
#   bash scripts/ops/cpu_isolation.sh verify
#
# Env: reads CPU_ISOLATION_ENABLED and *_CPUSET from .env when present.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

CPU_ISOLATION_ENABLED="${CPU_ISOLATION_ENABLED:-1}"
COMPOSE_FILE="${CPU_ISOLATION_COMPOSE_FILE:-deploy/compose/docker-compose.yaml}"

log() { printf 'cpu-isolation: %s\n' "$*"; }
warn() { printf 'cpu-isolation: WARN: %s\n' "$*" >&2; }
die() {
  printf 'cpu-isolation: ERROR: %s\n' "$*" >&2
  exit 1
}

inspect_container() {
  local name="$1"
  local cid
  cid="$(docker ps -q -f "name=${name}$" | head -1)"
  if [[ -z "$cid" ]]; then
    warn "container $name not running"
    return 1
  fi
  docker inspect "$cid" --format \
    'name={{.Name}} cpuset={{.HostConfig.CpusetCpus}} cpu_quota={{.HostConfig.CpuQuota}} cpu_shares={{.HostConfig.CpuShares}}'
}

verify_no_cpu_quota() {
  local line quota
  line="$(inspect_container "$1" || return 1)"
  quota="$(echo "$line" | sed -n 's/.*cpu_quota=\([^ ]*\).*/\1/p')"
  if [[ -n "$quota" && "$quota" != "0" ]]; then
    die "$1 has cpu_quota=$quota (want 0 — no CFS hard cap)"
  fi
  log "ok  $line"
}

cmd_status() {
  log "CPU_ISOLATION_ENABLED=$CPU_ISOLATION_ENABLED"
  log "TRACKER cpuset: ${TRACKER_0_CPUSET:-0} ${TRACKER_1_CPUSET:-1} ${TRACKER_2_CPUSET:-2} ${TRACKER_3_CPUSET:-3}"
  log "REDIS_CPUSET=${REDIS_CPUSET:-4,5} EDGE_CPUSET=${EDGE_CPUSET:-6} COLD_CPUSET=${COLD_CPUSET:-7}"
  if ! command -v docker > /dev/null 2>&1; then
    warn "docker not available"
    return 0
  fi
  for ctr in bidshard-tracker-0-1 espx-tracker-0-1 tracker-0; do
    if docker ps --format '{{.Names}}' | grep -qx "$ctr"; then
      inspect_container "$ctr" || true
    fi
  done
}

cmd_verify() {
  if [[ "$CPU_ISOLATION_ENABLED" != "1" ]]; then
    log "skip (CPU_ISOLATION_ENABLED!=1)"
    exit 0
  fi
  if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
    die "docker required for verify"
  fi
  # Resolve tracker container name from compose project.
  COMPOSE=(docker compose -f "$COMPOSE_FILE" -f deploy/compose/docker-compose.cpu-isolation.yaml --profile cpu-isolation)
  TRACKER_CID="$("${COMPOSE[@]}" ps -q tracker-0 2> /dev/null | head -1 || true)"
  if [[ -z "$TRACKER_CID" ]]; then
    die "tracker-0 not running — start stack with --profile cpu-isolation"
  fi
  name="$(docker inspect "$TRACKER_CID" --format '{{.Name}}' | sed 's/^\///')"
  cpuset="$(docker inspect "$TRACKER_CID" --format '{{.HostConfig.CpusetCpus}}')"
  quota="$(docker inspect "$TRACKER_CID" --format '{{.HostConfig.CpuQuota}}')"
  want="${TRACKER_0_CPUSET:-0}"
  if [[ -z "$cpuset" ]]; then
    die "$name cpuset empty (profile cpu-isolation not active?)"
  fi
  if [[ "$cpuset" != "$want" ]]; then
    die "$name cpuset=$cpuset want $want"
  fi
  if [[ -n "$quota" && "$quota" != "0" ]]; then
    die "$name cpu_quota=$quota (remove deploy.resources.limits.cpus)"
  fi
  log "ok  $name cpuset=$cpuset cpu_quota=$quota"
  printf 'fault_proof fault=cpu_isolation status=ok container=%s cpuset=%s harness=compose_cpuset cpu_quota=0 baseline_ok=true\n' "$name" "$cpuset"
}

MODE="${1:-status}"
case "$MODE" in
  status) cmd_status ;;
  verify) cmd_verify ;;
  *) die "usage: $0 {status|verify}" ;;
esac

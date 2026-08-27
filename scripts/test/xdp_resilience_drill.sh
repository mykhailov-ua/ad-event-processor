#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${XDP_RESILIENCE_LOG:-$CI_ARTIFACT_DIR/xdp-resilience.log}"
IFACE="${XDP_RESILIENCE_IFACE:-lo}"

log() { printf 'xdp_resilience_drill: %s\n' "$*"; }
die() {
  printf 'xdp_resilience_drill: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  log "skip (BTF vmlinux missing)"
  exit 0
fi

if [[ "$(id -u)" -ne 0 ]]; then
  log "re-exec with sudo for XDP attach on ${IFACE}"
  exec sudo -E bash "$0" "$@"
fi

[[ -f .env ]] && set -a && source .env && set +a

REDIS_PASS="${REDIS_PASSWORD:-your_redis_password_here}"
export REDIS_ADDRS="${REDIS_ADDRS:-127.0.0.1:6379}"
export REDIS_PASS

if ! command -v clang > /dev/null 2>&1; then
  log "installing clang for bpf2go"
  if command -v apt-get > /dev/null 2>&1; then
    apt-get update -qq
    apt-get install -y -qq clang llvm libbpf-dev linux-libc-dev
  else
    die "clang required (make bpf-dev)"
  fi
fi

if ! go test ./internal/edge/ -run='^TestXDP_dropBlocklistedSource$' -count=1 > /dev/null 2>&1; then
  log "generating edge BPF objects"
  go generate ./internal/edge/
fi

if ! go test ./internal/edge/ -run='^TestXDP_dropBlocklistedSource$' -count=1 > /dev/null 2>&1; then
  log "skip (BPF objects unavailable)"
  exit 0
fi

if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  log "start redis-0 for blocklist sync"
  docker compose up -d redis-0
  for _ in $(seq 1 30); do
    if docker compose exec -T redis-0 redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning PING 2> /dev/null | grep -q PONG; then
      break
    fi
    sleep 1
  done
else
  log "docker unavailable; expecting REDIS_ADDRS=${REDIS_ADDRS}"
fi

export XDP_RESILIENCE_DRILL=1
log "attach on ${IFACE}, sync blocklist, assert drop counter (prog.Test on same maps)"
go test ./internal/edge/ -run='^TestResilienceDrill_LoopbackBlocklistDrop$' -count=1 -v 2>&1 | tee "$LOG"

grep -q 'fault_proof fault=xdp_resilience_drill ' "$LOG" || die "missing fault_proof line in $LOG"

log "soft bench gate (generic mode)"
bash scripts/test/edge_xdp_bench_gate.sh

log "ok"

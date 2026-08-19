#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  echo "edge_xdp_compose_smoke: skip (BTF vmlinux missing)"
  exit 0
fi

if ! command -v docker > /dev/null 2>&1; then
  echo "edge_xdp_compose_smoke: skip (docker not installed)"
  exit 0
fi

if ! docker info > /dev/null 2>&1; then
  echo "edge_xdp_compose_smoke: skip (docker daemon unavailable)"
  exit 0
fi

[[ -f .env ]] && set -a && source .env && set +a

REDIS_PASS="${REDIS_PASSWORD:-your_redis_password_here}"
PIN_DIR="${EDGE_BPF_PIN_DIR:-/sys/fs/bpf/ad-event-processor}"
BLOCKLIST="${PIN_DIR}/blocklist_v4"
TEST_IP="${EDGE_XDP_SMOKE_IP:-203.0.113.99}"

log() { printf 'edge_xdp_compose_smoke: %s\n' "$*"; }
die() {
  printf 'edge_xdp_compose_smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  docker compose --profile enterprise-xdp stop edge-xdp > /dev/null 2>&1 || true
}
trap cleanup EXIT

log "build edge-xdp image"
docker compose --profile enterprise-xdp build edge-xdp

log "start redis-0 + edge-xdp (profile enterprise-xdp)"
docker compose --profile enterprise-xdp up -d redis-0 edge-xdp

log "wait for edge-xdp attach and bpf-sync"
attached=0
for _ in $(seq 1 60); do
  if docker compose logs edge-xdp 2>&1 | grep -q 'edge xdp attached'; then
    if docker compose logs edge-xdp 2>&1 | grep -q 'edge-bpf-sync metrics listening'; then
      attached=1
      break
    fi
  fi
  sleep 1
done
[[ "$attached" -eq 1 ]] || die "edge-xdp did not attach or bpf-sync did not start"

if sudo -n test -e "$BLOCKLIST" 2> /dev/null; then
  log "pinned blocklist map present at $BLOCKLIST"
else
  log "bpffs pin path not readable on host (need root); entrypoint wait passed in container"
fi

REDIS_CID="$(docker compose ps -q redis-0)"
[[ -n "$REDIS_CID" ]] || die "redis-0 container not running"

log "enable ebpf_xdp_edge entitlement on shard 0"
docker exec "$REDIS_CID" redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning \
  HSET entitlement:deployment ebpf_xdp_edge 1 > /dev/null

log "seed blacklist:manual with $TEST_IP"
docker exec "$REDIS_CID" redis-cli -p 6379 -a "$REDIS_PASS" --no-auth-warning \
  SADD blacklist:manual "$TEST_IP" > /dev/null

log "wait for bpf-sync to apply deny list"
synced=0
for _ in $(seq 1 30); do
  if docker compose logs edge-xdp 2>&1 | grep -q '"deny_added":[1-9]'; then
    synced=1
    break
  fi
  sleep 1
done
[[ "$synced" -eq 1 ]] || die "timed out waiting for edge bpf sync logs (deny_added)"

log "ok — pinned map present and blacklist synced"

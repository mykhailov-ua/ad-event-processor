#!/usr/bin/env bash
# Role: Enterprise XDP perimeter preflight (BTF, pin dir, optional edge-xdp service).
# Execution context: Host before enabling EDGE_XDP_ENABLED=1; reads .env for iface and pin path.
# Env knobs: EDGE_XDP_ENABLED (0 skip); EDGE_XDP_INGRESS_INTERFACE; EDGE_BPF_PIN_DIR; STRICT (1).
# Verify: bash scripts/ops/xdp_preflight.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ENV_FILE="${1:-$ROOT/.env}"
STRICT="${STRICT:-0}"

log() { printf 'xdp-preflight: %s\n' "$*"; }
warn() { printf 'xdp-preflight: WARN: %s\n' "$*" >&2; }
die() {
  printf 'xdp-preflight: ERROR: %s\n' "$*" >&2
  exit 1
}

fail=0
run_check() {
  local name=$1
  shift
  log "check: $name"
  if "$@"; then
    log "  OK: $name"
  else
    warn "  FAIL: $name"
    fail=1
  fi
}

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

EDGE_XDP_ENABLED="${EDGE_XDP_ENABLED:-0}"
if [[ "$EDGE_XDP_ENABLED" != "1" ]]; then
  log "EDGE_XDP_ENABLED!=1 - XDP checks skipped (optional P2)"
  exit 0
fi

IFACE="${EDGE_XDP_INGRESS_INTERFACE:-}"
PIN_DIR="${EDGE_BPF_PIN_DIR:-/sys/fs/bpf/ad-event-processor}"

run_check "BTF vmlinux" test -r /sys/kernel/btf/vmlinux
run_check "ingress iface not loopback" bash -c '[[ -n "$1" && "$1" != "lo" ]]' _ "$IFACE"
run_check "pin dir exists" test -d "$PIN_DIR"

if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  run_check "edge-xdp container" bash -c '
    docker ps --format "{{.Names}}" | grep -qE "edge-xdp"
  ' || warn "start: docker compose --profile enterprise-xdp up -d edge-xdp edge-bpf-sync"
else
  warn "docker unavailable - skip edge-xdp container check"
fi

run_check "edge-bpf-sync metrics" bash -c '
  port="${EDGE_BPF_SYNC_METRICS_PORT:-9191}"
  curl -sf --max-time 3 "http://127.0.0.1:${port}/metrics" | grep -q ad_event_processor
' || warn "edge-bpf-sync :${EDGE_BPF_SYNC_METRICS_PORT:-9191}/metrics unreachable"

if [[ -x "$SCRIPTS/test/edge/xdp_resilience_drill.sh" && "${XDP_PREFLIGHT_DRILL:-0}" == "1" ]]; then
  run_check "xdp resilience drill" sudo bash "$SCRIPTS/test/edge/xdp_resilience_drill.sh" || true
fi

if [[ "$fail" -ne 0 && "$STRICT" == "1" ]]; then
  die "XDP preflight failed (STRICT=1)"
fi

log "xdp preflight complete (license ebpf_xdp_edge required in JWT)"

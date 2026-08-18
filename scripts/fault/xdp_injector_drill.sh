#!/usr/bin/env bash
# Lab-only XDP fault injector drill (MILESTONE §2.2.8). Not part of appliance default.
# Precondition: BTF vmlinux, clang, BPF objects — skips exit 0 when BTF/clang/BPF build missing.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'xdp_injector_drill: %s\n' "$*"; }

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  log "skip (BTF vmlinux missing)"
  exit 0
fi

if ! command -v clang > /dev/null 2>&1; then
  log "skip (clang missing; run make bpf-dev)"
  exit 0
fi

if ! go test ./internal/edge/ -run='^TestXDP_dropBlocklistedSource$' -count=1 > /dev/null 2>&1; then
  log "generating BPF objects (go generate ./internal/edge/)"
  go generate ./internal/edge/
fi

if ! go test ./internal/edge/ -run='^TestXDP_dropBlocklistedSource$' -count=1 > /dev/null 2>&1; then
  log "skip (BPF objects unavailable after generate)"
  exit 0
fi

log "build edge-xdp-fault"
go build -o bin/edge-xdp-fault ./cmd/edge-xdp-fault

log "program-mode injection (userspace BPF)"
OUT="$(mktemp)"
./bin/edge-xdp-fault -mode program -iters 500 -flood 1000 | tee "$OUT"
grep -q 'fault_proof fault=xdp_injector_malformed ' "$OUT"
grep -q 'fault_proof fault=xdp_injector_syn_flood ' "$OUT"

IFACE="${EDGE_XDP_FAULT_IFACE:-}"
if [[ -n "$IFACE" ]] && [[ "$(id -u)" -eq 0 ]]; then
  log "iface-mode injection on ${IFACE} (requires attached XDP for kernel counters)"
  IFACE_OUT="$(mktemp)"
  ./bin/edge-xdp-fault -mode iface -iface "$IFACE" -iters 100 -flood 500 -dst 127.0.0.1 | tee "$IFACE_OUT"
  grep -q 'fault_proof fault=xdp_injector_iface ' "$IFACE_OUT"
else
  log "iface mode skipped (set EDGE_XDP_FAULT_IFACE=lo and run as root to enable)"
fi

log "ok"

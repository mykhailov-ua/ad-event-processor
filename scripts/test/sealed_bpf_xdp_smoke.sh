#!/usr/bin/env bash
# Sealed assets — XDP attach lab: valid JWT + sealed edge BPF attaches like dev baseline.
# Harness: kernel_xdp_attach_lo_generic; drop proof via prog.Test (prog_test_same_maps), not lo RX.
# Skips exit 0 when BTF/clang/root/bpf objects missing — see header in docs/PILOT_LICENSE.md.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${SEALED_BPF_XDP_LOG:-sealed-bpf-xdp-smoke.log}"
IFACE="${SEALED_BPF_XDP_IFACE:-lo}"
: > "$LOG"

log() { printf 'sealed_bpf_xdp_smoke: %s\n' "$*"; }
die() {
  printf 'sealed_bpf_xdp_smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
  log "skip (BTF vmlinux missing)"
  exit 0
fi

if [[ "$(id -u)" -ne 0 ]]; then
  if sudo -n true 2> /dev/null; then
    log "re-exec with sudo for XDP attach on ${IFACE}"
    exec sudo -E bash "$0" "$@"
  fi
  log "skip (root required — run: sudo SEALED_BPF_XDP_SMOKE=1 $0)"
  exit 0
fi

if ! command -v clang > /dev/null 2>&1; then
  if [[ "$(id -u)" -eq 0 ]] && command -v apt-get > /dev/null 2>&1; then
    log "installing clang for bpf2go"
    if ! apt-get update -qq || ! apt-get install -y -qq clang llvm libbpf-dev linux-libc-dev; then
      log "skip (clang install failed — run make bpf-dev)"
      exit 0
    fi
  else
    log "skip (clang required — run make bpf-dev)"
    exit 0
  fi
fi

log "regenerating edge BPF when needed"
go generate ./internal/edge/

log "smoke: sealed collection load (TestEdgeSealed_ValidLicenseLoadsCollection)"
if ! go test ./internal/edge/ -run='^TestEdgeSealed_ValidLicenseLoadsCollection$' -count=1 -v 2>&1 | tee -a "$LOG"; then
  die "sealed collection load failed (see $LOG)"
fi

log "smoke: MCK path unit (TestEdgeSealed_MCKMatchesLicenseFilePath)"
go test ./internal/edge/ -run='^TestEdgeSealed_MCKMatchesLicenseFilePath$' -count=1 >> "$LOG" 2>&1

export SEALED_BPF_XDP_SMOKE=1
log "attach lab on ${IFACE} (TestEdgeSealed_XDPAttachMatchesBaseline)"
go test ./internal/edge/ -run='^TestEdgeSealed_XDPAttachMatchesBaseline$' -count=1 -v 2>&1 | tee -a "$LOG"

grep -q 'fault_proof fault=sealed_bpf_xdp_smoke ' "$LOG" || die "missing fault_proof line in $LOG"
grep -q 'harness=kernel_xdp_attach_lo_generic' "$LOG" || die "missing harness label in $LOG"
grep -q 'drop_assertion=prog_test_same_maps' "$LOG" || die "missing drop_assertion label in $LOG"

log "ok"

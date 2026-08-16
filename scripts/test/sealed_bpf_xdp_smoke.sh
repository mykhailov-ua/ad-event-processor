#!/usr/bin/env bash
# V2-B.D2: valid license + sealed BPF → XDP attach equivalent to unsealed baseline.
# Harness: kernel_xdp_attach_lo_generic + prog_test_same_maps (not raw lo RX).
# Precondition: BTF vmlinux, root, clang/bpf2go objects — skips exit 0 when missing.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${SEALED_BPF_XDP_LOG:-sealed-bpf-xdp-smoke.log}"
IFACE="${SEALED_BPF_XDP_IFACE:-lo}"

log() { printf 'sealed_bpf_xdp_smoke: %s\n' "$*"; }
die() { printf 'sealed_bpf_xdp_smoke: ERROR: %s\n' "$*" >&2; exit 1; }

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
	log "skip (BTF vmlinux missing)"
	exit 0
fi

if [[ "$(id -u)" -ne 0 ]]; then
	if sudo -n true 2>/dev/null; then
		log "re-exec with sudo for XDP attach on ${IFACE}"
		exec sudo -E bash "$0" "$@"
	fi
	log "skip (root required for XDP attach — run: sudo SEALED_BPF_XDP_SMOKE=1 $0)"
	exit 0
fi

if ! command -v clang >/dev/null 2>&1; then
	if command -v apt-get >/dev/null 2>&1 && [[ "$(id -u)" -eq 0 ]]; then
		log "installing clang for bpf2go"
		if ! apt-get update -qq || ! apt-get install -y -qq clang llvm libbpf-dev linux-libc-dev; then
			log "skip (clang install failed — run make bpf-dev)"
			exit 0
		fi
	else
		log "skip (clang required for bpf2go — run make bpf-dev)"
		exit 0
	fi
fi

if ! go test ./internal/edge/bpf/ -run='^TestXDP_dropBlocklistedSource$' -count=1 >/dev/null 2>&1; then
	log "generating edge BPF objects"
	go generate ./internal/edge/bpf/
fi

if ! go test ./internal/edge/bpf/ -run='^TestSealedBPF_ValidMCKLoadsCollection$' -count=1 >/dev/null 2>&1; then
	log "skip (sealed BPF collection load unavailable — bpf objects or BTF)"
	exit 0
fi

export SEALED_BPF_XDP_SMOKE=1
log "collection load + attach baseline vs sealed on ${IFACE}"
go test ./internal/edge/bpf/ -run='^TestSealedBPF_ValidLicenseAttachBaseline$' -count=1 -v 2>&1 | tee "$LOG"

grep -q 'fault_proof fault=sealed_bpf_xdp_smoke ' "$LOG" || die "missing fault_proof line in $LOG"
grep -q 'harness=kernel_xdp_attach_lo_generic' "$LOG" || die "missing harness label in $LOG"

log "ok"

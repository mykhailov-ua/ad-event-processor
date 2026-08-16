#!/usr/bin/env bash
# B.D2 prereq: regenerate edge BPF when clang is available; otherwise skip.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'bpf_edge_prereq_gate: %s\n' "$*"; }

if [[ ! -r /sys/kernel/btf/vmlinux ]]; then
	log "skip (BTF vmlinux missing)"
	exit 0
fi

if ! command -v clang >/dev/null 2>&1; then
	log "skip (clang not in PATH — apt install clang llvm libbpf-dev)"
	exit 0
fi

log "regenerating edge BPF (bpf2go)"
go generate ./internal/edge/bpf/

log "smoke: sealed collection load"
go test ./internal/edge/bpf/ -run 'Sealed' -count=1

if [[ "$(id -u)" -eq 0 ]] && [[ "${SEALED_BPF_XDP_SMOKE:-0}" == "1" ]]; then
	log "running XDP attach smoke"
	bash scripts/test/sealed_bpf_xdp_smoke.sh
else
	log "skip XDP attach (set SEALED_BPF_XDP_SMOKE=1 as root to run attach proof)"
fi

log "OK"

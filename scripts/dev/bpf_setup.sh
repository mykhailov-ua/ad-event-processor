#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
	CHECK_ONLY=1
fi

log() { printf 'bpf-setup: %s\n' "$*"; }

log "preflight"
if ! bash "$SCRIPTS/load/bpf_requirements.sh"; then
	log "WARN: BPF preflight reported failures (attach may still work as root)"
fi

if [[ "$CHECK_ONLY" == "1" ]]; then
	exit 0
fi

log "building loadtest_probe.o"
bash "$SCRIPTS/load/bpf_build.sh"

log "building bpf-collector"
mkdir -p "$ROOT/bin"
go build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector

log "ready: deploy/dev/bpf/loadtest_probe.o bin/bpf-collector"
log "standalone session: sudo make bpf-session-start"
log "load test: sudo ESPX_BPF_PROBE=1 bash scripts/load/malformed.sh business"

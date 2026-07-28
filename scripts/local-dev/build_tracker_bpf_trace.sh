#!/usr/bin/env bash
# Build tracker with espx_bpf_trace markers for dev BPF uprobes.
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

OUT="${1:-$ROOT/bin/tracker-bpf-trace}"
log() { printf 'build-tracker-bpf-trace: %s\n' "$*"; }

log "building $OUT (-tags espx_bpf_trace)"
go build -tags espx_bpf_trace -o "$OUT" ./cmd/tracker
log "set ESPX_BPF_TRACKER_BINARY=$OUT when running bpf-collector"

#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${1:-$ROOT/bin/tracker-bpf-trace}"
log() { printf 'build-tracker-bpf-trace: %s\n' "$*"; }

log "building $OUT (-tags ad_event_processor_bpf_trace)"
go build -tags ad_event_processor_bpf_trace -o "$OUT" ./cmd/tracker
log "set AD_EVENT_PROCESSOR_BPF_TRACKER_BINARY=$OUT when running bpf-collector"

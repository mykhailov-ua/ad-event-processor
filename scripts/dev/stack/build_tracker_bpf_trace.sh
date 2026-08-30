#!/usr/bin/env bash
# Role: Build tracker binary with ad_event_processor_bpf_trace tag for native uprobes.
# Execution context: Dev host; output used by bpf-collector when AD_EVENT_PROCESSOR_BPF_NATIVE=1.
# Env knobs: first arg is output path (default bin/tracker-bpf-trace).
# Verify: bash scripts/dev/stack/build_tracker_bpf_trace.sh && test -x bin/tracker-bpf-trace
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${1:-$ROOT/bin/tracker-bpf-trace}"
log() { printf 'build-tracker-bpf-trace: %s\n' "$*"; }

log "building $OUT (-tags ad_event_processor_bpf_trace)"
go build -tags ad_event_processor_bpf_trace -o "$OUT" ./cmd/tracker
log "set AD_EVENT_PROCESSOR_BPF_TRACKER_BINARY=$OUT when running bpf-collector"

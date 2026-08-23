#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=1
fi

log() { printf 'bpf-setup: %s\n' "$*"; }

log "preflight"
if ! bash "$SCRIPTS/test/bpf_requirements.sh"; then
  log "WARN: BPF preflight reported failures (attach may still work as root)"
fi

if [[ "$CHECK_ONLY" == "1" ]]; then
  exit 0
fi

log "building loadtest_probe.o"
bash "$SCRIPTS/test/bpf_build.sh"

log "building bpf-collector"
mkdir -p "$ROOT/bin"
if ! ad_event_processor_go_build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector; then
  log "ERROR: bpf-collector build failed — set AD_EVENT_PROCESSOR_GO_BIN=/path/to/go"
  exit 1
fi

log "building tracker-bpf-trace (host uprobes for native tracker)"
if bash "$SCRIPTS/dev/build_tracker_bpf_trace.sh" "$ROOT/bin/tracker-bpf-trace"; then
  log "native uprobes: export AD_EVENT_PROCESSOR_BPF_TRACKER_BINARY=$ROOT/bin/tracker-bpf-trace"
else
  log "WARN: tracker-bpf-trace build failed (docker load-test trackers still carry bpf trace markers)"
fi

log "ready: deploy/dev/bpf/loadtest_probe.o bin/bpf-collector"
log "standalone session: sudo make bpf-session-start"
log "load test: sudo AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/malformed.sh business"

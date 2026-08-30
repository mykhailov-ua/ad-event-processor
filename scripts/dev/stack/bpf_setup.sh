#!/usr/bin/env bash
# Role: Build BPF load-test probe object, bpf-collector, and optional tracker uprobes binary.
# Execution context: Dev host with clang/llvm; invoked by stack.sh build or stack.sh bpf.
# Env knobs: AD_EVENT_PROCESSOR_GO_BIN (go binary path); --check skips build, requirements only.
# Verify: bash scripts/dev/stack/bpf_setup.sh --check
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=1
fi

log() { printf 'bpf-setup: %s\n' "$*"; }

log "preflight"
if ! bash "$SCRIPTS/test/bpf/requirements.sh"; then
  log "WARN: BPF preflight reported failures (attach may still work as root)"
fi

if [[ "$CHECK_ONLY" == "1" ]]; then
  exit 0
fi

log "building loadtest_probe.o"
bash "$SCRIPTS/test/bpf/build.sh"

log "building bpf-collector"
mkdir -p "$ROOT/bin"
if ! aed_go_build -o "$ROOT/bin/bpf-collector" ./cmd/bpf-collector; then
  log "ERROR: bpf-collector build failed - set AD_EVENT_PROCESSOR_GO_BIN=/path/to/go"
  exit 1
fi

log "building tracker-bpf-trace (host uprobes for AD_EVENT_PROCESSOR_BPF_NATIVE=1)"
if bash "$SCRIPTS/dev/stack/build_tracker_bpf_trace.sh" "$ROOT/bin/tracker-bpf-trace"; then
  log "native uprobes: AD_EVENT_PROCESSOR_BPF_NATIVE=1 uses $ROOT/bin/tracker-bpf-trace"
else
  log "WARN: tracker-bpf-trace build failed (docker load-test uses /proc/<tracker-pid>/exe for uprobes)"
fi

log "ready: deploy/dev/bpf/loadtest_probe.o bin/bpf-collector"
log "standalone session: sudo make bpf-session-start"
log "load test: sudo AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/load/malformed.sh business"

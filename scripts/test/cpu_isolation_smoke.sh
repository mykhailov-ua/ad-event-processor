#!/usr/bin/env bash

set -euo pipefail

# Role: CPU isolation smoke; verifies TRACKER_CPU_SET and compose cgroup profile when CPU_ISOLATION_ENABLED=1.
# Execution context: Operator host with .env; optional stack running tracker.
# Invariants/contracts enforced: Isolation env knobs present; tracker process affinity matches config when enabled.
# Verify: bash scripts/test/cpu_isolation_smoke.sh
# Env: CPU_ISOLATION_ENABLED, TRACKER_CPU_SET
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

CPU_ISOLATION_ENABLED="${CPU_ISOLATION_ENABLED:-1}"

log() { printf 'cpu-isolation-smoke: %s\n' "$*"; }

if [[ "$CPU_ISOLATION_ENABLED" != "1" && "${CPU_ISOLATION_SMOKE_FORCE:-0}" != "1" ]]; then
  log "skip (CPU_ISOLATION_ENABLED!=1)"
  exit 0
fi

GO_BIN="$(cd "$ROOT" && go env GOROOT)/bin/go"
if [[ ! -x "$GO_BIN" ]]; then
  GO_BIN="$(command -v go)"
fi
log "unit: internal/ingest cpuset"
"$GO_BIN" test ./internal/ingest/ -run TestCount -count=1

if [[ "$CPU_ISOLATION_ENABLED" != "1" ]]; then
  log "skip live (CPU_ISOLATION_ENABLED!=1; unit ok)"
  printf 'fault_proof fault=cpu_isolation_smoke status=partial proof=unit_only harness=cpuset_parse isolation=disabled\n'
  exit 0
fi

if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
  log "skip live (docker unavailable; unit ok)"
  printf 'fault_proof fault=cpu_isolation_smoke status=partial proof=unit_only harness=cpuset_parse docker=absent\n'
  exit 0
fi

bash "$SCRIPTS/ops/cpu_isolation.sh" verify
log "passed (live cpuset verify - see fault_proof from cpu_isolation.sh)"

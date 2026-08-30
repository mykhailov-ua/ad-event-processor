#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: License guard fault integration.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/guard_fault.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'license_guard_fault_gate: %s\n' "$*"; }

log "default build: license_guard not compiled in"
go test ./internal/licensing/ -run 'TestGuard_NotCompiledInDefaultBuild' -count=1

log "ingestion license paths (no license_guard tag)"
go test ./internal/ingest/ -run 'License' -short -count=1 -timeout=120s

log "malformed traffic soak is optional (docker + stack); dev builds omit license_guard"
if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  log "skip malformed.sh in gate (run manually: bash scripts/test/load/malformed.sh smoke)"
else
  log "skip malformed.sh (docker unavailable)"
fi

log "ok"

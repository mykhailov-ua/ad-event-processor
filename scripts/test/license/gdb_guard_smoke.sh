#!/usr/bin/env bash
# Role: GDB attach denial smoke for license_guard release build on Linux.
# Execution context: Requires gdb in PATH; skips on non-Linux or missing gdb.
# Env knobs: LICENSE_GDB_SMOKE (1 forced).
# Verify: bash scripts/test/license/gdb_guard_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'license_gdb_guard_smoke: %s\n' "$*"; }

if [[ "$(uname -s)" != "Linux" ]]; then
  log "skip (linux only)"
  exit 0
fi

if ! command -v gdb > /dev/null 2>&1; then
  log "skip (gdb not in PATH)"
  exit 0
fi

export LICENSE_GDB_SMOKE=1
log "running gdb attach smoke (harness=license_guard_release)"
go test -tags=license_guard ./internal/licensing/ \
  -run '^TestGuard_GDBAttachDenied$' \
  -count=1 \
  -timeout=2m \
  -v

log "ok"

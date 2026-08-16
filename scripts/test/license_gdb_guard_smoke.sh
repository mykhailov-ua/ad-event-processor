#!/usr/bin/env bash
# V2-C.D1: gdb attach denied when license_guard + ptrace watchdog active.
# Harness: license_guard_release (test binary subprocess, not full garbled tracker).
# Precondition: Linux, gdb in PATH, -tags=license_guard — skips exit 0 when missing.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'license_gdb_guard_smoke: %s\n' "$*"; }

if [[ "$(uname -s)" != "Linux" ]]; then
	log "skip (linux only)"
	exit 0
fi

if ! command -v gdb >/dev/null 2>&1; then
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

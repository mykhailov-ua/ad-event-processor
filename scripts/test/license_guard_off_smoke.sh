#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'license_guard_off_smoke: %s\n' "$*"; }
die() {
  printf 'license_guard_off_smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

log "config kill switches"
go test ./internal/config/ -run 'LicenseGuard' -count=1

if [[ "$(uname -s)" == "Linux" ]]; then
  log "guard disabled skips ptrace launcher (-tags=license_guard)"
  go test -tags=license_guard ./internal/licensing/ -run 'TestGuard_DisabledSkipsPtraceLauncher' -count=1
else
  log "skip linux-only guard launcher test"
fi

log "default CI build omits license_guard tag"
go test ./internal/licensing/ -run 'TestGuard_NotCompiledInDefaultBuild' -count=1

if ! grep -q 'AD_EVENT_PROCESSOR_LICENSE_GUARD=0' .cursor/rules/licensing.mdc; then
  die ".cursor/rules/licensing.mdc missing LICENSE_GUARD=0 ops note"
fi
if ! grep -q 'LICENSE_GUARD_PTRACE=0' .cursor/rules/licensing.mdc; then
  die ".cursor/rules/licensing.mdc missing LICENSE_GUARD_PTRACE=0 ops note"
fi

log "ok - set AD_EVENT_PROCESSOR_LICENSE_GUARD=0 (or GUARD_PTRACE=0) before gdb/strace/delve on release builds"

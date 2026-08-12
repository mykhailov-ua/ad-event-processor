#!/usr/bin/env bash
# Billing export held-out smoke (MILESTONE §1.1.2): ledger rows → non-zero export bytes.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

log() { printf 'billing_export_smoke: %s\n' "$*"; }

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	log "skip (docker unavailable for Postgres testcontainer)"
	exit 0
fi

GO_BIN="$(ad_event_processor_go_bin)"
log "integration test"
"$GO_BIN" test ./internal/controlplane/adminapi/ -run '^TestJobRunner_ExportLedgerNonZeroBytes$' -count=1

log "ok"

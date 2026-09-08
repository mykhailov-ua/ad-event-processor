#!/usr/bin/env bash
# Role: Operator wrapper for periodic compose fault drills (P1 resilience cadence).
# Execution context: Dedicated host with stack up; logs to var/fault-drill/.
# Env knobs: COMPOSE_FAULT_LOG; SCOPE (default spool - fast subset).
# Verify: bash scripts/ops/fault_drill_scheduled.sh spool
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

SCOPE="${1:-spool}"
LOG="${COMPOSE_FAULT_LOG:-$ROOT/var/fault-drill/scheduled-$(date -u +%Y%m%dT%H%M%SZ).log}"
mkdir -p "$(dirname "$LOG")"

printf 'fault-drill-scheduled: scope=%s log=%s\n' "$SCOPE" "$LOG"
COMPOSE_FAULT_LOG="$LOG" bash "$SCRIPTS/fault/compose_fault_drill.sh" "$SCOPE"
printf 'fault-drill-scheduled: PASS scope=%s\n' "$SCOPE"

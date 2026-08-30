#!/usr/bin/env bash

set -euo pipefail

# Role: Library: Tee CI job output to var/ci.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/tee_ci_log.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if (($# < 2)); then
  echo "usage: tee_ci_log.sh <log-filename> <command> [args...]" >&2
  exit 2
fi

log_name="$1"
shift
log_path="$(ci_artifact_path "$log_name")"
"$@" 2>&1 | tee "$log_path"

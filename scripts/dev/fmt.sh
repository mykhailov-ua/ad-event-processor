#!/usr/bin/env bash
set -euo pipefail

# Role: Operator alias for universal repo format (Go, shell, Lua, prettier, BPF clang).
# Execution context: Local dev; same contract as make fmt / scripts/ci/format.sh.
# Verify: bash scripts/dev/fmt.sh
#          bash scripts/dev/fmt.sh --check
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/fmt.sh"
aed_fmt "$@"

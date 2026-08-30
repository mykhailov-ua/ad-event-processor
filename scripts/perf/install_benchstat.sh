#!/usr/bin/env bash
# Role: Install golang.org/x/perf/cmd/benchstat when absent from PATH.
# Execution context: CI or local perf gate prep; no-op when benchstat already installed.
# Env knobs: none.
# Verify: bash scripts/perf/install_benchstat.sh && command -v benchstat
set -euo pipefail

if command -v benchstat > /dev/null 2>&1; then
  exit 0
fi

go install golang.org/x/perf/cmd/benchstat@latest

#!/usr/bin/env bash
# Role: Compare current bench output to baseline via cmd/perf-gate; seeds baseline on first run.
# Execution context: CI perf tier; two file arguments baseline.txt and current.txt.
# Env knobs: none.
# Verify: bash scripts/perf/perf_baseline_gate.sh /tmp/base.txt /tmp/cur.txt
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if [[ $# -ne 2 ]]; then
  echo "usage: perf_baseline_gate.sh <baseline.txt> <current.txt>" >&2
  exit 2
fi

BASELINE="$1"
CURRENT="$2"

if [[ ! -s "$BASELINE" ]]; then
  echo "WARN: no baseline at $BASELINE; seeding from current run (first nightly or cache miss)"
  cp "$CURRENT" "$BASELINE"
  exit 0
fi

go run ./cmd/perf-gate "$BASELINE" "$CURRENT"

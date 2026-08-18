#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if [[ $# -lt 2 ]]; then
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

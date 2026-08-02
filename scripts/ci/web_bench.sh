#!/usr/bin/env bash
# Run admin UI micro-benchmarks (node perf_hooks + heap delta).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

node --expose-gc "$ROOT/web/scripts/bench/run.mjs"

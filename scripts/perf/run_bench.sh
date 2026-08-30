#!/usr/bin/env bash
# Role: Generic microbench runner with fixed count and GOMAXPROCS=1 for reproducible output.
# Execution context: Repo root; args: bench_regex then package paths.
# Env knobs: none (benchtime 200ms, count 10).
# Verify: bash scripts/perf/run_bench.sh 'BenchmarkGetShard' ./internal/domain/shard/
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <bench_regex> <package...>" >&2
  exit 2
fi

PATTERN="$1"
shift

export GOMAXPROCS=1
exec go test -run='^$' \
  -bench="$PATTERN" \
  -benchmem \
  -benchtime=200ms \
  -count=10 \
  -cpu=1 \
  "$@"

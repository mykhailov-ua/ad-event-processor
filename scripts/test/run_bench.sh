#!/usr/bin/env bash
# Repeatable micro/integration benches (not make test-alloc-gate).
#
# Examples:
#   bash scripts/test/run_bench.sh 'BenchmarkUnifiedFilter_Check' ./internal/ingestion
#   bash scripts/test/run_bench.sh 'BenchmarkUnifiedFilter_Check_integration' ./internal/ingestion
#   bash scripts/test/run_bench.sh 'BenchmarkLuaScript_Happy' ./internal/ingestion
#   bash scripts/test/run_bench.sh 'BenchmarkPostgresStoreBatch_integration' ./internal/ingestion
#   bash scripts/test/run_bench.sh 'BenchmarkTrackE2E_accept' ./internal/ingestion
#
# Integration/Lua benches skip under -short; do not pass -short here.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ $# -lt 1 ]]; then
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

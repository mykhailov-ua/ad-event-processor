#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BASELINE_DIR="${1:-.ci-baselines/cache-miss}"
mkdir -p "$BASELINE_DIR"
CURRENT="$BASELINE_DIR/current.txt"
BASELINE="$BASELINE_DIR/perf_stat.txt"
RUN_LOG="${2:-cache_miss_perf_stat.txt}"

BENCH_PATTERN='Benchmark(FilterFraudBoost|GetShard|RunAuction$$)'
BENCH_PKGS='./internal/ingest ./internal/rtb'

export GOMAXPROCS=1

if ! command -v perf > /dev/null 2>&1; then
  echo "SKIPPED: perf not installed (self-hosted runner required)"
  exit 0
fi

if ! perf stat -e cache-misses,cache-references,instructions -r 3 -- \
  go test -run='^$' -bench="$BENCH_PATTERN" -benchtime=300ms -count=1 -cpu=1 \
  $BENCH_PKGS 2>&1 | tee "$RUN_LOG"; then
  echo "FAIL: perf stat bench run failed" >&2
  exit 1
fi

misses="$(awk '/cache-misses/ {gsub(/,/,"",$1); miss=$1} /cache-references/ {gsub(/,/,"",$1); ref=$1} END {print miss+0}' "$RUN_LOG")"
refs="$(awk '/cache-references/ {gsub(/,/,"",$1); print $1+0; exit}' "$RUN_LOG")"
instructions="$(awk '/instructions/ {gsub(/,/,"",$1); print $1+0; exit}' "$RUN_LOG")"

if [[ "$refs" -eq 0 ]]; then
  echo "FAIL: could not parse cache-references from $RUN_LOG" >&2
  exit 1
fi

miss_pct="$(awk -v m="$misses" -v r="$refs" 'BEGIN {printf "%.4f", (m/r)*100}')"
{
  echo "cache_misses=$misses"
  echo "cache_references=$refs"
  echo "instructions=$instructions"
  echo "cache_miss_pct=$miss_pct"
} | tee "$CURRENT"

if [[ ! -s "$BASELINE" ]]; then
  echo "WARN: no baseline at $BASELINE; seeding from current run"
  cp "$CURRENT" "$BASELINE"
  exit 0
fi

baseline_pct="$(awk -F= '/^cache_miss_pct=/ {print $2}' "$BASELINE" | tail -1)"
if [[ -z "$baseline_pct" || "$baseline_pct" == "0" ]]; then
  echo "WARN: invalid baseline; reseeding"
  cp "$CURRENT" "$BASELINE"
  exit 0
fi

limit_pct="$(awk -v b="$baseline_pct" 'BEGIN {printf "%.4f", b * 1.12}')"
fail="$(awk -v c="$miss_pct" -v l="$limit_pct" 'BEGIN {print (c > l) ? 1 : 0}')"
echo "cache_miss_pct baseline=$baseline_pct current=$miss_pct limit(+12%)=$limit_pct"

if [[ "$fail" -eq 1 ]]; then
  echo "FAIL: cache-miss rate regressed >12% vs baseline" >&2
  exit 1
fi

echo "PASS: cache-miss rate within +12% of baseline"
cp "$CURRENT" "$BASELINE"

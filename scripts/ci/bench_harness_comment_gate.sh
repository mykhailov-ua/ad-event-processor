#!/usr/bin/env bash
# Fail when a new internal/ingestion benchmark lacks a harness label in its godoc block.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

ALLOWLIST="$ROOT/scripts/ci/bench_harness_allowlist.txt"
fail=0

bench_has_harness_comment() {
	local file=$1 line=$2
	sed -n "$((line - 20)),${line}p" "$file" | rg -qi 'harness'
}

while IFS=: read -r file line rest; do
	name="${rest#func }"
	name="${name%%(*}"
	if rg -qx "$name" "$ALLOWLIST"; then
		continue
	fi
	if bench_has_harness_comment "$file" "$line"; then
		continue
	fi
	echo "bench_harness_comment_gate: missing harness label for $name ($file:$line)" >&2
	fail=1
done < <(rg -n '^func Benchmark' internal/ingestion/*_bench_test.go)

if [[ "$fail" -ne 0 ]]; then
	echo "bench_harness_comment_gate: add // harness: <label> godoc or allowlist entry" >&2
	exit 1
fi

echo "bench_harness_comment_gate: OK"

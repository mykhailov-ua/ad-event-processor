#!/usr/bin/env bash
set -euo pipefail

# Role: Behavioral holdout tests on hot paths must run under -race (no //go:build !race).
# Verify: bash scripts/ci/static/hot_path_race_holdout_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

HOLDOUT_TESTS=(
  internal/ingest/local_quanta_fullskip_test.go
  internal/ingest/local_quanta_filter_test.go
  internal/ingest/local_quanta_flush_test.go
  internal/stream/local_quanta_test.go
)

for path in "${HOLDOUT_TESTS[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "hot_path_race_holdout_gate: ERROR missing $path"
    failed=1
    continue
  fi
  if head -n 1 "$path" | rg -q '^//go:build !race'; then
    echo "hot_path_race_holdout_gate: FAIL $path must not exclude race detector"
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "hot_path_race_holdout_gate: OK"

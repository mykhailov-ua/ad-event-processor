#!/usr/bin/env bash
set -euo pipefail

# Role: Forbid detached goroutines on Tier B track/filter worker paths.
# Verify: bash scripts/ci/static/ingest_lifetime_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

LIFETIME_FILES=(
  internal/ingest/trackwire.go
  internal/ingest/gnet/worker.go
)

for path in "${LIFETIME_FILES[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "ingest_lifetime_gate: missing $path"
    failed=1
    continue
  fi
  if rg -n 'go func\(' "$path" 2> /dev/null; then
    echo "ingest_lifetime_gate: FAIL detached goroutine in $path"
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "ingest_lifetime_gate: OK"

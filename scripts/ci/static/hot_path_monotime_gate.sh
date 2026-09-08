#!/usr/bin/env bash
set -euo pipefail

# Role: Single runtime.nanotime linkname site in production hot packages.
# Verify: bash scripts/ci/static/hot_path_monotime_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

if rg -n 'linkname.*runtime\.nanotime' \
  internal/ingest internal/filter internal/openrtb internal/rtb \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  echo "hot_path_monotime_gate: FAIL duplicate runtime.nanotime linkname (use pkg/monotime)"
  failed=1
fi

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "hot_path_monotime_gate: OK"

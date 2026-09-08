#!/usr/bin/env bash
set -euo pipefail

# Role: Forbid returning unsafeString/UnsafeString over stack scratch buffers in ingest production code.
# Verify: bash scripts/ci/static/ingest_unsafe_string_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

if rg -n 'return (unsafeString|filter\.UnsafeString)\(append' internal/ingest/ \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  echo "ingest_unsafe_string_gate: FAIL stack-scratch unsafe string return in ingest"
  failed=1
fi

if rg -n 'sync\.Map' internal/ingest/stream_admission_metrics.go 2> /dev/null; then
  echo "ingest_unsafe_string_gate: FAIL sync.Map in stream_admission_metrics.go (use atomic.Pointer slots)"
  failed=1
fi

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "ingest_unsafe_string_gate: OK"

#!/usr/bin/env bash
# Fail on bare t.Skip() in Go tests — skip reason required (S7.7).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if rg -q 't\.Skip\(\)\s*$' internal/ pkg/ --glob '*_test.go'; then
  echo "Error: bare t.Skip() without reason (add integration skip message):"
  rg 't\.Skip\(\)\s*$' internal/ pkg/ --glob '*_test.go'
  exit 1
fi

echo "Integration skip reason gate: OK"

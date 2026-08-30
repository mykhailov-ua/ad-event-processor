#!/usr/bin/env bash
set -euo pipefail

# Role: Static gate: Cold-path JSON decode limits in handlers.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/cold_path_json.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "cold_path_json_gate: body limit regression tests..."

go test ./pkg/coldpath/ -run='TestReadLimitedBody|TestDecodeRequestOrBadRequest' -count=1
go test ./internal/controlplane/ -run='TestColdPathJSON|TestLoginBodySizeLimit' -count=1
go test ./internal/controlplane/ -run='TestColdPathJSON' -count=1
go test ./internal/payment/ -run='TestColdPathJSON' -count=1

echo "cold_path_json_gate: OK"

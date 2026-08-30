#!/usr/bin/env bash
set -euo pipefail

# Role: Race detector short suite on internal and pkg.
# Execution context: CI merge-race-short job.
# Invariants/contracts enforced: go test -race -short; any race fails the gate.
# Verify: bash scripts/ci/race_short.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
go test -race -short -count=1 -timeout=25m ./internal/... ./pkg/...

#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: License hot-path alloc microbench gate.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/alloc.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "license_alloc_gate: LicenseFilter zero allocs"
go test ./internal/filter/ -run 'TestFilterLicense_zeroAllocs' -count=1

echo "license_alloc_gate: LicenseRPSFilter seed coupling unit tests"
go test ./internal/ingest/ -run 'LicenseRPSFilter' -count=1

echo "license_alloc_gate: OK"

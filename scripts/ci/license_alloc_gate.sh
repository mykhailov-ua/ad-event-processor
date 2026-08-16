#!/usr/bin/env bash
# V2-C.D6 / V2-B.D3: license filter hot-path must stay 0 allocs/op.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "license_alloc_gate: LicenseFilter zero allocs"
go test ./internal/ingestion/ -run 'TestFilterLicense_zeroAllocs' -count=1

echo "license_alloc_gate: LicenseRPSFilter seed coupling unit tests"
go test ./internal/ingestion/ -run 'LicenseRPSFilter' -count=1

echo "license_alloc_gate: OK"

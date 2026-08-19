#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "license_alloc_gate: LicenseFilter zero allocs"
go test ./internal/ingestion/ -run 'TestFilterLicense_zeroAllocs' -count=1

echo "license_alloc_gate: LicenseRPSFilter seed coupling unit tests"
go test ./internal/ingestion/ -run 'LicenseRPSFilter' -count=1

echo "license_alloc_gate: OK"

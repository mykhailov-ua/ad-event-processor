#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
bash scripts/ci/license_route_gap_gate.sh
bash scripts/test/release_hardening_smoke.sh
bash scripts/ci/license_fuzz_nightly_gate.sh
go test ./internal/edge/ -run '^TestEdgeSealed_MCKMatchesLicenseFilePath$' -count=1
go test ./internal/licensing/ -run 'Property_P_C3|Property_P_C4|VerifyDeploymentBind|HWID_Deterministic' -count=1
bash scripts/ci/license_alloc_gate.sh
echo "license_admin_smoke: OK"

#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../../.."
bash scripts/ci/license/route_gap.sh
bash scripts/test/release/hardening_smoke.sh
bash scripts/ci/license/fuzz_nightly.sh
go test ./internal/edge/ -run '^TestEdgeSealed_MCKMatchesLicenseFilePath$' -count=1
go test ./internal/licensing/ -run 'Property_P_C3|Property_P_C4|VerifyDeploymentBind|HWID_Deterministic' -count=1
bash scripts/ci/license/alloc.sh
echo "license_admin_smoke: OK"

#!/usr/bin/env bash
set -euo pipefail

# Role: CI proxy for binary_patch_lab PT-D04 ingest seed-coupling holdouts (no garble build).
# Execution context: license red-team via scripts/security/license_red_team.sh.
# Invariants/contracts enforced: seed coupling blocks over-cap RPS, OpenRTB, and LicenseFilter ingest.
# Verify:
# bash scripts/ci/static/binary_patch_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

PASS=0
FAIL=0

run_case() {
  local name="$1"
  shift
  printf '  %-42s' "$name"
  if "$@"; then
    echo "PASS"
    PASS=$((PASS + 1))
  else
    echo "FAIL"
    FAIL=$((FAIL + 1))
  fi
}

echo "binary_patch_gate: PT-D04 proxy ingest seed coupling..."
run_case "pt_d04_seed_coupling_rps" \
  go test ./internal/ingest/ -run 'LicenseRPSFilter_seedCoupling' -count=1 -short
run_case "pt_d04_openrtb_seed_coupling" \
  go test ./internal/ingest/ -run 'OpenRTBLicenseAllowed_seedCoupling' -count=1 -short
run_case "pt_d04_license_filter_seed_coupling" \
  go test ./internal/filter/ -run 'TestLicenseFilter_seedCouplingBlocksIngest' -count=1 -short

echo ""
echo "binary_patch_gate: pass=$PASS fail=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
echo "binary_patch_gate: OK"

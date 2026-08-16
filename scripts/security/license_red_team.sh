#!/usr/bin/env bash
# Automates P0 license bypass scenarios (S0.5–S0.7). Requires Docker for integration tests.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0

run_case() {
	local name="$1"
	shift
	printf '%-42s' "$name"
	if "$@"; then
		echo "PASS"
		PASS=$((PASS + 1))
	else
		echo "FAIL"
		FAIL=$((FAIL + 1))
	fi
}

skip_case() {
	local name="$1"
	local reason="$2"
	printf '%-42s' "$name"
	echo "SKIP ($reason)"
	SKIP=$((SKIP + 1))
}

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	skip_case "integration_license_protection" "docker unavailable"
else
	run_case "integration_license_protection" \
		go test ./tests/integration/ -run 'LicenseProtection' -count=1
fi

run_case "licensing_unit_revoked_claim" \
	go test ./internal/licensing/ -run 'Revoked|DetermineState' -count=1

run_case "license_rps_filter" \
	go test ./internal/ingestion/ -run 'LicenseRPSFilter' -count=1

run_case "sync_entitlements_expired_default" \
	go test ./internal/ingestion/ -run 'SyncEntitlements' -count=1

run_case "license_seed_coupling_rps" \
	go test ./internal/ingestion/ -run 'LicenseRPSFilter_seedCoupling' -count=1

run_case "sealed_unified_filter_lua" \
	go test ./internal/ingestion/ -run 'ResolveUnifiedFilterLua' -count=1

run_case "licensing_mck_seal" \
	go test ./internal/licensing/ -run 'MCK|Seal|FeatureSeed' -count=1

run_case "licensing_skew_watch" \
	go test ./internal/licensing/ -run SkewWatch -count=1

if [[ -n "${ASSET_SEAL_SALT:-}" ]]; then
	run_case "asset_seal_salt_smoke" \
		bash scripts/ci/asset_seal_salt_smoke.sh
else
	skip_case "asset_seal_salt_smoke" "ASSET_SEAL_SALT unset"
fi

	if [[ "$(uname -s)" == "Linux" ]]; then
	run_case "edge_bpf_sealed_invalid_mck" \
		go test ./internal/edge/bpf/ -run 'Sealed' -count=1
	run_case "hwid_strings_gate" \
		bash scripts/ci/hwid_strings_gate.sh
	run_case "public_key_strings_gate" \
		bash scripts/ci/public_key_strings_gate.sh
	run_case "licensing_guard" \
		go test -tags=license_guard ./internal/licensing/ -run Guard -count=1
	run_case "license_guard_off_smoke" \
		bash scripts/test/license_guard_off_smoke.sh
else
	skip_case "edge_bpf_sealed_invalid_mck" "linux only"
	skip_case "hwid_strings_gate" "linux only"
	skip_case "public_key_strings_gate" "linux only"
	skip_case "licensing_guard" "linux only"
	skip_case "license_guard_off_smoke" "linux only"
fi

if [[ "$(uname -s)" == "Linux" ]]; then
	run_case "license_guard_fault_gate" \
		bash scripts/ci/license_guard_fault_gate.sh
else
	skip_case "license_guard_fault_gate" "linux only"
fi

echo ""
echo "license red-team summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [[ "$FAIL" -gt 0 ]]; then
	exit 1
fi

echo ""
bash scripts/test/license_red_team_extended.sh

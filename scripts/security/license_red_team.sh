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

echo ""
echo "license red-team summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [[ "$FAIL" -gt 0 ]]; then
	exit 1
fi

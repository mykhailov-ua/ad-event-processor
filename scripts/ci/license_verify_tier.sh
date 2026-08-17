#!/usr/bin/env bash
# M0/M1/M2 licensing verification tiers (.cursor/MILESTONE.md §8).
# Skips print reason and exit 0; failures exit 1.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0

log() { printf 'license_verify_tier: %s\n' "$*"; }

run_gate() {
	local id="$1"
	shift
	printf '  %-8s ' "$id"
	if "$@"; then
		echo "PASS"
		PASS=$((PASS + 1))
	else
		echo "FAIL"
		FAIL=$((FAIL + 1))
	fi
}

skip_gate() {
	local id="$1"
	local reason="$2"
	printf '  %-8s SKIP (%s)\n' "$id" "$reason"
	SKIP=$((SKIP + 1))
}

on_linux() { [[ "$(uname -s)" == "Linux" ]]; }

log "=== Tier M0 (baseline) ==="
run_gate M0.2 go test ./internal/licensing/ -count=1
run_gate M0.3 go test ./internal/licensing/ -run DeterministicVector -count=1
run_gate M0.4 bash scripts/security/license_red_team.sh
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	run_gate M0.5 go test ./tests/integration/ -run LicenseProtection -count=1
else
	skip_gate M0.5 "docker unavailable"
fi
run_gate M0.6 go test ./internal/licensing/ -run 'VerifyJWT/Tampered' -count=1

log "=== Tier M1 (crypto rigor) ==="
run_gate M1.1 go test ./internal/licensing/ -run HKDF_RFC5869 -count=1
run_gate M1.2 go test ./internal/licensing/ -run HWID -count=1
run_gate M1.3 go test ./internal/licensing/ -run HWID_Deterministic -count=1
run_gate M1.4 go test ./internal/licensing/ -rapid.checks=200 -run Property -count=1
if [[ "${LICENSE_VERIFY_FUZZ:-0}" == "1" ]]; then
	run_gate M1.5 go test ./internal/licensing/ -fuzz=FuzzVerifyJWT -fuzztime=10s -count=1
else
	skip_gate M1.5 "set LICENSE_VERIFY_FUZZ=1 for 10s fuzz smoke"
fi

log "=== Tier M2 (entanglement) ==="
run_gate M2.1 go test ./internal/licensing/ -run 'MCK|DeriveMCK' -count=1
run_gate M2.2 go test ./internal/licensing/ -run 'MCK_Sensitivity' -count=1
run_gate M2.3 go test ./internal/licensing/ -run Seal -count=1
if on_linux; then
	run_gate M2.4 go test ./internal/edge/bpf/ -run 'TestEdgeSealed_' -count=1
	run_gate M2.4b go test ./internal/ingestion/ -run ResolveUnifiedFilterLua -count=1
else
	skip_gate M2.4 "linux only"
	skip_gate M2.4b "linux only"
fi
run_gate M2.5 go test ./internal/licensing/ -run FeatureSeed -count=1
if command -v openssl >/dev/null 2>&1 && openssl kdf -help >/dev/null 2>&1; then
	run_gate M2.6 bash scripts/ci/license_differential_gate.sh
else
	skip_gate M2.6 "openssl kdf unavailable"
fi
if [[ "${SEALED_BPF_XDP_SMOKE:-0}" == "1" ]]; then
	run_gate M2.7 bash scripts/test/sealed_bpf_xdp_smoke.sh
else
	skip_gate M2.7 "set SEALED_BPF_XDP_SMOKE=1 as root for XDP attach lab"
fi
if [[ "${GARBLE_LITERALS_P99_SMOKE:-0}" == "1" ]]; then
	run_gate D1.4 bash scripts/test/garble_literals_p99_smoke.sh
else
	skip_gate D1.4 "set GARBLE_LITERALS_P99_SMOKE=1 + load-test stack for garble p99 lab"
fi

log "=== Tier M3 (release spot-check; optional) ==="
if [[ "${LICENSE_VERIFY_GARBLED:-0}" == "1" ]]; then
	run_gate M3.3 bash scripts/ci/license_red_team_garbled.sh
else
	skip_gate M3.3 "set LICENSE_VERIFY_GARBLED=1 for garbled release check"
fi
skip_gate M3.1 "TLC model optional (docs/formal/)"
skip_gate M3.2 "Alloy model optional (docs/formal/)"
run_gate M3.4 bash scripts/ci/license_fuzz_nightly_gate.sh
run_gate M3.5 bash scripts/test/license_red_team_extended.sh
if [[ "${LICENSE_VERIFY_RELEASE_QA:-0}" == "1" ]]; then
	run_gate M3.6 bash scripts/test/release_qa_smoke.sh
else
	skip_gate M3.6 "set LICENSE_VERIFY_RELEASE_QA=1 for fuzz+garble release QA smoke"
fi

log "=== Hot-path alloc (V2-C.D6 / V2-B.D3 subset) ==="
run_gate C.D6 bash scripts/ci/license_alloc_gate.sh

echo ""
log "summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [[ "$FAIL" -gt 0 ]]; then
	exit 1
fi

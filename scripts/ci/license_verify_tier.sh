#!/usr/bin/env bash

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

log "Tier baseline"
run_gate baseline-2 go test ./internal/licensing/ -count=1
run_gate baseline-3 go test ./internal/licensing/ -run DeterministicVector -count=1
run_gate baseline-4 bash scripts/security/license_red_team.sh
if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
  run_gate baseline-5 go test ./tests/integration/ -run LicenseProtection -count=1
else
  skip_gate baseline-5 "docker unavailable"
fi
run_gate baseline-6 go test ./internal/licensing/ -run 'VerifyJWT/Tampered' -count=1

log "Tier crypto"
run_gate crypto-1 go test ./internal/licensing/ -run HKDF_RFC5869 -count=1
run_gate crypto-2 go test ./internal/licensing/ -run HWID -count=1
run_gate crypto-3 go test ./internal/licensing/ -run HWID_Deterministic -count=1
run_gate crypto-4 go test ./internal/licensing/ -rapid.checks=200 -run Property -count=1
if [[ "${LICENSE_VERIFY_FUZZ:-0}" == "1" ]]; then
  run_gate crypto-5 go test ./internal/licensing/ -fuzz=FuzzVerifyJWT -fuzztime=10s -count=1
else
  skip_gate crypto-5 "set LICENSE_VERIFY_FUZZ=1 for 10s fuzz smoke"
fi

log "Tier entanglement"
run_gate entangle-1 go test ./internal/licensing/ -run 'MCK|DeriveMCK' -count=1
run_gate entangle-2 go test ./internal/licensing/ -run 'MCK_Sensitivity' -count=1
run_gate entangle-3 go test ./internal/licensing/ -run Seal -count=1
if on_linux; then
  run_gate entangle-4 go test ./internal/edge/ -run 'TestEdgeSealed_' -count=1
  run_gate entangle-4b go test ./internal/ingestion/ -run ResolveUnifiedFilterLua -count=1
else
  skip_gate entangle-4 "linux only"
  skip_gate entangle-4b "linux only"
fi
run_gate entangle-5 go test ./internal/licensing/ -run FeatureSeed -count=1
if command -v openssl > /dev/null 2>&1 && openssl kdf -help > /dev/null 2>&1; then
  run_gate entangle-6 bash scripts/ci/license_differential_gate.sh
else
  skip_gate entangle-6 "openssl kdf unavailable"
fi
if [[ "${SEALED_BPF_XDP_SMOKE:-0}" == "1" ]]; then
  run_gate entangle-7 bash scripts/test/sealed_bpf_xdp_smoke.sh
else
  skip_gate entangle-7 "set SEALED_BPF_XDP_SMOKE=1 as root for XDP attach lab"
fi
if [[ "${GARBLE_LITERALS_P99_SMOKE:-0}" == "1" ]]; then
  run_gate garble-p99 bash scripts/test/garble_literals_p99_smoke.sh
else
  skip_gate garble-p99 "set GARBLE_LITERALS_P99_SMOKE=1 + load-test stack for garble p99 lab"
fi

log "Tier release (optional spot-check)"
if [[ "${LICENSE_VERIFY_GARBLED:-0}" == "1" ]]; then
  run_gate release-3 bash scripts/ci/license_red_team_garbled.sh
else
  skip_gate release-3 "set LICENSE_VERIFY_GARBLED=1 for garbled release check"
fi
run_gate release-4 bash scripts/ci/license_fuzz_nightly_gate.sh
run_gate release-5 bash scripts/test/license_red_team_extended.sh
if [[ "${LICENSE_VERIFY_RELEASE_QA:-0}" == "1" ]]; then
  run_gate release-6 bash scripts/test/release_qa_smoke.sh
else
  skip_gate release-6 "set LICENSE_VERIFY_RELEASE_QA=1 for fuzz+garble release QA smoke"
fi

log "Hot-path alloc"
run_gate license-alloc bash scripts/ci/license_alloc_gate.sh

echo ""
log "summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi

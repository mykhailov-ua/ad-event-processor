#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0

log() { printf 'binary_patch_lab: %s\n' "$*"; }
err() { printf 'binary_patch_lab: ERROR: %s\n' "$*" >&2; }

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

skip_case() {
  local name="$1"
  local reason="$2"
  printf '  %-42s' "$name"
  echo "SKIP ($reason)"
  SKIP=$((SKIP + 1))
}

print_manual_steps() {
  log "manual PT-D04 / PT-D07 procedure: deploy/vendor/fixtures/binary_patch/README.md"
  log "1. Build: LICENSE_GUARD=1 bash scripts/ci/release_garble.sh /tmp/patch-lab tracker"
  log "2. Patch one byte in .text (hexedit on disk copy or gdb write if attach allowed)"
  log "3. Start tracker with valid license.jwt; guard on trips text_tamper within probe interval"
  log "4. Patch LicenseFilter only: seed coupling must still block over-cap RPS (PT-D04)"
}

if [[ "${LICENSE_BINARY_PATCH_MANUAL:-0}" == "1" ]]; then
  print_manual_steps
  if [[ "${LICENSE_BINARY_PATCH_BUILD:-0}" == "1" ]]; then
    if [[ -z "${GARBLE_SEED:-}" ]]; then
      err "set GARBLE_SEED for reproducible garble build"
      exit 2
    fi
    log "building guarded tracker to /tmp/patch-lab"
    LICENSE_GUARD=1 bash scripts/ci/release_garble.sh /tmp/patch-lab tracker
    log "binary: /tmp/patch-lab/tracker"
  fi
  exit 0
fi

log "automated catalog (PT-D04 proxy, PT-D07 proxy, PT-E08 proxy)"

run_case "pt_d04_seed_coupling_rps" \
  go test ./internal/ingest/ -run 'LicenseRPSFilter_seedCoupling' -count=1 -short
run_case "pt_d04_openrtb_seed_coupling" \
  go test ./internal/ingest/ -run 'OpenRTBLicenseAllowed_seedCoupling' -count=1 -short

if [[ "$(uname -s)" == "Linux" ]]; then
  run_case "pt_d07_text_tamper" \
    go test -tags=license_guard ./internal/licensing/ \
    -run '^TestGuard_TextTamper$' -count=1 -short
  run_case "pt_d07_tamper_stretch" \
    go test -tags=license_guard ./internal/licensing/ \
    -run '^TestGuard_TamperStretchBeforeTrip$' -count=1 -short
  run_case "pt_e08_trip_without_verify" \
    go test -tags=license_guard ./internal/licensing/ \
    -run '^TestGuard_TripWithoutVerifyCall$' -count=1 -short
else
  skip_case "pt_d07_text_tamper" "linux only"
  skip_case "pt_d07_tamper_stretch" "linux only"
  skip_case "pt_e08_trip_without_verify" "linux only"
fi

echo ""
log "summary: pass=$PASS fail=$FAIL skip=$SKIP"
log "full manual drill: LICENSE_BINARY_PATCH_MANUAL=1 bash scripts/lab/binary_patch_lab.sh"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi

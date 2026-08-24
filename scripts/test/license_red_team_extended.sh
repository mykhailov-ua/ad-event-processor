#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0

log() { printf 'license_red_team_extended: %s\n' "$*"; }

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

log "Step 7: IngestAllowed patch insufficient (MCK/seed coupling)"
run_case "redteam7_seed_coupling_rps" \
  go test ./internal/ingestion/ -run 'LicenseRPSFilter_seedCoupling' -count=1
run_case "redteam7_openrtb_seed_coupling" \
  go test ./internal/ingestion/ -run 'OpenRTBLicenseAllowed_seedCoupling' -count=1
run_case "redteam7_feature_seed_unit" \
  go test ./internal/licensing/ -run 'FeatureSeed' -count=1

log "Step 8: Sealed assets without valid JWT"
if [[ "$(uname -s)" == "Linux" ]]; then
  run_case "redteam8_sealed_bpf_invalid_mck" \
    go test ./internal/edge/ -run 'Sealed' -count=1
  run_case "redteam8_sealed_lua_resolve" \
    go test ./internal/ingestion/ -run 'ResolveUnifiedFilterLua' -count=1
else
  skip_case "redteam8_sealed_bpf_invalid_mck" "linux only"
  skip_case "redteam8_sealed_lua_resolve" "linux only"
fi

log "Step 9: gdb attach with guard on"
if [[ "$(uname -s)" == "Linux" ]] && command -v gdb > /dev/null 2>&1; then
  run_case "redteam9_gdb_guard_smoke" \
    env LICENSE_GDB_SMOKE=1 bash scripts/test/license_gdb_guard_smoke.sh
else
  skip_case "redteam9_gdb_guard_smoke" "linux + gdb required"
fi

log "Step 10: Clock rewind -> expired ingest"
run_case "redteam10_skew_watch_unit" \
  go test ./internal/licensing/ -run SkewWatch -count=1
run_case "redteam10_registry_skew_ingest" \
  go test ./internal/ingestion/ -run 'Registry_licenseRecheck_clockSkew' -count=1

echo ""
log "summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi

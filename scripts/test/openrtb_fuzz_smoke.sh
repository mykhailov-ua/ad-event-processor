#!/usr/bin/env bash
set -euo pipefail

# Role: OpenRTB 2.6 parser fuzz smoke across ingest and rtb packages.
# Execution context: Nightly parser-fuzz workflow; no Docker.
# Invariants/contracts enforced: No panic within OPENRTB_FUZZ_TIME (default 5s) per Fuzz* target.
# Verify: bash scripts/test/openrtb_fuzz_smoke.sh
# Env: OPENRTB_FUZZ_TIME
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${OPENRTB_FUZZ_TIME:-5s}"

run_fuzz() {
  local pkg="$1"
  shift
  for name in "$@"; do
    echo "openrtb_fuzz_smoke: $pkg Fuzz$name ($FUZZTIME)"
    go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" "$pkg"
  done
}

run_fuzz ./internal/ingest/ \
  ParseOpenRTB26Split \
  ParseOpenRTB26Helpers \
  ParseOpenRTB3FSM \
  OpenRTB26ImpSlotWalk

run_fuzz ./internal/openrtb/ \
  ValidateBytes \
  AppendApplyMacros \
  GzipCompress

echo "openrtb_fuzz_smoke: PASS"

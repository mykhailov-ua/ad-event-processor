#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$ROOT/scripts/lib/garble_literals_policy.sh"
source "$ROOT/scripts/lib/release_garble_policy.sh"

fail() {
  echo "garble_literals_policy_gate: $*" >&2
  exit 1
}

with_clean_env() {
  unset GARBLE_LITERALS
  unset GARBLE_LITERALS_TRACKER GARBLE_LITERALS_PROCESSOR GARBLE_LITERALS_CONTROL
}

with_clean_env
[[ "$(garble_literals_for_cmd tracker)" == "0" ]] || fail "tracker default must be 0"
[[ "$(garble_literals_for_cmd processor)" == "1" ]] || fail "processor default must be 1"
[[ "$(garble_literals_for_cmd control)" == "1" ]] || fail "control default must be 1"

with_clean_env
GARBLE_LITERALS=0
[[ "$(garble_literals_for_cmd tracker)" == "0" ]] || fail "GARBLE_LITERALS=0 must apply to tracker"
[[ "$(garble_literals_for_cmd control)" == "0" ]] || fail "GARBLE_LITERALS=0 must apply to control"

with_clean_env
GARBLE_LITERALS=1
[[ "$(garble_literals_for_cmd tracker)" == "1" ]] || fail "GARBLE_LITERALS=1 must apply to tracker"

with_clean_env
GARBLE_LITERALS_TRACKER=1
[[ "$(garble_literals_for_cmd tracker)" == "1" ]] || fail "GARBLE_LITERALS_TRACKER must override tracker default"
[[ "$(garble_literals_for_cmd control)" == "1" ]] || fail "control keeps default when only tracker overridden"

unset RELEASE_GARBLE GARBLE_SEED RELEASE_GARBLE_SKIP_SEED ASSET_SEAL_SALT AD_EVENT_PROCESSOR_ASSET_SEAL_SALT RELEASE_ASSET_SEAL_REQUIRED

RELEASE_GARBLE=1 GARBLE_SEED= release_garble_seed_ok && fail "release_garble must require GARBLE_SEED when RELEASE_GARBLE=1"
RELEASE_GARBLE=1 GARBLE_SEED=test-seed-fixed release_garble_seed_ok || fail "release_garble must accept GARBLE_SEED"
RELEASE_GARBLE=1 GARBLE_SEED= RELEASE_GARBLE_SKIP_SEED=1 release_garble_seed_ok || fail "RELEASE_GARBLE_SKIP_SEED=1 must allow missing seed"

RELEASE_GARBLE=1 ASSET_SEAL_SALT= release_asset_seal_salt_ok && fail "release_garble must require ASSET_SEAL_SALT when RELEASE_GARBLE=1"
RELEASE_GARBLE=1 ASSET_SEAL_SALT=deadbeef RELEASE_GARBLE_SKIP_SEED=1 release_asset_seal_salt_ok || fail "RELEASE_GARBLE_SKIP_SEED=1 must allow missing asset seal salt"

echo "garble_literals_policy_gate: OK"

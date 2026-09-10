#!/usr/bin/env bash

set -euo pipefail

# Role: Static gate: MCK seed coupling on release builds.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/mck_seed_coupling_release.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail() {
  echo "mck_seed_coupling_release_gate: $*" >&2
  exit 1
}

grep -q '^AD_EVENT_PROCESSOR_LICENSE_MODE=file' deploy/installer/install.env.example \
  || fail "deploy/installer/install.env.example must default LICENSE_MODE=file"

grep -q '^AD_EVENT_PROCESSOR_LICENSE_MODE=file' .env.example \
  || fail ".env.example must default LICENSE_MODE=file"

grep -q 'AD_EVENT_PROCESSOR_LICENSE_MODE file' scripts/install/appliance_bootstrap.sh \
  || fail "appliance_bootstrap must set LICENSE_MODE=file"

if grep -Eiq 'AD_EVENT_PROCESSOR_LICENSE_MODE=(dev|development)' deploy/installer/install.env.example; then
  fail "install.env.example must not ship dev license mode"
fi

grep -q 'AD_EVENT_PROCESSOR_ASSET_SEAL_SALT' deploy/installer/install.env.example \
  || fail "install.env.example must document AD_EVENT_PROCESSOR_ASSET_SEAL_SALT"

grep -qi 'seed coupling' deploy/installer/install.env.example \
  || fail "install.env.example must note LICENSE_MODE=file seed coupling"

grep -q 'AD_EVENT_PROCESSOR_ASSET_SEAL_SALT' .env.example \
  || fail ".env.example must document AD_EVENT_PROCESSOR_ASSET_SEAL_SALT"

grep -q 'AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB' .env.example \
  || fail ".env.example must document AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB"

grep -q 'unified_filter_sealed.bin' scripts/ci/verify_release_pack.sh \
  || fail "verify_release_pack must check sealed blob requirement for garbled releases"

grep -q 'edge_sealed.bin' scripts/ci/verify_release_pack.sh \
  || fail "verify_release_pack must warn on missing edge sealed blob for garbled releases"

grep -q 'processor_ch_ingest_sealed.bin' scripts/ci/verify_release_pack.sh \
  || fail "verify_release_pack must warn on missing processor CH ingest sealed blob for garbled releases"

grep -q 'control_runtime_sealed.bin' scripts/ci/verify_release_pack.sh \
  || fail "verify_release_pack must warn on missing control runtime sealed blob for garbled releases"

[[ -f scripts/ops/bundle_enterprise_release.sh ]] \
  || fail "scripts/ops/bundle_enterprise_release.sh must exist for enterprise release packs"

grep -q 'AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB' deploy/installer/install.env.example \
  || fail "install.env.example must document AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB"

grep -q 'AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB' deploy/installer/install.env.example \
  || fail "install.env.example must document AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB"

grep -q 'AD_EVENT_PROCESSOR_CONTROL_RUNTIME_SEALED_BLOB' deploy/installer/install.env.example \
  || fail "install.env.example must document AD_EVENT_PROCESSOR_CONTROL_RUNTIME_SEALED_BLOB"

go test ./internal/config/ -run 'LicenseSeedCoupling|LicenseAssetsUnsealed' -count=1

echo "mck_seed_coupling_release_gate: OK"

#!/usr/bin/env bash

set -euo pipefail

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

go test ./internal/config/ -run 'LicenseSeedCoupling|LicenseAssetsUnsealed' -count=1

echo "mck_seed_coupling_release_gate: OK"

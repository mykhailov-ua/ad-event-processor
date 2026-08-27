#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail() {
  echo "mck_info_label_gate: $*" >&2
  exit 1
}

grep -q 'buildMCKInfoLabel' scripts/ci/release_garble.sh \
  || fail "release_garble.sh must set -X licensing.buildMCKInfoLabel"

grep -q 'MCK_INFO_LABEL' scripts/ci/release_garble.sh \
  || fail "release_garble.sh must define MCK_INFO_LABEL default"

go test ./internal/licensing/ -run 'DeriveMCK_GoldenVector|MCKDerivation_releaseLabel' -count=1

echo "mck_info_label_gate: OK"

#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: HWID string static scan.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/hwid_strings.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "hwid_strings_gate: skip (linux only)"
  exit 0
fi

OUT="${TMPDIR:-/tmp}/hwid-strings-gate-$$"
mkdir -p "$OUT"
trap 'rm -rf "$OUT"' EXIT

echo "hwid_strings_gate: building tracker probe"
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUT/tracker" ./cmd/tracker

if strings "$OUT/tracker" | rg -qi 'machine-id|product_uuid|/sys/class/dmi'; then
  echo "hwid_strings_gate: forbidden HWID path literals found in tracker binary" >&2
  strings "$OUT/tracker" | rg -i 'machine-id|product_uuid|/sys/class/dmi' || true
  exit 1
fi

echo "hwid_strings_gate: OK"

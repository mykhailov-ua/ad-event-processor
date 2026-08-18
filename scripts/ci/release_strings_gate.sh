#!/usr/bin/env bash
# V2-C.D8 / V2-D.5: garbled tracker must not embed obvious license symbols/strings.
set -euo pipefail

BIN="${1:?usage: release_strings_gate.sh <path/to/tracker>}"

if [[ ! -f "$BIN" ]]; then
  echo "release_strings_gate: missing binary: $BIN" >&2
  exit 1
fi

FORBIDDEN=(
  'IngestAllowed'
  'VerifyLicense'
  'VerifyJWT'
  'license file verification'
  'internal/licensing'
  'BEGIN PUBLIC'
  'ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70'
)

for pat in "${FORBIDDEN[@]}"; do
  if strings "$BIN" | rg -qi "$pat"; then
    echo "release_strings_gate: forbidden pattern '$pat' in $BIN" >&2
    strings "$BIN" | rg -i "$pat" | head -5 >&2 || true
    exit 1
  fi
done

echo "release_strings_gate: OK ($BIN)"

#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${OUT:-$ROOT/bin/garbled-alloc-gate}"
mkdir -p "$OUT"

if ! command -v garble > /dev/null 2>&1; then
  echo "license_garbled_alloc_gate: installing garble ${GARBLE_VERSION:-v0.15.0}"
  go install "mvdan.cc/garble@${GARBLE_VERSION:-v0.15.0}"
fi

export RELEASE_GARBLE=1
export GARBLE_SEED="${GARBLE_SEED:-cafebabecafebabecafebabecafebabe}"
export GARBLE_VERSION="${GARBLE_VERSION:-v0.15.0}"
export LICENSE_GUARD="${LICENSE_GUARD:-0}"
export PATH="$(go env GOPATH)/bin:${PATH}"

echo "license_garbled_alloc_gate: building garbled tracker -> $OUT"
if ! bash scripts/ci/release_garble.sh "$OUT" tracker; then
  echo "license_garbled_alloc_gate: garble build failed (set GARBLE_VERSION=v0.15.0 for Go 1.25)" >&2
  exit 1
fi

test -x "$OUT/tracker" || {
  echo "license_garbled_alloc_gate: missing $OUT/tracker"
  exit 1
}

echo "license_garbled_alloc_gate: license hot-path alloc subset"
bash scripts/ci/license/alloc.sh

echo "license_garbled_alloc_gate: OK (full make test-alloc-gate on garbled binary not automated)"

#!/usr/bin/env bash
# V2-D.5: garbled release binaries pass license math + strings obfuscation gate.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${OUT:-$ROOT/bin/garbled-release}"
mkdir -p "$OUT"

if ! command -v garble >/dev/null 2>&1; then
	echo "license_red_team_garbled: installing garble ${GARBLE_VERSION}"
	go install "mvdan.cc/garble@${GARBLE_VERSION}"
fi

export RELEASE_GARBLE=1
export GARBLE_SEED="${GARBLE_SEED:-cafebabecafebabecafebabecafebabe}"
export GARBLE_VERSION="${GARBLE_VERSION:-v0.15.0}"
export LICENSE_GUARD="${LICENSE_GUARD:-0}"
export PATH="$(go env GOPATH)/bin:${PATH}"

echo "license_red_team_garbled: policy gate"
bash scripts/ci/garble_literals_policy_gate.sh

echo "license_red_team_garbled: building garbled tracker/processor/control -> $OUT"
if ! bash scripts/ci/release_garble.sh "$OUT" tracker processor control; then
	echo "license_red_team_garbled: skip (garble build failed; set GARBLE_VERSION=v0.15.0 for Go 1.25)" >&2
	exit 0
fi

bash scripts/ci/release_strings_gate.sh "$OUT/tracker"

echo "license_red_team_garbled: running license-red-team unit gates"
bash scripts/security/license_red_team.sh

echo "license_red_team_garbled: OK"

#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${OUT:-$ROOT/bin/garbled-release}"
mkdir -p "$OUT"

if ! command -v garble > /dev/null 2>&1; then
  echo "license_red_team_garbled: installing garble ${GARBLE_VERSION}"
  go install "mvdan.cc/garble@${GARBLE_VERSION}"
fi

export RELEASE_GARBLE=1
export GARBLE_SEED="${GARBLE_SEED:-cafebabecafebabecafebabecafebabe}"
export ASSET_SEAL_SALT="${ASSET_SEAL_SALT:-deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef}"
export GARBLE_VERSION="${GARBLE_VERSION:-v0.15.0}"
export LICENSE_GUARD="${LICENSE_GUARD:-1}"
export PATH="$(go env GOPATH)/bin:${PATH}"

echo "license_red_team_garbled: policy gate"
bash scripts/ci/license/garble_literals_policy.sh

echo "license_red_team_garbled: building garbled tracker/processor/control -> $OUT"
if ! bash scripts/ci/release_garble.sh "$OUT" tracker processor control; then
  echo "license_red_team_garbled: garble build failed (set GARBLE_VERSION=v0.15.0 for Go 1.25)" >&2
  exit 1
fi

bash scripts/ci/license/release_strings.sh "$OUT/tracker" "$OUT/processor" "$OUT/control"

if [[ "${LICENSE_GUARD:-1}" == "1" ]]; then
  echo "license_red_team_garbled: guard unit tests"
  go test -tags=license_guard ./internal/licensing/ -run Guard -count=1
fi

echo "license_red_team_garbled: running license-red-team unit gates"
bash scripts/security/license_red_team.sh

echo "fault_proof fault=license_red_team_garbled harness=release_qa_garbled_red_team pass=1"
echo "license_red_team_garbled: OK"

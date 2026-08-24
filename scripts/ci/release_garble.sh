#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$ROOT/scripts/lib/garble_literals_policy.sh"
cd "$ROOT"

OUT_DIR="${1:-bin}"
shift || true

if [[ $# -gt 0 ]]; then
  CMDS=("$@")
else
  CMDS=(tracker processor control)
fi

USE_GARBLE="${RELEASE_GARBLE:-1}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
export CGO_ENABLED="${CGO_ENABLED:-0}"

LDFLAGS="-s -w -buildid="
if [[ -n "${ASSET_SEAL_SALT:-}" ]]; then
  LDFLAGS="${LDFLAGS} -X ad-event-processor/internal/licensing.buildAssetSealSaltHex=${ASSET_SEAL_SALT}"
fi
TAGS="timetzdata"
if [[ "$USE_GARBLE" == "1" && "${LICENSE_GUARD:-1}" == "1" ]]; then
  TAGS="${TAGS},license_guard"
fi
BUILD_FLAGS=(-trimpath -tags "$TAGS" -ldflags "$LDFLAGS")

mkdir -p "$OUT_DIR"

build_one() {
  local cmd="$1"
  local out="$OUT_DIR/$cmd"
  local literals
  literals="$(garble_literals_for_cmd "$cmd")"
  echo "release_garble: $cmd -> $out (GOOS=$GOOS GOARCH=$GOARCH garble=$USE_GARBLE literals=$literals)"
  if [[ "$USE_GARBLE" == "1" ]]; then
    if ! command -v garble > /dev/null 2>&1; then
      GARBLE_VERSION="${GARBLE_VERSION:-v0.15.0}"
      echo "release_garble: installing garble ${GARBLE_VERSION}"
      go install "mvdan.cc/garble@${GARBLE_VERSION}"
      export PATH="$(go env GOPATH)/bin:${PATH}"
    fi
    if [[ -z "${GARBLE_SEED:-}" ]]; then
      echo "release_garble: GARBLE_SEED unset — build is non-reproducible (set for CI/release)" >&2
    fi
    garble_args=(-seed="${GARBLE_SEED:-}")
    if [[ "$literals" == "1" ]]; then
      garble_args+=(-literals)
    fi
    GOOS="$GOOS" GOARCH="$GOARCH" garble "${garble_args[@]}" build "${BUILD_FLAGS[@]}" -o "$out" "./cmd/$cmd"
  else
    GOOS="$GOOS" GOARCH="$GOARCH" go build "${BUILD_FLAGS[@]}" -o "$out" "./cmd/$cmd"
  fi
}

for cmd in "${CMDS[@]}"; do
  build_one "$cmd"
done

echo "release_garble: done (${#CMDS[@]} binaries in $OUT_DIR)"

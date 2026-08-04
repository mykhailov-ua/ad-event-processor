#!/usr/bin/env bash
# Release build with optional garble obfuscation (S1.1 / S1.8).
# Usage: release_garble.sh <out_dir> [cmd ...]
# Env: RELEASE_GARBLE=1|0, GARBLE_SEED, GOOS, GOARCH, CGO_ENABLED=0
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT_DIR="${1:-bin}"
shift || true

if [[ $# -gt 0 ]]; then
	CMDS=("$@")
else
	CMDS=(tracker processor control)
fi

USE_GARBLE="${RELEASE_GARBLE:-1}"
GARBLE_LITERALS="${GARBLE_LITERALS:-0}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
export CGO_ENABLED="${CGO_ENABLED:-0}"

LDFLAGS="-s -w -buildid="
TAGS="timetzdata"
BUILD_FLAGS=(-trimpath -tags "$TAGS" -ldflags "$LDFLAGS")

mkdir -p "$OUT_DIR"

build_one() {
	local cmd="$1"
	local out="$OUT_DIR/$cmd"
	echo "release_garble: $cmd -> $out (GOOS=$GOOS GOARCH=$GOARCH garble=$USE_GARBLE)"
	if [[ "$USE_GARBLE" == "1" ]]; then
		if ! command -v garble >/dev/null 2>&1; then
			GARBLE_VERSION="${GARBLE_VERSION:-v0.14.2}"
			echo "release_garble: installing garble ${GARBLE_VERSION}"
			go install "mvdan.cc/garble@${GARBLE_VERSION}"
		fi
		if [[ -z "${GARBLE_SEED:-}" ]]; then
			echo "release_garble: GARBLE_SEED unset — build is non-reproducible (set for CI/release)" >&2
		fi
		garble_args=(-seed="${GARBLE_SEED:-}")
		if [[ "$GARBLE_LITERALS" == "1" ]]; then
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

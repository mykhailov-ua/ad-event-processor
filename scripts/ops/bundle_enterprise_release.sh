#!/usr/bin/env bash
# Role: Build per-customer garbled enterprise installer tarball (bins + sealed blobs).
# Execution context: Vendor release host with Go toolchain and garble installed.
# Env:
#   GARBLE_SEED - required garble seed (64 hex chars)
#   ASSET_SEAL_SALT or AD_EVENT_PROCESSOR_ASSET_SEAL_SALT - required MCK asset seal salt
#   AD_EVENT_PROCESSOR_RELEASE_BIN_DIR - garbled bin output dir (set by this script)
#   AD_EVENT_PROCESSOR_RELEASE_SEALED_DIR - optional sealed blob source tree (default: repo root)
#   RELEASE_GARBLE - default 1; set 0 for plain go build (still requires seeds when garbling)
# Verify:
#   GARBLE_SEED=<hex> ASSET_SEAL_SALT=<hex> bash scripts/ops/bundle_enterprise_release.sh --version 1.0.0
#   tar -tzf dist/ad-event-processor-installer-1.0.0.tar.gz | grep -E 'bin/control|edge_sealed.bin'
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$ROOT/scripts/lib/release_garble_policy.sh"
cd "$ROOT"

VERSION=""
BIN_DIR=""
SEALED_DIR=""

usage() {
  echo "usage: bundle_enterprise_release.sh --version <ver> [--bin-dir <dir>] [--sealed-dir <dir>]" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || usage
      VERSION="${2#v}"
      shift 2
      ;;
    --bin-dir)
      [[ $# -ge 2 ]] || usage
      BIN_DIR="$2"
      shift 2
      ;;
    --sealed-dir)
      [[ $# -ge 2 ]] || usage
      SEALED_DIR="$2"
      shift 2
      ;;
    -h | --help)
      usage
      ;;
    *)
      echo "bundle_enterprise_release: unknown arg: $1" >&2
      usage
      ;;
  esac
done

[[ -n "$VERSION" ]] || usage

if [[ -z "$BIN_DIR" ]]; then
  BIN_DIR="$ROOT/dist/enterprise-release-bins"
fi
if [[ -z "$SEALED_DIR" ]]; then
  SEALED_DIR="$ROOT"
fi

if [[ "${RELEASE_GARBLE:-1}" == "1" ]]; then
  if ! release_garble_seed_ok; then
    echo "bundle_enterprise_release: GARBLE_SEED required for garbled enterprise release" >&2
    exit 1
  fi
  if ! release_asset_seal_salt_ok; then
    echo "bundle_enterprise_release: ASSET_SEAL_SALT required for garbled enterprise release" >&2
    exit 1
  fi
fi

echo "bundle_enterprise_release: garble bins -> ${BIN_DIR}"
bash "$ROOT/scripts/ci/release_garble.sh" "$BIN_DIR"

echo "bundle_enterprise_release: pack installer version=${VERSION}"
AD_EVENT_PROCESSOR_RELEASE_BIN_DIR="$BIN_DIR" \
  AD_EVENT_PROCESSOR_RELEASE_SEALED_DIR="$SEALED_DIR" \
  bash "$ROOT/scripts/install/release_pack.sh" "$VERSION"

TARBALL="$ROOT/dist/ad-event-processor-installer-${VERSION}.tar.gz"
echo "bundle_enterprise_release: ${TARBALL}"

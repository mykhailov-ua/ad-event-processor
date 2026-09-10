#!/usr/bin/env bash
# Role: Build installer tarball and publish to deploy/marketing/releases for bidshard.com/get.sh.
# Execution context: Vendor operator machine with Go toolchain; not CI garble tier.
# Env:
#   VERSION (arg or pilot-YYYY-MM-DD)
#   RELEASE_PACK_BUILD_PLAIN_BINS=1 (default) for pre-GA pilot tarballs without GHCR extract
#   SKIP_DEPLOY=1 to skip marketing rsync
# Verify:
#   bash scripts/ops/publish_installer_release.sh pilot-2026-09-10
#   curl -fsI https://bidshard.com/releases/ad-event-processor-installer.tar.gz
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="pilot-$(date -u +%Y-%m-%d)"
fi
VERSION="${VERSION#v}"

RELEASE_DIR="$ROOT/deploy/marketing/releases"
export RELEASE_PACK_BUILD_PLAIN_BINS="${RELEASE_PACK_BUILD_PLAIN_BINS:-1}"

log() { printf 'publish-installer-release: %s\n' "$*"; }
die() {
  printf 'publish-installer-release: ERROR: %s\n' "$*" >&2
  exit 1
}

log "build tarball version=${VERSION}"
RELEASE_PACK_BUILD_PLAIN_BINS="$RELEASE_PACK_BUILD_PLAIN_BINS" \
  bash "$ROOT/scripts/install/release_pack.sh" "$VERSION"

TARBALL="$ROOT/dist/ad-event-processor-installer-${VERSION}.tar.gz"
[[ -f "$TARBALL" ]] || die "missing ${TARBALL}"

mkdir -p "$RELEASE_DIR"
cp "$TARBALL" "$RELEASE_DIR/ad-event-processor-installer-${VERSION}.tar.gz"
cp "$TARBALL" "$RELEASE_DIR/ad-event-processor-installer.tar.gz"
ln -sf "ad-event-processor-installer-${VERSION}.tar.gz" "$RELEASE_DIR/latest.tar.gz"

log "published local copies:"
log "  ${RELEASE_DIR}/ad-event-processor-installer.tar.gz"
log "  ${RELEASE_DIR}/ad-event-processor-installer-${VERSION}.tar.gz"

if [[ "${SKIP_DEPLOY:-0}" == "1" ]]; then
  log "SKIP_DEPLOY=1 — not syncing to VPS"
  exit 0
fi

MARKETING_DOMAIN="${MARKETING_DOMAIN:-bidshard.com}" \
  bash "$ROOT/scripts/ops/deploy_marketing.sh"

DOMAIN="${MARKETING_DOMAIN:-bidshard.com}"
log "done — curl -fsI https://${DOMAIN}/releases/ad-event-processor-installer.tar.gz"

#!/usr/bin/env bash
# Pack a minimal installer tarball (no Go source) for GitHub Releases.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(git describe --tags --always --dirty 2> /dev/null || echo dev)"
fi
VERSION="${VERSION#v}"

STAGE="$ROOT/dist/ad-event-processor-installer-stage"
OUT_DIR="$ROOT/dist"
TARBALL="$OUT_DIR/ad-event-processor-installer-${VERSION}.tar.gz"
LEGACY_TARBALL="$OUT_DIR/bidshard-installer-${VERSION}.tar.gz"

rm -rf "$STAGE"
mkdir -p "$STAGE/bidshard"

copy_tree() {
  local src="$1"
  local dst="$2"
  mkdir -p "$dst"
  cp -a "$src" "$dst/"
}

# Root compose entry
cp "$ROOT/docker-compose.yaml" "$STAGE/bidshard/"
mkdir -p "$STAGE/bidshard/deploy/compose"
cp "$ROOT/deploy/compose/docker-compose.yaml" "$STAGE/bidshard/deploy/compose/"
cp "$ROOT/deploy/compose/docker-compose.release.yaml" "$STAGE/bidshard/deploy/compose/"

cp "$ROOT/.env.example" "$STAGE/bidshard/"
mkdir -p "$STAGE/bidshard/deploy/installer"
cp "$ROOT/deploy/installer/install.env.example" "$STAGE/bidshard/deploy/installer/"
cp "$ROOT/deploy/installer/install.yaml.example" "$STAGE/bidshard/deploy/installer/"
cp "$ROOT/deploy/installer/packages.yaml" "$STAGE/bidshard/deploy/installer/"

mkdir -p "$STAGE/bidshard/deploy/geoip"
touch "$STAGE/bidshard/deploy/geoip/.gitkeep"

mkdir -p "$STAGE/bidshard/scripts/install" "$STAGE/bidshard/scripts/dev" "$STAGE/bidshard/scripts/lib" "$STAGE/bidshard/scripts/ci"
cp "$ROOT/scripts/install/ad-event-processor-install.sh" "$STAGE/bidshard/scripts/install/"
cp "$ROOT/scripts/install/preflight.sh" "$STAGE/bidshard/scripts/install/"
cp "$ROOT/scripts/install/bidshard-install.sh" "$STAGE/bidshard/scripts/install/"
cp "$ROOT/scripts/install/get.sh" "$STAGE/bidshard/scripts/install/"
mkdir -p "$STAGE/bidshard/deploy/ingress/caddy"
cp "$ROOT/deploy/ingress/caddy/Caddyfile.example" "$STAGE/bidshard/deploy/ingress/caddy/"
mkdir -p "$STAGE/bidshard/deploy/ingress/caddy/generated" "$STAGE/bidshard/deploy/ingress/certs"
touch "$STAGE/bidshard/deploy/ingress/caddy/generated/.gitkeep"
touch "$STAGE/bidshard/deploy/ingress/certs/.gitkeep"
cp "$ROOT/scripts/install/render_ingress.sh" "$STAGE/bidshard/scripts/install/"
cp "$ROOT/scripts/dev/stack.sh" "$STAGE/bidshard/scripts/dev/"
cp "$ROOT/scripts/lib/paths.sh" "$STAGE/bidshard/scripts/lib/"
cp "$ROOT/scripts/lib/installer_env.sh" "$STAGE/bidshard/scripts/lib/"
cp "$ROOT/scripts/lib/safe_paths.sh" "$STAGE/bidshard/scripts/lib/"
cp "$ROOT/scripts/ci/deps.sh" "$STAGE/bidshard/scripts/ci/"

mkdir -p "$STAGE/bidshard/bin"
echo "release_pack: building linux/amd64 ad-event-processor-install CLI..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE/bidshard/bin/ad-event-processor-install" ./cmd/installer
chmod +x "$STAGE/bidshard/bin/ad-event-processor-install"

mkdir -p "$STAGE/bidshard/deploy/vendor"
if [[ -f "$ROOT/deploy/vendor/license_public.key" ]]; then
  cp "$ROOT/deploy/vendor/license_public.key" "$STAGE/bidshard/deploy/vendor/"
fi

mkdir -p "$OUT_DIR"
tar -czf "$TARBALL" -C "$STAGE" bidshard
cp -f "$TARBALL" "$LEGACY_TARBALL"
rm -rf "$STAGE"

bash "$ROOT/scripts/ci/verify_release_pack.sh" "$TARBALL"

echo "release_pack: $TARBALL"
echo "release_pack: $LEGACY_TARBALL (legacy alias tarball)"
echo "Upload to GitHub Releases as ad-event-processor-installer.tar.gz (and bidshard-installer.tar.gz alias) for tag v${VERSION}"

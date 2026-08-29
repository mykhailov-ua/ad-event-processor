#!/usr/bin/env bash

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

rm -rf "$STAGE"
mkdir -p "$STAGE/ad-event-processor"

copy_tree() {
  local src="$1"
  local dst="$2"
  mkdir -p "$dst"
  cp -a "$src" "$dst/"
}

cp "$ROOT/docker-compose.yaml" "$STAGE/ad-event-processor/"
mkdir -p "$STAGE/ad-event-processor/deploy/compose"
cp "$ROOT/deploy/compose/docker-compose.yaml" "$STAGE/ad-event-processor/deploy/compose/"
cp "$ROOT/deploy/compose/docker-compose.release.yaml" "$STAGE/ad-event-processor/deploy/compose/"

cp "$ROOT/.env.example" "$STAGE/ad-event-processor/"
mkdir -p "$STAGE/ad-event-processor/deploy/installer"
cp "$ROOT/deploy/installer/install.env.example" "$STAGE/ad-event-processor/deploy/installer/"
cp "$ROOT/deploy/installer/install.yaml.example" "$STAGE/ad-event-processor/deploy/installer/"
cp "$ROOT/deploy/installer/packages.yaml" "$STAGE/ad-event-processor/deploy/installer/"

mkdir -p "$STAGE/ad-event-processor/deploy/geoip"
touch "$STAGE/ad-event-processor/deploy/geoip/.gitkeep"

mkdir -p "$STAGE/ad-event-processor/scripts/install" "$STAGE/ad-event-processor/scripts/dev" "$STAGE/ad-event-processor/scripts/lib" "$STAGE/ad-event-processor/scripts/ci"
cp "$ROOT/scripts/install/ad-event-processor-install.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/preflight.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/get.sh" "$STAGE/ad-event-processor/scripts/install/"
mkdir -p "$STAGE/ad-event-processor/deploy/ingress/caddy"
cp "$ROOT/deploy/ingress/caddy/Caddyfile.example" "$STAGE/ad-event-processor/deploy/ingress/caddy/"
mkdir -p "$STAGE/ad-event-processor/deploy/ingress/caddy/generated" "$STAGE/ad-event-processor/deploy/ingress/certs"
touch "$STAGE/ad-event-processor/deploy/ingress/caddy/generated/.gitkeep"
touch "$STAGE/ad-event-processor/deploy/ingress/certs/.gitkeep"
cp "$ROOT/scripts/install/render_ingress.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/dev/stack/stack.sh" "$STAGE/ad-event-processor/scripts/dev/"
cp "$ROOT/scripts/lib/paths.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/installer_env.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/safe_paths.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/ci/deps.sh" "$STAGE/ad-event-processor/scripts/ci/"

mkdir -p "$STAGE/ad-event-processor/bin"
echo "release_pack: building linux/amd64 ad-event-processor-install CLI..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE/ad-event-processor/bin/ad-event-processor-install" ./cmd/installer
chmod +x "$STAGE/ad-event-processor/bin/ad-event-processor-install"

mkdir -p "$STAGE/ad-event-processor/deploy/vendor"
if [[ -f "$ROOT/deploy/vendor/license_public.key" ]]; then
  cp "$ROOT/deploy/vendor/license_public.key" "$STAGE/ad-event-processor/deploy/vendor/"
fi

mkdir -p "$OUT_DIR"
tar -czf "$TARBALL" -C "$STAGE" ad-event-processor
rm -rf "$STAGE"

bash "$ROOT/scripts/ci/verify_release_pack.sh" "$TARBALL"

echo "release_pack: $TARBALL"
echo "Upload to GitHub Releases as ad-event-processor-installer.tar.gz for tag v${VERSION}"

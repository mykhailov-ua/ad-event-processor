#!/usr/bin/env bash
# Role: Stage release installer tarball (compose, scripts, deploy trees) under dist/.
# Execution context: Release engineer machine from git checkout; VERSION arg or git describe.
# Env knobs: VERSION (arg or git describe); output dist/ad-event-processor-installer-<version>.tar.gz.
# Verify: bash scripts/install/release_pack.sh dev && ls dist/ad-event-processor-installer-dev.tar.gz
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
cp "$ROOT/deploy/compose/docker-compose.memory-dev.yaml" "$STAGE/ad-event-processor/deploy/compose/"
mkdir -p "$STAGE/ad-event-processor/deploy/compose/scripts"
cp "$ROOT/deploy/compose/scripts/init-run-volume.sh" "$STAGE/ad-event-processor/deploy/compose/scripts/"

cp "$ROOT/.env.example" "$STAGE/ad-event-processor/"
mkdir -p "$STAGE/ad-event-processor/deploy/installer"
cp "$ROOT/deploy/installer/install.env.example" "$STAGE/ad-event-processor/deploy/installer/"
cp "$ROOT/deploy/installer/TESTER_QUICKSTART.md" "$STAGE/ad-event-processor/TESTER_QUICKSTART.md"
cp "$ROOT/deploy/installer/install.yaml.example" "$STAGE/ad-event-processor/deploy/installer/"
cp "$ROOT/deploy/installer/packages.yaml" "$STAGE/ad-event-processor/deploy/installer/"

mkdir -p "$STAGE/ad-event-processor/deploy/geoip"
touch "$STAGE/ad-event-processor/deploy/geoip/.gitkeep"

mkdir -p "$STAGE/ad-event-processor/scripts/install" "$STAGE/ad-event-processor/scripts/dev" "$STAGE/ad-event-processor/scripts/lib" "$STAGE/ad-event-processor/scripts/ci" "$STAGE/ad-event-processor/deploy/systemd"
cp "$ROOT/scripts/install/ad-event-processor-install.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/install.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/mode_systemd.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/preflight.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/install/get.sh" "$STAGE/ad-event-processor/scripts/install/"
cp "$ROOT/scripts/lib/install_cli.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/deploy/systemd/ad-event-processor-control.service" "$STAGE/ad-event-processor/deploy/systemd/"
cp "$ROOT/deploy/systemd/ad-event-processor-broker.service" "$STAGE/ad-event-processor/deploy/systemd/"
cp "$ROOT/deploy/systemd/ad-event-processor-tracker.service" "$STAGE/ad-event-processor/deploy/systemd/"
cp "$ROOT/deploy/systemd/ad-event-processor-processor.service" "$STAGE/ad-event-processor/deploy/systemd/"
cp "$ROOT/scripts/install/install.sh" "$STAGE/ad-event-processor/install.sh"
mkdir -p "$STAGE/ad-event-processor/deploy/ingress/caddy"
cp "$ROOT/deploy/ingress/caddy/Caddyfile.example" "$STAGE/ad-event-processor/deploy/ingress/caddy/"
mkdir -p "$STAGE/ad-event-processor/deploy/ingress/caddy/generated" "$STAGE/ad-event-processor/deploy/ingress/certs"
touch "$STAGE/ad-event-processor/deploy/ingress/caddy/generated/.gitkeep"
touch "$STAGE/ad-event-processor/deploy/ingress/certs/.gitkeep"
cp "$ROOT/scripts/install/render_ingress.sh" "$STAGE/ad-event-processor/scripts/install/"
mkdir -p "$STAGE/ad-event-processor/scripts/dev/stack"
cp "$ROOT/scripts/dev/stack/stack.sh" "$STAGE/ad-event-processor/scripts/dev/stack/"
mkdir -p "$STAGE/ad-event-processor/scripts/ops"
cp "$ROOT/scripts/ops/bootstrap_pg_schema.sh" "$STAGE/ad-event-processor/scripts/ops/"
cp "$ROOT/scripts/lib/paths.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/installer_env.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/safe_paths.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/ci_artifacts.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/go.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/redis_topology.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/lib/dev_bind_mounts.sh" "$STAGE/ad-event-processor/scripts/lib/"
cp "$ROOT/scripts/ci/deps.sh" "$STAGE/ad-event-processor/scripts/ci/"

mkdir -p "$STAGE/ad-event-processor/internal/ingest/migrations" \
  "$STAGE/ad-event-processor/internal/identity/migrations" \
  "$STAGE/ad-event-processor/internal/ledger/migrations"
cp -a "$ROOT/internal/ingest/migrations/." "$STAGE/ad-event-processor/internal/ingest/migrations/"
cp -a "$ROOT/internal/identity/migrations/." "$STAGE/ad-event-processor/internal/identity/migrations/"
cp -a "$ROOT/internal/ledger/migrations/." "$STAGE/ad-event-processor/internal/ledger/migrations/"

mkdir -p "$STAGE/ad-event-processor/bin"
echo "release_pack: building linux/amd64 ad-event-processor-install CLI..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE/ad-event-processor/bin/ad-event-processor-install" ./cmd/installer
chmod +x "$STAGE/ad-event-processor/bin/ad-event-processor-install"
echo "release_pack: building linux/amd64 migrate-cold-path..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE/ad-event-processor/bin/migrate-cold-path" ./cmd/migrate-cold-path
chmod +x "$STAGE/ad-event-processor/bin/migrate-cold-path"
echo "release_pack: building linux/amd64 broker..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$STAGE/ad-event-processor/bin/broker" ./cmd/broker
chmod +x "$STAGE/ad-event-processor/bin/broker"

BIN_SRC="${AD_EVENT_PROCESSOR_RELEASE_BIN_DIR:-}"
SEALED_SRC_BASE="${AD_EVENT_PROCESSOR_RELEASE_SEALED_DIR:-$ROOT}"

bundle_sealed_blob() {
  local stage_rel="$1"
  local src="$2"
  local hint="$3"
  if [[ -s "$src" ]]; then
    mkdir -p "$STAGE/ad-event-processor/$(dirname "$stage_rel")"
    install -m 0640 "$src" "$STAGE/ad-event-processor/$stage_rel"
  else
    echo "release_pack: warning: garbled release missing ${src}; ${hint}" >&2
  fi
}

if [[ -n "$BIN_SRC" ]]; then
  echo "release_pack: bundling garbled binaries from ${BIN_SRC}..."
  for bin in control tracker processor broker; do
    if [[ -x "${BIN_SRC}/${bin}" ]]; then
      install -m 0755 "${BIN_SRC}/${bin}" "$STAGE/ad-event-processor/bin/${bin}"
    else
      echo "release_pack: warning: missing ${BIN_SRC}/${bin}" >&2
    fi
  done
  bundle_sealed_blob "internal/ingestion/unified_filter_sealed.bin" \
    "${SEALED_SRC_BASE}/internal/ingestion/unified_filter_sealed.bin" \
    "LICENSE_MODE=file installs need sealed blobs (cmd/license-asset-seal)"
  bundle_sealed_blob "internal/edge/edge_sealed.bin" \
    "${SEALED_SRC_BASE}/internal/edge/edge_sealed.bin" \
    "LICENSE_MODE=file installs need edge sealed blob (cmd/license-asset-seal)"
  bundle_sealed_blob "internal/ingestion/processor_ch_ingest_sealed.bin" \
    "${SEALED_SRC_BASE}/internal/ingestion/processor_ch_ingest_sealed.bin" \
    "LICENSE_MODE=file installs need processor CH ingest sealed blob (cmd/license-asset-seal)"
  bundle_sealed_blob "internal/control/control_runtime_sealed.bin" \
    "${SEALED_SRC_BASE}/internal/control/control_runtime_sealed.bin" \
    "LICENSE_MODE=file installs need control runtime sealed blob (cmd/license-asset-seal)"
  LICENSE_PUBLIC_KEY="$ROOT/deploy/vendor/license_public.key"
  if [[ ! -f "$LICENSE_PUBLIC_KEY" ]]; then
    echo "release_pack: warning: garbled release missing ${LICENSE_PUBLIC_KEY}; appliance JWT verify needs license_public.key" >&2
  fi
elif [[ "${RELEASE_PACK_BUILD_PLAIN_BINS:-0}" == "1" ]]; then
  echo "release_pack: building plain linux/amd64 control tracker processor (pre-GA pilot channel only)..."
  for bin in control tracker processor; do
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" \
      -o "$STAGE/ad-event-processor/bin/${bin}" "./cmd/${bin}"
    chmod +x "$STAGE/ad-event-processor/bin/${bin}"
  done
fi

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

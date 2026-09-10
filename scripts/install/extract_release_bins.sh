#!/usr/bin/env bash
# Role: Extract garbled control/tracker/processor from a pilot GHCR image into a host bin dir.
# Execution context: release-images CI release-installer job after publish.
# Env knobs: IMAGE (arg1), DEST_DIR (arg2).
# Verify: bash scripts/install/extract_release_bins.sh ghcr.io/owner/ad-event-processor:v0.1.0 /tmp/bins
set -euo pipefail

IMAGE="${1:?usage: extract_release_bins.sh <image> <dest_dir>}"
DEST="${2:?usage: extract_release_bins.sh <image> <dest_dir>}"

if ! command -v docker > /dev/null 2>&1; then
  echo "extract_release_bins: docker required" >&2
  exit 1
fi

mkdir -p "$DEST"
CID="$(docker create "$IMAGE")"
trap 'docker rm -f "$CID" >/dev/null 2>&1 || true' EXIT

for bin in control tracker processor; do
  docker cp "${CID}:/${bin}" "${DEST}/${bin}"
  chmod 0755 "${DEST}/${bin}"
done

echo "extract_release_bins: wrote control tracker processor to ${DEST}"

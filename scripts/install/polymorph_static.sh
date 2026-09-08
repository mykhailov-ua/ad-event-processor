#!/usr/bin/env bash
# Build per-install static polymorph variants (attest.wasm + track_pixel.js).
# Execution context: appliance install or operator re-run; needs clang/wasm-ld or WASI SDK.
# Env: STATIC_POLYMORPH_SEED (default: install_id file or hostname hash), AD_EVENT_PROCESSOR_INSTALL_ROOT.
# Verify: bash scripts/install/polymorph_static.sh && ls -la var/static-polymorph/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
INSTALL_ROOT="${AD_EVENT_PROCESSOR_INSTALL_ROOT:-${ROOT}}"
OUT_DIR="${STATIC_POLYMORPH_DIR:-${INSTALL_ROOT}/var/static-polymorph}"
SEED="${STATIC_POLYMORPH_SEED:-}"

if [[ -z "${SEED}" && -f "${INSTALL_ROOT}/var/install_id" ]]; then
  SEED="$(tr -d '[:space:]' < "${INSTALL_ROOT}/var/install_id")"
fi
if [[ -z "${SEED}" ]]; then
  SEED="$(printf '%s' "$(hostname)-${INSTALL_ROOT}" | sha256sum | awk '{print $1}' | head -c 32)"
fi

mkdir -p "${OUT_DIR}"

WASM_ATTEST_SEED="${SEED}" \
  WASM_ATTEST_SKIP_EMBED=1 \
  WASM_ATTEST_OUT="${OUT_DIR}/attest.wasm" \
  bash "${ROOT}/scripts/build/wasm_attest.sh" --seed="${SEED}"

node "${ROOT}/web/scripts/build_track_pixel.mjs" --seed="${SEED}" --variant-out="${OUT_DIR}/track_pixel.js"

WASM_SHA="$(sha256sum "${OUT_DIR}/attest.wasm" | awk '{print $1}')"
PIXEL_SHA="$(sha256sum "${OUT_DIR}/track_pixel.js" | awk '{print $1}')"
{
  echo "# static polymorph manifest"
  echo "seed=${SEED}"
  echo "attest_wasm_sha256=${WASM_SHA}"
  echo "track_pixel_sha256=${PIXEL_SHA}"
} > "${OUT_DIR}/polymorph.manifest"

echo "polymorph_static: wrote ${OUT_DIR} seed=${SEED}"
echo "polymorph_static: attest.wasm sha256=${WASM_SHA}"
echo "polymorph_static: track_pixel.js sha256=${PIXEL_SHA}"

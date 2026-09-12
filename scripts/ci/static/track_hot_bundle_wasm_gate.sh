#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
HOT=(
  "${ROOT}/internal/track/track_pixel.js"
  "${ROOT}/internal/track/track_telemetry.js"
  "${ROOT}/internal/track/track_biometrics.js"
  "${ROOT}/web/src/static/track.js"
)
FORBIDDEN='tag\.wasm|tag-w\.js|tagW|WebAssembly'

for f in "${HOT[@]}"; do
  if [[ ! -f "${f}" ]]; then
    continue
  fi
  if rg -n "${FORBIDDEN}" "${f}" > /dev/null 2>&1; then
    echo "track_hot_bundle_wasm_gate: forbidden wasm reference in ${f}" >&2
    rg -n "${FORBIDDEN}" "${f}" >&2 || true
    exit 1
  fi
done

echo "track_hot_bundle_wasm_gate: ok"

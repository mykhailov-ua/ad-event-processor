#!/usr/bin/env bash
set -euo pipefail

# Role: Fail when browser-served pixel bundles contain legacy moderator-trigger lexicon
#   (antifraud, trackEvent, track.js paths, attest filenames, etc.).
# Execution context: pr_fast static tier.
# Verify: bash scripts/ci/static/track_pixel_surface_gate.sh

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SURFACE=(
  web/src/static/track.js
  internal/track/track_pixel.js
  internal/track/track_telemetry.js
  internal/track/track_biometrics.js
  internal/track/antifraud_telemetry.js
  internal/track/telemetry_stealth_poc.js
  internal/track/wasm_attest_loader.js
  internal/track/safe_page_hydrator.js
  internal/track/safe_page_stealth_boot.js
)

# Pipe-separated ripgrep alternation; case-sensitive literal substrings in shipped JS.
FORBIDDEN='track\.js|/_aed/track|trackEvent|trackTelemetry|trackBiometrics|trackAntifraud|antifraud-telemetry|track-telemetry|track-biometrics|telemetry-stealth|wasm-attest|attest\.wasm|aedWasmAttest|aedSensBootstrap|/track/antifraud|telemetry_mac|\bantifraud\b|\bfingerprint\b|\bbiometric'

failed=0
for f in "${SURFACE[@]}"; do
  if [[ ! -f "${f}" ]]; then
    echo "track_pixel_surface_gate: missing ${f}" >&2
    failed=1
    continue
  fi
  if rg -n "${FORBIDDEN}" "${f}" > /dev/null 2>&1; then
    echo "track_pixel_surface_gate: forbidden pixel surface literal in ${f}" >&2
    rg -n "${FORBIDDEN}" "${f}" >&2 || true
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

echo "track_pixel_surface_gate: ok"

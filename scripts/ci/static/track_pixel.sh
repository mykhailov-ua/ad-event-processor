#!/usr/bin/env bash
set -euo pipefail

# Role: Fail when go:embed track_pixel.js drifts from web/src/static/track.js source
#   or trackEvent contract vectors regress.
# Execution context: pr_fast static tier.
# Verify: bash scripts/ci/static/track_pixel.sh

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [ ! -f web/src/static/track.js ]; then
  echo "Error: missing web/src/static/track.js"
  exit 1
fi

if [ ! -f web/package.json ]; then
  echo "Error: missing web/package.json for esbuild track_pixel build"
  exit 1
fi

node web/scripts/build_track_pixel.mjs --check

(
  cd web
  node --import ./scripts/test_aliases.mjs --test --experimental-strip-types src/static/track_event.test.mjs
)

#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST="web/dist"

if [ ! -d "$DIST/src" ]; then
  echo "Error: $DIST/src missing. Run: node web/scripts/build.mjs"
  exit 1
fi

missing=0
require() {
  local path="$1"
  if [ ! -e "$DIST/$path" ]; then
    echo "Error: dist missing $path (required for go:embed admin UI)"
    missing=1
  fi
}

require "index.html"
require "login.html"
require "src/main.js"
require "src/login.js"
require "src/workers/parse_json.worker.js"
require "src/workers/report_aggregate.worker.js"
require "src/styles/tokens.css"
require "src/styles/system.css"
require "src/styles/main.css"
require "src/static/ad-event-processor-track.js"

if ! grep -q '/src/main.js' "$DIST/index.html"; then
  echo "Error: index.html must reference /src/main.js (go:embed + static routes)"
  missing=1
fi
if ! grep -q '/src/login.js' "$DIST/login.html"; then
  echo "Error: login.html must reference /src/login.js"
  missing=1
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

TRACK_SNIPPET="$DIST/src/static/ad-event-processor-track.js"
if [ -f "$TRACK_SNIPPET" ]; then
  GZ_BYTES="$(gzip -c "$TRACK_SNIPPET" | wc -c | tr -d ' ')"
  MAX_GZ=2048
  if [ "$GZ_BYTES" -gt "$MAX_GZ" ]; then
    echo "Error: ad-event-processor-track.js gzip size ${GZ_BYTES}B exceeds ${MAX_GZ}B SLA"
    exit 1
  fi
  echo "ad-event-processor-track.js gzip: ${GZ_BYTES}B (limit ${MAX_GZ}B)"
fi

echo "Web dist hygiene: OK (embed entry + workers + styles + track snippet)"

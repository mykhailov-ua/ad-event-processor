#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Web dist hygiene when web/dist or synced embed exists.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/web_dist.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST="${ADMIN_WEB_DIST_ROOT:-web/dist}"

if [ ! -d "$DIST/src" ]; then
  echo "Error: $DIST/src missing. Run: cd web && npm run build"
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
require "src/styles/app.css"
require "src/static/track.js"

if ! grep -q '/src/main.js' "$DIST/index.html"; then
  echo "Error: index.html must reference /src/main.js (go:embed + static routes)"
  missing=1
fi
if ! grep -q '/src/login.js' "$DIST/login.html"; then
  echo "Error: login.html must reference /src/login.js"
  missing=1
fi
if ! grep -q '/src/styles/app.css' "$DIST/index.html"; then
  echo "Error: index.html must reference /src/styles/app.css (tailwind build output)"
  missing=1
fi
if ! grep -q '/src/styles/app.css' "$DIST/login.html"; then
  echo "Error: login.html must reference /src/styles/app.css"
  missing=1
fi
if grep -q 'import "./.*\.css"' "$DIST/src/login.js" 2> /dev/null; then
  echo "Error: login.js must not import .css as ESM (breaks browser module load)"
  missing=1
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

TRACK_SNIPPET="$DIST/src/static/track.js"
if [ -f "$TRACK_SNIPPET" ]; then
  GZ_BYTES="$(gzip -c "$TRACK_SNIPPET" | wc -c | tr -d ' ')"
  MAX_GZ=2048
  if [ "$GZ_BYTES" -gt "$MAX_GZ" ]; then
    echo "Error: track.js gzip size ${GZ_BYTES}B exceeds ${MAX_GZ}B SLA"
    exit 1
  fi
  echo "track.js gzip: ${GZ_BYTES}B (limit ${MAX_GZ}B)"
fi

echo "Web dist hygiene: OK (embed entry + app.css + track snippet)"

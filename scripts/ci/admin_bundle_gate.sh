#!/usr/bin/env bash
# Admin bundle size gate (uncompressed bytes in web/dist/src/*.js).
# Limits (post-React Phase 0): total ≤ 1.25 MB, main.js ≤ 500 KB, each lazy chunk ≤ 180 KB.
# Still forbids chart.js / uPlot in dist/. React in dist/ is allowed after Phase 0.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST_DIR="$ROOT/web/dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: $DIST_DIR does not exist. Run: npm run build --prefix web"
  exit 1
fi

if [ ! -f "$DIST_DIR/src/main.js" ]; then
  echo "Error: $DIST_DIR/src/main.js missing (esbuild entry)"
  exit 1
fi

if [ ! -f "$DIST_DIR/src/login.js" ]; then
  echo "Error: $DIST_DIR/src/login.js missing (esbuild entry)"
  exit 1
fi

if grep -riE '\bchart\.js\b|\buplot\b' "$DIST_DIR" > /dev/null 2>&1; then
  echo "Error: forbidden chart library reference in dist/"
  exit 1
fi

# Bundled entry + chunks (gzip-ish budget via uncompressed bytes).
TOTAL_BYTES=$(find "$DIST_DIR/src" -name '*.js' -print0 | xargs -0 cat | wc -c)
MAX_BYTES=$((1280 * 1024))

echo "Bundled JS total: $TOTAL_BYTES bytes (soft limit: $MAX_BYTES bytes)"

if [ "$TOTAL_BYTES" -gt "$MAX_BYTES" ]; then
  echo "Error: admin JS bundle exceeds soft limit"
  exit 1
fi

MAIN_BYTES=$(wc -c < "$DIST_DIR/src/main.js")
MAIN_MAX=$((500 * 1024))
echo "main.js: $MAIN_BYTES bytes (soft limit: $MAIN_MAX bytes)"
if [ "$MAIN_BYTES" -gt "$MAIN_MAX" ]; then
  echo "Error: main.js entry exceeds soft limit (check code-splitting)"
  exit 1
fi

CHUNK_MAX=$((180 * 1024))
CHUNK_DIR="$DIST_DIR/src/chunks"
if [ -d "$CHUNK_DIR" ]; then
  while IFS= read -r -d '' chunk; do
  chunk_bytes=$(wc -c < "$chunk")
  chunk_name=$(basename "$chunk")
  echo "chunk $chunk_name: $chunk_bytes bytes (soft limit: $CHUNK_MAX bytes)"
  if [ "$chunk_bytes" -gt "$CHUNK_MAX" ]; then
    echo "Error: lazy chunk $chunk_name exceeds soft limit"
    exit 1
  fi
  done < <(find "$CHUNK_DIR" -name '*.js' -print0)
fi

echo "Admin bundle gate PASSED."

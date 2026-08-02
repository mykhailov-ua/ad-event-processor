#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST_DIR="$ROOT/web/dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: $DIST_DIR does not exist. Run npm run build first."
  exit 1
fi

# Check for forbidden chart.js dependency in build output
if grep -ri "chart\.js" "$DIST_DIR" > /dev/null 2>&1; then
  echo "Error: chart.js found in admin bundle! Forbidden dependency."
  exit 1
fi

# Check gzip size of main index JS chunk
MAIN_JS=$(find "$DIST_DIR/assets" -name "main-*.js" | head -n 1)

if [ -z "$MAIN_JS" ]; then
  MAIN_JS=$(find "$DIST_DIR/assets" -name "index-*.js" | head -n 1)
fi

if [ -z "$MAIN_JS" ]; then
  echo "Error: No main JS chunk main-*.js or index-*.js found in $DIST_DIR/assets"
  exit 1
fi

GZIP_SIZE_BYTES=$(gzip -c "$MAIN_JS" | wc -c)
MAX_BYTES=$((250 * 1024))

echo "Main chunk: $MAIN_JS"
echo "Gzip size: $GZIP_SIZE_BYTES bytes (limit: $MAX_BYTES bytes)"

if [ "$GZIP_SIZE_BYTES" -gt "$MAX_BYTES" ]; then
  echo "Error: Main JS bundle gzip size ($GZIP_SIZE_BYTES bytes) exceeds 250 KB limit!"
  exit 1
fi

echo "Admin bundle budget check PASSED."

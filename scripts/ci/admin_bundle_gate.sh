#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST_DIR="$ROOT/web/dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: $DIST_DIR does not exist. Run: node web/scripts/build.mjs"
  exit 1
fi

if [ ! -f "$DIST_DIR/src/main.js" ]; then
  echo "Error: $DIST_DIR/src/main.js missing"
  exit 1
fi

if grep -riE '\bchart\.js\b|\buplot\b|from ['\''\"]react' "$DIST_DIR" > /dev/null 2>&1; then
  echo "Error: forbidden chart or framework reference in dist/"
  exit 1
fi

TOTAL_BYTES=$(find "$DIST_DIR/src" -name '*.js' -print0 | xargs -0 cat | wc -c)
MAX_BYTES=$((512 * 1024))

echo "ESM src total: $TOTAL_BYTES bytes (soft limit: $MAX_BYTES bytes)"

if [ "$TOTAL_BYTES" -gt "$MAX_BYTES" ]; then
  echo "Error: admin ESM tree exceeds 512 KB uncompressed limit"
  exit 1
fi

echo "Admin bundle gate PASSED."

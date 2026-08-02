#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST="web/dist"

if [ ! -d "$DIST/assets" ]; then
  echo "Error: $DIST/assets missing. Run npm run build in web/."
  exit 1
fi

if find "$DIST/assets" -name '*.map' -print -quit | grep -q .; then
  echo "Error: sourcemap files found in production dist/assets."
  exit 1
fi

echo "Web dist hygiene: OK"

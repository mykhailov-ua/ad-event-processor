#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST="web/dist"

if [ ! -d "$DIST/src" ]; then
  echo "Error: $DIST/src missing. Run: node web/scripts/build.mjs"
  exit 1
fi

if [ ! -f "$DIST/index.html" ] || [ ! -f "$DIST/login.html" ]; then
  echo "Error: dist HTML entry files missing"
  exit 1
fi

echo "Web dist hygiene: OK"

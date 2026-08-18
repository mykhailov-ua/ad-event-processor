#!/usr/bin/env bash
# Self-serve shell must not link operator routes.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

SHELL_TS="$ROOT/web/src/components/selfserve_shell_layout.tsx"
if [ ! -f "$SHELL_TS" ]; then
  echo "Error: missing $SHELL_TS"
  exit 1
fi

FORBIDDEN=(
  "/ops"
  "/customers"
  "/shards"
  "/audit"
  "/settings/license"
)

for path in "${FORBIDDEN[@]}"; do
  if grep -q "$path" "$SHELL_TS"; then
    echo "Error: selfserve shell must not reference operator path $path"
    exit 1
  fi
done

echo "Self-serve nav gate PASSED."

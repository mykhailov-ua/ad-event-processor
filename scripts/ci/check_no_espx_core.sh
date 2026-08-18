#!/usr/bin/env bash
# Strict naming gate: zero legacy espx strings in core Go/Lua trees.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PATTERN='(\bespx\b|ESPX_|espx_)'
TARGETS=(internal pkg cmd deploy/nginx)

if rg -n "$PATTERN" "${TARGETS[@]}" --glob '*.go' --glob '*.lua' 2> /dev/null; then
  echo "check_no_espx_core: forbidden legacy naming in ${TARGETS[*]}" >&2
  exit 1
fi

echo "check_no_espx_core: ok"

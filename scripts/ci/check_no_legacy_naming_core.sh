#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PATTERN='(\bespx\b|\bESPX_[A-Z0-9_]*\b|BidShard|\bbidshard\b|X-BidShard)'
TARGETS=(internal pkg cmd deploy/nginx)

if rg -n "$PATTERN" "${TARGETS[@]}" --glob '*.go' --glob '*.lua' 2> /dev/null; then
  echo "check_no_legacy_naming_core: forbidden legacy naming in ${TARGETS[*]}" >&2
  exit 1
fi

echo "check_no_legacy_naming_core: ok"

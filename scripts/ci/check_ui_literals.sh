#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if rg -n '[А-Яа-яЁё]' web/src/views --glob '*.js' 2>/dev/null; then
  echo "Error: Cyrillic UI strings found in web/src/views (use English)."
  exit 1
fi

if rg -n '\$\$\{' web/src/views --glob '*.js' 2>/dev/null; then
  echo "Error: raw dollar template in views; use money.js helpers."
  exit 1
fi

echo "UI literal check: OK"

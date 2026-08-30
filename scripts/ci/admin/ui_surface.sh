#!/usr/bin/env bash
set -euo pipefail

# Role: Admin gate: CSS module surface ownership.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/ui_surface.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_SRC="$ROOT/web/src"
fail=0

echo "ui surface: web/src/ui must not use BEM __ modifiers"
if rg -n '__[a-z0-9-]+' "$WEB_SRC/ui" --glob '*.tsx' --glob '*.ts' --glob '*.css' 2> /dev/null; then
  echo "Error: BEM-style __ class names are banned under web/src/ui (use CSS Modules + CVA)"
  fail=1
fi

echo "ui surface: pages must not import *.module.css"
if rg -n "from ['\"].*\\.module\\.css['\"]" "$WEB_SRC/pages" 2> /dev/null; then
  echo "Error: pages import CSS modules directly; use web/src/ui components"
  fail=1
fi

echo "ui surface: pages must not use legacy btn-- BEM classes"
if rg -n 'className="[^"]*\\bbtn--' "$WEB_SRC/pages" 2> /dev/null; then
  echo "Error: pages use btn--* BEM classes; use <Button> from components/button"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi

echo "ui surface gate: OK"

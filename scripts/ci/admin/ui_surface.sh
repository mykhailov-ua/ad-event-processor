#!/usr/bin/env bash
set -euo pipefail

# Role: Admin gate: UI surface ownership (Tailwind + first-party primitives).
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: No BEM on pages; pages stay thin; no stylesheet imports on pages.
# Verify: bash scripts/ci/admin/ui_surface.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_SRC="$ROOT/web/src"
fail=0

bem_in_class() {
  local target="$1"
  rg -n \
    'className="[^"]*\b[a-z0-9-]+__[a-z0-9-]+' \
    'className=\{`[^`]*\b[a-z0-9-]+__[a-z0-9-]+' \
    'className=\{cn\([^)]*\b[a-z0-9-]+__[a-z0-9-]+' \
    "$target" --glob '*.tsx' --glob '*.ts' 2> /dev/null
}

for surface_dir in ui pages; do
  target="$WEB_SRC/$surface_dir"
  [ -d "$target" ] || continue
  echo "ui surface: web/src/$surface_dir must not use BEM __ modifiers in className"
  if bem_in_class "$target"; then
    echo "Error: BEM-style __ class names are banned under web/src/$surface_dir (use Tailwind + @/components/ui)"
    fail=1
  fi
done

echo "ui surface: pages must not import stylesheets"
if rg -n "from ['\"].*\\.(module\\.)?css['\"]" "$WEB_SRC/pages" 2> /dev/null; then
  echo "Error: pages import CSS directly; use web/src/domains components"
  fail=1
fi

echo "ui surface: pages must not use legacy btn-- BEM classes"
if rg -n 'className="[^"]*\\bbtn--' "$WEB_SRC/pages" 2> /dev/null; then
  echo "Error: pages use btn--* BEM classes; use <Button> from components/ui/button"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi

echo "ui surface gate: OK"

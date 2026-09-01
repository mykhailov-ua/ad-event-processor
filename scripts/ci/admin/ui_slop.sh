#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Admin UI anti-slop patterns.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Shipped copy, Button sizing, PageChrome badge mount rules.
# Verify: bash scripts/ci/admin/ui_slop.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -d web/src ]]; then
  echo "UI slop check: skipped (web/src absent)"
  exit 0
fi

COPY_SRC=(
  web/src/domains
  web/src/pages
  web/src/ui
  web/src/views
)
BUTTON_SRC=(
  web/src/domains
  web/src/pages
  web/src/ui
  web/src/views
)

failed=0

check_rg_dirs() {
  local desc="$1"
  shift
  local dir
  for dir in "${COPY_SRC[@]}"; do
    [ -d "$dir" ] || continue
    if rg -n "$@" "$dir" \
      --glob '*.tsx' --glob '*.ts' --glob '*.js' \
      --glob '!**/placeholder.ts' \
      --glob '!**/report_stub.ts' \
      2> /dev/null; then
      echo "Error: UI slop - $desc (under ${dir})"
      failed=1
    fi
  done
}

check_rg_dirs '"(skeleton)" in shipped page copy' '\(skeleton\)'
check_rg_dirs 'API-not-ready admission in user copy' '(?i)(not fully available yet|Skeleton shows expected)'
check_rg_dirs 'empty table blames user to "connect API"' '(?i)connect [A-Za-z ]+ API'

for dir in "${BUTTON_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n 'size="sm"' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - mixed Button size=\"sm\" is banned under ${dir}; use default --control-height-md"
    failed=1
  fi
done

for dir in "${COPY_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n 'badge=\{loading \? null' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - PageChrome badge must not mount/unmount on load; use LoadingCountBadge (under ${dir})"
    failed=1
  fi
done

# Pages and domains must call API modules, not raw fetch (cold-path boundary).
for dir in web/src/domains web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '\bfetch\(' "$dir" --glob '*.tsx' --glob '*.ts' 2> /dev/null; then
    echo "Error: UI slop - raw fetch() in ${dir}; use web/src/api/* wrappers"
    failed=1
  fi
done

# Raw HTML tables in domain UI are banned; use shadcn Table.
for dir in web/src/domains web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '<(table|thead|tbody)\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - raw <table> in ${dir}; use @/components/ui/table"
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "Remediation: .cursor/rules/ui.mdc anti-slop section."
  exit 1
fi

echo "UI slop check: OK"

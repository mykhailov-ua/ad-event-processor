#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Admin UI anti-slop patterns.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Shipped copy, control height contract, PageChrome badge mount rules.
# Verify: bash scripts/ci/admin/ui_slop.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -d web/src ]]; then
  echo "UI slop check: skipped (web/src absent)"
  exit 0
fi

COPY_SRC=(
  web/src/domains
  web/src/shell
  web/src/pages
  web/src/views
)
BUTTON_SRC=(
  web/src/domains
  web/src/shell
  web/src/pages
  web/src/views
)
CONTROL_SRC=(
  web/src/components
  web/src/domains
  web/src/shell
  web/src/pages
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
    echo "Error: UI slop - mixed Button size=\"sm\" is banned under ${dir}; use default Button (h-control)"
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

# Manual h-* overrides on form controls drift from --control-height; use web/src/lib/control_size.ts shells.
for dir in "${CONTROL_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n 'SelectTrigger[^>]*className="[^"]*\bh-[789]\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - SelectTrigger manual h-7/h-8/h-9 under ${dir}; use default SelectTrigger"
    failed=1
  fi
  if rg -n '<Input[^>]*className="[^"]*\bh-[789]\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - Input manual h-7/h-8/h-9 under ${dir}; use default Input"
    failed=1
  fi
  if rg -n '<Button[^>]*className="[^"]*\bh-[789]\b' "$dir" --glob '*.tsx' \
    --glob '!**/dashboard_surface_radius_demo.tsx' \
    --glob '!**/dashboard_time_series_plot.tsx' \
    2> /dev/null; then
    echo "Error: UI slop - Button manual h-7/h-8/h-9 under ${dir}; use default Button"
    failed=1
  fi
done

# Pages and UI domains must call API modules, not raw fetch (cold-path boundary).
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '\bfetch\(' "$dir" --glob '*.tsx' --glob '*.ts' 2> /dev/null; then
    echo "Error: UI slop - raw fetch() in ${dir}; use web/src/api/* wrappers"
    failed=1
  fi
done

# Raw HTML tables in domain UI are banned; use @/components/ui/table or DirectoryTable.
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '<(table|thead|tbody)\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - raw <table> in ${dir}; use @/components/ui/table"
    failed=1
  fi
done

# Raw admin-btn on interactive elements bypasses Button contract (ui.mdc).
for dir in "${BUTTON_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n '<button[^>]*className="[^"]*admin-btn' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - raw <button className=\"admin-btn\"> under ${dir}; use @/components/ui/button"
    failed=1
  fi
  if rg -n "<button[^>]*className=\\{[^}]*'admin-btn" "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - raw <button> with admin-btn cn() under ${dir}; use @/components/ui/button"
    failed=1
  fi
  if rg -n 'className="admin-btn' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - className=\"admin-btn\" under ${dir}; use @/components/ui/button (multiline <button> included)"
    failed=1
  fi
  if rg -n "className='admin-btn" "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - className='admin-btn' under ${dir}; use @/components/ui/button"
    failed=1
  fi
  if rg -n 'className=\{[^}]*admin-btn' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - admin-btn in className expression under ${dir}; use @/components/ui/button"
    failed=1
  fi
done

if rg -n '@radix-ui|class-variance-authority|ui\.shadcn\.com' web/src web/package.json 2> /dev/null; then
  echo "Error: UI slop - shadcn/Radix dependencies are banned; use web/src/components/ui first-party primitives"
  failed=1
fi

if [ -f web/components.json ]; then
  echo "Error: UI slop - web/components.json (shadcn CLI) must not exist"
  failed=1
fi

# Legacy ui-table-frame wrapper is retired; use DirectoryTable.
if rg -n 'ui-table-frame' web/src --glob '*.tsx' 2> /dev/null; then
  echo "Error: UI slop - ui-table-frame is retired; use DirectoryTable from web/src/shell/directory_table.tsx"
  failed=1
fi

# Hand-rolled text inputs bypass Input height contract (web/src/lib/control_size.ts).
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '<input[^>]*className="[^"]*h-8 w-full rounded-md border' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - hand-rolled text <input> under ${dir}; use @/components/ui/input + Label"
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "Remediation: .cursor/rules/ui.mdc anti-slop section; control height: web/src/lib/control_size.ts"
  exit 1
fi

echo "UI slop check: OK"

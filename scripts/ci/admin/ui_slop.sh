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
  if rg -n 'size="icon"|size=\{[^}]*['\''"]icon['\''"]' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - Button size=\"icon\" is banned under ${dir}; use default Button + className size-7 p-0"
    failed=1
  fi
done

for dir in "${COPY_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n 'truncate[^"\n`]*tabular-nums|tabular-nums[^"\n`]*truncate' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - truncate with tabular-nums on same node under ${dir}; widen column (VL-18)"
    failed=1
  fi
  if rg -n '<(h[1-6])\b[^>]*\btruncate\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - truncate on heading under ${dir}; use whitespace-nowrap (VL-18)"
    failed=1
  fi
done

# Column drag highlight must not use React state on dragover (repaints tbody).
while IFS= read -r drag_file; do
  [ -n "$drag_file" ] || continue
  if rg -n 'useState\([^)]*[Dd]rag' "$drag_file" 2> /dev/null \
    && rg -n 'onDragOver' "$drag_file" 2> /dev/null; then
    echo "Error: UI slop - useState drag highlight with onDragOver in ${drag_file}; use ref classList toggle (P4)"
    failed=1
  fi
done < <(rg -l 'onDragOver' web/src/domains web/src/shell --glob '*.tsx' 2> /dev/null || true)

if [ -d web/e2e ]; then
  if rg -n 'or\(.*empty' web/e2e --glob '*.spec.js' 2> /dev/null; then
    echo "Error: UI slop - deprecated table.or(empty) pattern in web/e2e specs (FE2 / L1s)"
    failed=1
  fi
fi

for dir in "${COPY_SRC[@]}"; do
  [ -d "$dir" ] || continue
  if rg -n 'badge=\{loading \? null' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - PageChrome badge must not mount/unmount on load; use LoadingCountBadge (under ${dir})"
    failed=1
  fi
done

# Manual h-* overrides on form controls drift from --control-height; use admin_kit.controlHeight / default primitives.
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

# Hand-rolled text inputs bypass Input height contract (admin_kit.controlHeight).
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '<input[^>]*className="[^"]*h-8 w-full rounded-md border' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - hand-rolled text <input> under ${dir}; use @/components/ui/input + Label"
    failed=1
  fi
done

if rg -n 'className="[^"]*admin-[^"]*![-a-z]' web/src/domains web/src/shell web/src/pages --glob '*.tsx' 2> /dev/null; then
  echo "Error: UI slop - Tailwind important utility on same className as admin-* BEM; see frontend-slop.mdc Tailwind vs global CSS collision"
  failed=1
fi
if rg -n "className=\\{cn\\([^)]*admin-[^)]*!" web/src/domains web/src/shell web/src/pages --glob '*.tsx' 2> /dev/null; then
  echo "Error: UI slop - Tailwind important utility in cn() with admin-* BEM; see frontend-slop.mdc Tailwind vs global CSS collision"
  failed=1
fi

# Legacy BEM hooks in app.css surface tokens (flat ui-*-segment only).
if [ -f web/src/styles/app.css ]; then
  if rg -n '\.ui-[a-z0-9-]+(__|--)[a-z0-9-]+' web/src/styles/app.css 2> /dev/null; then
    echo "Error: UI slop - BEM __ or -- modifier in web/src/styles/app.css; use flat ui-*-segment class names"
    failed=1
  fi
fi

for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '\!bg-' "$dir" --glob '*.tsx' --glob '*.ts' 2> /dev/null; then
    echo "Error: UI slop - Tailwind !bg-* row/cell overrides under ${dir}; use data-row-accent / app.css pin rules"
    failed=1
  fi
done

# Spacing and typography drift in domains (canonical: web/src/lib/admin_spacing.ts).
for dir in web/src/domains web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n '\bgap-[5678]\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - gap-5+ banned under ${dir}; use adminSpacing.gap (xs|sm|md|lg|xl) from admin_kit"
    failed=1
  fi
  if rg -n '\btext-sm\b|\btext-base\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - text-sm/text-base banned under ${dir}; use adminTypography.body / sectionTitle"
    failed=1
  fi
  if rg -n '\btext-xs\b' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - text-xs banned under ${dir}; use adminTypography.badge or monoData"
    failed=1
  fi
  if rg -n 'text-\[(8|9|1[0-9])px\]' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - arbitrary text-[Npx] banned under ${dir}; use adminTypography.* from admin_spacing.ts"
    failed=1
  fi
done

# Legacy admin-* BEM hooks in class strings (text-ui-* typography tokens are allowed).
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n "'admin-[a-z][a-z0-9-]*'|\"admin-[a-z][a-z0-9-]*\"" "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - legacy admin-* BEM class hook under ${dir}; use Tailwind utilities in className"
    failed=1
  fi
  if rg -n "'[a-z0-9-]+--[a-z0-9-]+'|\"[a-z0-9-]+--[a-z0-9-]+\"" "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - BEM -- modifier class under ${dir}; use Tailwind utilities in className"
    failed=1
  fi
  if rg -n "'[a-z0-9-]+__[a-z0-9-]+'|\"[a-z0-9-]+__[a-z0-9-]+\"" "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - BEM __ element class under ${dir}; use Tailwind utilities in className"
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "Remediation: .cursor/rules/ui.mdc (**Spacing and typography**); web/src/lib/admin_spacing.ts; frontend-slop.mdc layout contract"
  exit 1
fi

echo "ui slop: overlay React loop guards (OV-*)"
OVERLAY_UI=(
  web/src/components/ui/select.tsx
  web/src/components/ui/popover.tsx
  web/src/components/ui/dropdown-menu.tsx
)
for overlay_file in "${OVERLAY_UI[@]}"; do
  [ -f "$overlay_file" ] || continue
  if ! rg -n 'mergeOverlayPosition' "$overlay_file" 2> /dev/null; then
    echo "Error: UI slop - ${overlay_file} must use mergeOverlayPosition (OV-3)"
    failed=1
  fi
  if ! rg -n 'useMemo\(' "$overlay_file" 2> /dev/null; then
    echo "Error: UI slop - ${overlay_file} overlay context must use useMemo (OV-1)"
    failed=1
  fi
  if rg -n '\[ctx,' "$overlay_file" 2> /dev/null; then
    echo "Error: UI slop - ${overlay_file} bans [ctx, in effect deps (OV-2)"
    failed=1
  fi
done

echo "ui slop: route error boundary reset (RB-E*)"
if ! rg -n 'AppRouteErrorBoundary' web/src/app_shell.tsx 2> /dev/null; then
  echo "Error: UI slop - app_shell must wrap Outlet in AppRouteErrorBoundary (RB-E1)"
  failed=1
fi

echo "ui slop: UUID helper on HTTP paths (UUID-*)"
for dir in web/src/domains web/src/api web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n 'crypto\.randomUUID' "$dir" --glob '*.ts' --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - crypto.randomUUID under ${dir}; use newRandomUuid from @/lib/uuid (UUID-1)"
    failed=1
  fi
done

echo "ui slop: CSP inline swatch ban (CSP-1)"
for dir in web/src/domains web/src/shell web/src/pages; do
  [ -d "$dir" ] || continue
  if rg -n 'style=\{\{\s*backgroundColor' "$dir" --glob '*.tsx' 2> /dev/null; then
    echo "Error: UI slop - inline backgroundColor style under ${dir}; use CHART_SWATCH_CLASS / Tailwind (CSP-1)"
    failed=1
  fi
done

echo "ui slop: react-day-picker global CSS (CSP-2)"
if [ -f web/src/main.tsx ]; then
  if ! rg -n "react-day-picker/style\.css" web/src/main.tsx 2> /dev/null; then
    echo "Error: UI slop - main.tsx must import react-day-picker/style.css (CSP-2)"
    failed=1
  fi
fi

if [ "$failed" -ne 0 ]; then
  echo "Remediation: frontend-slop.mdc OV-*/RB-E*/CSP-*/UUID-*; overlay_position_state.ts; app_error_boundary.tsx"
  exit 1
fi

# Class A mobile nav: Sheet dock with dedicated nav scroll (frontend-slop.mdc overlays).
echo "ui slop: mobile nav sheet contract"
if ! rg -n 'AppMobileNavSheet' web/src/app_shell.tsx 2> /dev/null; then
  echo "Error: UI slop - app_shell must render AppMobileNavSheet for mobile navigation (VL-13c)"
  failed=1
fi
if ! rg -n 'overflow-hidden' web/src/shell/app_sidebar.tsx 2> /dev/null; then
  echo "Error: UI slop - AppMobileNavSheet SheetContent must use overflow-hidden dock layout"
  failed=1
fi
if rg -n 'DropdownMenu' web/src/app_shell.tsx web/src/shell/tracker_shell_header.tsx 2> /dev/null; then
  echo "Error: UI slop - shell nav must not use DropdownMenu; use AppSidebar / AppMobileNavSheet (VL-13c)"
  failed=1
fi

echo "ui slop: DropdownMenu default list cap"
if ! rg -n 'max-h-60' web/src/components/ui/dropdown_menu_scroll.ts 2> /dev/null; then
  echo "Error: UI slop - DropdownMenuContent scroll wrapper must cap height with max-h-60 (VL-13b)"
  failed=1
fi
if ! rg -n 'stopPropagation' web/src/components/ui/dropdown-menu.tsx 2> /dev/null; then
  echo "Error: UI slop - DropdownMenu scroll body must stop wheel propagation (VL-13b)"
  failed=1
fi

if [ "$failed" -ne 0 ]; then
  echo "Remediation: .cursor/rules/ui.mdc (**Spacing and typography**); web/src/lib/admin_spacing.ts; frontend-slop.mdc layout contract"
  exit 1
fi

echo "UI slop check: OK"

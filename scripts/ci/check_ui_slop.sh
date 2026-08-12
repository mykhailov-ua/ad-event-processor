#!/usr/bin/env bash
# Fail on user-visible "fake shipped" copy in production admin routes.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

VIEWS="web/src/views"
failed=0

check_rg() {
  local desc="$1"
  shift
  if rg -n "$@" "$VIEWS" \
    --glob '*.ts' --glob '*.js' \
    --glob '!**/placeholder.ts' \
    --glob '!**/report_stub.ts' \
    2>/dev/null; then
    echo "Error: UI slop — $desc"
    failed=1
  fi
}

check_rg '"(skeleton)" in shipped page copy' '\(skeleton\)'
check_rg 'API-not-ready admission in user copy' '(?i)(not fully available yet|Skeleton shows expected)'
check_rg 'empty table blames user to "connect API"' '(?i)connect [A-Za-z ]+ API'

if [ "$failed" -ne 0 ]; then
  echo "Remediation: web/DESIGN.md §11; backlog .cursor/MILESTONE.md §1.0 (anti-slop)."
  exit 1
fi

echo "UI slop check: OK"

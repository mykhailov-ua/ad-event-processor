#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: forbid new client-side sort over list/table payloads (cold-path Class A).
# Execution context: CI via admin/web.sh when web/src exists.
# Invariants/contracts enforced: No new [...items].sort / sortRows / sortCampaignListItemsClient outside allowlist.
# Verify: bash scripts/ci/admin/client_list_sort.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -d web/src ]]; then
  echo "client_list_sort_gate: skipped (web/src absent)"
  exit 0
fi

ALLOWLIST="$SCRIPTS/ci/admin/allowlists/client_list_sort_paths.txt"
if [[ ! -f "$ALLOWLIST" ]]; then
  echo "Error: missing allowlist $ALLOWLIST"
  exit 1
fi

mapfile -t ALLOWED_PATHS < <(grep -v '^[[:space:]]*#' "$ALLOWLIST" | grep -v '^[[:space:]]*$' || true)

is_allowlisted() {
  local rel="$1"
  local entry
  for entry in "${ALLOWED_PATHS[@]}"; do
    if [[ "$rel" == "$entry" ]]; then
      return 0
    fi
  done
  return 1
}

SCAN_DIRS=(web/src/pages web/src/domains)
PATTERNS=(
  'sortCampaignListItemsClient'
  'function sortRows\('
  '\[\.\.\.(items|rows)\]\.sort\('
)

failed=0

for dir in "${SCAN_DIRS[@]}"; do
  [[ -d "$dir" ]] || continue
  for pattern in "${PATTERNS[@]}"; do
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      file="${line%%:*}"
      if is_allowlisted "$file"; then
        continue
      fi
      echo "Error: client list sort pattern in non-allowlisted file: $line"
      failed=1
    done < <(rg -n "$pattern" "$dir" --glob '*.ts' --glob '*.tsx' 2>/dev/null || true)
  done
done

if [[ "$failed" -ne 0 ]]; then
  echo "Remediation: server sort via query params; shrink allowlist in $ALLOWLIST"
  exit 1
fi

echo "Client list sort gate: OK (${#ALLOWED_PATHS[@]} allowlisted paths)"

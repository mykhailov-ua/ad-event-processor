#!/usr/bin/env bash
set -euo pipefail

# Role: Admin gate: UI literal ban scan.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/ui_literals.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if rg -P -n '\p{Cyrillic}' web/src/ui web/src/helpers web/src/pages web/src/components --glob '*.{js,ts,tsx}' 2> /dev/null; then
  echo "Error: Cyrillic UI strings found (use English)."
  exit 1
fi

if rg -n '\$\$\{' web/src/views web/src/ui --glob '*.{js,ts}' 2> /dev/null; then
  echo "Error: raw dollar template in views; use money.js helpers."
  exit 1
fi

if rg -n '[^\x00-\x7E]' web/src web/e2e --glob '*.{js,ts,tsx}' 2> /dev/null; then
  echo "Error: non-ASCII characters in web UI or e2e (use ASCII only)."
  exit 1
fi

echo "UI literal check: OK"

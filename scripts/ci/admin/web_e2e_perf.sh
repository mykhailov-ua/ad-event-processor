#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web Playwright perf tier (wall-clock budgets on Hot surfaces).
# Execution context: Nightly or operator with stack on :8188; not pr_fast default.
# Verify: ADMIN_WEB_E2E_PERF=1 bash scripts/ci/admin/web_e2e_perf.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

E2E_DIR="$ROOT/web/e2e"

if [ ! -d "$E2E_DIR/perf" ]; then
  echo "admin web e2e perf: skipped (web/e2e/perf absent)"
  exit 0
fi

if [ "${ADMIN_E2E_SKIP:-0}" = "1" ]; then
  echo "admin web e2e perf: skipped (ADMIN_E2E_SKIP=1)"
  exit 0
fi

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "admin web e2e perf: npm ci"
  (cd "$E2E_DIR" && npm ci)
  (cd "$E2E_DIR" && npx playwright install chromium)
fi

echo "admin web e2e perf: Playwright perf specs (serial; wall-clock budgets)"
(
  cd "$E2E_DIR" && npx playwright test perf/
)

echo "admin web e2e perf PASSED"

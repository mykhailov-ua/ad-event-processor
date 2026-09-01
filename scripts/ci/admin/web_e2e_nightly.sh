#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web nightly e2e (full Playwright matrix).
# Execution context: Nightly tier; not invoked from web.sh (too heavy for pr_fast).
# Requires: control plane stack listening on :8188 (CONTROL_URL default http://127.0.0.1:8188).
# Verify: ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/test/admin_stack_e2e.sh
# Or (stack already up): ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

E2E_DIR="$ROOT/web/e2e"

if [ ! -d "$E2E_DIR" ]; then
  echo "admin web e2e nightly: skipped (web/e2e absent)"
  exit 0
fi

if [ "${ADMIN_E2E_SKIP:-0}" = "1" ]; then
  echo "admin web e2e nightly: skipped (ADMIN_E2E_SKIP=1)"
  exit 0
fi

SPEC_COUNT="$(find "$E2E_DIR" -maxdepth 1 -name '*.spec.js' | wc -l | tr -d ' ')"
echo "admin web e2e nightly: ${SPEC_COUNT} spec file(s)"

if [ "$SPEC_COUNT" -eq 0 ]; then
  echo "admin web e2e nightly: ERROR: no *.spec.js under web/e2e" >&2
  exit 1
fi

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "admin web e2e nightly: npm ci"
  (cd "$E2E_DIR" && npm ci)
fi

echo "admin web e2e nightly: playwright full matrix (stack on :8188)"
(
  cd "$E2E_DIR" && npx playwright test
)

echo "admin web e2e nightly PASSED (${SPEC_COUNT} specs)"

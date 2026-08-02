#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_DIR="$ROOT/web"

if [ ! -f "$WEB_DIR/package.json" ]; then
  echo "Error: $WEB_DIR/package.json not found"
  exit 1
fi

cd "$WEB_DIR"

echo "== admin: npm ci =="
npm ci

echo "== admin: web deps =="
node "$WEB_DIR/scripts/ci/check_web_deps.mjs"

echo "== admin: ui literals =="
bash "$SCRIPTS/ci/check_ui_literals.sh"

echo "== admin: lint =="
npm run lint

echo "== admin: vitest =="
npm test

echo "== admin: build =="
npm run build

echo "== admin: dist hygiene =="
bash "$SCRIPTS/ci/check_web_dist.sh"

echo "== admin: security literals =="
bash "$SCRIPTS/ci/check_web_security.sh"

echo "== admin: bundle gate =="
bash "$SCRIPTS/ci/admin_bundle_gate.sh"

echo "== admin: lighthouse checklist =="
bash "$SCRIPTS/ci/admin_lighthouse_checklist.sh"

echo "== admin: confirm registry audit =="
bash "$SCRIPTS/ci/confirm_registry_audit.sh"

if [ "${ADMIN_SKIP_E2E:-}" = "1" ]; then
  echo "== admin: e2e skipped (ADMIN_SKIP_E2E=1) =="
else
  echo "== admin: playwright e2e =="
  npx playwright install chromium
  npm run test:e2e
fi

echo "Admin web checks PASSED."

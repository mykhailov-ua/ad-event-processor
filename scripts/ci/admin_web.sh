#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_DIR="$ROOT/web"

if [ ! -f "$WEB_DIR/scripts/build.mjs" ]; then
  echo "Error: $WEB_DIR/scripts/build.mjs not found"
  exit 1
fi

if [ ! -d "$WEB_DIR/node_modules/esbuild" ]; then
  echo "admin: npm ci (web)"
  (cd "$WEB_DIR" && npm ci)
fi

echo "admin: typecheck (tsc --noEmit + tests)"
(cd "$WEB_DIR" && npm run typecheck)

echo "admin: build (esbuild)"
(cd "$WEB_DIR" && node scripts/build.mjs)

echo "admin: ui literals"
bash "$SCRIPTS/ci/check_ui_literals.sh"

echo "admin: ui anti-slop"
bash "$SCRIPTS/ci/check_ui_slop.sh"

echo "admin: jsdoc exports (legacy .js; .ts uses tsc)"
bash "$SCRIPTS/ci/check_jsdoc.sh"

echo "admin: dist hygiene"
bash "$SCRIPTS/ci/check_web_dist.sh"

echo "admin: go:embed static routes"
go test ./internal/controlplane/ -run 'TestAdminStaticRoutes|TestInjectAdminBoot' -count=1

echo "admin: live report routes"
bash "$SCRIPTS/ci/report_live_routes_gate.sh"

echo "admin: security literals"
bash "$SCRIPTS/ci/check_web_security.sh"

echo "admin: forbidden chart libs / size"
bash "$SCRIPTS/ci/admin_bundle_gate.sh"
bash "$SCRIPTS/ci/selfserve_nav_gate.sh"

echo "admin: micro-benchmarks"
bash "$SCRIPTS/ci/web_bench.sh"

echo "admin: unit tests"
(cd "$WEB_DIR" && npm run test)

echo "admin: confirm registry audit"
bash "$SCRIPTS/ci/confirm_registry_audit.sh"

if [ "${ADMIN_SKIP_E2E:-1}" = "1" ]; then
  echo "admin: e2e skipped (ADMIN_SKIP_E2E=1, default)"
else
  E2E_DIR="$WEB_DIR/e2e"
  if [ ! -f "$E2E_DIR/package.json" ]; then
    echo "Error: e2e requires $E2E_DIR/package.json (set ADMIN_SKIP_E2E=1 to skip)"
    exit 1
  fi
  echo "admin: playwright e2e"
  (cd "$WEB_DIR" && node scripts/build.mjs)
  set +e
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium && npm run test:e2e)
  e2e_status=$?
  set -e
  if [ "$e2e_status" -ne 0 ]; then
    echo "admin: playwright e2e FAILED (exit $e2e_status)"
    exit "$e2e_status"
  fi
fi

echo "Admin web checks PASSED."

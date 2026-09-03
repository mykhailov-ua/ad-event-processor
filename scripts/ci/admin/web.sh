#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web gate: stub-only when no dist/embed; full hygiene when dist or synced embed exists.
# Execution context: CI via pr_fast.
# Invariants/contracts enforced: go test TestAdminStaticRoutes and TestInjectAdminBoot must pass.
# Verify: bash scripts/ci/admin/web.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

DIST="web/dist"
STUB="internal/controlplane/admin_static_stub"

has_dist() {
  [ -d "$DIST/src" ]
}

has_fresh_embed() {
  [ -f "$STUB/src/styles/app.css" ] && [ -f "$STUB/src/main.js" ]
}

run_static_tests() {
  go test ./internal/controlplane/ -run 'TestAdminStaticRoutes|TestInjectAdminBoot' -count=1
}

if has_dist || has_fresh_embed; then
  echo "admin: dist/embed hygiene + typecheck + static routes"
  if [ -d web/src ]; then
    bash "$SCRIPTS/ci/admin/ui_slop.sh"
    bash "$SCRIPTS/ci/admin/ui_surface.sh"
    bash "$SCRIPTS/ci/admin/ui_literals.sh"
    bash "$SCRIPTS/ci/admin/web_security.sh"
    bash "$SCRIPTS/ci/admin/live_routes.sh"
    bash "$SCRIPTS/ci/admin/client_list_sort.sh"
  fi
  if has_dist; then
    bash "$SCRIPTS/ci/admin/web_dist.sh"
  else
    ADMIN_WEB_DIST_ROOT="$STUB" bash "$SCRIPTS/ci/admin/web_dist.sh"
  fi
  if [ -f web/package.json ]; then
    (cd web && npm run typecheck)
  fi
  if [ "${ADMIN_WEB_E2E_SMOKE:-0}" = "1" ] && [ -d web/e2e ]; then
    bash "$SCRIPTS/ci/admin/web_e2e_smoke.sh"
  fi
  run_static_tests
  echo "Admin web checks PASSED (dist/embed + typecheck)."
else
  echo "admin: static stub embed + boot inject"
  if [ -d web/src ]; then
    bash "$SCRIPTS/ci/admin/ui_slop.sh"
    bash "$SCRIPTS/ci/admin/ui_surface.sh"
    bash "$SCRIPTS/ci/admin/ui_literals.sh"
    bash "$SCRIPTS/ci/admin/web_security.sh"
    bash "$SCRIPTS/ci/admin/live_routes.sh"
    bash "$SCRIPTS/ci/admin/client_list_sort.sh"
  fi
  run_static_tests
  echo "Admin web checks PASSED (stub only)."
fi

echo "admin: set ADMIN_WEB_E2E_SMOKE=1 to run smoke playwright bundle"
# Full Playwright matrix (nightly only): ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh

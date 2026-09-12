#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web smoke e2e (curated Playwright bundle).
# Execution context: Optional tier when ADMIN_WEB_E2E_SMOKE=1 via web.sh.
# Not handler wiring proof: route mount + subset of L1/L3; see web_e2e_keep_proof.sh for KEEP matrix.
# Verify: ADMIN_WEB_E2E_SMOKE=1 bash scripts/ci/admin/web.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

E2E_DIR="$ROOT/web/e2e"

if [ ! -d "$E2E_DIR" ]; then
  echo "admin web e2e smoke: skipped (web/e2e absent)"
  exit 0
fi

if [ "${ADMIN_E2E_SKIP:-0}" = "1" ]; then
  echo "admin web e2e smoke: skipped (ADMIN_E2E_SKIP=1)"
  exit 0
fi

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "admin web e2e smoke: npm ci"
  (cd "$E2E_DIR" && npm ci)
fi

echo "admin web e2e smoke: playwright bundle (mount smoke; not full L1/L3 matrix)"
(
  cd "$E2E_DIR" && npx playwright test --workers=1 \
    smoke_matrix.spec.js \
    login.spec.js \
    bootstrap.spec.js \
    sidebar.spec.js \
    customers_list.spec.js \
    customer_detail_billing.spec.js \
    campaigns_filters.spec.js \
    campaign_editor.spec.js \
    export_hub.spec.js \
    settings.spec.js \
    audit.spec.js \
    ops_console.spec.js \
    permission_route_audit.spec.js \
    integrations_hub.spec.js \
    integrations_postbacks_health.spec.js \
    team_invite.spec.js \
    team.spec.js \
    freeze_redirect.spec.js \
    click_log.spec.js
)

echo "admin web e2e smoke PASSED"

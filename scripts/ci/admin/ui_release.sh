#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Admin UI release orchestrator.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/ui_release.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_DIR="$ROOT/web"
E2E_DIR="$WEB_DIR/e2e"

if [[ ! -f "$WEB_DIR/scripts/build.mjs" ]]; then
  echo "admin ui release: skipped (web/ absent; OpenAPI contracts remain under api/openapi/)"
  bash "$SCRIPTS/ci/admin/web.sh"
  exit 0
fi

skip_e2e="${ADMIN_RELEASE_SKIP_E2E:-1}"

echo "admin ui release: CI gates (e2e skipped by default)"
ADMIN_SKIP_E2E=1 bash "$SCRIPTS/ci/admin/web.sh"

# E2E off by default; ADMIN_RELEASE_SKIP_E2E=0 runs Playwright bundle
if [ "$skip_e2e" = "1" ]; then
  echo "admin ui release: e2e skipped (set ADMIN_RELEASE_SKIP_E2E=0 to run Playwright bundle)"
  echo "admin ui release gate PASSED (typecheck + unit + slop gates only)."
  exit 0
fi

echo "admin ui release: e2e running (ADMIN_RELEASE_SKIP_E2E=0)"

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "admin ui release: playwright install"
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium)
fi

echo "admin ui release: playwright bundle"
(cd "$WEB_DIR" && node scripts/build.mjs)
(
  cd "$E2E_DIR" && npx playwright test \
    ops_compliance.spec.js \
    ops_dlq.spec.js \
    billing_exports.spec.js \
    billing_disputes.spec.js \
    cpa_held_out.spec.js \
    integrations_postbacks.spec.js \
    report_actions.spec.js \
    invoice_deliveries.spec.js \
    ops_shards_catchup.spec.js \
    ops_billing_invariant.spec.js \
    campaign_config_edit.spec.js \
    campaign_integration.spec.js \
    customer_tax_profile.spec.js \
    rtb_deals.spec.js \
    audit_export.spec.js \
    ops_recon.spec.js \
    rtb_integration.spec.js \
    customer_balance_export.spec.js \
    settings_license.spec.js \
    sidebar.spec.js \
    billing_selfserve.spec.js \
    admin_regression.spec.js
)

echo "admin ui release gate PASSED (CI + e2e)."

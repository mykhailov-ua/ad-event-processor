#!/usr/bin/env bash
# Milestone §1.5 acceptance gate — CI regression + optional milestone e2e bundle.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_DIR="$ROOT/web"
E2E_DIR="$WEB_DIR/e2e"

echo "== milestone §1.5: admin web CI (e2e skipped by default) =="
ADMIN_SKIP_E2E=1 bash "$SCRIPTS/ci/admin_web.sh"

if [ "${MILESTONE_SKIP_E2E:-1}" = "1" ]; then
  echo "== milestone §1.5: e2e skipped (set MILESTONE_SKIP_E2E=0 to run Playwright bundle) =="
  echo "Milestone UI gate PASSED (CI only — typecheck + unit + slop gates; not full Playwright)."
  exit 0
fi

echo "== milestone §1.5: e2e running (MILESTONE_SKIP_E2E=0) =="

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "== milestone §1.5: playwright install =="
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium)
fi

echo "== milestone §1.5: milestone e2e bundle =="
(cd "$WEB_DIR" && node scripts/build.mjs)
(cd "$E2E_DIR" && npx playwright test \
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
  milestone_regression.spec.js \
)

echo "Milestone UI gate PASSED (CI + e2e)."

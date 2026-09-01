#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web smoke e2e (curated Playwright bundle).
# Execution context: Optional tier when ADMIN_WEB_E2E_SMOKE=1 via web.sh.
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

echo "admin web e2e smoke: playwright bundle"
(
  cd "$E2E_DIR" && npx playwright test \
    smoke_matrix.spec.js \
    login.spec.js \
    bootstrap.spec.js \
    sidebar.spec.js \
    customers_list.spec.js \
    customer_detail_billing.spec.js \
    campaigns_filters.spec.js \
    campaign_editor.spec.js \
    campaign_editor_deep.spec.js \
    campaign_integrations.spec.js \
    campaign_publish.spec.js \
    campaign_diff.spec.js \
    settings_patch.spec.js \
    settings_apply.spec.js \
    fraud_hub.spec.js \
    fraud_presets.spec.js \
    fraud_labels_write.spec.js \
    fraud_overrides_write.spec.js \
    team_invite.spec.js
)

echo "admin web e2e smoke PASSED"

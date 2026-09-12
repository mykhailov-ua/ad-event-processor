#!/usr/bin/env bash
set -euo pipefail

# Role: KEEP route L1 + L3 Playwright proof bundle (one read + one error contract per core CP route).
# Execution context: Nightly tier via web_e2e_nightly.sh; not pr_fast wiring proof.
# Requires: control plane stack on :8188 (ADMIN_E2E_BASE_URL default http://127.0.0.1:8188).
# Verify: ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

E2E_DIR="$ROOT/web/e2e"

if [ ! -d "$E2E_DIR" ]; then
  echo "admin web e2e keep proof: skipped (web/e2e absent)"
  exit 0
fi

if [ "${ADMIN_E2E_SKIP:-0}" = "1" ]; then
  echo "admin web e2e keep proof: skipped (ADMIN_E2E_SKIP=1)"
  exit 0
fi

if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "admin web e2e keep proof: npm ci"
  (cd "$E2E_DIR" && npm ci)
fi

KEEP_PROOF_SPECS=(
  customers_list.spec.js
  customer_detail_billing.spec.js
  campaigns_filters.spec.js
  campaign_editor.spec.js
  export_hub.spec.js
  settings.spec.js
  audit.spec.js
  ops_console.spec.js
  permission_route_audit.spec.js
  integrations_hub.spec.js
  integrations_postbacks_health.spec.js
  integrations_postbacks_save.spec.js
  team_invite.spec.js
  team.spec.js
  freeze_redirect.spec.js
  click_log.spec.js
)

echo "admin web e2e keep proof: ${#KEEP_PROOF_SPECS[@]} spec file(s)"
(
  cd "$E2E_DIR" && npx playwright test --workers=1 "${KEEP_PROOF_SPECS[@]}"
)

echo "admin web e2e keep proof PASSED"

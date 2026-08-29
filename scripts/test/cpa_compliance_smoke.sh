#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"

if [[ "${CPA_COMPLIANCE_SKIP_ADMIN:-0}" != "1" ]]; then
  echo "cpa compliance: route gap"
  bash scripts/ci/admin/cpa_route_gap.sh

  echo "cpa compliance: admin web gates"
  bash scripts/ci/admin/web.sh
  bash scripts/test/billing_export_smoke.sh
fi

echo "cpa compliance: held-out go unit (no Docker)"
go test ./internal/controlplane/ -run 'DLQInbox|DlqInbox|CPA_HeldOut' -short -count=1
go test ./internal/controlplane/ -run 'DLQInbox|ConsentProofs' -count=1

if [[ "${CPA_HELD_OUT_INTEGRATION:-0}" = "1" ]]; then
  echo "cpa compliance: held-out go integration (Docker testcontainers)"
  go test ./internal/controlplane/ -run 'CPA_HeldOut' -count=1
fi

if [[ "${CPA_COMPLIANCE_SKIP_E2E:-${CPA_HELD_OUT_SKIP_E2E:-0}}" = "1" ]]; then
  echo "cpa_compliance_smoke: OK (e2e skipped - set CPA_COMPLIANCE_SKIP_E2E=0 for Playwright)"
  exit 0
fi

E2E_DIR="$ROOT/web/e2e"
if [[ ! -f "$E2E_DIR/package.json" ]]; then
  echo "cpa_compliance_smoke: OK (e2e skipped; web/ absent)"
  exit 0
fi
if [[ ! -d "$E2E_DIR/node_modules/@playwright/test" ]]; then
  echo "cpa compliance: playwright install"
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium)
fi

echo "cpa compliance: web build for preview"
(cd "$ROOT/web" && node scripts/build.mjs)

echo "cpa compliance: playwright held-out suite"
(cd "$E2E_DIR" && npx playwright test cpa_held_out.spec.js --grep 'ops consolidation')

echo "cpa compliance: playwright ops compliance"
(cd "$E2E_DIR" && npx playwright test ops_compliance.spec.js)

echo "cpa_compliance_smoke: OK"

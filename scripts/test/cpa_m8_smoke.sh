#!/usr/bin/env bash
# CPA-M8 smoke — unified DLQ, consent browser, ops compliance e2e.
# Run from repo root: bash scripts/test/cpa_m8_smoke.sh
# Skip Playwright: CPA_M8_SKIP_E2E=1 bash scripts/test/cpa_m8_smoke.sh
# Integration (Docker): CPA_M8_INTEGRATION=1 bash scripts/test/cpa_m8_smoke.sh
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"

echo "== CPA-M8: route gap =="
bash scripts/ci/cpa_route_gap_gate.sh

echo "== CPA-M8: go unit (no Docker) =="
go test ./internal/controlplane/ -run 'DLQInbox|DlqInbox|CPA_M8' -short -count=1
go test ./internal/controlplane/ -run 'DLQInbox|ConsentProofs' -count=1

if [ "${CPA_M8_INTEGRATION:-0}" = "1" ]; then
  echo "== CPA-M8: go integration (Docker testcontainers) =="
  go test ./internal/controlplane/ -run 'CPA_M8' -count=1
fi

if [ "${CPA_M8_SKIP_E2E:-0}" = "1" ]; then
  echo "cpa_m8_smoke: OK (e2e skipped — CPA_M8_SKIP_E2E=1)"
  exit 0
fi

E2E_DIR="$ROOT/web/e2e"
if [ ! -f "$E2E_DIR/package.json" ]; then
  echo "error: missing $E2E_DIR/package.json" >&2
  exit 1
fi
if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
  echo "== CPA-M8: playwright install =="
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium)
fi

echo "== CPA-M8: web build for preview =="
(cd "$ROOT/web" && node scripts/build.mjs)

echo "== CPA-M8: playwright held-out M8 =="
# Must use @playwright/test from web/e2e — not bare `playwright` from npx cache.
(cd "$E2E_DIR" && npx playwright test cpa_held_out.spec.js --grep 'CPA-M8')

echo "== CPA-M8: playwright ops compliance =="
(cd "$E2E_DIR" && npx playwright test ops_compliance.spec.js)

echo "cpa_m8_smoke: OK"

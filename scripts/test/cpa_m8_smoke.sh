#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"

echo "== cpa held-out: route gap =="
bash scripts/ci/cpa_route_gap_gate.sh

echo "== cpa held-out: go unit (no Docker) =="
go test ./internal/controlplane/ -run 'DLQInbox|DlqInbox|CPA_M8' -short -count=1
go test ./internal/controlplane/ -run 'DLQInbox|ConsentProofs' -count=1

if [ "${CPA_M8_INTEGRATION:-0}" = "1" ]; then
  echo "== cpa held-out: go integration (Docker testcontainers) =="
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
  echo "== cpa held-out: playwright install =="
  (cd "$E2E_DIR" && npm ci && npx playwright install chromium)
fi

echo "== cpa held-out: web build for preview =="
(cd "$ROOT/web" && node scripts/build.mjs)

echo "== cpa held-out: playwright held-out suite =="

(cd "$E2E_DIR" && npx playwright test cpa_held_out.spec.js --grep 'ops consolidation')

echo "== cpa held-out: playwright ops compliance =="
(cd "$E2E_DIR" && npx playwright test ops_compliance.spec.js)

echo "cpa_m8_smoke: OK"

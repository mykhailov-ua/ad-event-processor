#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "admin release: confirm registry audit"
bash "$SCRIPTS/ci/confirm_registry_audit.sh"

echo "admin release: web security literals"
bash "$SCRIPTS/ci/check_web_security.sh"

echo "admin release: govulncheck"
bash "$SCRIPTS/ci/govulncheck.sh"

echo "admin release: lighthouse INP checklist"
bash "$SCRIPTS/ci/admin_lighthouse_checklist.sh"

echo "admin release: pre-tag UI e2e (manual; not run in this gate)"
echo "Before release tag, run once locally with Playwright installed:"
echo "  ADMIN_RELEASE_SKIP_E2E=0 bash scripts/ci/admin_ui_release_gate.sh"

echo "Admin release security gate PASSED."

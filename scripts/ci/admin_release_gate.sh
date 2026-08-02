#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "== admin release: confirm registry audit =="
bash "$SCRIPTS/ci/confirm_registry_audit.sh"

echo "== admin release: web security literals =="
bash "$SCRIPTS/ci/check_web_security.sh"

echo "== admin release: govulncheck =="
bash "$SCRIPTS/ci/govulncheck.sh"

echo "== admin release: lighthouse INP checklist =="
bash "$SCRIPTS/ci/admin_lighthouse_checklist.sh"

echo "Admin release security gate PASSED."

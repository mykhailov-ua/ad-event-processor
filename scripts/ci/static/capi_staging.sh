#!/usr/bin/env bash

set -euo pipefail

# Role: Static gate: CAPI staging route and handler contract.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/capi_staging.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SCRIPT="$SCRIPTS/test/capi/meta_staging.sh"
TEST="$SCRIPTS/test/capi/meta_staging_test.sh"
GOOGLE_SCRIPT="$SCRIPTS/test/capi/google_offline_staging.sh"
GOOGLE_TEST="$SCRIPTS/test/capi/google_offline_staging_test.sh"

[[ -x "$SCRIPT" ]] || chmod +x "$SCRIPT"
[[ -x "$TEST" ]] || chmod +x "$TEST"
[[ -x "$GOOGLE_SCRIPT" ]] || chmod +x "$GOOGLE_SCRIPT"
[[ -x "$GOOGLE_TEST" ]] || chmod +x "$GOOGLE_TEST"

CAPI_STAGING_DRY_RUN=1 bash "$SCRIPT"
bash "$TEST"
GOOGLE_OFFLINE_STAGING_DRY_RUN=1 bash "$GOOGLE_SCRIPT"
bash "$GOOGLE_TEST"

echo "capi_staging_gate: OK"

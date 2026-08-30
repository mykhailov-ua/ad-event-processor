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

[[ -x "$SCRIPT" ]] || chmod +x "$SCRIPT"
[[ -x "$TEST" ]] || chmod +x "$TEST"

CAPI_STAGING_DRY_RUN=1 bash "$SCRIPT"
bash "$TEST"

echo "capi_staging_gate: OK"

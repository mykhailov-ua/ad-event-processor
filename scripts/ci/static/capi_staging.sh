#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SCRIPT="$SCRIPTS/test/capi/meta_staging.sh"
TEST="$SCRIPTS/test/capi/meta_staging_test.sh"

[[ -x "$SCRIPT" ]] || chmod +x "$SCRIPT"
[[ -x "$TEST" ]] || chmod +x "$TEST"

CAPI_STAGING_DRY_RUN=1 bash "$SCRIPT"
bash "$TEST"

echo "capi_staging_gate: OK"

#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Admin web microbench.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/web_bench.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

node --expose-gc --experimental-strip-types --import "$ROOT/web/scripts/register_ts.mjs" "$ROOT/web/scripts/bench/run.mjs"
node --expose-gc --experimental-strip-types --import "$ROOT/web/scripts/register_ts.mjs" "$ROOT/web/scripts/bench/hot_path.mjs"

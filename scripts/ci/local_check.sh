#!/usr/bin/env bash
set -euo pipefail

# Role: Operator pre-push check: pr_fast plus full module build.
# Execution context: Operator machine only; heavier than pr_fast alone.
# Invariants/contracts enforced: Fails on first pr_fast or make build error.
# Verify: bash scripts/ci/local_check.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/pr_fast.sh"
make build

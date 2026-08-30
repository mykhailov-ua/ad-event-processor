#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: rangeint and modernize static rules.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/go_modernize.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "lint_go_modernize_gate: rangeint static scan..."
bash "$SCRIPTS/ci/static/go_rangeint.sh"

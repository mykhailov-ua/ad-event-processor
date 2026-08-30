#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: actionlint on GitHub workflows.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/workflows.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

ACTIONLINT_IMAGE="${ACTIONLINT_IMAGE:-rhysd/actionlint:latest}"

if [[ ! -d "$ROOT/.github/workflows" ]]; then
  echo "lint_workflows_gate: skip (.github/workflows missing)"
  exit 0
fi

run_actionlint() {
  if command -v actionlint > /dev/null 2>&1; then
    actionlint -color
    return $?
  fi
  if command -v docker > /dev/null 2>&1; then
    docker run --rm -v "$ROOT:/work" -w /work "$ACTIONLINT_IMAGE"
    return $?
  fi
  echo "lint_workflows_gate: need actionlint or docker" >&2
  return 1
}

echo "lint_workflows_gate: actionlint (.github/workflows)..."
run_actionlint
echo "lint_workflows_gate: OK"

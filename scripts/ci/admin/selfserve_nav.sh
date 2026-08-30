#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: Self-serve nav contract.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/selfserve_nav.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

SHELL_TS="$ROOT/web/src/components/selfserve_shell_layout.tsx"
if [ ! -f "$SHELL_TS" ]; then
  echo "Self-serve nav gate: skipped (selfserve shell removed)"
  exit 0
fi

FORBIDDEN=(
  "/ops"
  "/customers"
  "/shards"
  "/audit"
  "/settings/license"
)

for path in "${FORBIDDEN[@]}"; do
  if grep -q "$path" "$SHELL_TS"; then
    echo "Error: selfserve shell must not reference operator path $path"
    exit 1
  fi
done

echo "Self-serve nav gate PASSED."

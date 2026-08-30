#!/usr/bin/env bash
set -euo pipefail

# Role: Config file validation before gates.
# Execution context: CI pr_fast entry.
# Invariants/contracts enforced: Invalid env/compose configs fail early.
# Verify: bash scripts/ci/validate_configs.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

safe_validate_codegen_configs
echo "codegen configs OK"

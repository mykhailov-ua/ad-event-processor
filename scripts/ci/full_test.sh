#!/usr/bin/env bash
set -euo pipefail

# Role: Main-branch full test orchestrator: gen, shard control, integration, and fault tiers.
# Execution context: CI Gate main-full-test on push to main; needs Docker and testcontainers.
# Invariants/contracts enforced: Runs make gen before integration/fault; does not re-invoke pr_fast.
# Verify: bash scripts/ci/full_test.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
bash "$SCRIPTS/ci/naming/state_shard_control.sh"
bash "$SCRIPTS/ci/compose_profile_check.sh"
make lint
make test-integration
make test-fault

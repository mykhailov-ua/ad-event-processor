#!/usr/bin/env bash
set -euo pipefail

# Role: Merge-integration gate: codegen then testcontainers integration suite.
# Execution context: CI merge-integration job; operator with Docker for local reproduction.
# Invariants/contracts enforced: make gen must succeed before make test-integration.
# Verify: bash scripts/ci/integration_test.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
make test-integration

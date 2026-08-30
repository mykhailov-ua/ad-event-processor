#!/usr/bin/env bash
set -euo pipefail

# Role: Resilience fault test wrapper; delegates to scripts/test/run_resilience.sh.
# Execution context: CI Gate main-resilience on main; operator for local fault proofs.
# Invariants/contracts enforced: Downstream log must contain required fault_proof lines (see resilience_fault_gates.sh).
# Verify: bash scripts/fault/run.sh
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/test/run_resilience.sh" "$@"

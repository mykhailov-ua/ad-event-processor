#!/usr/bin/env bash

set -euo pipefail

# Role: Fault/resilience: Grep fault_proof lines from resilience log.
# Execution context: CI main-resilience or operator fault tier; needs Docker for compose drills.
# Invariants/contracts enforced: Success logs fault_proof fault=<name>; resilience_fault_gates.sh greps required proofs.
# Verify: bash scripts/fault/resilience_fault_gates.sh
LOG="${1:-${RESILIENCE_LOG:-/tmp/ad-event-processor-resilience.log}}"
if [ ! -f "$LOG" ]; then
  echo "FAIL: resilience log not found: $LOG" >&2
  exit 1
fi

REQUIRED_FAULTS=(
  noscript_storm
  spool_disk_block
  sentinel_promotion_isolation
)

echo "resilience_fault_gates: checking fault_proof lines in $LOG"
for fault in "${REQUIRED_FAULTS[@]}"; do
  if ! grep -q "fault_proof fault=${fault} " "$LOG"; then
    echo "FAIL: missing fault_proof fault=${fault}" >&2
    exit 1
  fi
  echo "  OK fault=${fault}"
done

echo "resilience_fault_gates: OK"

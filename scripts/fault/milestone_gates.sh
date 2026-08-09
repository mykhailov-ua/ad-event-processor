#!/usr/bin/env bash
# Milestone §5 fault gates (E/F/G). Requires resilience log from scripts/fault/run.sh.
set -euo pipefail

LOG="${1:-${RESILIENCE_LOG:-/tmp/espx-resilience.log}}"
if [ ! -f "$LOG" ]; then
	echo "FAIL: resilience log not found: $LOG" >&2
	exit 1
fi

REQUIRED_FAULTS=(
	noscript_storm
	spool_disk_block
	sentinel_promotion_isolation
)

echo "milestone_gates: checking fault_proof lines in $LOG"
for fault in "${REQUIRED_FAULTS[@]}"; do
	if ! grep -q "fault_proof fault=${fault} " "$LOG"; then
		echo "FAIL: missing fault_proof fault=${fault}" >&2
		exit 1
	fi
	echo "  OK fault=${fault}"
done

echo "milestone_gates: E/F/G passed"

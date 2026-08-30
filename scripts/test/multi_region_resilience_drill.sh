#!/usr/bin/env bash
set -euo pipefail

# Role: Multi-region fault subset (region proxy, lease, quorum); counts mr_ fault_proof lines in controlplane tests.
# Execution context: Operator or nightly; testcontainers Docker required.
# Invariants/contracts enforced: MIN_MR_PROOFS (default 12); named mr_* faults must appear in log.
# Verify: bash scripts/test/multi_region_resilience_drill.sh
# Env: MR_RESILIENCE_DRILL_LOG, RESILIENCE_MIN_PROOFS_MR
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${MR_RESILIENCE_DRILL_LOG:-/tmp/ad-event-processor-mr-resilience-drill.log}"
MIN_MR_PROOFS="${RESILIENCE_MIN_PROOFS_MR:-12}"

echo "Multi-region resilience drill"
echo "SLA targets:"
echo "  - Regional proxy failover RTO < 120 s"
echo "  - AssertBudgetInvariant after lease partition heal"
echo "  - Zero duplicate global apply (proposal_rows=1)"
echo ""
echo "Running MR fault subset (testcontainers; requires Docker)..."
echo ""

go test -count=1 -v -timeout 25m \
  -run 'TestFault_(Score|OperationLease|Region|Proxy|Disk|Global|Quorum)' \
  ./internal/controlplane/... \
  2>&1 | tee "$LOG"

MR_PROOFS="$(grep -c 'fault_proof fault=mr_' "$LOG" || true)"
echo ""
echo "mr fault_proof lines: $MR_PROOFS (min $MIN_MR_PROOFS)"
test "$MR_PROOFS" -ge "$MIN_MR_PROOFS"

for fault in \
  mr_score_cold_node \
  mr_lease_pg_partition \
  mr_lease_standby_executor \
  mr_lease_dual_cas \
  mr_quorum_book \
  mr_global_pg_partition; do
  if ! grep -q "fault_proof fault=${fault}" "$LOG"; then
    echo "FAIL: missing fault_proof fault=${fault}"
    exit 1
  fi
done

echo ""
echo "OK: multi-region fault drill proofs present"
echo "Next: follow docs/DEVELOPMENT.md Multi-region resilience drill (90 min operator checklist)"

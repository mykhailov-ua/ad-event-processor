#!/usr/bin/env bash
# M7.4 multi-region game day drill — operator dry-run for regional proxy failover and lease heal.
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

LOG="${MR_GAME_DAY_LOG:-/tmp/espx-mr-game-day.log}"
MIN_MR_PROOFS="${CHAOS_MIN_PROOFS_MR:-12}"

echo "=== M7 Multi-Region Game Day Drill ==="
echo "SLA targets:"
echo "  - Regional proxy failover RTO < 120 s"
echo "  - AssertBudgetInvariant after lease partition heal"
echo "  - Zero duplicate global apply (proposal_rows=1)"
echo ""
echo "Running MR chaos subset (testcontainers; requires Docker)..."
echo ""

go test -count=1 -v -timeout 25m \
	-run 'TestChaos_(Score|OperationLease|Region|Proxy|Disk|Global|Quorum)' \
	./internal/management/... \
	2>&1 | tee "$LOG"

MR_PROOFS="$(grep -c 'chaos_proof fault=mr_' "$LOG" || true)"
echo ""
echo "mr chaos_proof lines: $MR_PROOFS (min $MIN_MR_PROOFS)"
test "$MR_PROOFS" -ge "$MIN_MR_PROOFS"

for fault in \
	mr_score_cold_node \
	mr_lease_pg_partition \
	mr_lease_ghost_executor \
	mr_lease_dual_cas \
	mr_quorum_book \
	mr_global_pg_partition; do
	if ! grep -q "chaos_proof fault=${fault}" "$LOG"; then
		echo "FAIL: missing chaos_proof fault=${fault}"
		exit 1
	fi
done

echo ""
echo "OK: multi-region game day chaos proofs present"
echo "Next: follow docs/DEVELOPMENT.md § Multi-region game day (90 min operator checklist)"

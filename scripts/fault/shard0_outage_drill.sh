#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "=== Shard-0 outage drill ==="
echo "Expected during redis-0 outage:"
echo "  - Shards 1-3 track continue (p99 within SLA)"
echo "  - Shard-0 campaigns return 503 shard_unavailable (or registry_stale for unknown IDs)"
echo "  - Management outbox UPDATE_SETTINGS stays PENDING until redis-0 recovers"
echo "  - AssertBudgetInvariant holds on surviving shards"
echo ""

go test -count=1 -v -run 'TestFault_Shard0Outage' -timeout 15m ./tests/resilience/ 2>&1 | tee /tmp/espx-shard0-outage.log

if grep -q 'fault_proof fault=shard_0_outage' /tmp/espx-shard0-outage.log || \
   grep -q 'fault_proof fault=shard0_survival_shards_1_3' /tmp/espx-shard0-outage.log; then
  echo "OK: shard-0 survival fault_proof present"
  exit 0
fi
echo "FAIL: missing fault_proof for shard-0 survival"
exit 1

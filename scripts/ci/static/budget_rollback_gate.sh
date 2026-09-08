#!/usr/bin/env bash
set -euo pipefail

# Role: Post-debit rollback idempotency and ingest admission/rollback holdouts.
# Execution context: CI merge-pr-fast via pr_fast.sh; race tier is merge-race-short (full ./internal/... -race).
# Invariants/contracts enforced: budget-rollback.lua SET NX guard; at most one Redis refund per click_id;
#   ingest RollbackDebit wiring on local-quanta and publish-failure paths.
# Verify:
# bash scripts/ci/static/budget_rollback_gate.sh
# go test -race -short -count=1 -run TestBudgetRollback_ ./internal/filter/unified/
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "budget_rollback_gate: unified filter idempotent rollback (miniredis)..."
go test -short -count=1 -timeout=120s \
  ./internal/filter/unified/ \
  -run 'TestRollbackDebit_eligibleLuaDebitMisroute|TestBudgetRollback_'

echo "budget_rollback_gate: ingest admission + post-debit rollback holdouts..."
go test -short -count=1 -timeout=180s \
  ./internal/ingest/ \
  -run 'TestUnifiedFilter_RollbackDebit|TestUnifiedFilter_SetDeferStreamToProducer|TestPublishAcceptedOrRollback|TestStreamProducerAdmission|TestStreamProducerReserve|TestPublishAcceptedTrack_holdout|TestLocalQuantaPendingDebit_publishFail|TestEligibleLuaDebit_publishFail_redisRollback'

echo "budget_rollback_gate: OK"

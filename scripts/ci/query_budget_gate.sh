#!/usr/bin/env bash
set -euo pipefail

# Role: merge-integration query-budget gate; sublinear PG query growth vs input N on bulk admin routes.
# Execution context: CI merge-integration via integration_test.sh; local reproduction with Docker.
# Invariants/contracts enforced: TestQueryBudget_Sublinear* must pass without -short.
# Verify: bash scripts/ci/query_budget_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen

echo "query-budget-gate: helper unit checks..."
go test -count=1 -timeout=2m ./internal/database/ -run 'TestAssertQueryBudgetSublinear'

echo "query-budget-gate: HTTP sublinear bulk endpoints..."
go test -count=1 -timeout=15m ./internal/controlplane/ -run 'TestQueryBudget_Sublinear'

echo "query-budget-gate: import batch upsert sublinear..."
go test -count=1 -timeout=15m ./internal/campaign/importexport/ -run 'TestBatchUpsertLandersByNameURL_queryBudgetSublinear'

echo "query-budget-gate: OK"

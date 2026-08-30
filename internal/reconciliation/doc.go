// Package reconciliation owns spend recon windows, budget snapshot compare, global spend sync, and adjust applier.
//
// Role:
//   - ReconWorker (worker.go): periodic PG/Redis/CH drift checks; tick also runs governance.QuotaRepairRunner.
//   - ReconService (recon.go): ReconcileWindow compares CH spend vs PG/Redis; enqueues reconciliation adjust outbox.
//   - AdjustApplier (adjust_applier.go): applies RECONCILIATION_ADJUST outbox payloads (ledger + Redis delta).
//   - GlobalSpendReconciler (global_spend.go): batches regional spend into PG/Redis under sync_idempotency.
//   - ListRuns (runs.go): management/payment recon run history for opsadmin reader.
//   - RTBCHStats (rtb_export.go): ClickHouse stats helper for RTB reconcile reads.
//
// Topology:
//   - controlplane/governance_bridge.go: Service implements reconciliation.Host (ReconInfraHost, ReconOpsHost, ReconRepairHost).
//   - controlplane/outbox_bridge.go: outbox worker calls NewAdjustApplier(s).Apply for adjust events.
//   - ReconWorker started from serve_workers.go; GlobalSpendReconciler wired in serve.go when multi-region enabled.
//
// Invariants:
//   - Adjust idempotency hash recon_adjust_outbox_{outboxEventID}; Redis marker recon:redis_applied:{id}.
//   - parseReconciliationAdjustPayload requires campaign_id, customer_id, and non-zero ledger or redis delta.
//   - Budget snapshot auto-adjust capped by autoAdjustChunkMicro; drift above chunk skips auto-adjust (alerts ops).
//   - Stale unresolved discrepancies alert via Host.Alerter without mutating live budgets until adjust outbox applies.
//
// Forbidden:
//   - Synchronous recon on /track accept path.
//   - Adjust apply without outbox idempotency hash or PG commit before Redis retry.
//
// Verify:
//
//	go test ./internal/reconciliation/ -short -count=1
//	go test ./internal/reconciliation/ -short -run TestRecon -count=1
//	go test ./internal/reconciliation/ -short -run TestGlobalSpendReconciler -count=1
//	go test ./internal/reconciliation/ -short -run TestListRuns -count=1
//	go test ./internal/reconciliation/ -short -run TestReconcileBudgetSnapshot -count=1
//
// Integration tier (TestReconcileWindow_* skips under -short):
// make test-integration
package reconciliation

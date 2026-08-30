// Package reconciliation owns spend recon windows, budget snapshot compare, global spend sync, and adjust applier.
//
// Role:
//   - ReconWorker runs periodic PG/Redis/CH drift checks; ReconService applies reconciliation adjust outbox payloads.
//   - GlobalSpendReconciler syncs aggregate spend; list endpoints expose recon run history for opsadmin reader.
//
// Topology:
//   - Wired via governance_bridge.go and reconciliation bridge aliases; depends on governance for quota repair tick.
//   - Host port supplies Redis eval, ledger writes, and CH stats readers.
//
// Invariants:
//   - Adjust outbox idempotency key recon_adjust_outbox_{id}; Redis applied marker prevents double apply.
//   - Payload requires campaign_id, customer_id, and non-zero ledger or redis delta.
//   - Stale recon runs flagged without mutating live budgets until operator approves adjust.
//
// Forbidden:
//   - Synchronous recon on /track accept path.
//   - Adjust apply without outbox idempotency hash.
//
// Verify:
//
//	go test ./internal/reconciliation/ -short -count=1
//	go test ./internal/reconciliation/ -short -run TestRecon -count=1
package reconciliation

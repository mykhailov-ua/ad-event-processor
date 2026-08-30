// Package settlement drains payment.payment_outbox into balance_ledger via PaymentSettlement.
//
// Role:
//   - OutboxWorker: FOR UPDATE SKIP LOCKED claim, call in-process SettlementAPI, mark processed/failed.
//   - SettlementLedgerClient for direct settlement batches when configured.
//   - CryptoHoldWorker releases held crypto deposits after confirmation policy.
//   - ReconService periodic PG vs provider reconciliation with optional ops alerts.
//
// Topology:
//   - Started from payment.Module.StartWorkers alongside webhook HTTP server.
//   - controlplane.NewSettlementHandler implements domain.PaymentSettlement for monolith wiring.
//
// Invariants:
//   - Settlement idempotency via ledger_idempotency_key in outbox payload.
//   - Stale PROCESSING reclaim on payment outbox (same 1 min policy as public outbox).
//   - PostSettlementMarkHook test seam only; production marks rows in worker transaction.
//
// Defaults and limits:
//   - Outbox poll interval 100 ms when idle; immediate re-poll when batch processed.
//   - ProcessOutbox batch limit 100 rows per iteration.
//
// Forbidden:
//   - Synchronous settlement in Stripe HTTP handler (webhook package).
//
// Verify:
//
//	go test ./internal/payment/settlement/ -short -count=1
package settlement

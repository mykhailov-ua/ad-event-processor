// Package settlement drains payment.payment_outbox into balance_ledger via PaymentSettlement.
//
// Role:
//   - OutboxWorker: FOR UPDATE SKIP LOCKED claim on payment.payment_outbox, call in-process SettlementAPI, mark processed/failed.
//   - SettlementLedgerClient for direct settlement batches when configured.
//   - CryptoHoldWorker releases held crypto deposits after confirmation policy.
//   - ReconService periodic PG vs provider reconciliation with optional ops alerts.
//
// Topology:
//   - Started from payment.Module.StartWorkers alongside webhook HTTP on PAYMENT_WEBHOOK_PORT 8187.
//   - controlplane.NewSettlementHandler (billing_bridge.go) implements domain.PaymentSettlement for monolith wiring.
//   - serve.go injects SettlementAPI into payment.Module.SetSettlementAPI before StartWorkers.
//
// Settlement outbox path:
//   - Table payment.payment_outbox (payment schema); sqlc queries in internal/payment/queries/payment.sql.
//   - Claim: GetPendingOutboxEventsForUpdate (PENDING or PROCESSING with lease_until < now()), batch limit 100.
//   - Lease: status PROCESSING, lease_until now + 30 s; reclaimStaleProcessing clears expired leases to PENDING.
//   - Dispatch: worker_outbox.handleOutboxEvent -> outbox_settlement.applySettlementOutbox -> PaymentSettlement API.
//   - Event types: SETTLE_BALANCE, REVERSE_BALANCE, APPLY_CHARGEBACK, REVERSE_CHARGEBACK.
//   - Payload ledger_idempotency_key drives idempotency; default intent key payment:{intent_uuid} from webhook.
//
// Poll intervals (payment.payment_outbox; not public outbox 20 ms / 250 ms):
//   - Module.StartWorkers passes 100 ms as OutboxWorker.Start seed interval.
//   - Empty batch: pollTimer.Reset(interval) -> 100 ms idle wait.
//   - Rows processed: pollTimer.Reset(0) -> immediate repoll (no exponential backoff).
//   - Iteration error: 2 s retry; recovery ticker every 5x interval reclaims stale PROCESSING leases.
//
// Invariants:
//   - Settlement idempotency via ledger_idempotency_key in outbox payload and ledger host.
//   - PostSettlementMarkHook test seam only; production marks rows in worker after Apply* succeeds.
//   - Permanent outbox failure may set payment_intent SETTLEMENT_FAILED for SETTLE_BALANCE fatals.
//
// Forbidden:
//   - Synchronous settlement in Stripe/crypto HTTP handler (webhook package).
//   - Assuming public.outbox_events poll tuning applies here (different table and 100 ms loop).
//
// Verify:
//
//	go list -e ./internal/payment/settlement/
//	go test ./internal/payment/settlement/ -list '.*'
//	go test ./internal/payment/settlement/ -short -count=1
//	go test ./internal/payment/ -short -run TestFault_PaymentSettlement -count=1
package settlement

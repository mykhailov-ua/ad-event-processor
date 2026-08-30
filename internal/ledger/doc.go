// Package ledger owns billing schema writes: balance_ledger, invoices, tax, and invoice delivery.
//
// Role:
//   - Service: sqlc-backed invoice generation, ledger lines, wallet reads, PDF render hooks.
//   - Module (OpenModule): in-process billing API for controlplane; invoice cron worker when enabled.
//   - Worker (margin-guard path): CH spend vs revenue threshold checks; started from internal/control/run.go.
//   - BillingClient gRPC/HTTP facade used by controlplane handlers for operator billing calls.
//
// Topology:
//   - Payment settlement (internal/payment/settlement) calls PaymentSettlement into balance_ledger via controlplane.
//   - cmd/processor does not insert balance_ledger rows (settlement consumer writes events/stats only).
//   - Invoice delivery retries use billingadmin.InvoiceDeliveryRetryer wired in serve.go.
//
// Invariants:
//   - Financial truth: Postgres balance_ledger + current_spend; Redis budgets are operational limits.
//   - Invoice worker runs on billing schema; schedule 1st of month 00:15 UTC when InvoiceWorkerEnabled.
//   - Idempotent ledger inserts via idempotency keys on settlement paths.
//
// Forbidden:
//   - balance_ledger mutations from processor stream consumers or tracker ingest.
//   - Synchronous balance credit in Stripe webhook handler (payment outbox only).
//
// Env defaults:
//   - Margin guard interval: MarginGuardIntervalSec (WorkerInterval default 5 min when unset).
//   - Cost-over-revenue threshold: MarginGuardDefaultThresholdBps default 500 bps.
//
// Verify:
//
//	go test ./internal/ledger/ -short -count=1
//	go test ./internal/ledger/ -short -run Invoice -count=1
package ledger

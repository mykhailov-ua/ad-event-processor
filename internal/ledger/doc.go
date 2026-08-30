// Package ledger owns billing-schema invoice generation, ledger drift checks, margin-guard
// worker, and in-process BillingAPI for the control monolith.
//
// Role:
//   - Service (service.go): sqlc-backed invoice create from balance_ledger spend windows,
//     tax profiles (TaxCalculator), PDF render hooks, invoice delivery and retry paths.
//   - Module / OpenModule (open.go): wires Handler and BillingAPI when BillingInternalToken set;
//     StartWorkers runs InvoiceWorker on 1st of month 00:15 UTC when InvoiceWorkerEnabled.
//   - Worker (worker.go): CH placement spend vs revenue margin guard; pauses placements when
//     cost-over-revenue exceeds licensed threshold; started from internal/control/run.go.
//   - BillingClient (client.go): thin domain.BillingAPI facade for controlplane operator billing calls.
//   - CheckLedgerBalanceInvariant (ledger_invariant_assert.go): customers.balance vs ledger sum
//     before invoice generation; drift increments LedgerDriftTotal metrics.
//
// Topology:
//   - OpenModule from internal/control/monolith.go; margin Worker from internal/control/run.go.
//   - Payment webhook credits write balance_ledger via controlplane Service + payment/settlement
//     outbox (domain.PaymentSettlement); this package reads ledger rows for invoicing, not
//     payment credit inserts.
//   - cmd/processor stream consumers do not insert balance_ledger (events/stats only).
//   - billingadmin.NewInvoiceDeliveryRetryer wires mod.Ledger().RetryInvoiceDelivery in serve.go.
//
// Invariants:
//   - Financial truth: Postgres customers.balance and balance_ledger; Redis budgets are operational limits.
//   - Invoice generation idempotent per customer+billing_month (GetInvoiceByCustomerMonth).
//   - Settlement idempotency keys on payment paths live in controlplane; ledger invariant excludes
//     rtb_cost, operator_margin, publisher_payout from wallet sum comparison.
//   - Margin guard requires licensing.Entitlements.Features.MarginGuard on customer snapshot.
//
// Defaults and limits:
//   - MarginGuardIntervalSec: WorkerInterval default 5 min when unset (threshold.go).
//   - MarginGuardDefaultThresholdBps default 500 bps when unset.
//   - Invoice worker schedule: 1st of month 00:15 UTC (worker_invoice.go).
//
// Forbidden:
//   - balance_ledger mutations from processor stream consumers or tracker ingest.
//   - Synchronous balance credit in Stripe webhook handler (payment outbox only).
//
// Verify:
//
//	go list -e ./internal/ledger/...
//	go test ./internal/ledger/ -short -run TestService_GenerateInvoice -count=1
//	go test ./internal/ledger/ -short -run TestFault_LedgerDriftCheck -count=1
//	go test ./internal/ledger/ -short -run TestFault_InvoiceCronIdempotent -count=1
//	go test ./internal/ledger/ -short -run TestCostOverRevenueThresholdBps -count=1
package ledger

// Package payment is the in-process payment module: checkout intents, webhooks, settlement outbox.
//
// Role:
//   - Service facade over checkout.Service and webhook.Service for controlplane API client.
//   - Module (OpenModule): webhook HTTP server, payment outbox worker, crypto-hold worker, financial recon.
//   - Settlement applies provider events to balance_ledger via domain.PaymentSettlement injected at StartWorkers.
//
// Topology:
//   - Webhook listener: PAYMENT_WEBHOOK_PORT default 8187 on cmd/control when payment module enabled.
//     Routes: POST /webhooks/stripe, POST /webhooks/crypto, GET /health, GET /metrics (internal/payment/webhook).
//     Legacy GET/POST /ui/payment/* on the same listener returns 410 Gone (checkout.RegisterLegacyUIRoutes).
//   - Management JSON for self-serve intents: MANAGEMENT_PORT default 8188 via controlplane/selfserve (not :8187).
//   - Payment Postgres schema payment.* (intents, webhook_events, payment_outbox); DSN from PAYMENT_DB_DSN.
//
// Settlement outbox path (async ledger; no sync INSERT in webhook handler):
//   1. webhook.Service validates provider event, updates payment_intent in same PG txn.
//   2. INSERT payment.payment_outbox (event_type SETTLE_BALANCE, REVERSE_BALANCE, APPLY_CHARGEBACK, ...).
//   3. settlement.OutboxWorker polls payment.payment_outbox (100 ms idle; immediate repoll when batch processed).
//   4. domain.PaymentSettlement.ApplyPaymentCredit/Refund/Chargeback (controlplane.NewSettlementHandler).
//   5. public.balance_ledger rows via controlplane settlement host.
//
// Invariants:
//   - Webhook bodies limited to pkg/coldpath.PaymentWebhookMaxBody (64 KiB).
//   - Stripe signature max age 5 min on webhook/http_webhook.go.
//   - Idempotent webhook processing by provider event ID before outbox enqueue.
//
// Forbidden:
//   - Legacy /ui/payment HTML checkout (checkout registers 410 Gone on webhook mux only).
//   - Settlement RPC between processes in default monolith (in-process SettlementAPI only).
//   - balance_ledger writes inside webhook HTTP handler goroutine.
//
// Env defaults:
//   - PAYMENT_WEBHOOK_PORT default 8187 (internal/config/env_controlplane.go).
//   - Payment outbox and crypto-hold poll interval 100 ms in Module.StartWorkers (open.go).
//   - PAYMENT_FINANCIAL_RECON_INTERVAL_MS when > 0 enables recon worker tick.
//
// Verify:
//
//	go list -e ./internal/payment/
//	go test ./internal/payment/ -list '.*'
//	go test ./internal/payment/ -short -count=1
//	go test ./internal/payment/webhook/ -short -count=1
//	go test ./internal/payment/settlement/ -short -count=1
package payment

// Package payment is the in-process payment module: checkout intents, webhooks, settlement outbox.
//
// Role:
//   - Service facade over checkout.Service and webhook.Service for controlplane API client.
//   - Module (OpenModule): webhook HTTP server, payment outbox worker, crypto-hold worker, financial recon.
//   - Settlement applies provider events to balance_ledger via domain.PaymentSettlement injected at StartWorkers.
//
// Topology:
//   - Webhook listener: PAYMENT_WEBHOOK_PORT default 8187 (/webhooks/stripe, /webhooks/crypto, /health, /metrics).
//   - Management JSON for self-serve intents is on MANAGEMENT_PORT via controlplane/selfserve (not this port).
//   - payment.payment_outbox polled by settlement.OutboxWorker; no synchronous ledger write in webhook handler.
//
// Invariants:
//   - Webhook bodies limited to pkg/coldpath.PaymentWebhookMaxBody (64 KiB).
//   - Stripe signature max age 5 min on webhook/http_webhook.go.
//   - Idempotent webhook processing by provider event ID before outbox enqueue.
//
// Forbidden:
//   - Legacy /ui/payment HTML routes (checkout registers 410 Gone).
//   - Settlement RPC between processes in default monolith (in-process SettlementAPI only).
//
// Env defaults:
//   - Payment outbox poll interval 100 ms in Module.StartWorkers.
//   - PAYMENT_FINANCIAL_RECON_INTERVAL_MS when > 0 enables recon worker.
//
// Verify:
//
//	go test ./internal/payment/ -short -count=1
//	go test ./internal/payment/webhook/ -short -count=1
package payment

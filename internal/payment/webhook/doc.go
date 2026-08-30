// Package webhook ingests Stripe and crypto provider callbacks into payment.payment_outbox.
//
// Role:
//   - WebhookHandler HTTP on payment module port: POST /webhooks/stripe, POST /webhooks/crypto.
//   - Service validates signatures, normalizes amounts, inserts outbox rows (no direct ledger write).
//   - Dispute and refund replay paths share coldpath body limits and metrics counters.
//
// Topology:
//   - Registered from payment.Module.startWebhookServer; separate from management :8188 listener.
//   - GET /health and GET /metrics on the same webhook listener (promhttp).
//
// Invariants:
//   - Missing Stripe webhook secret returns 503 (fail closed).
//   - Invalid signature increments WebhookSignatureFailuresTotal and returns 400.
//
// Forbidden:
//   - balance_ledger INSERT in webhook handler goroutine (settlement outbox worker only).
//
// Verify:
//
//	go test ./internal/payment/webhook/ -short -count=1
package webhook

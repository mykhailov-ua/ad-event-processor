// Package webhook ingests Stripe and crypto provider callbacks into payment.payment_outbox.
//
// Role:
//   - WebhookHandler HTTP on payment module port: POST /webhooks/stripe, POST /webhooks/crypto.
//   - Service validates signatures, normalizes amounts, inserts outbox rows in the webhook PG transaction.
//   - Dispute and refund replay paths share coldpath body limits and metrics counters.
//
// Topology:
//   - Registered from payment.Module.startWebhookServer on PAYMENT_WEBHOOK_PORT default 8187.
//   - Separate listener from MANAGEMENT_PORT 8188 admin/self-serve JSON API.
//   - GET /health and GET /metrics on the same :8187 mux (promhttp).
//   - checkout.RegisterLegacyUIRoutes also on this mux (/ui/payment/* -> 410 Gone).
//
// Settlement outbox enqueue (no balance_ledger in this package):
//   - Stripe payment success: CreateOutboxEvent SETTLE_BALANCE with SettleBalancePayload JSON.
//   - Refunds/disputes: REVERSE_BALANCE, APPLY_CHARGEBACK, REVERSE_CHARGEBACK via service_dispute.go.
//   - Rows land in payment.payment_outbox status PENDING; settlement.OutboxWorker drains to balance_ledger.
//
// Invariants:
//   - Missing Stripe webhook secret returns 503 (fail closed).
//   - Invalid signature increments WebhookSignatureFailuresTotal and returns 400.
//   - Webhook body max pkg/coldpath.PaymentWebhookMaxBody 64 KiB.
//   - Provider event dedup via payment.webhook_events before outbox insert (same transaction).
//
// Forbidden:
//   - balance_ledger INSERT in webhook handler or Service methods (settlement outbox worker only).
//   - Calling domain.PaymentSettlement.Apply* synchronously from HTTP handlers.
//
// Verify:
//
//	go list -e ./internal/payment/webhook/
//	go test ./internal/payment/webhook/ -list '.*'
//	go test ./internal/payment/webhook/ -short -count=1
//	go test ./internal/payment/ -short -run TestProcessStripeWebhook -count=1
package webhook

// Package checkout creates and reads payment intents; internal PaymentAPI for controlplane.
//
// Role:
//   - CreatePaymentIntent, Get/List intents in Postgres payment.payment_intents.
//   - Provider adapters: Stripe checkout session, Cryptomus, BTCPay (deposit QR, crypto metadata).
//   - Handler exposes domain.PaymentAPI (gRPC metadata x-internal-token) to payment.Module and controlplane.
//
// Topology:
//   - Self-serve POST /api/v1/selfserve/payment-intents on MANAGEMENT_PORT default 8188 (campaign/selfserve).
//   - Operator billing routes on :8188; not on PAYMENT_WEBHOOK_PORT 8187.
//   - checkout.RegisterLegacyUIRoutes mounts 410 Gone on the webhook mux (:8187) for /ui/payment/{path...}.
//
// Invariants:
//   - Self-serve intent body limit: pkg/coldpath.SelfServePaymentIntentMaxBody 16 KiB.
//   - Amount validation in micro-units before provider call.
//   - Successful provider checkout does not write balance_ledger; webhook -> payment.payment_outbox only.
//
// Forbidden:
//   - HTMX or server-rendered payment pages in this package.
//   - balance_ledger INSERT or settlement.OutboxWorker calls from checkout handlers.
//
// Verify:
//
//	go list -e ./internal/payment/checkout/
//	go test ./internal/payment/checkout/ -list '.*'
//	go test ./internal/payment/checkout/ -short -count=1
package checkout

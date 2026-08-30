// Package checkout creates and reads payment intents; internal PaymentAPI for controlplane.
//
// Role:
//   - CreatePaymentIntent, Get/List intents in Postgres payment schema.
//   - Provider adapters: Stripe, Cryptomus, BTCPay (deposit QR, crypto checkout metadata).
//   - Handler exposes gRPC/HTTP checkout API to payment.Module; self-serve uses controlplane routes.
//
// Topology:
//   - Self-serve POST /api/v1/selfserve/payment-intents on management port (campaign/selfserve handlers).
//   - Legacy HTML checkout UI removed; /ui/payment/* returns 410 Gone (api_gone.go).
//
// Invariants:
//   - Self-serve intent body limit: pkg/coldpath.SelfServePaymentIntentMaxBody 16 KiB.
//   - Amount validation in micro-units before provider call.
//
// Forbidden:
//   - HTMX or server-rendered payment pages in this package.
//
// Verify:
//
//	go test ./internal/payment/checkout/ -short -count=1
package checkout

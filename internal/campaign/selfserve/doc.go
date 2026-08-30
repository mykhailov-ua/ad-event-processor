// Package selfserve serves advertiser API-key scoped HTTP under /api/v1/selfserve/*.
//
// Role:
//   - Campaign create/list, templates, payment intents, invoices, API key management callbacks.
//   - DenyScopedAPIKeyOperatorReport blocks operator-only report keys on self-serve tokens.
//   - Wired from controlplane adminapi_wire_domains.go as SelfServeHTTPHandlers.
//
// Invariants:
//   - Self-serve permissions are a subset of operator RBAC (RequireSelfServePermission).
//   - Payment intent POST uses SelfServePaymentIntentMaxBody 16 KiB.
//
// Forbidden:
//   - shards:write or audit-only routes on self-serve mux registration.
//
// Verify:
//
//	go test ./internal/campaign/selfserve/ -short -count=1
package selfserve

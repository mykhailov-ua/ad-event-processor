// Package inbound verifies affiliate S2S postback auth on conversion /track requests.
//
// Role:
// - Optional per-customer IP CIDR allowlist and HMAC-SHA256 signature verification.
//
// Invariants:
// - No verification when customer has empty allowlist and no secret configured.
// - POSTBACK_INBOUND_AUTH_DISABLED=1 skips checks (dev only).
//
// Verify:
// go test ./internal/postback/inbound/ -short -count=1
package inbound

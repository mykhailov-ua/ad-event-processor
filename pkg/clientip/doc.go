// Package clientip parses trusted proxy CIDRs and extracts client IP from X-Forwarded-For and RemoteAddr.
//
// Role:
//   - Trusted.ParseTrusted builds exact IP and CIDR sets from config entries.
//   - FromRequest walks X-Forwarded-For right-to-left until untrusted hop, else RemoteAddr.
//
// Topology:
//   - Used by control HTTP middleware and cold-path audit; edge nginx sets forwarded headers upstream.
//   - Stdlib net/http only; no internal/* imports.
//
// Invariants:
//   - Nil or empty trusted config treats only RemoteAddr as client IP.
//   - Invalid CIDR entries skipped silently during parse.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/clientip/... -short -count=1
package clientip

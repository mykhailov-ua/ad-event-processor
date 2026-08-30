// Package clientip parses trusted proxy CIDRs and extracts client IP from X-Forwarded-For and RemoteAddr.
//
// Role:
//   - ParseTrusted builds exact IP and CIDR sets from config entries.
//   - FromRequest resolves client IP for net/http handlers behind reverse proxies.
//   - FromProxyPeer resolves IP when only peer address and forwarded headers are available (gnet/edge glue).
//
// Topology:
//   - Used by control HTTP middleware and cold-path audit; edge nginx sets forwarded headers upstream.
//   - Stdlib net/http only; no internal/* imports.
//
// Invariants:
//   - Untrusted RemoteAddr: X-Real-IP and X-Forwarded-For ignored; peer host returned.
//   - Trusted peer: X-Real-IP checked before X-Forwarded-For; first matching public IP wins.
//   - X-Forwarded-For scan walks right-to-left (append model); private, loopback, and link-local hops skipped.
//   - Nil request or empty trusted config: only RemoteAddr host used (never trust headers).
//   - Invalid CIDR or IP entries skipped silently during ParseTrusted; other entries still apply.
//
// Tradeoffs:
//   - Rightmost public XFF hop matches nginx append semantics; leftmost client IP rejected when chain includes proxies.
//   - Skip RFC1918/link-local in forwarded headers even from trusted peers (forged internal hops cannot become client IP).
//   - Silent skip of bad trust-list entries vs fail-closed parse: one typo does not disable all trusted proxies.
//   - X-Real-IP preferred over XFF when both present on trusted path (single-hop operator override).
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/clientip/... -short -count=1
//	go test ./pkg/clientip/ -short -run TestFromRequest -count=1
package clientip

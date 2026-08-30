// Package netintel provides GeoIP, CIDR LPM, proxy/VPN, and TLS impersonation feeds for local filters.
//
// Role:
//   - GeoProvider (MaxMind) for country and anonymous-IP lookup on filter path.
//   - CIDR block tables with RCU snapshot reload; DC ASN, residential proxy ring, moderator intel feeds.
//   - ProxyVPNConnType classification used by Geo and segment filters before UnifiedFilter.
//
// Topology:
//   - Feed loaders run on background ticks or startup; hot path reads atomic.Pointer snapshots only.
//   - Fail-closed on refresh error after first successful load; fail-open on first boot when feed missing.
//
// Forbidden:
//   - Per-request HTTP download or Postgres fetch inside FilterEngine.Check.
//
// Verify:
//
//	go test ./internal/filter/netintel/... -short -count=1
//	go test ./internal/filter/netintel/ -short -run TestCIDR_RCUSwap -count=1
package netintel

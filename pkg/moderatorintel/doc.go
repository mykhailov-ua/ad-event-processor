// Package moderatorintel verifies signed moderator IP prefix feeds for edge fraud tier hints.
//
// Role:
//   - ParseFeedV1 decodes moderator_intel_v1 JSON into typed prefixes and network IDs.
//   - VerifySignature checks HMAC-SHA256 over raw body (base64 raw URL encoding).
//   - NetworkID and NetworkName map meta/google/tiktok/other wire tokens to uint8 IDs 1..4.
//
// Topology:
//   - Consumed by internal/filter/netintel feed loader and internal/ingest landing_bundle metrics labels.
//   - Edge sync writes moderator_intel_v1.json and optional .sig sidecar; stdlib crypto only.
//   - Callers verify signature before ParseFeedV1; parse does not read HTTP headers.
//
// Invariants:
//   - VerifySignature uses subtle.ConstantTimeCompare; false on empty secret, body, or signature.
//   - ParseFeedV1 requires format moderator_intel_v1, RFC3339 expires_at strictly after now (UTC).
//   - ParseFeedV1 requires at least one entry; unknown network name or invalid prefix rejects whole feed.
//   - Prefixes stored masked via netip.Prefix.Masked(); IPv4 and IPv6 both supported.
//   - NetworkID accepts aliases (facebook/fb, goog/google_ads, tt, unknown/empty -> other).
//
// Tradeoffs:
//   - Signed feed vs inline IP lists: vendor rotation without tracker redeploy; secret distribution cost.
//   - Expiry in feed vs TTL in loader: hard reject at parse time; loader may retain last good snapshot on refresh fail.
//   - uint8 network IDs vs string labels: compact LPM table in netintel; NetworkName for Prometheus only.
//   - Separate VerifySignature: HTTP layer can reject bad MAC before JSON decode; tests cover each path.
//
// Forbidden:
//   - Import internal/* packages.
//   - Hot-path per-request HTTP fetch (loader refreshes on background tick only).
//
// Verify:
//
//	go test ./pkg/moderatorintel/... -short -count=1
package moderatorintel

// Package proxyupstream validates click proxy upstream URLs for SSRF-safe redirect vs proxy delivery modes.
//
// Role:
//   - ValidateURL rejects userinfo, fragments, private or link-local resolved IPs, and non-HTTPS schemes unless lab flag set.
//   - ValidateDeliveryPair enforces proxy_upstream_url when click_delivery is proxy; redirect mode skips URL checks.
//   - ClickDeliveryRedirect and ClickDeliveryProxy match campaign click_delivery wire values.
//   - Exported errors (ErrEmptyURL, ErrPrivateUpstream, ErrInvalidScheme, ...) for errors.Is in handlers and tests.
//
// Topology:
//   - Hot: internal/ingest click_proxy checks ClickDeliveryProxy before outbound hop (validation at publish time in campaign).
//   - Cold: internal/campaign/runtime ops and publish_bundle ValidateDeliveryPair on campaign save/publish.
//   - Stdlib net, net/url, context only; no internal/* imports.
//
// Invariants:
//   - ValidateURL trims whitespace; empty string returns ErrEmptyURL.
//   - Literal IP hosts checked with isBlockedIP without DNS; hostnames resolved via DefaultResolver with 10s context timeout.
//   - All resolved A/AAAA addresses must pass isBlockedIP (private, loopback, link-local, multicast, CGNAT 100.64.0.0/10).
//   - HTTPS required by default; HTTP allowed only when allowHTTP true (lab / AllowHTTPInsecure campaign flag).
//   - ValidateDeliveryPair defaults empty delivery to ClickDeliveryRedirect.
//   - Redirect delivery returns nil regardless of upstream string (including empty).
//   - Proxy delivery requires non-empty upstream passing ValidateURL.
//
// Tradeoffs:
//   - DNS lookup at validation time vs connect-time only: blocks obvious SSRF to internal names at publish; DNS may change later.
//   - allowHTTP lab escape hatch vs production HTTPS-only: local lander stacks without TLS certs; default remains fail-closed.
//   - Redirect mode skips URL validation entirely: operator destination URL is browser-handled, not server-fetched.
//   - Reject userinfo and fragment vs full URL normalization: credentials and #anchors are never valid proxy upstream targets.
//   - CGNAT 100.64/10 blocked in addition to net.IP private helpers: covers shared-address-space ranges Go omits from IsPrivate.
//
// Forbidden:
//   - Import internal/* packages.
//   - Using ValidateURL alone when click_delivery may be redirect (use ValidateDeliveryPair).
//
// Verify:
//
//	go test ./pkg/proxyupstream/... -short -count=1
//	go test ./pkg/proxyupstream/ -short -run TestValidateURL_blocksPrivateLiteral -count=1
//	go test ./pkg/proxyupstream/ -short -run TestValidateDeliveryPair -count=1
package proxyupstream

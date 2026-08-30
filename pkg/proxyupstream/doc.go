// Package proxyupstream validates click proxy upstream URLs for SSRF-safe redirect vs proxy delivery modes.
//
// Role:
//   - ValidateURL rejects userinfo, fragments, private IPs, and non-HTTPS unless lab flag set.
//   - ClickDeliveryRedirect and ClickDeliveryProxy constants match campaign delivery enum.
//
// Topology:
//   - Called from campaign editor and platformconfig validation; stdlib net/url only.
//
// Invariants:
//   - DNS lookup performed for hostname; private resolved addresses rejected.
//   - Empty URL allowed only when delivery mode is redirect.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/proxyupstream/... -short -count=1
package proxyupstream

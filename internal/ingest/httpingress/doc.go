// Package httpingress parses HTTP/1, HTTP/2, and HTTP/3 ingress wire formats.
//
// Role:
//   - HTTP/1 DFA (ParseHTTP1), chunked body, header-order fingerprint, edge policy hooks.
//   - HTTP/2 HPACK decode and H2 fingerprint; HTTP/3 QUIC ingress helpers.
//   - Called from gnet Tier A epoll and pinned-worker parse paths; no filter or Redis here.
//
// Invariants:
//   - HTTP/1 happy path reuses scratch buffers for zero-alloc parse when caller pools them.
//   - Track and OpenRTB body limits enforced before handler React (see config ORTB_SCAN_MAX_BYTES).
//   - Chunked bodies allowed on /openrtb/bid only (not /track).
//
// Forbidden:
//   - encoding/json on production /track or /openrtb/bid parse path.
//   - FilterEngine.Check, UnifiedFilter, or synchronous Redis from this package.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestHTTP1Parse -count=1
//	go test ./internal/ingest/ -short -run TestChaos_CrossHop_NginxGnet -count=1
package httpingress

// Package httpingress parses HTTP/1, HTTP/2, and HTTP/3 ingress wire formats.
//
// Role:
//   - HTTP/1 DFA (ParseHTTP1), chunked body, header-order fingerprint, edge policy hooks.
//   - HTTP/2 HPACK decode and H2 fingerprint; HTTP/3 QUIC ingress helpers.
//   - Called from gnet Tier A epoll and pinned-worker parse paths; no filter or Redis here.
//
// Invariants:
//   - HTTP/1 happy path reuses scratch buffers for zero-alloc parse when caller pools them.
//   - POST /track requires Content-Length; chunked Transfer-Encoding rejected (http1TrackEdgePolicy).
//   - Chunked bodies allowed on /openrtb/bid only when TE is exactly "chunked" (not /track).
//   - Body slices for Content-Length path alias caller wire until copied or discarded upstream.
//   - Invalid TE combinations (duplicate TE header, TE+CL on chunked) return ErrInvalid.
//   - Header order recording capped at HeaderOrderMax (16) tokens per request.
//
// Contracts:
//
// ParseHTTP1(data, maxBody, scratchPtr):
//   - maxBody from MAX_REQUEST_BODY_SIZE (bytes, default 1048576); ErrPayloadTooLarge when CL or
//     chunked aggregate exceeds maxBody.
//   - Returns ErrIncomplete when headers or body not fully present in data.
//   - POST /track with missing CL or chunked TE: ErrInvalid before body read.
//
// Defaults and limits:
//   - MaxBufferedOverhead = 8192 bytes (inbound buffer headroom above maxBody).
//   - HTTP1HeaderOrder array size HeaderOrderMax = 16.
//   - Fast path: 22-byte compare for canonical POST /track request line (http1_fsm.go).
//   - Chunked parser rejects chunk extensions on /track paths; OpenRTB chunked uses scratchPtr pool.
//
// Tradeoffs:
//   - Hand-rolled HTTP/1 DFA vs net/http per request:
//     DFA runs on gnet peek buffer with zero per-request net/http allocation. Rejected net/http server
//     on tracker ingest (hot-path.mdc): incompatible with gnet epoll and alloc gate.
//   - Body aliases wire buffer vs always copying body:
//     Content-Length bodies reference data[i:i+clValue] for zero-copy parse on Tier A. Pin/copy
//     happens in gnet PinParsedHTTPRequest before Tier B handler; peek frame discarded after offload.
//   - Chunked TE on /openrtb/bid only vs allowing on /track:
//     /track rejects chunked to block TE.TE obfuscation and slow-body attacks aligned with nginx edge.
//     OpenRTB bid accepts chunked for exchange wire compatibility.
//   - POST /track requires Content-Length vs implicit empty body:
//     Missing CL on /track is ErrInvalid; forces explicit body size for edge parity (NginxTrackCorpus).
//   - Header-order fingerprint vs ignoring order:
//     Records up to 16 classified header tokens for L7WireFilter; mismatch is a fraud signal, not a
//     parse reject. Rejected rejecting non-Chrome order at parse layer: would false-positive legitimate clients.
//   - Edge/nginx parity vs gnet-only rules:
//     HTTP1IngressCanonical and TestChaos_CrossHop_NginxGnet require differential_count=0; edge Lua and
//     gnet must agree on disposition for shared corpora.
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

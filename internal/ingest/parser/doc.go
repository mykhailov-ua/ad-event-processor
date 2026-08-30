// Package parser provides zero-alloc JSON scan, UUID formatting, attestation tokens, and wire encoding for ingest.
//
// Role:
//   - Hand-rolled JSON scanner (scan.go) with depth, whitespace, string, escape, and key-pair budgets.
//   - ParseUUID and delimiter tables for broker/registry payloads (uuid_keys.go).
//   - AppendJSONString, MarshalExtra for eval wire and cold-path JSON assembly (encode.go).
//   - Attestation HMAC cookie mint/verify for landing protection (attestation.go).
//   - ConfigureSecurity toggles strict UTF-8 rejection via atomic flag (JSON_STRICT_UTF8).
//
// Thread model (hot-path.mdc Tracker thread model):
//
//	Tier A (gnet epoll): may peek HTTP framing only; must not run full /track JSON accept scan here.
//	Tier B (PinnedWorkerPool, LockOSThread): parseTrackIngest and OpenRTB split parse run the scanner
//	  over OffloadHTTPPin / worker-arena bytes copied on enqueue — not the discarded gnet peek frame.
//
// Invariants:
//   - ErrMalformed when depth, quote checks, key-pair budgets, or strict UTF-8 rules are exceeded.
//   - Track keys ASCII-only; Unicode-escaped keys rejected (TrackKeyOK).
//   - OpenRTB parse uses OrtbMaxJSONDepth and OrtbMaxQuoteChecks in addition to ScanBudget.
//   - ScanBudget nil receiver consumes are no-ops (test helpers only; production always NewScanBudget).
//
// Contracts:
//
// Env knobs:
//   - JSON_STRICT_UTF8 (bool, default true): ConfigureSecurity sets strictUTF8Enabled atomic.
//   - ORTB_SCAN_MAX_BYTES (bytes, default 262144): caps OpenRTB body scan window (OrtbScanMaxBytes).
//
// Defaults and limits (scan.go):
//   - MaxJSONDepth = 16 (track / general SkipValueBudget).
//   - OrtbMaxJSONDepth = 32 (OpenRTB nested object depth).
//   - MaxWSkip = 256 per SkipWSBudget local burst.
//   - MaxJSONTotalWSkip = 4096 (ScanBudget whitespace budget).
//   - MaxJSONStringScanBytes = 65536 (per-string scan budget).
//   - MaxJSONStringEscapes = 16384 (escape-sequence budget).
//   - MaxJSONKeyPairs = 10000 (object key-pair budget).
//   - OrtbScanMaxBytes = 262144 (OpenRTB wire scan cap).
//   - OrtbMaxQuoteChecks = 65536 (OpenRTB quote/escape walk cap).
//
// Tradeoffs:
//   - Hand-rolled ScanBudget scanner vs encoding/json.Unmarshal:
//     Scanner extracts required fields with fixed budgets and zero heap allocs on /track accept path.
//     Rejected json.Unmarshal on hot accept: allocates map/reflect, no per-field caps, harder to prove
//     0 allocs/op (make test-alloc-gate). encoding/json remains on cold paths (e.g. brand creative replica).
//   - Multiple budget counters vs single max-bytes check:
//     Separate ws/str/esc/pairs counters defeat whitespace bombs, escape walks, and key-pair floods
//     that fit under a naive byte limit (TestChaos_ParserSecurity_PS_*). Rejected one cap only: attacker
//     shifts cost between dimensions.
//   - Strict UTF-8 atomic toggle vs always permissive:
//     JSON_STRICT_UTF8 rejects overlong UTF-8 and lone surrogates at scan time. Runtime toggle allows
//     dev corpus without recompile; production default true.
//   - OrtbMaxJSONDepth 32 vs track MaxJSONDepth 16:
//     OpenRTB bid objects nest deeper; track JSON is flat. Separate caps limit OpenRTB CPU without
//     widening track attack surface.
//   - Delimiter table scan vs strings.Index for UUID fields:
//     256-byte delimiterTable gives branchless field boundary for broker/registry hot extracts.
//
// Forbidden:
//   - encoding/json.Unmarshal on production /track or /openrtb/bid accept path.
//   - Holding unsafe.String views past Tier B handler return (copy via string() when escaping arena).
//
// Verify (tests live in parent internal/ingest; subpackage has no *_test.go):
//
//	go test ./internal/ingest/ -short -run FuzzParseTrackJSON -count=1
//	go test ./internal/ingest/ -short -run TestChaos_CrossHop_NginxGnet -count=1
//	go test ./internal/ingest/ -short -run TestChaos_ParserSecurity -count=1
//	go test ./internal/ingest/ -short -run TestMintAttestationToken -count=1
package parser

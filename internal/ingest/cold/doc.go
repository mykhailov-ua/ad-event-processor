// Package cold hosts ingest-side matcher, attribution payload, and composite encoding helpers.
//
// Role:
//   - AppendAttributionPayload and JSON field append without encoding/json on hot paths.
//   - Composite fingerprint (FormatUUIDCanonical, ComputeCompositeHashUUID) and OpenRTB pb reset helpers.
//   - Header/fingerprint classifiers (Sec-Fetch, Accept-Encoding, Client Hints, TLS ALPN) wired into filter via cold_bridge init.
//   - EnsureIngestGeo, health probe metrics, conversion datacenter IP checker, and category_mask scan.
//
// Topology:
//   - Pure functions over []byte, string, and domain.Event; called from Tier B handler path after parse.
//   - ingest/cold_bridge.go forwards symbols to parent ingest package and registers filter hook funcs in init().
//   - Uses internal/ingest/parser AppendJSONString and internal/track SubIDSlots.
//   - No background goroutines, no atomic snapshots; stateless except filter/geo deps passed in.
//
// Invariants:
//   - ComputeCompositeHashUUID is zero-alloc (stack [36]byte canonical UUID + CRC32-IEEE; parent bench in composite_routing_test.go).
//   - AppendAttributionPayload returns dst[:0] when only empty object would be emitted (len(dst)==1 after '{').
//   - SecFetchAnomaly fires only for Chrome desktop UA (not in-app webview): missing Sec-Fetch trio or navigate+document pair.
//   - AcceptEncodingBrowserMismatch requires Chrome UA, br present, and zstd when Chrome major >= 123.
//   - ClientHintsPlatformMismatch skips in-app webview; compares Sec-CH-UA-Platform and ?0/?1 mobile vs ScanUAFamily.
//   - ConversionDatacenterIPChecker: geo IsAnonymous true, or ASN from GeoProvider + DCASNTable.IsDatacenter.
//   - CampaignTripletPick.PickShard: composite hash mod 100 maps 0-39 PrimaryA, 40-79 PrimaryB, 80-99 Reserve.
//
// Contracts:
//   - Wire Sec-Fetch enums: site/mode/dest packed as uint8; WireSecFetchAllBits = site|mode|dest present.
//   - Wire Accept-Encoding flags: gzip, deflate, br, zstd, identity as bit mask (WireEnc*).
//   - ParseCategoryMask scans raw JSON for "category_mask":<digits> without full unmarshal.
//   - ParseConnTimingMSHeader reuses httpingress.ParseContentLengthStrict (0..65535).
//   - RttSplitDeltaMS saturates at 65535 when ttfbApp > rttSyn.
//
// Tradeoffs:
//   - Rejected encoding/json on synchronous /track accept path; parser.AppendJSONString and manual append only.
//   - Rejected map[string] for composite routing keys: stack canonical UUID buffer + CRC32 in one pass.
//   - Rejected background reload for classifiers: pure sync eval on request bytes (no RCU layer needed).
//   - Classifiers fail-open (return false / no anomaly) when UA empty, in-app webview, or corpus/header unset.
//   - AcceptLangGeoMismatch delegates to filter.AcceptLangGeoMismatch to avoid duplicate country/lang tables.
//   - ResetAdEventInPlace truncates pb slice fields in place for sync.Pool reuse instead of new allocations.
//
// Forbidden:
//   - encoding/json on synchronous /track accept path.
//   - Postgres, ClickHouse, outbox, or background I/O side effects.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestAppendAttributionPayload -count=1
//	go test ./internal/ingest/ -short -run TestFormatUUIDCanonical -count=1
package cold

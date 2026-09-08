// Package h2frametrace hashes HTTP/2 frame-type timing traces for corpus matching.
//
// Role:
// - Normalizes edge-captured X-H2-FRAME-TRACE header values (comma-separated frame:ms_bucket tokens).
// - Tracker DeviceFilter compares CRC32-IEEE hash against an embedded UA-family corpus.
//
// Wire format example: "settings:0,headers:1,window_update:2,data:0".
//
// Invariants:
// - Fail-open when header absent, parse error, or corpus row missing.
// - Max trace 512 bytes; max token 64 bytes (parity with edge-tcp-sig-v2.lua).
//
// Verify:
// go test ./pkg/h2frametrace/ -count=1
// go test ./internal/ingest/ -short -run H2Frame -count=1
package h2frametrace

// Package parser provides zero-alloc JSON scan, UUID formatting, and wire encoding helpers for ingest.
//
// Role:
//   - Hand-rolled JSON scanner with depth and quote budgets for /track and OpenRTB paths.
//   - AppendJSONString, attestation helpers; ConfigureSecurity toggles strict UTF-8 via atomic flag.
//
// Invariants:
//   - ErrMalformed when depth, quote checks, or key-pair budgets are exceeded.
//   - OrtbScanMaxBytes and OrtbMaxQuoteChecks cap OpenRTB parse cost on hot path.
//
// Forbidden:
//   - encoding/json.Unmarshal on production /track accept path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run FuzzParseTrackJSON -count=1
//	go test ./internal/ingest/ -short -run TestChaos_CrossHop_NginxGnet -count=1
package parser

// Package openrtb provides OpenRTB 2.6 wire codec without importing ingest handlers.
//
// Role:
//   - DFA parse (ParseOpenRTB26, ParseOpenRTB26Split) with OrtbScanMaxBytes and OrtbMaxQuoteChecks scan budgets; no encoding/json on production bid path.
//   - WriteBidHTTPResponse and WriteNoBidHTTPResponse: fixed HTTP header plus in-place JSON body; ApplyMacros for adm and nurl.
//   - ValidateBytes and Validate for cold admin validate-bid-request (USD/EUR cur whitelist); OpenRTB 3.0 limited FSM paths (ParseOpenRTB3FSM, ortbMaxDepth=32).
//
// Topology:
//   - Imported by internal/ingest openrtb handlers and internal/ingest/ortbreact on PinnedWorkerPool Tier B workers.
//   - Must not import internal/ingest or internal/controlplane (wire-only boundary).
//
// Invariants:
//   - Auction glue uses micro-units; bidfloor parsed via ParseDecimalMicro.
//   - Chunked request bodies allowed on POST /openrtb/bid only (ingest handler policy).
//   - Default scan caps: OrtbScanMaxBytes=262144, OrtbMaxQuoteChecks=65536 (config may override via ConfigureOrtbScanLimits).
//
// Forbidden:
//   - json.Marshal on production bid response encode path.
//   - FilterEngine.Check or full ingest filter chain from this package.
//
// Verify:
//	go test ./internal/openrtb/ -short -count=1
//	go test ./internal/openrtb/ -short -run TestValidateOpenRTB26 -count=1
//	go test ./internal/ingest/ -short -run OpenRTB -count=1
package openrtb

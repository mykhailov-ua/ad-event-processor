// Package openrtb provides OpenRTB 2.6 wire codec without importing ingest handlers.
//
// Role:
//   - DFA parse (ParseOpenRTB26) with scan byte and quote-check budgets; no encoding/json on hot bid path.
//   - WriteBidHTTPResponse fixed header plus in-place JSON body; macro expansion for adm and nurl.
//   - ValidateOpenRTB26 for cold admin validate-bid-request API; OpenRTB 3.0 FSM for limited paths.
//
// Topology:
//   - Imported by internal/ingest/ortbreact and ingest openrtb handlers on PinnedWorkerPool Tier B.
//   - Must not import internal/ingest or internal/controlplane (wire-only boundary).
//
// Invariants:
//   - OrtbMaxJSONDepth=32; micro-units throughout auction glue.
//   - Chunked request bodies allowed on /openrtb/bid only.
//
// Forbidden:
//   - json.Marshal on production bid response path.
//   - FilterEngine.Check from this package.
//
// Verify:
//
//	go test ./internal/openrtb/... -short -count=1
//	go test ./internal/openrtb/ -short -run TestValidateOpenRTB26 -count=1
package openrtb

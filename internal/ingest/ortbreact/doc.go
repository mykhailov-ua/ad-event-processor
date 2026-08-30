// Package ortbreact maps OpenRTB 2.6 wire flags into rtb.RtbTargetingInput for ingest handlers.
//
// Role:
//   - WireTargeting scratch structs and FillFromOpenRTB26 for /openrtb/bid React path.
//   - Bridges internal/openrtb parse output to internal/rtb auction inputs without importing ingest from openrtb.
//
// Topology:
//   - Runs on PinnedWorkerPool Tier B after DFA parse; RunAuction in-process, no full FilterEngine chain.
//   - Uses ingest/cold for shared targeting helpers where needed.
//
// Forbidden:
//   - Full FilterEngine.Check on /openrtb/bid exchange path.
//   - encoding/json bid response on production hot path (use openrtb.WriteBidHTTPResponse).
//
// Verify:
//
//	go test ./internal/ingest/ -short -run OpenRTB -count=1
//	go test ./internal/openrtb/... -short -count=1
package ortbreact

// Package httpresponse writes JSON success and error envelopes for admin and webhook handlers.
//
// Role:
//   - WriteJSON and WriteError encode responses with pooled buffers.
//   - grpc.go maps gRPC-style status codes for region-proxy admin glue.
//
// Topology:
//   - Imported by internal/controlplane and domain admin handlers via pkg/coldpath patterns.
//   - encoding/json on cold path only.
//
// Invariants:
//   - ErrorResponse shape {error:{code,message}} stable for OpenAPI clients.
//   - Write sets Content-Type application/json.
//
// Forbidden:
//   - Import internal/* packages.
//   - Use on /track hot path (ingest uses prebuilt bytes).
//
// Verify:
//
//	go test ./pkg/httpresponse/... -short -count=1
package httpresponse

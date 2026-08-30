// Package httpresponse writes JSON success and error envelopes for admin and webhook handlers.
//
// Role:
//   - JSON encodes arbitrary data with encoding/json and trailing newline.
//   - Error writes {"error":{"code","message"}} with a pooled bytes.Buffer (no json.Marshal on errors).
//   - WriteGRPCError maps google.golang.org/grpc/status codes to HTTP status + Error envelope
//     (region-proxy admin glue and gRPC-backed handlers).
//
// Topology:
//   - Imported by internal/controlplane, internal/campaign, internal/fraudadmin, internal/reports,
//     internal/trafficoptimizer, and other cold-path admin handlers.
//   - Not imported by cmd/tracker or internal/ingest; hot path uses prebuilt response bytes.
//
// Invariants:
//   - ErrorResponse shape {error:{code,message}} is stable for OpenAPI clients and admin SPA parsers.
//   - JSON and Error set Content-Type application/json before WriteHeader.
//   - Error appends a single trailing newline after the JSON object.
//   - WriteGRPCError unknown or non-gRPC errors map to 500 INTERNAL with message "request failed".
//
// Zero-alloc / performance:
//   - Error uses sync.Pool of *bytes.Buffer; manual string assembly avoids json.Marshal on the error path.
//   - JSON uses encoding/json.Marshal (heap allocates); acceptable on cold admin path only.
//   - BenchmarkError in response_test.go is the alloc reference for Error; not cited as /track SLA.
//
// Fail-closed:
//   - WriteGRPCError: unmapped gRPC code -> 500 INTERNAL (no silent 200).
//   - JSON: if json.Marshal fails after WriteHeader, body may be empty while status is already sent;
//     callers should pass marshal-safe DTOs (admin handlers use typed structs).
//   - Error does not HTML-escape code/message; callers must not pass untrusted markup in message fields.
//
// Tradeoffs:
//   - Pooled manual JSON for errors vs encoding/json everywhere: fewer allocs on high-volume 4xx paths.
//   - Separate package vs pkg/coldpath: coldpath owns body limits/decode; httpresponse owns write envelopes.
//   - gRPC code table is fixed subset; extend WriteGRPCError when new status surfaces in admin API.
//
// Forbidden:
//   - Import internal/* packages.
//   - Use on /track, /click, or internal/ingest gnet handlers (prebuilt bytes + filter rejects only).
//
// Verify:
//   go test ./pkg/httpresponse/... -short -run 'TestError|TestWriteGRPCError' -count=1
//   go test ./pkg/httpresponse/... -short -bench=BenchmarkError -benchmem -count=1
package httpresponse

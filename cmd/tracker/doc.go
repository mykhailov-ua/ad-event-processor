// Binary tracker: gnet ingest on /track, /click, OpenRTB, filters, stream/broker.
//
// Build:
//   go build -o bin/tracker ./cmd/tracker/
//
// Verify (scoped):
//   go test ./internal/ingestion/ -short -count=1
//   make test-alloc-gate
//
package main

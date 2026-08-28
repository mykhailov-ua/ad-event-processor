// Package ingestion is the tracker hot path: gnet HTTP/1-2, /track and /click,
// unified filter, stream/broker producers, OpenRTB ingress, and RTB auction glue.
//
// Boundaries (hard):
//   - Must NOT import internal/controlplane admin handlers.
//   - Must NOT import internal/fraud ML scoring (batch sidecar only).
//
// SLA: ad_http_request_duration_seconds p95 < 50 ms (core.mdc).
//
// Verify (scoped):
//   go test ./internal/ingestion/ -short -count=1
//   go build -o /dev/null ./cmd/tracker/
//
// Alloc / parser gates (when touching handlers, filters, parsers):
//   make test-alloc-gate
//   bash scripts/ci/escape_heap_gate.sh
//
// Benchmarks (examples):
//   go test ./internal/ingestion/ -run='^$' -bench='BenchmarkAdsPacketHandlerProto|BenchmarkParseOpenRTB26' -benchmem -count=1
//
// Integration / fault (operator machine):
//   make test-integration
//   make test-fault
//
package ingestion

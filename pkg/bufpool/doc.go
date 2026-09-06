// Package bufpool provides sync.Pool-backed []byte reuse for cold and async paths.
//
// Role:
//   - Stream/broker encode, audit log, and other packages outside ingest/filter Tier B hot files.
//
// Forbidden:
//   - ingest/gnet, handler.go, unified-filter hot path (use local pools with alloc-gate baselines).
//
// Verify:
//
//	go test ./pkg/bufpool/ -count=1
package bufpool

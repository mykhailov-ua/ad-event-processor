// Package codec hosts sync.Pool-backed protobuf and byte-buffer helpers for stream enqueue.
//
// Role:
//   - StreamEventPool, AdLogRecordPool, ByteBufPool for zero-alloc encode on hot enqueue path.
//   - MicroUnitFactor alias to pkg/money; unsafe slice helpers for vtproto wire assembly.
//
// Topology:
//   - Used by stream producer, broker producer, fraud stream writer, and auditlog on tracker Tier B.
//
// Forbidden:
//   - encoding/json builders for production stream payloads.
//
// Verify:
//
//	go test ./internal/stream/... -short -count=1
package codec

// Package logger implements the tracker hot-path ring-buffer log shard and segment persistence.
//
// Role:
//   - LogShard: 65536-slot MPSC ring (RingCapacity, RingMask); per-slot LogPayload max 500 bytes data.
//   - Background flush to length-prefixed active.log segments on disk.
//   - Optional AES encryption via LOG_ENCRYPTION_KEY; zstd compression on segment roll.
//   - StartEvacuator uploads closed segment_*.log to S3 when uploader wired (log pipeline).
//
// Topology:
//   - Wired from cmd/tracker/wire.go; tailed by cmd/log-shipper -> broker topic tracker-logs.
//   - Compacted by cmd/log-compactor via internal/logpipeline.
//
// Defaults and limits:
//   - RingCapacity 65536; ringUsable RingCapacity-1 (one slot sentinel).
//   - False-sharing padding: 64-byte cache line pads around atomic cursors.
//   - Evacuator ticker 5s when S3 uploader enabled.
//
// Forbidden:
//   - Not structured slog replacement on cold path (tracker uses slog alongside ring buffer).
//   - pkg/* must not import internal/*.
//
// Verify:
// go test ./pkg/logger/... -short -count=1
// make test-alloc-gate
package logger

// Package logger implements the tracker hot-path ring-buffer log shard and segment persistence.
//
// Role:
//   - LogShard: 65536-slot MPSC ring (RingCapacity, RingMask); per-slot LogPayload max 500 bytes data.
//   - Logger.WriteToShard enqueues to shard; StartDrainer batches into length-prefixed active.log segments.
//   - StartPersister fdatasync(2) flushes buffers; StartDiskMonitor sheds on latency or low free space.
//   - Optional AES-GCM encryption via LOG_ENCRYPTION_KEY; zstd compression on segment roll and evacuator.
//   - StartEvacuator uploads closed segment_*.log.zst to S3 when uploader wired (log pipeline).
//
// Topology:
//   - Wired from cmd/tracker/wire.go; tailed by cmd/log-shipper -> broker topic tracker-logs.
//   - Compacted by cmd/log-compactor via internal/logpipeline; DecryptSegment for downstream readers.
//   - Processor and stream workers import Logger for audit side paths; not a slog replacement on cold admin.
//
// Defaults and limits:
//   - RingCapacity 65536; ringUsable RingCapacity-1 (one slot sentinel for full detection).
//   - False-sharing padding: 64-byte cache-line pads around atomic cursors.
//   - Persist queue depth 64..4096 from ComputePersistQueueDepth; default enqueue timeout 25ms.
//   - Drainer tick 5ms; flush batch when buffer full or 50ms since first line in batch.
//   - Disk degraded when free space < 1 GiB or write EMA exceeds DiskLatencyLimit.
//   - Evacuator ticker 5s when S3 uploader enabled.
//
// Invariants:
//   - LogShard.Write returns false when ring full after spin/sleep shed (caller must drop or retry).
//   - WriteToShard hot path targets 0 allocs/op when ring has space (TestLoggerZeroAlloc).
//   - Drainer skips priority 0 lines while diskDegraded; priority > 0 still persisted.
//   - Persist enqueue timeout increments persistQueueDrops and persistQueueDropBytes on overflow.
//   - Segment format: uint32 length prefix per record; encryption uses monotonic nonce per buffer.
//   - Close drains shards, closes persistCh, waits on wg (drainer, persister, disk monitor).
//
// Tradeoffs:
//   - MPSC ring vs channel per request: zero-alloc enqueue; bounded loss under overload vs unbounded heap.
//   - Async persist vs sync write on /track: disk latency isolated from accept path; shed under NVMe stall.
//   - Default passphrase when LOG_ENCRYPTION_KEY unset: dev convenience; operators must set key in production.
//   - zstd on roll: CPU for smaller evacuator uploads; segment still readable after decrypt without zstd on active file.
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Not structured slog replacement on cold path (tracker uses slog alongside ring buffer).
//   - fmt.Sprintf or per-request heap growth on WriteToShard path (hot-path.mdc).
//
// Verify:
//   go test ./pkg/logger/... -short -count=1
//   go test ./pkg/logger/... -short -run TestLoggerZeroAlloc -count=1
//   go test ./pkg/logger/... -run='^$' -bench=BenchmarkLoggerWriteToShard -benchmem -count=1
package logger

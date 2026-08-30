// Package log implements mmap append-only WAL segments for broker partition storage.
//
// Role:
//   - PartitionLog: segment chain per topic-partition directory under broker data dir.
//   - Leader append (Append / AppendFenced), follower replicate (AppendReplicatedAt), fetch
//     read path (ReadRawMessages), segment roll, retention, durability/fsync tiers.
//   - Consumed by internal/broker server; not imported on tracker hot path.
//
// pkg/internal boundary:
//   - Library only: no gnet, Redis coord, or consumer offset store (internal/broker).
//   - Optional pkg/iogate.DiskWriteGate for pressure-aware append and group-commit fsync.
//
// WAL on-disk layout (per partition directory):
//   - Active segment pair: {baseOffset:020d}.log + {baseOffset:020d}.index.
//   - fencing.epoch: 8-byte big-endian uint64; monotonic HA leader fence floor.
//   - Log record: length u32 (includes 8-byte offset prefix) + offset u64 + payload bytes.
//   - Sparse index entry (every indexInterval bytes appended, default 4 KiB): offset u64 +
//     byte position u64 in segment log. Index size capped from maxSegSize / indexInterval.
//   - Writable segments mmap full maxSegSize (default 64 MiB from cmd/broker -max-seg-mb);
//     4 KiB page pre-touch on writable mmap to avoid first-write fault stalls.
//   - Recover() on active segment scans log tail; truncates torn records; rebuilds nextOffset.
//   - Fetch blob (ReadRawMessages): concatenated on-disk records for client MessageIterator.
//
// Durability tiers (DurabilityConfig, ParseDurabilityMode):
//   - async (default): background flush ticker calls Sync on interval (FlushInterval 100ms).
//   - group_commit: fsync after GroupCommitRecords (64) pending appends, or when iogate
//     NoteAppend/FlushDueByInterval fires; balances latency vs batch fsync.
//   - sync: fsync (Segment.Sync) on every leader append via syncLocked.
//   - fsync failure or gate AcquireFsync error sets PartitionLog.DiskOK false (fail-closed
//     for further leader append until operator clears degraded gate).
//
// Fencing:
//   - AppendFenced(epoch, payload): rejects epoch < stored fencing.epoch with ErrStaleFencingEpoch.
//   - epoch 0 skips fence check (single-node / tests).
//   - AdvanceFencingEpoch CAS-persists monotonic epoch to fencing.epoch (tmp + rename).
//   - internal/broker maps stale epoch to produce/fetch response status 5.
//
// Retention:
//   - RetentionPolicy: MaxAge, MaxBytes, FloorOffset (consumer HWM), SafetyMessages tail guard.
//   - ApplyRetention never deletes the active writable segment; sealed segments only.
//
// Defaults and limits:
//   - ErrSegmentNotFound, ErrStaleFencingEpoch, ErrReplicationGap sentinel errors.
//   - FetchBufPool: 1 MiB pooled buffers for fetch reads <= 1 MiB.
//
// Tradeoffs:
//   - mmap MAP_SHARED append vs streaming write: zero-copy hot append, segment-sized VM map.
//   - async/group_commit vs sync: ingest latency vs crash window (group_commit default in HA).
//   - Pre-fault writable mmap pages vs lazy fault: pay startup touch cost for steady-state p99.
//   - Segment roll syncs active segment before seal; roll uses iogate TierLow when gated.
//   - findActualIndexSize trims corrupt partial index writes on open (torn tail discard).
//
// Forbidden:
//   - Not Redis Streams; budget Lua remains on Redis when broker is CH-ingest-only.
//   - pkg/* must not import internal/*.
//
// Verify:
// go test ./pkg/broker/log/ -short -count=1
package log

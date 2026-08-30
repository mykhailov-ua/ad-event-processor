// Package log implements mmap WAL segments for the broker ingest daemon.
//
// Role:
//   - Segment append with sparse index (index interval from serve -index-kb, default 4 KiB).
//   - Fencing epoch file (fencing.epoch) for HA leader handoff.
//   - Fetch via mmap with FetchBufPool (1 MiB pooled read buffers).
//   - Retention and durability hooks for segment rotation and fsync policy.
//
// Topology:
//   - Data dir default /var/lib/ad-event-processor/broker (cmd/broker -data-dir).
//   - Max segment size from -max-seg-mb (default 64 MiB).
//   - DiskWriteGate from pkg/iogate backs pressure-aware segment writes.
//
// Defaults and limits:
//   - ErrStaleFencingEpoch, ErrReplicationGap, ErrSegmentNotFound sentinel errors.
//   - Segment base offset monotonic; torn tail discarded on Recover paths in tests.
//
// Forbidden:
//   - Not Redis Streams; budget Lua remains on Redis when broker is CH ingest only.
//   - pkg/* must not import internal/*.
//
// Verify:
// go test ./pkg/broker/log/... -short -count=1
// go test ./pkg/broker/... -short -count=1
package log

// Package iogate sheds or batches disk-append I/O for broker WAL and mmap paths under a latency budget.
//
// Role:
//   - DiskWriteGate caps concurrent appends (appendSem) and serializes fsync (fsyncSem).
//   - TierHigh (WAL/broker mmap): proceeds when degraded; waits on appendSem only.
//   - TierLow (forward/dedup): returns ErrShed when degraded or DiskWritable is false.
//   - NoteAppend triggers group-commit fsync when record count or interval threshold is met.
//   - FlushVectored (linux): writev(2) scatter write + optional group-commit fsync via gate.
//   - BindMetrics wires Prometheus observers from internal/metrics/disk_gate_wire.go.
//
// Defaults and limits:
//   - DefaultDiskLatencyBudget 50ms (fsync EMA above this sets degraded).
//   - DefaultGroupCommitRecords 64 (NoteAppend fsync trigger by count).
//   - DefaultGroupCommitInterval 100ms (NoteAppend fsync trigger by wall clock).
//   - DefaultAppendCapacity 32 (appendSem buffered capacity).
//
// Env defaults:
//   - DISK_LATENCY_BUDGET_MS overrides default budget at gate construction (invalid/zero falls back).
//
// Topology:
//   - Used by internal/broker server, pkg/broker/log PartitionLog, pkg/regionproxy/wal, cmd/region-proxy.
//   - golang.org/x/sys/cpu.CacheLinePad separates inFlight from fsyncInFlight atomics (false-sharing).
//   - writev_linux.go is linux-only; writev_stub.go provides non-linux fallback (buffered Write).
//
// Invariants:
//   - At most one fsync holder at a time (fsyncSem capacity 1).
//   - EMA alpha 0.1 on fsync latency; first sample seeds EMA; budget breach sets degraded.
//   - Nil *DiskWriteGate methods are no-ops (callers may pass nil in tests).
//   - ErrShed is wrapped with tier context: errors.Is(err, ErrShed).
//
// Zero-alloc / performance:
//   - AcquireAppend + ReleaseAppend: 0 allocs/op on hot gate path (TestDiskGateAcquire_zeroAlloc).
//   - Group commit targets >=70% fsync reduction vs per-append fsync (TestGroupCommit_FsyncReduction70Percent).
//   - VectoredWrite allocates iovecs slice per call; acceptable on WAL append path, not /track.
//   - BenchmarkDiskGateAcquire in disk_gate_bench_test.go measures gate acquire/release only.
//
// Fail-closed:
//   - TierLow: ErrShed when degraded or DiskWritable reports false (dedup/forward paths drop work).
//   - TierHigh: fail-open on degraded flag (WAL still acquires append slot; may block on full appendSem).
//   - DiskWritable false sets degraded and sheds TierLow immediately.
//   - Context cancel on AcquireAppend/AcquireFsync returns ctx.Err() (no silent success).
//
// Tradeoffs:
//   - Tier split: durability-critical WAL (TierHigh) vs sheddable side lanes (TierLow).
//   - Group commit trades fsync latency for throughput; interval + count thresholds are tunable via Config.
//   - Degraded is sticky until operator clears via SetDegraded(false) or process restart; no auto-heal loop here.
//   - EMA on fsync latency vs instantaneous spike detection: smooths brief blips, two-sample budget cross in tests.
//
// Forbidden:
//   - Import internal/* packages.
//   - Use TierLow for budget debit or stream admission on tracker hot path.
//
// Verify:
//
//	go test ./pkg/iogate/... -short -run TestDiskGateAcquire_zeroAlloc -count=1
//	go test ./pkg/iogate/... -short -run TestDiskWriteGate_DegradedShedsTierLowBlocksTierHigh -count=1
//	go test ./pkg/iogate/... -short -run TestGroupCommit_FsyncReduction70Percent -count=1
package iogate

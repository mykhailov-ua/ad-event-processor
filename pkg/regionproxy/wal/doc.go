// Package wal is the mmap append-only WAL for cmd/region-proxy regional ingress.
//
// Role:
//   - Open(dir, DiskWriteGate): single wal.segment file mmap'd at defaultSegSize 64 MiB (grown if file already larger).
//   - Append / AppendBatch assign monotonic seq; Recover on open truncates torn tail and rebuilds writePos/nextSeq.
//   - ProcessPendingKeyGen and ScanDedupReady populate FactorU and WalFlagDedupReady for opkey/uplink.
//   - TryClaimForward, UnclaimForward, MarkRemoteAcked implement uplink forward lifecycle on header flags.
//   - Partition wraps WAL for broker-style AppendReplicatedAt / AppendLeader with fencing epoch.
//
// Topology:
//   - Wired from internal/regionproxy Server (data-dir on cmd/region-proxy); keygen and opkey read the same segment.
//   - Group commit via pkg/iogate.DiskWriteGate (TierHigh append, AcquireFsync on NoteAppend threshold).
//
// Invariants:
//   - Record layout: HeaderSize 128 bytes LE header + payload; MaxPayloadSize 4096; empty payload rejected (ErrEmptyPayload).
//   - Recover stops at first record missing WalFlagAppended or with invalid length; truncates file tail to last valid byte offset.
//   - Flag order: WalFlagAppended -> WalFlagDedupReady -> WalFlagForwardClaimed -> WalFlagRemoteAcked (MarkRemoteAcked requires forward claimed).
//   - TryClaimForward no-ops when dedup not ready or already claimed; UnclaimForward preserves remote-acked records.
//   - Append holds WAL mutex; forward flag updates use TierLow gate when configured.
//   - ErrSegmentFull when append would exceed mmap capacity; ErrCorrupt on bounds/header mismatch.
//
// Defaults and limits:
//   - walSegmentFile wal.segment under data dir; seq starts at 0 after empty recover.
//   - ReadRawMessages emits broker wire framing for partition fetch consumers.
//
// Tradeoffs:
//   - Single fixed mmap segment vs rotating files: simpler recover/truncate, bounded regional retention per cell.
//   - Mmap append with optional group fsync balances durability vs regional ingest latency (not Postgres spend authority).
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy (hot path boundary).
//   - Not Postgres balance_ledger or global spend authority; global control reconciles accepted batches.
//
// Verify:
// go test ./pkg/regionproxy/wal/... -short -run 'TestWAL_AppendRecoverRoundTrip|TestWAL_RecoverTruncatesTornTail|TestWAL_ProcessPendingKeyGen|TestWAL_ForwardClaimAndRemoteAck|TestWAL_FaultSIGKILLReplayIdempotent' -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package wal

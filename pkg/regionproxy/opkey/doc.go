// Package opkey pools operation keys with an MPSC queue and WAL watermark backpressure.
//
// Role:
//   - Pool drains ScanDedupReady WAL records into TryEnqueue slots: 16-byte OpID + FactorU + seq.
//   - idGen mixes node FNV-64 hash, wall ms, and per-node seq for OpID uniqueness.
//   - Slot CAS flags: Derived, ReplicaBooked, Executing, LeaseRenewed (local lease before Redis quorum).
//   - BatchCommitter coordinates pkg/regionproxy/quorum Book/Transition with uplink forward gating.
//
// Topology:
//   - Wired from internal/regionproxy.Server.SetOpKey; uplink.Worker Dequeue consumes the same Pool.
//   - drainWAL advances lastWalSeq; ScanDedupReady callback stops enqueue on shed or full queue.
//   - BatchCommitter optional on uplink.Config; fault tests wire Redis for multi-replica forward proofs.
//
// Defaults and limits:
//   - MPSCQueue power-of-two QueueSize default 4096; invalid size resets to 4096.
//   - Watermark default 1000: TryEnqueue returns false when Depth > Watermark (shedTotal + incIngressShed metric).
//   - PollInterval default 1ms; BatchSize default 64 in New (cmd/region-proxy sets 256).
//   - Push failure when ring full returns false without incrementing enqueued counter.
//
// Invariants:
//   - One Slot per WAL seq in flight; Release zeroes flags and returns Slot to sync.Pool.
//   - BatchCommitter.PrepareForward requires quorum.QuorumMet before Transition to executing; uplink skips slot when not ready.
//   - Without Redis BatchCommitter, PrepareForward uses local Slot TryBook/TryClaimExecuting only.
//   - OpID on uplink JSON must match slot.OpID for global operation_lease dedup paths.
//
// Forbidden:
//   - Unbounded enqueue without watermark (production cmd/region-proxy sets Watermark 1000).
//   - internal/ingest hot path importing pkg/regionproxy/opkey.
//
// Verify:
// go test ./pkg/regionproxy/opkey/... -short -count=1
// go test ./pkg/regionproxy/opkey/ -short -run TestBatchCommitter -count=1
// go test ./internal/controlplane/ -short -run TestFault_RegionProxyQuorumBook -count=1
package opkey

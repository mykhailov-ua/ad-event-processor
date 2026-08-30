// Package keygen assigns regional dedup FactorU keys to WAL records for uplink idempotency.
//
// Role:
//   - KeyGen background loop polls WAL tail via ProcessPendingKeyGen; sets WalFlagDedupReady + FactorU header field.
//   - derive callback: dedupkey.WriteCanonicalProxyBatchPayload(seq, payload) then dedupkey.FactorU (32-byte UUID-shaped hash).
//   - Metrics hooks via BindMetrics (rate, queue depth, per-record lag histogram).
//
// Topology:
//   - Started from internal/regionproxy.Server.SetKeyGen; cmd/region-proxy -region-code and -node-id flags.
//   - Runs after WAL Append/AppendBatch; opkey.Pool ScanDedupReady blocks until DedupReady flag set.
//   - NodeID in Config is wired for observability; FactorU derivation uses seq + payload only.
//
// Defaults and limits:
//   - PollInterval default 1ms; BatchSize default 64 in New (cmd/region-proxy sets 256).
//   - Processed counter exposed via Processed(); KeyGenQueueDepth from WAL counts records missing DedupReady.
//   - LockOSThread on loop goroutine; stack canonicalBuf sized wal.MaxPayloadSize + 64 for zero-alloc derive path.
//
// Invariants:
//   - Records processed in WAL seq order; stops at first record without WalFlagAppended or with DedupReady already set.
//   - FactorU written at header FactorUOffset; flag mutation uses iogate TierLow when gate present.
//   - Global ingest and internal/regionproxy IngestBatch reject factor_u mismatch vs recomputed canonical hash.
//
// Forbidden:
//   - Skipping keygen while uplink enabled (opkey drain stalls at first non-DedupReady record).
//   - internal/ingest hot path importing pkg/regionproxy/keygen.
//
// Verify:
// go test ./pkg/regionproxy/keygen/... -short -count=1
// go test ./pkg/regionproxy/wal/... -short -run KeyGen -count=1
// go test ./tests/e2e/ -short -run TestE2E_RegionProxyUplink -count=1
package keygen

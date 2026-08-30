// Package regionproxy is the root for Enterprise multi-region cell helpers (license feature multi_region).
//
// Role:
//   - Shared libraries for cmd/region-proxy and internal/regionproxy gnet WAL ingress server.
//   - Regional cmd/processor spend sync uses client + ingress topic only (MULTI_REGION_ENABLED=1).
//   - Global control IngestRegionProxyBatch consumes uplink JSON; quorum leases gate replica-forward paths.
//
// Pipeline (primary region-proxy cell):
//  1. Ingress: gnet produce-batch or broker fan-in appends payloads to mmap WAL (pkg/regionproxy/wal).
//  2. Keygen: background goroutine stamps WalFlagDedupReady + FactorU per record (pkg/regionproxy/keygen).
//  3. Opkey: MPSC pool assigns 16-byte operation IDs and queues dedup-ready slots (pkg/regionproxy/opkey).
//  4. Uplink (optional): HTTP POST each slot to GLOBAL_INGEST_URL; WAL forward/remote-ack flags (pkg/regionproxy/uplink).
//  5. Quorum (optional): Redis lease book/ack before forward when opkey.BatchCommitter wired (pkg/regionproxy/quorum).
//
// Subpackages:
//   - wal: mmap append-only wal.segment; group commit via iogate.DiskWriteGate; Recover discards torn tail.
//   - client: broker ProduceSpendSyncPayload to ingress.DefaultTopic (regional processor path).
//   - ingress: broker topic name constant region-proxy-ingress (matches internal/regionproxy.DefaultIngressTopic).
//   - keygen: dedupkey.FactorU over canonical (seq, payload); polls WAL KeyGenQueueDepth.
//   - opkey: TryEnqueue/Dequeue slots; watermark load shed; BatchCommitter bridges quorum + uplink.
//   - quorum: Redis HSET/SADD leases ad_event_processor:op:lease:{hex}; states booked/executing/completed.
//   - uplink: Worker drains opkey.Pool; TryClaimForward/MarkRemoteAcked on WAL; X-Admin-API-Key header.
//
// WAL (pkg/regionproxy/wal):
//   - Open(dir, gate): single 64 MiB mmap file wal.segment; monotonic seq per Append/AppendBatch.
//   - 128-byte header (HeaderSize): Seq, PayloadLen, FactorU[32], Flags (WalFlagAppended, DedupReady,
//     ForwardClaimed, RemoteAcked); MaxPayloadSize 4096 bytes after header.
//   - TierHigh gate on append; TierLow on keygen/forward header mutations; NoteAppend triggers group fsync.
//   - Recover walks records until first incomplete header or torn payload; resets writePos and nextSeq.
//   - ScanDedupReady, ProcessPendingKeyGen, TryClaimForward, MarkRemoteAcked for downstream consumers.
//
// Quorum (pkg/regionproxy/quorum):
//   - Book/AckBook add node_id to ack set; ReadStatus returns AckCount vs Required(replicaCount).
//   - Required: quorum 1 when replicaCount <= 1; defaultQuorum 2 when replicaCount > 1.
//   - Transition CAS state booked -> executing -> completed; leaseTTL 48h on Redis keys.
//   - opkey.BatchCommitter calls Book + Transition before uplink forward; Complete marks completed.
//   - Global control operation_lease paths use the same store when Enterprise multi_region enabled.
//
// Uplink (pkg/regionproxy/uplink):
//   - Worker loop LockOSThread; Dequeue from opkey.Pool up to BatchSize per tick.
//   - Optional BatchCommitter.PrepareForward gates forward until quorum met; Complete after HTTP 2xx.
//   - forwardSlot: TryClaimForward, POST JSON (region_code, node_id, source_epoch, seq, factor_u, payload, op_id),
//     MarkRemoteAcked on success; UnclaimForward on retryable failure (ForwardMaxAttempts default 3).
//   - Wired from cmd/region-proxy when -global-ingest-url or GLOBAL_INGEST_URL set; BatchCommitter optional in tests.
//
// Topology:
//   - Not default single_vps; compose profile multi-region (regions.mdc).
//   - Production hot ingest (internal/ingest, non-_test) must not import pkg/regionproxy.
//   - pkg/* must not import internal/*; tracker /track must not block on regional uplink or quorum Redis.
//
// Invariants:
//   - factor_u on uplink must match dedupkey.FactorU over canonical proxy batch bytes (global ingest rejects mismatch).
//   - WAL is regional durability buffer, not Postgres balance_ledger authority; global control reconciles spend.
//   - Opkey watermark shed (TryEnqueue false) is explicit load shed; disk gate saturated returns proxyBackpressure on gnet.
//
// Forbidden:
//   - internal/ingest hot path importing pkg/regionproxy.
//   - Treating broker client mock or unit WAL test as multi-region cutover proof without integration tier.
//
// Verify:
// go test ./pkg/regionproxy/... -short -count=1
// go test ./internal/regionproxy/... -short -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package regionproxy

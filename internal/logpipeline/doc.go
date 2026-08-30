// Package logpipeline owns shared log-tier pipeline logic for evacuator and compactor sidecars.
//
// Role:
//   - Evacuator (evacuator.go): watches hot log dir for *.log.zst.ready segments, uploads to ObjectStore
//     (S3 or memory tier), persists evacuator checkpoint; optional compactor marker gate.
//   - Compactor (compactor.go): claims hot segments, decrypts zstd streams, filters/samples impressions
//     (sample.go, segment.go), key-compacts by click_id (key_compact.go), writes warm tier via TierStore.
//   - ColdRolluper (cold_rollup.go, cold_clickhouse.go): aggregates warm segments into hourly CH rollups
//     and filter-reject slice tables when cold tier enabled.
//   - CheckpointStore (checkpoint.go): JSONL records with source/dest SHA256 digests for idempotent compaction.
//   - Leader lock (leader.go), lag gauges (lag.go, metrics.go), lifecycle helpers (lifecycle.go).
//
// Topology:
//   - Node-local I/O on mmap/zstd log segments; wired from cmd/log-evacuator and cmd/log-compactor.
//   - Optional S3 tier (s3_tier_store.go) and in-memory tier for tests (memory_tier_store.go).
//   - Protobuf events use internal/ingest/pb.AdStreamEvent framing (length-prefixed payloads in segments).
//
// Invariants:
//   - Compaction checkpoint keyed by source_key + source_sha256; duplicate source digest skipped.
//   - Billable events (click, conversion, fraud, silent_reject) always kept; impressions sampled by click_id hash.
//   - Evacuator single-flight per segment name; stuck .evacuating segments recovered on startup.
//   - Compactor leader election optional; only lock holder compacts when configured.
//
// Forbidden:
//   - Tracker hot path (internal/ingest) imports.
//   - Direct /track or Redis stream writes from this package.
//
// Verify:
//
//	go test ./internal/logpipeline/... -short -count=1
//	go test ./internal/logpipeline/ -short -run TestFault_logCompactorCheckpointCrashRecovery -count=1
//	go test ./internal/logpipeline/ -short -run TestEvacuator_uploadsReadySegment -count=1
package logpipeline

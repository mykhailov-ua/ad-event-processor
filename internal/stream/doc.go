// Package stream implements hot-path event enqueue: Redis StreamProducer and broker mmap producers.
//
// Role:
//   - StreamProducer.TryReserve / BrokerProducer.TryReserve before UnifiedFilter Lua debit (Tier B worker).
//   - Async XADD or broker Produce after accept; not in the same Lua script as budget debit.
//   - Subpackages: broker (mmap WAL client), codec, fraud stream writer, recon helpers, auditlog, breaker.
//
// Invariants:
//   - TryReserve at STREAM_PRODUCER_ADMISSION_PCT (default 85%) fail-closed before debit.
//   - SetDeferStreamToProducer(true) -> Lua stream key fcap:ignored; single Go writer.
//   - Post-debit reject increments ad_stream_producer_post_debit_rejected_total (~0 in prod).
//
// Forbidden:
//   - Sync stream write before HTTP 202 on /track accept path.
//   - Claiming CH ingest wired without integration/fault flush proof.
//
// Verify:
//
//	go test ./internal/stream/... -short -count=1
//	go test ./internal/ingest/ -short -run TestStreamProducerAdmissionRaceWithoutReserve -count=1
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_SetDeferStreamToProducer -count=1
package stream

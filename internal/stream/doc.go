// Package stream implements event enqueue on the tracker and cold-path stream/broker consumers.
//
// Role:
//   - Hot path (tracker Tier B): StreamProducer.TryReserve before UnifiedFilter Lua debit; async XADD after accept.
//   - Broker mmap producer lives in internal/stream/broker (TryReserve + Produce); not in the same Lua as debit.
//   - Cold path (cmd/processor): StreamConsumer, ClickHouse store/spool, settlement workers, local quanta ledger,
//     budget-rollback.lua / local-quota-refill.lua companions.
//   - Subpackages: broker, codec, fraud (fraud stream writer), recon, auditlog, breaker.
//
// Invariants:
//   - TryReserve at STREAM_PRODUCER_ADMISSION_PCT (default 85%) fail-closed before debit; occupied = depth + reserved.
//   - SetDeferStreamToProducer(true) -> Lua stream key fcap:ignored; single Go writer (StreamProducer or BrokerProducer).
//   - Post-debit enqueue fail -> budget-rollback.lua + local quanta refund; ad_stream_producer_post_debit_rejected_total (~0 in prod).
//   - StreamConsumer uses circuit breaker, poison split, and optional ProcessorWeightController throttling.
//   - No synchronous XADD or Produce before HTTP 202 on /track accept path.
//
// Tradeoffs:
//   - TryReserve before debit: reserves a queue/ring slot before UnifiedFilter Lua debit so overload returns 503
//     filterRejectProducerOverload instead of debiting budget then failing enqueue (ad_stream_producer_post_debit_rejected_total).
//     Rejected alternative: debit-first admission race (TestStreamProducerAdmissionRaceWithoutReserve holdout).
//   - Dual stream write history: older paths could XADD in Lua and again in Go StreamProducer (duplicate CH rows).
//     SetDeferStreamToProducer sets KEYS[9]/local-quanta stream to fcap:ignored; Go producer is sole writer
//     (TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix).
//   - Broker ring vs Redis XADD: CH_INGEST_SOURCE=broker uses mmap WAL BrokerProducer (default ring 32768, power-of-two MPSC)
//     with the same TryReserve admission path; CH_INGEST_SOURCE=redis uses per-shard StreamProducer chan queue + async XADD.
//     Lua never holds Redis thread for stream append; broker cutover runs BROKER_SHADOW_MODE before live CH commit.
//   - Local quanta full-skip: zero sync EVALSHA when eligible; stream lane still defers to fcap:ignored when any Go producer
//     is wired so local-quanta async lane cannot race the authoritative sink.
//
// Forbidden:
//   - Sync stream write before HTTP 202 on /track accept path.
//   - Claiming CH ingest wired without integration/fault flush proof.
//
// Verify:
//
//	go test ./internal/stream/ -short -run TestLocalQuantaLedger_TrySpendLocal -count=1
//	go test ./internal/stream/ -short -run TestStreamConsumer_FlushBatch_XAckError -count=1
//	go test ./internal/stream/broker/ -short -run TestBrokerProducer_RingOverflow -count=1
//	go test ./internal/ingest/ -short -run TestStreamProducerAdmissionRaceWithoutReserve -count=1
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix -count=1
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_RollbackDebit_LocalQuanta -count=1
package stream

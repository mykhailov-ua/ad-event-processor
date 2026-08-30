// Package broker implements mmap WAL producer and consumer for CH_INGEST_SOURCE=broker cutover.
//
// Role:
//   - broker_producer.go: BrokerProducer power-of-two MPSC ring; async flush goroutine calls pkg/broker/client.Produce.
//   - broker_consumer.go: BrokerStreamConsumer reads WAL batches, optional shadow mode, CH StoreBatch with breaker.
//   - broker_payload.go: ParseBrokerPayload decodes vtproto AdStreamEvent or AdLogRecord via stream/codec pools.
//   - broker_reconcile.go: BrokerReconcileWorker compares broker offsets vs Redis stream length (divergence metric).
//   - fraud_broker_sink.go: length-prefixed fraud batch Produce when fraud path uses broker instead of Redis XADD.
//
// Topology:
//   - Producer runs on tracker Tier B after accept (mirror StreamProducer admission path).
//   - Consumer runs in cmd/processor; BROKER_SHADOW_MODE=1 (cfg.ShadowMode) counts without CH commit or offset advance.
//   - Default topic ad-events; default ring capacity 32768 (rounded to power of two).
//
// Invariants:
//   - TryReserve(admissionPct) before UnifiedFilter Lua debit; occupied = (head - tail) + reserved.
//   - EnqueueReserved consumes reservation then Enqueue; ReleaseReserve on post-debit reject paths.
//   - ErrRingBufferFull must surface before debit when TryReserve is wired (ingest tryAcquireStreamAdmission).
//   - Post-debit enqueue failure must not swallow Produce errors without rollback metric (ad_stream_producer_post_debit_rejected_total).
//   - Shadow mode increments ad_broker_shadow_messages_total; live mode commits offsets after durable CH write.
//   - Corrupt broker payloads skipped with metric; consumer breaker opens on repeated StoreBatch failure.
//
// Tradeoffs:
//   - Broker ring vs Redis XADD: mmap WAL + partition HA replaces Redis stream RAM and PEL for CH ingest when
//     CH_INGEST_SOURCE=broker; tracker still runs UnifiedFilter Lua on Redis for budget/dedup. Rejected: Lua XADD in
//     the same script as debit (blocks Redis single thread; split to Go producer + budget-rollback.lua).
//   - Power-of-two MPSC ring (default 32768): fixed memory, cache-friendly slots, ErrRingBufferFull at capacity instead
//     of unbounded heap growth. Same STREAM_PRODUCER_ADMISSION_PCT TryReserve as StreamProducer (default 85% fail-closed).
//   - Shadow cutover: BROKER_SHADOW_MODE=1 observes WAL without CH commit or offset advance; ad_broker_ingest_divergence_high
//     must stay quiet before live cutover (TestFault_BrokerShadowCutover_NoEventLoss). Rejected: big-bang broker flip without shadow soak.
//   - Fraud broker sink: length-prefixed batches on a separate topic when CH_INGEST_SOURCE=broker; skips per-shard fraud XADD
//     while preserving analytical/critical lane semantics from internal/stream/fraud.
//
// Forbidden:
//   - Swallowing Produce errors after budget debit without rollback metric.
//   - Claiming broker-primary CH ingest from mockBrokerClient unit tests alone.
//
// Verify:
//
//	go test ./internal/stream/broker/ -short -run TestBrokerProducer_RingOverflow -count=1
//	go test ./internal/stream/broker/ -short -run TestBrokerStreamConsumer_ShadowMode -count=1
//	go test ./internal/stream/broker/ -short -run TestBrokerStreamConsumer_CorruptPayload -count=1
//	go test ./internal/ingest/ -short -run TestFault_BrokerShadowCutover_NoEventLoss -count=1
//	go test ./internal/ingest/ -short -run TestFault_BrokerLiveConsumer_CorruptPayload -count=1
package broker

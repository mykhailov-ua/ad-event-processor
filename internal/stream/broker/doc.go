// Package broker implements mmap WAL producer and consumer for CH_INGEST_SOURCE=broker cutover.
//
// Role:
//   - BrokerProducer ring buffer enqueue with TryReserve admission before Lua debit.
//   - BrokerStreamConsumer flushes batches to ClickHouse; shadow mode for divergence checks.
//   - Payload codec for AdStreamEvent and AdLogRecord vtproto wire.
//
// Topology:
//   - Producer runs on tracker Tier B after accept; consumer runs in cmd/processor.
//   - BROKER_SHADOW_MODE=1 compares against Redis stream path before live cutover.
//
// Invariants:
//   - ErrRingBufferFull fail-closed before debit when TryReserve wired correctly.
//   - Corrupt payloads skipped with metric on consumer path (TestFault_BrokerLiveConsumer_CorruptPayload).
//
// Forbidden:
//   - Swallowing Produce errors after budget debit without rollback metric.
//
// Verify:
//
//	go test ./internal/stream/broker/... -short -count=1
//	go test ./internal/ingest/ -short -run TestFault_BrokerShadowCutover_NoEventLoss -count=1
package broker

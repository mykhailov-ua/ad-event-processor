// Package unified implements UnifiedFilter: the last filter gate and Redis Lua budget debit.
//
// Role:
//   - Single EVALSHA (unified-filter.lua) per accept when not local-quanta full-skip eligible.
//   - Go-local prechecks (schedule, geo floor, ingress RPD) before Lua when configured.
//   - SetDeferStreamToProducer(true) sets stream key fcap:ignored; Go StreamProducer/BrokerProducer is sole writer.
//
// Topology:
//   - Called synchronously from FilterEngine.Check on PinnedWorkerPool Tier B worker.
//   - Redis keys use {campaign_id} hash tag; sharder picks master for multi-key atomicity.
//
// Invariants:
//   - TryReserve on stream producer must succeed before Lua debit (enforced in ingest tryAcquireStreamAdmission).
//   - Post-debit enqueue failure triggers RollbackDebit (budget-rollback.lua or local quanta refund).
//   - FILTER_TIMEOUT_MS enforced via monotonic deadline inside Check, same worker goroutine.
//
// Forbidden:
//   - Sync XADD in the same Lua script as budget debit.
//   - Postgres writes on synchronous Check path (budget miss may read registry snapshot only).
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_RollbackDebit -count=1
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_SetDeferStreamToProducer -count=1
//	go test ./internal/filter/... -short -count=1
package unified

// Package unified implements UnifiedFilter: the last filter gate and Redis Lua budget debit.
//
// Role:
//   - Single EVALSHA (unified-filter.lua) per accept when not local-quanta full-skip eligible.
//   - Go-local prechecks (schedule, geo floor, ingress RPD, TTC when ttc_in_go) before Lua when configured.
//   - budget-fast.lua for debit-only fast path; budget-rollback.lua on post-debit enqueue failure.
//   - SetDeferStreamToProducer(true) sets stream key fcap:ignored; Go StreamProducer/BrokerProducer is sole writer.
//
// Topology:
//   - Called synchronously from FilterEngine.Check on PinnedWorkerPool Tier B worker.
//   - Redis keys use {campaign_id} hash tag; sharder picks master for multi-key atomicity.
//   - Holdout tests for debit/admission/rollback currently live in internal/ingest (pre-drain wiring).
//
// Lua return codes (unified-filter.lua):
//
//	-1 budget key missing; 0 success; 2 duplicate (dedup SET NX failed after checks);
//	3 insufficient budget; 4 daily pacing quota; 5 frequency cap; 6 TTC below minimum;
//	7 TTC missing when fail-closed; 10 TTC bypass (missing imp ts, fail-open); 11 routing epoch / migration fence;
//	20 degraded accept (remaining deadline < degrade_ns, default 2ms).
//
// Budget invariants:
//   - TryReserve on stream producer must succeed before Lua debit (ingest tryAcquireStreamAdmission).
//   - MGET spend_key, dedup, daily_spent, fcap before debit; budget and pacing checks before SET NX dedup.
//   - Debit: INCRBY spend_key -amount; campaign/customer sync counters on first spend in window.
//   - Dedup SET NX on KEYS[2] after budget gates, before side effects; duplicate returns 2 without debit.
//   - XADD in Lua only when KEYS[9] is set and not fcap:ignored; deferred producer mode uses fcap:ignored.
//   - Post-debit enqueue failure triggers RollbackDebit (budget-rollback.lua or local quanta refund).
//   - Local-quanta full-skip (LOCAL_QUOTA_MODE live): zero sync EVALSHA when eligible; async publish only.
//   - FILTER_TIMEOUT_MS enforced via monotonic deadline (ARGV[29..31]) inside Check, same worker goroutine.
//   - skip_budget (ARGV[24]) and migration/routing epoch barrier (return 11) must not debit past fence.
//
// Forbidden:
//   - Sync XADD in the same Lua script as budget debit when stream deferred to Go producer.
//   - Postgres writes on synchronous Check path (budget miss may read registry snapshot only).
//
// Verify:
// go test ./internal/filter/... -short -count=1
// go test ./internal/ingest/ -short -run TestUnifiedFilter_RollbackDebit -count=1
// go test ./internal/ingest/ -short -run TestUnifiedFilter_SetDeferStreamToProducer -count=1
// go test ./internal/ingest/ -short -run TestStreamProducerAdmissionRaceWithoutReserve -count=1
// go test ./internal/ingest/ -short -run TestTryAcquireStreamAdmission_holdoutDeferredNoPublisher -count=1
//
// Tradeoffs:
//   - Lua numeric return codes vs string errors on Redis wire: ints keep script fast and branch tables
//     small in Go; mapping to filterRejectKind happens once in unified_eval.go. Rejected Redis error
//     strings or JSON payloads from Lua (extra parse + alloc on hot path).
//   - Return code semantics (unified-filter.lua): -1 missing budget key (fail closed, no debit);
//     0 full accept; 2 duplicate after gates (dedup SET NX, no second debit); 3 insufficient budget;
//     4 daily pacing; 5 frequency cap; 6 TTC below minimum; 7 TTC missing when fail-closed; 10 TTC
//     bypass (missing impression ts, fail-open accept); 11 routing epoch / migration fence (no debit past
//     fence); 20 degraded accept when remaining monotonic deadline < degrade_ns. Codes 4-7 reject before
//     debit; 20 skips pacing/fcap/TTC side effects but still debits when budget path runs (load shed).
//   - degrade_ns default 2ms (ARGV[31], 2000000 ns): when FILTER_TIMEOUT_MS budget is nearly exhausted,
//     Lua returns 20 and skips expensive pacing/fcap/TTC branches to bound tail latency inside one EVALSHA.
//     Rejected: separate short Lua script per tier (double RTT); rejected wall-clock deadline in Lua
//     (ARGV uses monotonic ns only); rejected raising degrade_ns above ~2ms (p99 Redis Lua budget in
//     core.mdc); rejected disabling degrade (Redis thread holds under pacing/fcap at timeout tail).
//   - Local quanta full-skip vs sync EVALSHA: eligible traffic debits in Go TrySpendDebit (~16 ns) with
//     zero sync EVALSHA; crash before async publish is eventually consistent via local-quota-refill.lua /
//     local-quota-return.lua. Rejected: always sync Lua (p99 Redis); rejected full-skip without refill
//     path (budget invariant drift). LOCAL_QUOTA_MODE=live gates eligibility in localQuantaFullSkipEligible.
//   - fcap:ignored stream deferral (SetDeferStreamToProducer): Lua KEYS[9] set to fcap:ignored so script
//     does not XADD; StreamProducer or BrokerProducer is sole CH/broker writer. Prevents dual XADD when
//     Go producer also publishes (holdout TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix).
//     Rejected: Lua XADD in same script as debit (holds Redis thread, tradeoffs.mdc); rejected deferred
//     mode without wired producer (503 filterRejectInfra before debit;
//     TestTryAcquireStreamAdmission_holdoutDeferredNoPublisher).
//   - Rollback path: TryReserve on stream/broker before Lua debit; post-debit enqueue failure calls
//     RollbackDebit -> budget-rollback.lua on Redis path or local quanta ledger refund on full-skip path
//     (~200 ms rollback timeout). Rejected: accept without rollback on producer overload (orphan debit);
//     rejected debit-after-enqueue ordering (admission race holdouts). Monitor
//     ad_stream_producer_post_debit_rejected_total ~0.
//   - Go-local prechecks (schedule, geo floor, ingress RPD, ttc_in_go) before EVALSHA when configured:
//     trims Redis work for known rejects; rejected moving all gates into Lua only (larger script, no mmap
//     geo); rejected removing prechecks entirely (higher Redis CPU per reject).
package unified

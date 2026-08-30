// Package dedup adapts pkg/dedupkey scopes to Postgres dedup_claim_confirm and sync_idempotency.
//
// Role:
//   - Adapter.ClaimConfirm calls dedup_claim_confirm for multi-region proposal/confirm dedup.
//   - Adapter.RecordApply and NeedsResumeApply gate sync_idempotency rows after side effects apply.
//   - LoadRoutingEpoch reads MAX(epoch_id) from control_plane_epochs for routing epoch bootstrap.
//   - RegionScope builds dedupkey.Scope from region code and adapter source epoch.
//
// Topology:
//   - Cold path only: regionproxy ingest batches, controlplane region outbox relay, shardadmin lease
//     workers, processor broker consumer, domain.SyncWorker spend flush, stream/broker consumers.
//   - Nil pool on Adapter is a no-op (ClaimConfirm returns OutcomeConfirmed; RecordApply skipped).
//
// Invariants:
//   - Claim outcomes: confirmed, already_confirmed, hash_mismatch, rejected, pending.
//   - ShouldApply is true for confirmed and already_confirmed only.
//   - GuardOutcome maps hash_mismatch and rejected to ErrHashMismatch and ErrRejected.
//   - Dedup keys use pkg/dedupkey canonical formatting; replay of confirmed scope is idempotent.
//
// Forbidden:
//   - Synchronous dedup_claim_confirm on tracker /track or filter hot path.
//   - Partial apply after hash_mismatch or rejected outcome (callers must GuardOutcome first).
//
// Defaults and limits:
//   - Metrics: ad_event_processor_dedup_proposal_total, dedup_confirm_latency, dedup_mismatch_total.
//
// Verify:
//
//	go test ./internal/dedup/ -short -run TestDedupFormatKey_SQLGoldenVector -count=1
//	go test ./internal/dedup/ -run TestDedupClaimConfirm_GoMatchesPGFormat -count=1
package dedup

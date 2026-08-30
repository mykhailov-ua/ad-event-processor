// Package governance owns quota manager ticks, quota repair outbox applier, spend-cap checks, and CTV settlement helpers.
//
// Role:
//   - QuotaManager polls Postgres quota rows and applies Redis chunk refills when threshold crossed.
//   - QuotaRepairRunner applies outbox quota_repair events; spend_cap.go enforces media-buyer caps before spend paths.
//
// Topology:
//   - Wired via governance_bridge.go; Host supplies Redis shards, config knobs, and audit hooks.
//   - reconciliation package calls quota repair tick; team budget approval remains in controlplane.
//
// Invariants:
//   - Quota refill chunks respect domain.QuotaRepo limits; empty chunk skipped without error storm.
//   - Outbox quota repair ordering preserved (quota_repair_ordering tests).
//   - Spend cap reject returns validation error before ledger debit.
//
// Forbidden:
//   - Hot-path unified filter imports.
//   - O(N) quota DB scans per request (batch poll only).
//
// Verify:
//
//	go test ./internal/governance/ -short -count=1
//	go test ./internal/governance/ -short -run TestQuota -count=1
package governance

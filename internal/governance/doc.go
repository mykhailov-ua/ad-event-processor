// Package governance owns quota manager ticks, quota repair outbox applier, spend-cap checks, and CTV settlement helpers.
//
// Role:
//   - QuotaManager polls Postgres quota rows and applies Redis chunk refills when threshold crossed.
//   - QuotaRepairRunner reconciles drift, enqueues quota_repair outbox rows, and monitors shard quorum.
//   - OutboxWorker.ApplyQuotaRepair applies quota_repair payloads from the control-plane outbox.
//   - spend_cap.go enforces media-buyer caps before campaign budget increases.
//   - ApplyCTVSettlement posts CTV settlement spend to Postgres via domain/budget helpers.
//
// Topology:
//   - Wired via controlplane/governance_bridge.go; Host supplies Redis shards, config knobs, and audit hooks.
//   - reconciliation.ReconWorker calls QuotaRepairRunner.ReconcileQuotas on its tick.
//   - Team budget approval and RBAC remain in controlplane (team_budget_approval.go).
//
// Invariants:
//   - Quota refill chunks respect domain.QuotaRepo limits; empty chunk skipped without error storm.
//   - Outbox quota repair ordering preserved (TestApplyQuotaRepair_topUpRedis_*).
//   - Spend cap reject returns validation error before ledger debit.
//   - Quota repair Redis top-up is idempotent per outbox event ID.
//
// Forbidden:
//   - internal/ingest tracker hot-path imports (quota state is async refill/repair only).
//   - O(N) quota DB scans per request (batch poll and repair ticks only).
//
// Verify:
//
//	go test ./internal/governance/ -short -count=1
//	go test ./internal/governance/ -short -run TestQuotaManager -count=1
//	go test ./internal/governance/ -short -run TestApplyQuotaRepair -count=1
//	go test ./internal/governance/ -short -run TestCheckMediaBuyerBudgetCap -count=1
package governance

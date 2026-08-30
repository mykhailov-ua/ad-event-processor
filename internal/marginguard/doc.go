// Package marginguard owns margin-guard policy CRUD, breach reads, and placement pause outbox enqueue.
//
// Role:
//   - Store (store.go): Postgres margin_guard_policies and margin_guard_activity; sqlc margin window sums
//     (SumCampaignMarginWindow*) for 1 h rolling breach detection vs cost_over_revenue_threshold_bps.
//   - HTTPHandlers (http_handlers.go): GET/POST /api/v1/margin-guard/policies; GET /api/v1/margin-guard/activity;
//     POST /api/v1/margin-guard/overrides (remove placement override).
//   - AttachCampaignListMarginBreach decorates campaign list DTOs; GetCampaignMargin returns window snapshot.
//   - BlockCampaignPlacement enqueues PAUSE_PLACEMENT outbox_events (cold-path apply, not synchronous filter).
//   - Host port supplies DefaultCostOverRevenueThresholdBps (controlplane Service implements Host).
//
// Topology:
//   - Wired from controlplane marginguard_bridge.go and adminapi_wire_domains.go.
//   - Background margin evaluation and activity rows also run in internal/ledger Worker when margin-guard
//     sidecar or ledger tick is enabled (ledger_margin_batch.go); this package is store + admin HTTP only.
//   - License gate: licensing entitlements MarginGuard SKU flag.
//
// Invariants:
//   - Default cost-over-revenue threshold 500 bps when policy unset (ledger.CostOverRevenueLimitMicro).
//   - MarginBreach when rtb_cost_micro > limit and advertiser_spend_micro > 0 in the window.
//   - PAUSE_PLACEMENT outbox insert is the only automatic enforcement path from Store; no direct budget_limit mutation.
//
// Forbidden:
//   - Tracker hot path (internal/ingest) imports.
//   - Synchronous placement pause on /track (outbox worker applies pause).
//
// Verify:
//
//	go list -e ./internal/marginguard/...
//	go test ./internal/controlplane/ -short -run TestFault_MarginGuardPause -count=1
package marginguard

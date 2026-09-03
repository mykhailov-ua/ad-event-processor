// Package campaign owns campaign admin DTOs, HTTP handlers, lifecycle helpers, and
// orchestration ports for publish, import/export, and fraud preview on cmd/control.
//
// Role:
//   - CampaignsHTTPHandlers (handlers.go): GET/PATCH /api/v1/campaigns/{id}, list with
//     customer_id/status/q/pacing_mode sort; GET /api/v1/campaigns/target-countries for filter
//     options; owner assign; events, margin, placement-blocks,
//     clone, export, import; migration and conversion-mapping routes via bundle registrars.
//   - Publish routes (publish_bundle.go): POST /api/v1/campaigns/{id}/publish and publish-check.
//   - Fraud preview (fraud_bundle.go): POST /api/v1/campaigns/{id}/fraud/preview.
//   - PostbackHTTPHandlers (misc_helpers.go): postback config, DLQ, campaign status probe.
//   - lifecycle.go: CancelCampaign / FinalizeCancelledCampaign with outbox in same PG txn.
//   - runtime subpackage: PG list/get/patch/publish without HTTP (Effects delegation).
//   - worker subpackage: schedule, delivery optimizer, autoscale, drain ticks (see worker/doc.go).
//   - editor, wizard, integration, selfserve, importexport subpackages register extra routes
//     via route_registrars.go init hooks (each has doc.go).
//
// Topology:
//   - Host/Effects ports (effects_hosts.go) implemented by controlplane bridges
//     (campaign_*_bridge.go, campaign_delivery_bridge.go).
//   - Imports reports, reportjob, flow, controlplane/authz; must not import controlplane root
//     (acyclic composition).
//   - Report validation jobs use reportjob.ReportJobRunner on handlers when wired.
//   - Fraud campaign config reads integrate fraudadmin types; enforcement stays cold path.
//
// Invariants:
//   - Publish, budget, and status mutations enqueue outbox_events in the same PG transaction
//     as domain row updates (via Effects / lifecycle helpers).
//   - Media-buyer scope via authz snapshot and AssertMediaBuyerCampaignAccess on Effects paths.
//   - Cold-path JSON bodies use pkg/coldpath limits on handlers.
//   - List search/sort/pacing filters run in SQL (management.sql ListCampaigns/CountCampaigns).
//   - Status chip totals: CountCampaignsStatusTotals (single GROUP BY status query).
//   - Metric list sort (clicks, impressions, conversions) uses ListCampaignsSortedByStats (hash join on pre-aggregated campaign_stats).
//   - Extended metric sort aggregates customer-scoped PG stats/margin in one query each, then CH sort metrics in one query per chunk.
//   - Filename convention: no campaign_ prefix on files inside this directory (naming.mdc).
//
// Forbidden:
//   - Tracker hot path (internal/ingest), UnifiedFilter, or filter engine imports.
//   - Direct Redis writes from handlers (Effects/outbox/bridge only).
//
// Verify:
//
//	go list -e ./internal/campaign/...
//	go test ./internal/campaign/ -short -run 'TestApplyCampaignStatusCount|TestCampaignListExtendedSortMaxKeys' -count=1
//	go test ./internal/campaign/editor/ -short -run TestValidateCampaignPatch -count=1
//	go test ./internal/campaign/worker/ -short -count=1
package campaign

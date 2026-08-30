// Package clickhouse builds ClickHouse SQL for admin reports and maps rows to handler DTOs.
//
// Role:
//   - Query helpers: placement/keyword stats, IVT rates, spend velocity, daypart heatmap, geo-device, discrepancy, true ROI.
//   - campaign.go: campaign economics CH reads, Telegram export rows, daily event totals keyed by campaign+day.
//   - dimensions.go: coalesce column vs JSONExtractString dimension expressions shared with fraud reports.
//   - math.go: ROI and CPA helpers for report row shaping (no HTTP).
//   - Parent reports package re-exports selected entry points via clickhouse_exports.go.
//
// Topology:
//   - Called from internal/reports handlers and export writers; receives *database.ClickHouseQuery from controlplane.
//   - PG pool used only to list customer campaign IDs (ListCustomerCampaignIDs) before CH IN (?) filters.
//
// Invariants:
//   - Read-only CH via database.ClickHouseQuery; ClickHouseQueryContext applies ReportClickHouseQueryTimeout (10s).
//   - Default lookback ParseReportRange: 7 days when from/to omitted.
//   - Dimension SQL prefers typed CH columns over payload JSONExtract when both exist (holdout in dimensions_test.go).
//   - No balance_ledger reads; financial truth stays in Postgres billing paths.
//
// Forbidden:
//   - HTTP handlers or authz in this package.
//   - PG writes or outbox side effects.
//
// Verify:
// go list -e ./internal/reports/clickhouse/
// go test ./internal/reports/clickhouse/ -short -count=1
// go test ./internal/reports/clickhouse/ -short -run TestPlacementReportQuery_usesColumnPredicate_holdout -count=1
// go test ./internal/reports/clickhouse/ -short -run TestKeywordReportQuery_prefersDimensionColumns_holdout -count=1
package clickhouse

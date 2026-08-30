// Package reports serves synchronous report HTTP and ClickHouse query orchestration.
//
// Role:
//   - ReportsHTTPHandlers (handlers*.go): /api/v1/reports/* catalog routes, /api/v1/campaigns/{id}/stats,
//     POST /api/v1/forecast/campaign, GET /api/v1/reports/edge-parity.
//   - Freshness metadata (stale=true) when CH lag exceeds policy; PG fallbacks where catalog specifies.
//   - Subpackages: clickhouse (query builders), fraud (IVT/silent-reject routes), export (async CSV/ZIP writers),
//     views (/api/v1/views CRUD via ViewsHTTPHandlers, wired separately in adminapi_wire_domains.go).
//   - Fraud routes registered via reports.SetFraudRegistrar; blank import _ "reports/fraud" in controlplane/register.go.
//
// Topology:
//   - ClickHouseQuery and auth callbacks injected from controlplane Service (reports_bridge.go).
//   - reportjob calls reports/export hooks for job output; reports must not import reportjob (one-way).
//   - Fraud export API surface wired through fraud_exports.go SetFraudExports for report jobs.
//
// Invariants:
//   - Campaign-scoped reports call AuthorizeCampaignAccess; customer reports call AuthorizeCustomerAccess.
//   - Fraud evidence routes blocked for masked API key actors (reports/fraud RBAC tests).
//   - Compare/previous-period query params validated in wrapReport helpers.
//   - Silent-reject analytics use silent_reject_event column semantics (not legacy ghost_event keys).
//
// Forbidden:
//   - balance_ledger as analytics source (use billingadmin for money truth).
//   - ghost_* report keys in new code (canonical silent_reject_* wire names).
//   - Import of internal/reportjob (async jobs own polling; export hooks only).
//
// Verify:
//
//	go test ./internal/reports/ -short -count=1
//	go test ./internal/reports/fraud/ -short -count=1
//	go test ./internal/reports/ -short -run TestFilterReportCatalog -count=1
//	go test ./internal/reports/ -short -run TestReports_Placements -count=1
//	go test ./internal/reports/views/ -short -run TestValidateSavedViewActorPolicy -count=1
package reports

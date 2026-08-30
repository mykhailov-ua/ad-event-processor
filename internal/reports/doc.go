// Package reports serves synchronous report HTTP and ClickHouse query orchestration.
//
// Role:
//   - ReportsHTTPHandlers: /api/v1/reports/*, /api/v1/campaigns/{id}/stats, POST /api/v1/forecast/campaign.
//   - Freshness metadata (stale=true) when CH lag exceeds policy; PG fallbacks where catalog specifies.
//   - Subpackages: clickhouse (query builders), fraud (IVT/silent-reject routes), export (async CSV/ZIP writers), views (saved views).
//   - Fraud registrar hooked via reports.SetFraudRegistrar; blank import _ "reports/fraud" in controlplane register.go.
//
// Topology:
//   - ClickHouseQuery injected from controlplane Service; edge metrics panel via opsadmin adapter.
//   - reportjob calls reports export hooks for job output; reports must not import reportjob (one-way).
//
// Invariants:
//   - Campaign-scoped reports call AuthorizeCampaignAccess; customer reports call AuthorizeCustomerAccess.
//   - Fraud evidence routes blocked for masked API key actors.
//   - Compare/previous-period query params validated in wrapReport helpers.
//
// Forbidden:
//   - balance_ledger as analytics source (use billingadmin for money truth).
//   - Ghost_* report keys in new code (canonical silent_reject_* wire names).
//
// Verify:
//
//	go test ./internal/reports/ -short -count=1
//	go test ./internal/reports/fraud/ -short -count=1
package reports

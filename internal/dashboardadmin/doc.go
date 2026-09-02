// Package dashboardadmin serves role-specific dashboard HTTP and ClickHouse-backed KPI panels.
//
// Role:
//   - HTTPHandlers under /api/v1/dashboards/* (operator, campaign, buyer, adops, cfo, accountant, fraud).
//   - PublisherHTTPHandlers under /api/v1/publisher/dashboard and /api/v1/publisher/statements.
//   - RoleService and portfolio.go assemble buyer portfolio, billing, fraud ML snapshot, and edge metrics per role.
//   - portfolio_commercial.go and portfolio_recommendations.go derive utilization, overspend alerts, and recommendation cards.
//
// Topology:
//   - Wired via dashboardadmin_bridge.go; RoleHost, RoleReportHost, CampaignHost, PortfolioHost, and PublisherHost ports read PG, CH, and ledger via injected deps.
//   - Uses internal/reports scrub helpers for masked campaign reads; publisher routes scope by customer permission.
//
// Invariants:
//   - Masked API keys use campaigns:read:masked permission branch on campaign dashboard route.
//   - CH queries respect ReportCHTimeout from Host; empty campaign scope returns zeroed metrics not errors.
//   - Billing invariant block surfaces ledger drift without mutating balances from dashboard handlers.
//
// Forbidden:
//   - Spend mutation or budget PATCH from dashboard handlers (read-only aggregates).
//   - Tracker hot path imports.
//
// Verify:
//
//	go test ./internal/dashboardadmin/ -short -count=1
//	go test ./internal/dashboardadmin/ -short -run TestGetOperatorDashboard -count=1
//	go test ./internal/dashboardadmin/ -short -run TestGetBuyerDashboard -count=1
//	go test ./internal/dashboardadmin/ -short -run TestGetBuyerDrilldown -count=1
//	go test ./internal/reports/ -short -run TestParseDashboardDrilldownDimension -count=1
package dashboardadmin

// Package dashboardadmin serves role-specific dashboard HTTP and ClickHouse-backed KPI panels.
//
// Role:
//   - HTTP under /api/v1/dashboards/* (operator, campaign, buyer, adops, cfo, accountant, fraud) and /api/v1/publisher/*.
//   - portfolio.go and service_role.go assemble billing, fraud ML snapshot, and edge metrics blocks per role.
//
// Topology:
//   - Wired via dashboardadmin_bridge.go; RoleHost and RoleReportHost ports read PG, CH, and ledger via injected deps.
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
//	go test ./internal/dashboardadmin/ -short -run TestOperator -count=1
package dashboardadmin

// Package views provides saved report view CRUD and schedule validation helpers.
//
// Role:
//   - HTTP: /api/v1/views CRUD registered as ViewsHTTPHandlers from controlplane wire.
//   - ValidateReportScheduleForActor used by reportjob schedule handlers for RBAC + license gates.
//
// Invariants:
//   - View JSON schema validated before insert; customer scope from AuthorizeCustomerAccess.
//
// Verify:
//
//	go test ./internal/reports/views/ -short -count=1
package views

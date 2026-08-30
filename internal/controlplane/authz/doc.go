// Package authz holds RBAC permission constants, policy snapshots, and request context helpers.
//
// Role:
//   - Snapshot from context: Has/HasAny permission checks for handlers and commandpalette nav filter.
//   - PermCampaigns*, PermBilling*, scope (global/customer/team), mask level (full/masked).
//   - Policy loaded from roles YAML + DB overrides (policy_db.go); attached by adminauth middleware.
//
// Invariants:
//   - Snapshot is immutable per request; handlers must not mutate Permissions map.
//   - Wildcard permission "*" grants all checks in Snapshot.Has.
//
// Forbidden:
//   - HTTP handlers in this package (middleware lives in adminauth / controlplane).
//
// Verify:
//
//	go test ./internal/controlplane/authz/ -short -count=1
package authz

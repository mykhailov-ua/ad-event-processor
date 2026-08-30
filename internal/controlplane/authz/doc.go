// Package authz holds RBAC permission constants, policy snapshots, and request context helpers.
//
// Role:
//   - Snapshot per request: Has/HasAny permission checks for handlers, reports catalog, and commandpalette nav filter.
//   - Core Perm* constants (campaigns, billing, blacklist); extended operator matrix lives in control/http/rbac.go.
//   - Scope (global/customer/team), mask level (full/masked), AuthenticatedUser (user_context.go).
//   - Store merges embedded roles, deploy/operator/roles.yaml (roles_yaml.go), and per-user DB grants (policy_db.go).
//   - Attached by adminauth middleware after PASETO or API-key authentication.
//
// Invariants:
//   - Snapshot is immutable per request; handlers must not mutate Permissions map.
//   - Wildcard permission "*" grants all checks in Snapshot.Has.
//   - MaskLevelFromPermissions: campaigns:read or "*" -> MaskFull; otherwise MaskMasked.
//
// Forbidden:
//   - HTTP handlers or mux registration in this package (middleware lives in adminauth; routes in controlplane/domains).
//
// Verify:
//
//	go test ./internal/controlplane/authz/ -short -count=1
//	go test ./internal/controlplane/authz/ -short -run 'TestMaskLevelFromPermissions|TestPolicyPermissionMatrix' -count=1
package authz

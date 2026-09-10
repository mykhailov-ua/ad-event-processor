// Package access implements the RBAC role constructor: capability catalog,
// YAML compile/validate, atomic roles file persistence, and HTTP handlers.
//
// Role:
// - Cold-path admin API under /api/v1/access/*.
// - Source of truth: OPERATOR_ROLES_YAML (default deploy/operator/roles.yaml).
// - Capabilities compile to wire permissions; authz.Store is updated on apply.
//
// Invariants:
// - Invalid YAML must not mutate disk or in-memory policy (holdout tests).
// - Apply uses atomic rename + single .bak backup.
// - Built-in role A cannot be deleted; wildcard * rejected unless allow_wildcard.
//
// Forbidden:
// - Hot-path imports from tracker ingest.
// - Per-user grant logic (v1 uses users.role only).
//
// Verify:
// go test ./internal/access/ -short -count=1
// go test ./internal/access/ -short -run TestAccessApply_invalidYAMLDoesNotMutateStore_holdout -count=1
// go test ./internal/access/ -short -run TestAccess_customRole_deniedPostbackWrite_holdout -count=1
package access

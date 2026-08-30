// Package domains holds platform custom domain export and DNS verification helpers for platformadmin.
//
// Role:
//   - domains_export.go builds domain manifest for edge TLS provisioning.
//   - Verification state read from PG; mutations remain in platformadmin store.
//
// Topology:
//   - Subpackage of platformadmin; imported by platformadmin workers and handlers only.
//
// Invariants:
//   - Export omits unverified domains when require_verified flag set at caller.
//
// Forbidden:
//   - Hot-path imports.
//
// Verify:
//
//	go test ./internal/platformadmin/domains/... -short -count=1
package domains

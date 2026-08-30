// Package migrationsource lists embedded SQL migration directories for multi-schema cold-path bootstrap.
//
// Role:
//   - Returns ordered migration paths for pkg/coldpath.ApplyTrackedSchemaMigrations per service schema.
//   - Used by cmd/migrate-cold-path and control OpenModule startup.
//
// Topology:
//   - No I/O in package; embed or static path tables only.
//
// Invariants:
//   - Migration order stable; duplicate version numbers rejected at apply time.
//
// Forbidden:
//   - Runtime migration from hot-path binaries.
//
// Verify:
//
//	go test ./internal/migrationsource/... -short -count=1
package migrationsource

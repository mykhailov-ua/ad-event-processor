// Package platformadmin owns platform config, customers, team, session, and public activation HTTP.
//
// Role:
//   - HTTP under /api/v1/settings/platform*, /api/v1/customers/*, /api/v1/team/*, /api/v1/session, /api/v1/meta, /api/v1/public/*, /api/v1/support/*, /api/v1/platform-campaigns/*.
//   - store.go reads/writes platformconfig via pkg/platformconfig; workers handle nginx sync, audit export, vendor telemetry.
//
// Topology:
//   - Wired via platform_bridge.go; Host port supplies install token verify, audit writers, and edge expose sync.
//   - domains/ subpackage holds domain-export helpers; governance hooks for budget approval live beside customers HTTP.
//
// Invariants:
//   - Bootstrap requires valid install token; apply writes install.yaml and may flag restart_required fields.
//   - Customer and team mutations audit through Host; public activate/invite routes are rate-limited without session.
//   - Platform patch merges with existing platformconfig; invalid keys rejected before PG write.
//
// Forbidden:
//   - Tracker hot path imports.
//   - Per-request platformconfig load on /track (snapshots only on hot path).
//
// Verify:
//
//	go test ./internal/platformadmin/ -short -count=1
package platformadmin

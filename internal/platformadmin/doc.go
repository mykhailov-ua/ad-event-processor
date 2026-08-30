// Package platformadmin owns platform config, customers, team, session, and public activation HTTP.
//
// Role:
//   - HTTP under /api/v1/settings/platform*, /api/v1/customers/*, /api/v1/team/*, /api/v1/session,
//     /api/v1/meta, /api/v1/public/*, /api/v1/support/*, /api/v1/platform-campaigns/*.
//   - domains/ subpackage registers /api/v1/domains/* and /api/v1/ops/domains/* (see domains/doc.go).
//   - store.go reads/writes platformconfig via pkg/platformconfig; Bootstrap/Update/Apply call Host audit hooks
//     and SyncEdgeExpose after PG write.
//   - campaign_sync_handlers.go enqueues platformsync mutations and wires platformsync.Worker for vendor sync.
//   - Workers: NginxWorker (Redis blacklist export), AuditExportWorker (CSV retention), vendor probe worker,
//     telemetry pulse, SystemStateWorker (Redis system state), domains.StartDomainHealthWorker (via bridge).
//   - governance.go and team_members_handlers.go implement budget approval invite/approve/deny flows.
//   - list.go provides server-side ListEnvelope pagination helpers for customer list routes.
//
// Topology:
//   - Wired via platform_bridge.go; Host port supplies install token verify, audit writers, and edge expose sync.
//   - Imports billingadmin and controlplane/authz; must not import controlplane root (no import cycle).
//
// Invariants:
//   - Bootstrap requires valid install token; apply writes install.yaml and may flag restart_required fields.
//   - Customer and team mutations audit through Host; public activate/invite routes are rate-limited without session.
//   - Platform patch merges with existing platformconfig; invalid keys rejected before PG write.
//   - Campaign sync preview runs dry-run via platformsync.PreviewMutation before enqueueing pending mutations.
//
// Forbidden:
//   - Tracker hot path imports.
//   - Per-request platformconfig load on /track (snapshots only on hot path).
//
// Verify:
//
//	go list -e ./internal/platformadmin/
//	go test ./internal/platformadmin/ -short -count=1
//	go test ./internal/platformadmin/ -short -run TestInviteToken_roundTrip -count=1
//	go test ./internal/platformadmin/ -short -run TestRedactJSONPII_masksEmailAndIP -count=1
//	go test ./internal/platformadmin/domains/ -short -run TestCloudflareClient_ListZones -count=1
package platformadmin

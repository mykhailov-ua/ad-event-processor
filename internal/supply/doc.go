// Package supply owns sellers.json, ads.txt chain validation, and supply file export for the admin API.
//
// Role:
//   - HTTP under /api/v1/supply/* (sellers, ads.txt entries, validation, export).
//   - worker_audit.go audits supply file changes; export builds on-disk artifacts for edge/nginx serve.
//
// Topology:
//   - Wired via supply_admin_adapter.go; Host supplies audit logging and EnqueueSupplyFilesUpdate outbox hook.
//   - ValidateSupplyFiles checks chain integrity before publish; BuildSellersJSON and BuildAdsTxt are read paths for export.
//
// Invariants:
//   - ads.txt DIRECT/RESELLER lines must pass validate.go chain rules.
//   - Supply export path comes from Host.SupplyExportPath; writes are atomic rename at caller.
//   - Mutations audit with actor user id from Host.ActorUserID.
//
// Forbidden:
//   - Hot-path ingest imports; supply files are cold-path artifacts.
//
// Verify:
//
//	go test ./internal/supply/ -short -count=1
package supply

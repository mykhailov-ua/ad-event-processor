// Package importexport implements campaign export bundles and migration import validation jobs.
//
// Role:
//   - ExportCampaign / ImportCampaign orchestration using campaign.ImportExportHost port.
//   - Migration source adapters (migrationsource) and validation job enqueue via reportjob when async.
//   - No HTTP routes here; handlers in campaign/handlers.go call these functions.
//
// Invariants:
//   - Export bundles carry ExportVersion; import rejects unknown versions fail closed.
//   - Validation jobs respect media-buyer campaign access same as export.
//
// Verify:
//
//	go test ./internal/campaign/importexport/ -short -count=1
package importexport

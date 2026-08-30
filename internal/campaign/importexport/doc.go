// Package importexport implements campaign export bundles, JSON import, and migration validation output.
//
// Role:
//   - ExportCampaign and ImportCampaign assemble CampaignExportBundle (campaign, flow, postback,
//     conversion mappings, integration schema names) via campaign.ImportExportHost.
//   - WriteCampaignImportValidationJSON runs migrationsource.Preview for reportjob async validation jobs.
//   - No HTTP routes; parent campaign/handlers.go and runtime.Runtime call these entrypoints.
//
// Topology:
//   - ImportExportHost implemented by controlplane Service (campaign_import_bridge.go).
//   - Runtime.ExportCampaign / ImportCampaign delegate here through Effects.CampaignImportExportHost().
//   - Validation jobs: reportjob.ReportJobSpec with ReportKey campaign_import_validation; HTTP enqueue
//     in parent import_validate_job_http_test.go / reportjob runner (not in this package).
//
// Invariants:
//   - Export rejects deleted campaigns and enforces AssertMediaBuyerCampaignAccess on export.
//   - Import requires customer_id, idempotency key, positive budget_limit_micro, and matching export_version.
//   - Unknown export_version fails closed with validation error (no partial PG insert).
//   - Import PG work runs in a transaction; flow lander/offer refs upserted by name+URL.
//
// Forbidden:
//   - HTTP transport, tracker ingest (internal/ingest), direct Redis writes.
//
// Defaults and limits:
//   - CampaignExportVersion: 1 (also on parent campaign.CampaignExportVersion).
//   - MaxCampaignImportBytes: 65536 (64 KiB) for marshaled export/import bundle JSON.
//   - migrationsource.MaxPayloadBytes: 1048576 (1 MiB) for validation job import_payload (separate limit).
//
// Verify:
//
//	go test ./internal/campaign/importexport/ -short -run TestWriteCampaignImportValidationJSON -count=1
//	go test ./internal/campaign/importexport/ -short -run TestCampaignImportValidateJob -count=1
package importexport

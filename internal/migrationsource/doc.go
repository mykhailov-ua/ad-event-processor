// Package migrationsource parses and previews third-party tracker exports for campaign import.
//
// Role:
//   - Source kinds (types.go): keitaro_json, keitaro_admin_api, binom_json, binom_report_api, native_v1.
//   - Parse/Preview (adapter.go, preview.go): normalize vendor payloads to MappedCampaign/MappedFlow DTOs
//     with macro mapping (macro_mapper.go) and traffic-source schema resolution (schema_resolver.go).
//   - Maps (maps.go): YAML macro/source tables from deploy/vendor/migration (or install root override).
//   - FetchRemotePayload (pull.go): live pull for Keitaro admin API and Binom report API (30 s timeout).
//   - Transform helpers (transform.go, payload_decode.go, keitaro_streams.go) for flow paths and warnings.
//
// Topology:
//   - Called from controlplane campaign_import_bridge.go and campaign import/export validation jobs.
//   - No Postgres writes in package; preview-only mapping returned to admin import handlers.
//   - MaxPayloadBytes = 1 MiB per preview/parse request.
//
// Invariants:
//   - Unsupported source_kind returns error at Parse; native_v1 uses separate campaign export path.
//   - Secrets stripped count surfaced in PreviewResult; unmapped macros emit Warning slugs.
//   - Budget fields mapped from vendor cost/spend columns must not be confused (holdout tests).
//
// Forbidden:
//   - Tracker hot path imports.
//   - SQL schema migration listing (see pkg/coldpath.ApplyTrackedSchemaMigrations and cmd/migrate-cold-path).
//
// Verify:
//
//	go test ./internal/migrationsource/ -short -run TestPreviewKeitaro_streamsMultiPath_holdout -count=1
//	go test ./internal/migrationsource/ -short -run TestParseKeitaroJSON_adminAPIWire_holdout -count=1
//	go test ./internal/migrationsource/ -short -run TestParseBinomReportAPI_spendNotBudget_holdout -count=1
package migrationsource

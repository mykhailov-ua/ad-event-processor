// Package licensingadmin serves license status/apply HTTP, EULA acceptance, admin feature gates,
// and the license revoke-queue worker.
//
// Role:
//   - handlers.go: GET /api/v1/license/status, POST /api/v1/license/apply.
//   - eula_handlers.go / eula.go: GET /api/v1/eula, POST /api/v1/eula/accept (any authenticated session).
//   - gate.go: FeatureAllowed and RequireLicenseFeature for admin route middleware (403 feature_required).
//   - service.go: ApplyLicenseToken (VerifyJWTResolved, CheckHostActivation, InstallToken, ReloadLicense).
//   - worker.go: RevokeQueueWorker polls Postgres revoke queue and reloads when row matches active license key.
//
// Topology:
//   - Wired via controlplane/licensingadmin_bridge.go and adminapi_wire.go / adminapi_wire_domains.go.
//   - Host port supplies pool, ReloadLicense, DeploymentLimits, FeatureAllowed, audit, and EULA persistence.
//   - Uses internal/licensing facade; pkg/branding for support URLs on status DTO.
//
// Invariants:
//   - Apply validates JWT signature and host activation before writing AD_EVENT_PROCESSOR_LICENSE_PATH.
//   - GET /api/v1/license/status returns UNCONFIGURED when billing.license_status row is missing.
//   - Nil FeatureChecker leaves gated routes open; wired checker fails closed with 403 when feature denied.
//   - EULA accept records legal.Version once per deployment in system_settings.
//   - Deployment caps (CapHost): 0 = unlimited for max_tenants, max_api_keys, max_events_per_month; max_export_chunk_bytes 0 uses default chunk size.
//
// Limit enforcement map:
//   - max_tenants: EnforceDeploymentTenantCap -> CreateCustomer, ActivateOwner -> 429 LIMIT_EXCEEDED
//   - max_api_keys: EnforceDeploymentAPIKeyCap -> identity.CreateAPIKey -> 429 LIMIT_EXCEEDED
//   - max_active_campaigns: EnforceDeploymentCampaignCap -> CreateCampaign -> 429 LIMIT_EXCEEDED
//   - max_regions: EnforceDeploymentRegionCap -> control serve multi_region startup -> error
//   - max_export_chunk_bytes: ExportChunkMaxBytes chunk sizing only (0 = default 10 MiB); tier no longer disables exports
//   - max_events_per_month: EnforceDeploymentMonthlyEventsCap (cold); LicenseMonthlyEventsFilter (hot snapshot)
//
// Forbidden:
//   - License server ping on appliance (file-based JWT mode only).
//   - Per-request JWT crypto on tracker hot path.
//
// Verify:
//
//	go test ./internal/licensingadmin/ -short -count=1
//	go test ./internal/licensingadmin/ -short -run TestLicenseFeatureAllowed -count=1
//	go test ./internal/licensingadmin/ -short -run TestEnforceDeploymentExportAllowed -count=1
package licensingadmin

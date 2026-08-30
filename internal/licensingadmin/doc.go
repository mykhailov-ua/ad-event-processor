// Package licensingadmin serves license status/apply HTTP, EULA acceptance, admin feature gates,
// and the license revoke-queue worker.
//
// Role:
//   - handlers.go: GET /api/v1/license/status, POST /api/v1/license/apply.
//   - eula_handlers.go / eula.go: GET /api/v1/eula, POST /api/v1/eula/accept (system_settings row).
//   - gate.go: FeatureAllowed and RequireLicenseFeature for admin route middleware (403 feature_required).
//   - service.go: ApplyLicenseToken (VerifyJWTResolved, CheckHostActivation, InstallToken, ReloadLicense).
//   - worker.go: RevokeQueueWorker polls PG revoke queue and reloads when row matches active license key.
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
//
// Forbidden:
//   - License server ping on appliance (file-based JWT mode only).
//   - Per-request JWT crypto on tracker hot path.
//
// Verify:
//
//	go test ./internal/licensingadmin/ -short -count=1
//	go test ./internal/licensingadmin/ -short -run TestLicenseFeatureAllowed -count=1
//	go test ./internal/licensingadmin/ -short -run TestRevokeQueue -count=1
package licensingadmin

// Package entitlements maps JWT and deployment claims to feature flags, tier limits,
// license state, ingest gates, and license-epoch pubsub.
//
// Role:
//   - entitlements.go / jwt_claims.go: Limits, FeatureSet, LicenseClaims, and admin DTOs.
//   - effective.go: Effective merges deployment and customer entitlements (min non-zero limits, AND features).
//   - tier_policy.go / sku_yaml.go: SanitizeFeaturesForSKU, OpenRTBAllowed, deploy/vendor/sku.yaml loader.
//   - state.go / heartbeat_policy.go: DetermineState, DetermineEffectiveState, IngestAllowed, banner severity.
//   - ingest_gate.go: IngestAllowed (P-C3-03) - false only for EXPIRED or REVOKED.
//   - deployment_gate.go: LoadDeploymentSnapshot from billing.license_status (cold path).
//   - license_epoch_pubsub.go / epoch_gate.go: Redis license:epoch fan-out and local invalidation.
//   - volume.go / tier_usage.go: billable units, volume bands, renewal warnings.
//   - vendor_revoke.go: PG vendor revoke lookup; skew_watch*.go: clock-skew watchdog (linux).
//
// Topology:
//   - Consumed by internal/licensing root facade, internal/licensing/verify (EntitlementsFromClaims),
//     and controlplane license watcher snapshot reload.
//   - Tracker EntitlementsFilter (internal/filter) reads registry snapshot built from these types;
//     per-customer ingress RPD uses Redis INCR in the filter, not PG in this package.
//
// Invariants:
//   - Effective customer limits and features never exceed deployment ceiling (P-C4-03; verify/properties_test.go).
//   - IngestAllowed false only for StateExpired and StateRevoked (licensing.mdc P-C3-03).
//   - SanitizeFeaturesForSKU enforces SKU tier feature matrix before claims reach Effective.
//
// Forbidden:
//   - Per-request Postgres fetch for entitlements on tracker hot path.
//
// Verify:
//
//	go test ./internal/licensing/entitlements/ -short -run 'TestIngestAllowed|TestDetermineEffectiveState|TestSanitizeFeaturesForSKU' -count=1
//	go test ./internal/licensing/verify/ -short -run TestProperty_P_C4_03_DeploymentCeiling -count=1
package entitlements

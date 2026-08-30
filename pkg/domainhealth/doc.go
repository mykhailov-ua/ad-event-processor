// Package domainhealth probes tracking domain TLS, DNS, and HTTP health for operator dashboards.
//
// Role:
//   - Probe runs timed GET against tracking domain; classifies healthy/degraded/down and SSL state.
//   - reputation.go scores domain reputation signals for supply and ops panels.
//
// Topology:
//   - Called from opsadmin/platform readers; uses pkg/branding and pkg/platformconfig for URL defaults.
//   - Outbound HTTP from cold path only.
//
// Invariants:
//   - Probe context timeout bounded; errors map to HealthUnknown not panic.
//   - SSL expiring threshold documented in probe constants.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/domainhealth/... -short -count=1
package domainhealth

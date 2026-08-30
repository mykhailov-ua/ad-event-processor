// Package doctor runs install health probes and exposes ops doctor HTTP checklist.
//
// Role:
//   - HTTP GET /api/v1/ops/doctor returns component checklist (license, redis, ch, rtb, slotmap, xdp).
//   - Probes read local state only: tcp listen counters, slotmap parity, xdp attach, bundle manifest.
//
// Topology:
//   - Wired into opsadmin handler registration; DoctorHTTPHandlers receive probe funcs from control bootstrap.
//   - remediation.go maps failed check ids to operator remediation strings (no auto-fix).
//
// Invariants:
//   - Doctor endpoint read-only; no mutation side effects on GET.
//   - Probe failures aggregate; overall status worst-of components.
//   - Requires shards:read permission.
//
// Forbidden:
//   - Auto-apply license or nginx reload from doctor handlers.
//   - Hot-path imports.
//
// Verify:
//
//	go test ./internal/doctor/ -short -count=1
//	go test ./internal/doctor/ -short -run TestChecklist -count=1
package doctor

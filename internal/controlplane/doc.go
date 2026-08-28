// Package controlplane is the modular monolith composition root for cmd/control:
// Service, outbox workers, billing hooks, and *_bridge.go wiring to domain packages.
//
// Domain HTTP lives in internal/campaign, brand, flow, fraudadmin, etc. Bridges
// implement domain Host/Effects ports; they must not accumulate business rules
// (modular-monolith.mdc).
//
// Routes: /api/v1/* on :8188. Config mutations enqueue outbox_events in the same
// PG transaction as the domain change.
//
// Verify:
//   go test ./internal/controlplane/ -short -count=1
//   bash scripts/ci/cold_path_static_gate.sh
//   bash scripts/ci/anti_slop_gate.sh
//
// Must NOT be imported by internal/ingestion.
//
package controlplane

// Package control boots the control-plane modular monolith for cmd/control.
//
// Role:
//   - Load config, optional license guard watchdog CLI, then Run or RunCLI into controlplane.ServeWithOptions.
//   - Toggle in-process modules via Options (auth, management, payment, billing, notifier, margin-guard, cost-sync).
//   - Optional sidecars in the same process: margin-guard ledger worker, costsync worker, platform-campaign-sync.
//
// Topology:
//   - Management gateway: internal/controlplane on MANAGEMENT_PORT default 8188 (/api/v1, GET /metrics).
//   - Module wiring: buildServeOptions opens identity, payment, ledger/billing, notifier pools and StartWorkers hooks.
//   - StartControlServers from this package supplies TCP/UDP control publishers when management is enabled.
//
// Invariants:
//   - Run blocks on management server until context cancel when Management is true.
//   - Margin-guard and cost-sync paths use their own PG pools; they do not serve admin HTTP.
//
// Forbidden:
//   - Business rules in this package; domain logic stays in internal/<domain>/ and controlplane bridges.
//
// Env defaults:
//   - Module flags from config.Control.Enable* (see config/env_controlplane.go).
//
// Verify:
//
//	go test ./internal/control/ -short -count=1
//	go build -o bin/control ./cmd/control/
package control

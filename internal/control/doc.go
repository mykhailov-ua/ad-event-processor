// Package control boots the control-plane modular monolith for cmd/control.
//
// Role:
//   - RunCLI / RunFromCLI: config.Load, licensing guard watchdog, runtime policy init, then Run.
//   - Run: optional in-process modules (identity, payment, ledger, notifier) via buildServeOptions;
//     controlplane.ServeWithOptions when Management enabled.
//   - Sidecar goroutines in same process: margin-guard (ingest registry + CH), cost-sync, platform-campaign-sync.
//   - StartControlServers: optional TCP/UDP control publishers for tracker shard config push.
//   - ProbeHealth: --health-probe URL exit hook for container orchestration.
//
// Topology:
//   - Management gateway: controlplane on MANAGEMENT_PORT default 8188 (/api/v1, GET /metrics).
//   - HTTP middleware and auth live in internal/control/http (subpackage); wired from controlplane/serve.go.
//   - serveMarginGuard uses internal/ingest.Registry sync; not tracker gnet ingress.
//   - OptionsFromConfig maps cfg.Control.Enable* to Options toggles.
//
// Invariants:
//   - Run blocks on management server until context cancel when Management is true.
//   - Margin-guard and cost-sync use dedicated PG pools; they do not serve admin HTTP.
//   - Module cleanups run in reverse order on shutdown when auth/billing/payment/notifier opened.
//   - InitRuntimePolicy loads control_runtime_policy.json; dev mode may use embed fallback.
//
// Env defaults:
//   - Module flags from config.Control.EnableAuth, EnableManagement, EnablePayment, EnableBilling,
//     EnableNotifier, EnableMarginGuard, EnableCostSync, EnablePlatformCampaignSync.
//   - TCP/UDP control servers gated by cfg.TCPControlEnabled and cfg.UDPControlEnabled.
//
// Forbidden:
//   - Business rules in this package; domain logic stays in internal/<domain>/ and controlplane bridges.
//   - Tracker /track handler wiring (cmd/tracker only).
//
// Verify:
//
//	go test ./internal/control/ -short -count=1
//	go test ./internal/control/ -short -run TestRuntimePolicy_devModeUsesEmbed -count=1
//	go test ./internal/control/ -short -run TestFault_TCP_SnapshotHMACACK -count=1
//	go list -e ./cmd/control/
package control

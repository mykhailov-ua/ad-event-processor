// Package main boots the control plane modular monolith (cmd/control).
//
// Role:
//   - Optional ProbeHealth CLI exit before boot; licensing.MaybeRunGuardWatchdogCLI for garbled release guard.
//   - Default path: internal/control.RunCLI -> controlplane.ServeWithOptions on MANAGEMENT_PORT.
//
// Topology:
//   - Admin /api/v1/* and GET /metrics on MANAGEMENT_PORT default 8188 (same listener).
//   - Payment webhooks on PAYMENT_WEBHOOK_PORT default 8187 when payment module enabled.
//   - In-process workers: outbox (~20 ms poll), billing, payment settlement outbox, notifier, domain ticks.
//   - Admin static stub from internal/controlplane/admin_static_stub (web/ rebuild pending).
//
// Invariants:
//   - Admin mutation + outbox_events in same PG transaction (enforced in domain stores).
//   - Cold-path body limit 64 KiB (pkg/coldpath.DefaultMaxBody).
//   - Tracker is separate; control must not run FilterEngine on /track.
//
// Forbidden:
//   - Direct Redis config writes from HTTP handlers (outbox appliers only).
//
// Verify:
//
//	go test ./internal/controlplane/ -short -count=1
//	bash scripts/ci/admin/openapi.sh
//	go build -o bin/control ./cmd/control/
package main

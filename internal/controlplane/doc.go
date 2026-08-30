// Package controlplane is the admin API composition root for cmd/control.
//
// Role:
//   - Wire domain HTTP handlers (campaign, reports, billingadmin, fraudadmin, …) via RouteRegistry.
//   - Own Service: Postgres mutations, lazy domain stores, Redis read helpers, audit/outbox enqueue.
//   - Start background workers (outbox, recon, campaign delivery ticks, report jobs) from serve.go.
//   - Serve management HTTP on MANAGEMENT_PORT (default 8188): /api/v1/*, GET /metrics, admin static stub.
//
// Topology:
//   - HTTP: opsadmin.RegisterOpsRoutes registers GET /metrics on the same mux as /api/v1 (no separate metrics port).
//   - Optional CONTROL_UNIX_SOCKET serves the same handler tree as TCP management.
//   - Config side effects: outbox_events row + domain PG change in one transaction; OutboxWorker applies Redis.
//   - Bridges (*_bridge.go) implement domain Host/Effects ports; no domain SQL or validation in bridges.
//   - Payment webhooks listen on PAYMENT_WEBHOOK_PORT (default 8187) inside internal/payment when enabled.
//
// Routes:
//   - /api/v1/* — operators and automation (catalog in routecatalog).
//   - /api/v1/selfserve/* — advertiser API keys (campaign/selfserve).
//   - /admin/* — HTTP 410 Gone; do not extend legacy HTMX surface.
//
// Workers (control process only; never tracker):
//   - OutboxWorker (internal/outbox): poll 20 ms active, idle backoff to 250 ms; PROCESSING reclaim after 1 min.
//   - RegionOutboxRelay on multi-region cells; skipped on global coordinator.
//   - Recon, volume meter, ledger invariant, slot migration, shard orchestrator, report job/schedule workers.
//   - campaign/worker ticks: schedule, delivery optimizer, autoscale, drain outbox merge.
//
// Invariants:
//   - Admin mutation that changes tracker-visible config must enqueue outbox_events in the same PG txn.
//   - Handlers must not write Redis config keys directly; outbox appliers own Redis side effects.
//   - Cold-path bodies: pkg/coldpath.DefaultMaxBody 64 KiB unless a narrower limit applies.
//   - Tracker is a separate binary; controlplane must not run FilterEngine on /track.
//
// Forbidden:
//   - New domain handler bodies here when internal/<domain>/ exists (use bridge wiring only).
//   - *_aliases.go, service_<domain>_*.go growth, bridges > 200 lines with business rules.
//   - Import internal/ingest hot handlers or internal/fraud scoring into request paths.
//
// Defaults and limits:
//   - MANAGEMENT_PORT default 8188; PAYMENT_WEBHOOK_PORT default 8187.
//   - Outbox poll: outbox.PollActiveInterval 20 ms; outbox.PollIdleMax 250 ms; outbox.WorkerTimeout 30 s.
//   - RECON_WORKER_INTERVAL_MS default 3_600_000 (1 h).
//   - LEDGER_INVARIANT_INTERVAL_HOURS default 24.
//
// Verify:
//
//	go test ./internal/controlplane/ -short -count=1
//	bash scripts/ci/admin/openapi.sh
//	bash scripts/ci/static/cold_path_static.sh
package controlplane

// Package outbox polls public.outbox_events and applies Redis side effects for config propagation.
//
// Role:
//   - Worker claims rows with FOR UPDATE SKIP LOCKED, marks PROCESSING, dispatches by event type.
//   - RegionOutboxRelay forwards events from regional cells to global coordinator when licensed.
//   - Blacklist batch applier and campaign/settings/fraud/RTB payload handlers live in worker.go.
//
// Topology:
//   - Host port implemented by controlplane.Service via outbox_bridge.go (OutboxInfraHost, OutboxApplyHost).
//   - Started from controlplane.startBuiltinServiceWorkers at 20 ms initial timer (PollActiveInterval).
//   - Recovery ticker runs stale PROCESSING reclaim every 5× poll interval.
//
// Invariants:
//   - Duplicate event IDs must not double-apply Redis mutations (idempotent appliers, ON CONFLICT upstream).
//   - Handler failure leaves row retryable; permanent failures mark failed and emit ops metrics.
//   - PROCESSING rows older than 1 minute reset to PENDING (reclaimStaleProcessing).
//   - Per-iteration worker context timeout: WorkerTimeout 30 s.
//
// Defaults and limits:
//   - PollActiveInterval 20 ms when work found; idle exponential backoff capped at PollIdleMax 250 ms.
//   - ProcessOutboxWithCount batch limit 1000 rows per claim.
//   - Alert: ad_management_outbox_oldest_pending_seconds > 30 s (ops scrape).
//
// Forbidden:
//   - Running OutboxWorker inside cmd/tracker (config reaches tracker via Redis pub/sub only).
//   - LISTEN/NOTIFY driven dispatch (polling only; see tradeoffs.mdc).
//
// Verify:
//
//	go test ./internal/outbox/ -short -count=1
//	go test ./internal/controlplane/ -short -run Outbox -count=1
package outbox

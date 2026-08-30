// Package outbox polls public.outbox_events and applies Redis side effects for config propagation.
//
// Role:
//   - Worker claims rows with FOR UPDATE SKIP LOCKED, marks PROCESSING, dispatches by event type.
//   - RegionOutboxRelay forwards events from regional cells to global coordinator when licensed.
//   - Blacklist batch applier and campaign/settings/fraud/RTB payload handlers live in worker.go.
//
// Topology:
//   - Host port implemented by controlplane.Service via outbox_bridge.go (OutboxInfraHost, OutboxApplyHost).
//   - Started from controlplane.startBuiltinServiceWorkers: OutboxWorker and RegionOutboxRelay at 20 ms timer seed.
//   - Recovery ticker runs stale PROCESSING reclaim every 5x current poll interval.
//
// Poll intervals (public.outbox_events):
//   - PollActiveInterval 20 ms: baseline after work found; PollBackoff.Next resets idle step to this value.
//   - Work found (processed > 0): Next returns 0 so the loop repolls immediately (timer Reset(0)).
//   - Idle backoff: idle step doubles each empty iteration, capped at PollIdleMax 250 ms.
//   - Iteration error (not shutdown): fixed 2 s retry before next claim.
//   - Metric: metrics.OutboxPollIntervalMs records chosen delay per tick.
//
// Invariants:
//   - Duplicate event IDs must not double-apply Redis mutations (idempotent appliers, ON CONFLICT upstream).
//   - Handler failure leaves row retryable; permanent failures mark failed and emit ops metrics.
//   - PROCESSING rows with processing_started_at older than 1 minute reset to PENDING (reclaimStaleProcessing).
//   - Per-iteration worker context timeout: WorkerTimeout 30 s.
//
// Defaults and limits:
//   - ProcessOutboxWithCount batch limit 1000 rows per claim.
//   - Alert: ad_management_outbox_oldest_pending_seconds > 30 s (ops scrape).
//
// Forbidden:
//   - Running OutboxWorker inside cmd/tracker (config reaches tracker via Redis pub/sub only).
//   - LISTEN/NOTIFY driven dispatch (polling only; see tradeoffs.mdc).
//   - Confusing this loop with payment.payment_outbox (100 ms poll in internal/payment/settlement).
//
// Verify:
//
//	go list -e ./internal/outbox/
//	go test ./internal/outbox/ -list '.*'
//	go test ./internal/outbox/ -short -run TestPollBackoff_ActiveThenIdle -count=1
//	go test ./internal/outbox/ -short -run TestWorker_strictOrder -count=1
//	go test ./internal/controlplane/ -short -run Outbox -count=1
package outbox

// Package smartalerts owns smart alert rule CRUD, evaluation worker, and webhook drain.
//
// Role:
//   - HTTP under /api/v1/smart-alerts/rules and /api/v1/smart-alerts/events.
//   - worker_batch.go evaluates rules against CH/metrics windows; drain.go clears stuck firing state.
//
// Topology:
//   - Wired via smartalerts_bridge.go; Store persists rules and event history in Postgres.
//   - Webhook delivery uses outbound HTTP from worker tick, not synchronous on rule write.
//
// Invariants:
//   - Rules scoped per customer_id; campaign_id optional filter on metric evaluation.
//   - Ack endpoint records actor id and timestamp; duplicate ack is idempotent.
//   - Enabled=false rules skipped by worker without deleting history.
//
// Forbidden:
//   - Alert evaluation on tracker request path.
//   - Unbounded goroutine per webhook (batch worker only).
//
// Verify:
//
//	go test ./internal/smartalerts/ -short -count=1
package smartalerts

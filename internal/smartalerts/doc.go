// Package smartalerts owns smart alert rule CRUD, CH-backed evaluation worker, and webhook delivery.
//
// Role:
//   - HTTP under /api/v1/smart-alerts/rules, /api/v1/smart-alerts/history,
//     POST /api/v1/smart-alerts/events/{id}/ack.
//   - worker_batch.go batches CH metric windows per customer/campaign; store.go persists rules and fired events.
//   - drain.go (CheckStuckDrainJobs) alerts on stuck redis_slot_migration drain rows via Host.AlertDrainStuck.
//
// Topology:
//   - Wired via controlplane/settingsadmin_bridge.go; Store and Worker use smartalertsHost port.
//   - Webhook POST runs from worker tick (deliverWebhook), not on rule write HTTP path.
//   - Valid metrics: clicks, cr, roi_pct, bot_clicks; window clamped 5-1440 minutes.
//
// Invariants:
//   - Rules scoped per customer_id; optional campaign_id filters evaluation to one campaign.
//   - Ack updates alert_rule_events only when acked_at IS NULL; repeat ack returns error (no double-apply).
//   - Enabled=false rules skipped by worker without deleting history.
//   - One firing per rule per window_start (existing event lookup before insert).
//
// Forbidden:
//   - Alert evaluation on tracker request path.
//   - Unbounded goroutine per webhook (batch worker tick only).
//
// Verify:
//
//	go test ./internal/smartalerts/ -short -count=1
//	go test ./internal/smartalerts/ -short -run TestAlertThresholdBreached -count=1
package smartalerts

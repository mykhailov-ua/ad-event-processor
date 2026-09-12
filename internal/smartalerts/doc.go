// Package smartalerts owns smart alert rule CRUD, ClickHouse-backed evaluation worker, and webhook delivery.
//
// Role:
//   - HTTP under /api/v1/smart-alerts/rules, /api/v1/smart-alerts/history,
//     POST /api/v1/smart-alerts/events/{id}/ack.
//   - worker_batch.go batches ClickHouse metric windows per customer/campaign; store.go persists rules and fired events.
//   - worker_templates.go evaluates template:* rules via fixed PG/CH queries (M5 lite).
//   - action_apply.go applies margin_breach rule actions (notify, pause_campaign, blacklist_placement)
//     through Host.PauseCampaign and Host.BlacklistPlacement (same paths as automation executor).
//   - drain.go (CheckStuckDrainJobs) alerts on stuck redis_slot_migration drain rows via Host.AlertDrainStuck.
//
// Topology:
//   - Wired via controlplane/settingsadmin_bridge.go; Store and Worker use smartalertsHost port.
//   - Webhook POST runs from worker tick (deliverWebhook), not on rule write HTTP path.
//   - Valid metrics: clicks, cr, roi_pct, bot_clicks; template:* metrics for M5 lite templates.
//
// Invariants:
//   - Rules scoped per customer_id; optional campaign_id filters evaluation to one campaign.
//   - Ack updates alert_rule_events only when acked_at IS NULL; repeat ack returns error (no double-apply).
//   - Enabled=false rules skipped by worker without deleting history.
//   - One firing per rule per window_start (existing event lookup before insert).
//   - Template create path rejects arbitrary metric/operator from client when template is set.
//   - pause_campaign and blacklist_placement actions are allowed only for margin_breach template rules.
//
// Forbidden:
//   - Alert evaluation on tracker request path.
//   - Unbounded goroutine per webhook (batch worker tick only).
//   - Client-supplied SQL or arbitrary metric names on template create path.
//
// Verify:
//
//	go test ./internal/smartalerts/ -short -count=1
//	go test ./internal/smartalerts/ -short -run 'TestResolveTemplate|TestResolveUpsert' -count=1
package smartalerts

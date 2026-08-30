// Package automation owns campaign automation rule CRUD, preset catalog, ClickHouse
// metric evaluation, and background rule firing for the control plane.
//
// Role:
//   - HTTP (handlers.go): GET /api/v1/automation/presets; GET/POST /api/v1/automation/rules;
//     PUT/DELETE /api/v1/automation/rules/{id}; POST /api/v1/automation/rules/{id}/dry-run.
//   - RulesService persists rules in Postgres (sqlc automation_rules) and validates presets,
//     metrics, operators, and actions.
//   - Worker (worker.go) polls enabled rules on a ticker, evaluates CH windows (eval.go),
//     dedupes fires via ActionHash + automation_rule_fires, and applies actions through Executor.
//   - Dry-run and HTTP read paths never insert fire rows or call Executor mutations.
//
// Topology:
//   - Wired from controlplane adminapi_wire_domains.go; RulesService and HTTPHandlers on Service.
//   - Worker started by controlplane.Service.StartAutomationWorker when AUTOMATION_RULES_ENABLED.
//   - ExecutorHost (executor.go) delegates pause/blacklist to Service; platform_pause enqueues
//     platform_campaign_mutations via sqlc.
//   - worker_tick.go records LastWorkerTick for ops health only; evaluation lives in worker.go.
//
// Invariants:
//   - Eval interval must be one of 5, 10, 15, 30, 60 minutes and >= license eval floor
//     (RulesService.evalFloorMinutes, default 15; clamp 5-60).
//   - Window minutes clamp 15-1440; default 60 when unset on create. Cooldown clamp 15-10080;
//     default 60 when unset.
//   - Action hash idempotency: duplicate (rule, campaign, placement, window_end, action) skips
//     second fire in the same window.
//   - Cooldown suppresses re-fire until CooldownMinutes elapsed since last_fired_at.
//   - Per-customer eval cap per worker tick (default 50) skips excess rules until next tick.
//   - blacklist_placement requires group_by placement_id; platform_pause requires network and
//     ad_platform_campaign_api license when LicenseGate is wired.
//   - CH queries use 15 s timeout per eval window.
//
// Defaults and limits:
//   - Worker ticker interval: AUTOMATION_RULES_INTERVAL_MIN (default 15), clamped 5-60 min in
//     NewWorker; skipped when ClickHouse query client is nil.
//   - AUTOMATION_RULES_MAX_EVALS_PER_CUSTOMER_PER_TICK default 50 (clamp 1-500 in config).
//   - Webhook notify timeout 45 s; User-Agent via pkg/branding.
//   - Request bodies limited to pkg/coldpath.DefaultMaxBody (64 KiB).
//
// Forbidden:
//   - Tracker hot path (internal/ingest) imports.
//   - Postgres/Redis mutation on dry-run or rules CRUD without going through sqlc paths above.
//
// Verify:
//
//	go list -e ./internal/automation/...
//	go test ./internal/automation/ -short -run TestNormalizeEvalIntervalMinutes -count=1
//	go test ./internal/automation/ -short -run TestExpandPreset -count=1
//	go test ./internal/automation/ -short -run TestThresholdBreached -count=1
package automation

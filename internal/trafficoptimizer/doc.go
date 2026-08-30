// Package trafficoptimizer evaluates ClickHouse bandit stats and applies flow or creative weight patches on the control-plane cold path.
//
// Role:
//   - HTTP under /api/v1/traffic-optimizer/* for preset listing, rule CRUD, and dry-run.
//   - rules_service.go persists rules in Postgres; worker.go ticks due rules and calls optimize_tx.go / creative_tx.go.
//   - applier.go delegates Thompson and proportional updates to internal/flow bandit helpers.
//
// Topology:
//   - Wired from controlplane via trafficoptimizer_bridge.go; Host port supplies CH queries, bandit stats, and brand-creative outbox enqueue.
//   - PublishHost.PublishCampaignUpdate triggers catalog reload after successful apply.
//   - Tracker reads flow/creative weights from Redis/catalog snapshots only after outbox reload.
//
// Invariants:
//   - Rule apply is transactional; partial flow or creative weight update rolls back on PG error.
//   - RuleSupported gates scope/objective/algorithm pairs (EPC requires proportional; creative ROI requires brand scope).
//   - Disabled rules, cooldown windows, and eval-interval floors are skipped by the worker tick.
//   - Empty CH stats or unsupported rule shape yields no-op apply, not an error storm.
//
// Forbidden:
//   - Weight mutation on /track synchronous path.
//   - Import internal/ingest hot handlers.
//
// Verify:
//
//	go test ./internal/trafficoptimizer/ -short -count=1
//	go test ./internal/trafficoptimizer/ -short -run TestRuleSupported_holdout -count=1
//	go test ./internal/trafficoptimizer/ -short -run TestApplyRuleTx_holdout -count=1
//	go test ./internal/trafficoptimizer/ -short -run TestWorker_ruleOnCooldown_holdout -count=1
package trafficoptimizer

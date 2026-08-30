// Package trafficoptimizer evaluates ClickHouse stats and updates flow or creative weights on the control-plane cold path.
//
// Role:
//   - HTTP under /api/v1/traffic-optimizer/* for rule CRUD and manual optimize triggers.
//   - rules_service.go and applier.go evaluate CH windows; optimize_tx.go writes weight patches transactionally.
//
// Topology:
//   - Wired from controlplane; tracker reads weights from Redis/catalog snapshots only after outbox reload.
//   - Host port supplies campaign scope, audit, and publish reload hooks.
//
// Invariants:
//   - Optimize apply is transactional; rollback on partial flow/creative update failure.
//   - Rules require explicit metric, operator, and threshold; disabled rules skipped by worker.
//   - CH query timeout bounded; empty stats yield no-op apply not error storm.
//
// Forbidden:
//   - Weight mutation on /track synchronous path.
//   - Import internal/ingest handlers.
//
// Verify:
//
//	go test ./internal/trafficoptimizer/ -short -count=1
//	go test ./internal/trafficoptimizer/ -short -run TestRuleApply -count=1
package trafficoptimizer

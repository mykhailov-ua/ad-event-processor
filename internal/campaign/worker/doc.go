// Package worker runs campaign background ticks wired from controlplane.Service.
//
// Role:
//   - ScheduleWorker: cron-like campaign schedule transitions.
//   - Delivery optimizer + closed-loop pacing + autoscale budget ticks (RunDeliveryOptimizerTick).
//   - MAB / flow bandit hooks and brand delivery outbox merge (delivery_tick.go).
//   - DrainWorker: merges pending delivery outbox rows every 20 ms (controlplane workers.go).
//
// Topology:
//   - DeliveryHost port implemented by controlplane campaign_delivery_bridge.go.
//   - Uses domain.SyncWorker slice for post-tick Redis budget sync on tracker shards.
//
// Invariants:
//   - Delivery tick PG batch timeout 2 min per RunDeliveryOptimizerTick.
//   - Ticks run on worker goroutines started at control boot; no per-request goroutine spawn.
//
// Forbidden:
//   - HTTP handlers in this package (campaign handlers only).
//
// Verify:
//
//	go test ./internal/campaign/worker/ -short -count=1
package worker

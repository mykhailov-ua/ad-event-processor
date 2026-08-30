// Package worker runs campaign background ticks wired from controlplane.Service.
//
// Role:
//   - Worker: schedule transitions (ProcessScheduleTick) via campaign/runtime and DeliveryHost effects.
//   - Loop workers (loops.go): ScheduleWorker (1 min), PacingControllerWorker, AutoscaleBudgetWorker,
//     DeliveryOptimizerWorker; intervals from config via serve_workers.go.
//   - RunDeliveryOptimizerTick: closed-loop pacing, autoscale budgets, optional MAB + flow bandit in one PG txn.
//   - RunVPPPacingController: VPP ratio writes to Redis shards for campaigns in pacing_mode=vpp.
//   - DrainWorker: finalizes campaigns in draining status; started every 20 ms from controlplane/workers.go.
//
// Topology:
//   - DeliveryHost (delivery_hosts.go) implemented by controlplane campaign_delivery_bridge.go.
//   - Worker constructed in campaign_worker_bridge.go; LoopHost satisfied by *controlplane.Service.
//   - DeliveryOutboxMerge coalesces pacing/autoscale/sync events per campaign before outbox flush.
//   - domain.SyncWorker slice passed into pacing/delivery ticks for post-tick Redis budget sync on tracker shards.
//
// Invariants:
//   - Schedule tick: batch timeout 2 min; at most 200 campaigns claimed per tick.
//   - Delivery optimizer tick: batch timeout 2 min; MAB sub-tick when MABInterval elapsed (default 15 min).
//   - VPP pacing batch timeout 2 min; ratios pipelined per Redis shard.
//   - Drain batch timeout 30 s; uses WithPostgresHigh gate on DrainHost.
//   - Ticks run on worker goroutines started at control boot; no per-request goroutine spawn.
//   - Outbox merge priority: brand creatives (1) < create campaign (2) < pacing (3) < pause (4).
//
// Forbidden:
//   - HTTP handlers in this package (campaign and wizard packages own routes).
//   - Tracker hot-path imports or synchronous /track work.
//
// Verify:
//
//	go test ./internal/campaign/worker/ -short -count=1
//	go test ./internal/campaign/worker/ -short -run TestDeliveryOutboxMerge_priority -count=1
//	go test ./internal/campaign/worker/ -short -run TestComputeMABWeights_proportionalCTR -count=1
//	go test ./internal/campaign/worker/ -short -run TestPipelineWriteVPPRatios_batchesPerShard -count=1
package worker

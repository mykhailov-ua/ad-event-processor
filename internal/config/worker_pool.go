package config

import "runtime"

// Worker pool capacity defaults (must match internal/ingest/gnet/arena.go and pools.go).
const (
	DefaultWorkerPoolQueueDepth     = 8192
	DefaultWorkerArenaSlots         = 4
	DefaultWorkerArenaSlotBytes     = 1 << 20
	DefaultWorkerMaxPoolObjectBytes = 64 * 1024
	DefaultPostgresPoolConnHeadroom = 20
)

// WorkerPoolConfig is the canonical PinnedWorkerPool sizing surface for tracker wire.
//
// Capacity model (operator tuning):
//
//	in_flight ~= min(queue_depth, workers) concurrent offloads per worker queue;
//	arena_slots_per_worker x arena_slot_bytes bounds zero-alloc wire copies;
//	when arena slots are busy, requestBufferPool or heap fallback applies (see ad_worker_* metrics).
type WorkerPoolConfig struct {
	Workers            int
	QueueDepth         int
	ArenaSlots         int
	ArenaSlotBytes     int
	MaxPoolObjectBytes int
}

func (c *Config) WorkerPoolConfig() WorkerPoolConfig {
	workers := c.MaxWorkers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
		if workers <= 0 {
			workers = 1
		}
	}
	queue := c.WorkerPoolQueueDepth
	if queue <= 0 {
		queue = DefaultWorkerPoolQueueDepth
	}
	return WorkerPoolConfig{
		Workers:            workers,
		QueueDepth:         queue,
		ArenaSlots:         DefaultWorkerArenaSlots,
		ArenaSlotBytes:     DefaultWorkerArenaSlotBytes,
		MaxPoolObjectBytes: DefaultWorkerMaxPoolObjectBytes,
	}
}

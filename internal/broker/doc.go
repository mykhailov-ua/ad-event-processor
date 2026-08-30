// Package broker implements the mmap WAL ingest broker gnet server, Redis-coordinated HA
// leader election, consumer offset storage, retention, and offline replay tooling.
//
// Role:
//   - server.go: gnet listener for pkg/broker/protocol produce/fetch/metadata; mmap segments
//     via pkg/broker/log; health/metrics HTTP on configurable health addr.
//   - coord.go + registry_redis.go: per-topic leader lease, fencing epoch, topic id registry.
//   - offset_redis.go / offset_store.go: consumer high-water marks (memory default, Redis HA).
//   - replay.go: cutover drills and CH/compare replay from on-disk WAL segments.
//   - retention.go: segment age/size eviction on leader nodes (default check every 5 min).
//
// Topology:
//   - Standalone cmd/broker binary; tracker clients use pkg/broker/client via
//     internal/stream/broker BrokerProducer.
//   - Coordinator Redis URL from -redis-url, BROKER_REDIS_URL, or REDIS_ADDRS; Sentinel via
//     BROKER_REDIS_SENTINEL_MASTER and BROKER_REDIS_SENTINEL_ADDRS.
//   - Default gnet listen 127.0.0.1:9092 (or runtimepaths unix socket); health 127.0.0.1:8084.
//   - Metrics registered through internal/metrics broker log wire helpers.
//
// Invariants:
//   - Only topic leader appends; stale epoch rejected after fencing floor advance
//     (fencing_test, fault_leader_safety_test).
//   - Retention runs only on leader for each topic; FloorOffset from consumer HWM when set.
//   - Admission shedding when connection count exceeds max: produce returns overloaded status
//     (not silent accept).
//   - Disk write gate blocks append when diskOK is false (ENOSPC paths).
//   - Durability tier from log.DurabilityConfig (DefaultDurabilityConfig at server create).
//
// Defaults and limits:
//   - CoordConfig defaults: LeaseTTL 15 s, renew Interval 3 s, RenewFailThreshold 3,
//     DebounceWindow 2 s (coord_config.go).
//   - defaultRetentionCheckInterval 5 min when unset on server.
//   - NewServer default maxSegSize and indexInterval passed from cmd/broker flags.
//   - Replay default batch size 50000 events.
//
// Forbidden:
//   - Budget Lua, UnifiedFilter, or Postgres settlement inside broker server path.
//   - Unbounded in-memory topic registry in HA mode without Redis registry backing.
//
// Verify:
//
//	go list -e ./internal/broker/...
//	go test ./internal/broker/ -short -count=1
//	go test ./internal/broker/ -short -run TestAdmissionShedding_ProduceOverloaded -count=1
//	go test ./internal/broker/ -short -run TestFault_LeaderTakeover_HWMNeverRegresses -count=1
package broker

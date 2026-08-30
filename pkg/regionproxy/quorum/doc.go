// Package quorum implements Redis-backed operation lease book/ack for Enterprise multi-region batches.
//
// Role:
//   - Book, AckBook, ReadStatus, and Transition for 16-byte operation IDs across a replica node list.
//   - Lease hash key ad_event_processor:op:lease:{hex32}; ack members in ad_event_processor:op:lease:ack:{hex32}.
//   - Required(replicaCount) returns 1 when replicaCount <= 1, else defaultQuorum (2).
//
// Topology:
//   - internal/shardadmin OperationLeaseWorker books via host ControlRedis().
//   - pkg/regionproxy/opkey BatchCommitter calls Book/Transition before regional uplink forward (optional).
//   - Pairs with global control operation_leases PG rows and region-proxy uplink batches.
//
// Invariants:
//   - QuorumMet when SCard(ack) >= Required(replicaCount); stored replica_count on the lease hash overrides caller default when present.
//   - Book and AckBook are idempotent adds to the ack set; empty replicaNodes or replicaCount <= 0 treated as single-node (quorum 1).
//   - Transition is compare-and-set on state (booked -> executing -> completed); mismatch returns error without mutating state.
//   - leaseTTL (48h) refreshed on Book/AckBook via EXPIRE on both lease hash and ack set.
//
// Defaults and limits:
//   - States: booked, executing, completed (StateBooked, StateExecuting, StateCompleted).
//   - defaultQuorum 2 for multi-node cells; ParseReplicaCount returns 1 on invalid input.
//
// Tradeoffs:
//   - Redis lease book/ack for sub-ms coordination; Postgres operation_leases remains durable audit, not the hot quorum path.
//   - 2-of-N default instead of full replica ack keeps regional failover RTO bounded (see regions.mdc drill).
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy (hot path boundary).
//
// Verify:
// go test ./pkg/regionproxy/quorum/... -short -run 'TestBook_2of3Quorum|TestTransition_bookedExecutingCompleted' -count=1
// go test ./pkg/regionproxy/opkey/... -short -run TestBatchCommitter_quorumBeforeForward -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package quorum

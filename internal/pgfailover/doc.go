// Package pgfailover coordinates Postgres primary failover for control-plane and processor PG pools.
//
// Role:
//   - Coordinator (coordinator.go) is the Redis-lease leader: health-checks primary DSN, promotes standby
//     via Promoter, bumps fencing epoch, and publishes the active DSN to Redis keys under ad_event_processor:pg:global:*.
//   - Subscriber (subscriber.go) polls and listens on the Redis notify channel; reconnects pgxpool on DSN or epoch change.
//   - StandbyPromoter (promote.go) runs optional snapshot sync (snapshot.go), operator promote command, and writable check.
//   - FencingGate (fencing.go) rejects stale writes with ErrStalePgFencingEpoch after epoch bump.
//   - IngestRuntime (ingest.go) starts subscribers on processor Redis shards for hot-path PG readers (async background only).
//   - audit.go counts duplicate balance_ledger rows after failover for operator audit.
//
// Topology:
//   - Wired from shardadmin.StartPostgresFailover via controlplane serve.go; processor calls StartIngestSubscribers.
//   - Uses broker.Coordinator for single-leader promotion; not Redis Sentinel (Sentinel is a separate Redis HA path).
//   - Not imported on tracker /track handler path.
//
// Invariants:
//   - Fencing epoch monotonic; PublishDSN sets dsn, dsn_epoch, and fencing_epoch atomically in one pipeline.
//   - Reconnect closes the previous pgxpool after the new pool pings (shardadmin reconnect callback).
//   - Coordinator runs at most one executeFailover at a time (atomic failover flag).
//   - Optional SyncSnapshot copies customers and balance_ledger pages before promote when PostgresFailoverSnapshotSync is set.
//
// Forbidden:
//   - Synchronous failover or PG pool swap on /track accept path.
//   - Hot-path ingest imports of controlplane admin handlers.
//
// Verify:
//
//	go list -e ./internal/pgfailover/
//	go test ./internal/pgfailover/ -short -count=1
//	go test ./internal/pgfailover/ -run TestEnsureWritablePrimary -count=1
//	go test ./internal/pgfailover/ -run TestStartIngestSubscribers_reconnectOnPublish -count=1
//	go test ./internal/controlplane/ -run TestFault_PostgresMasterFailover -count=1
package pgfailover

# Redis sharding and shard 0 runbook

**Naming:** **ad-event-processor** stack — [NAMING.md](NAMING.md).

Shard 0 hosts pub/sub control plane, edge blacklist sync source, fraud aggregate writes, and several global-key conventions. Budget traffic on shards 1..N can continue when shard 0 is down if `REDIS_SHARD0_OPTIONAL_STARTUP=1` and the registry is warm. See [ARCHITECTURE.md](ARCHITECTURE.md) section 4.2.

## Goals

- Hot-path SLA for campaigns on shards 1..N (`/track`, budget Lua, streams).
- Fail-closed `503 shard_unavailable` for campaigns mapped to shard 0.
- Control-plane liveness (no crash loops; outbox best-effort fan-out) during shard-0 outage.

## Proof harness

```bash
go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/
```

| Package | File |
| --- | --- |
| `internal/controlplane` | `shard0_nil_spof_test.go` |
| `internal/ingestion` | `shard0_nil_spof_test.go` |

Scenario: `rdbs[0] == nil` with shards 1..N connected (miniredis or `REDIS_SHARD0_OPTIONAL_STARTUP=1`).

## Failure matrix (test-backed)

### Control plane — nil shard 0

| Symptom | Test | Behavior |
| --- | --- | --- |
| SyncWorker panic | `TestShard0Nil_SyncWorkerSyncAllNoOpWithoutPanic` | skip nil in `serve.go` |
| Readiness / shutdown panic | `TestShard0Nil_ReadinessProbeSkipsNilShard0`, `GracefulShutdownSkipsNilShard0` | `pingConnectedRedisShards` / `closeConnectedRedisShards` |
| Consent / purge panic | `TestShard0Nil_SyncUserConsentWritesHealthyShards`, `PurgeUserDataRedisHealthyShards` | skip nil; write shards 1..N |
| Outbox fan-out stall | `TestShard0Nil_OutboxHandleUpdateSettings`, `SyncKeyToAllShardsHealthyShards` | `forEachConnectedShard` skips nil; metric `ad_control_shard_fanout_skipped_total` |

### Tracker — nil shard 0

| Symptom | Test | Behavior |
| --- | --- | --- |
| Health panic | `TestShard0Nil_TrackerHealthSkipsNilShard0` | skip nil in ping |
| Fraud aggregate XADD | `TestShard0Nil_FraudStreamAggregateFlushesHealthyShard` | `firstConnectedRedisShard` |
| Brand / RTB globals | `TestShard0Nil_BrandCreativeStoreLoadsFromHealthyShard`, `DealFloorCacheRefreshFromHealthyShard` | `firstConnectedRedis(rdbs)` in `cmd/tracker/main.go` |
| Campaign on shard 0 slot | Docker `TestFault_Shard0Outage` | `503 shard_unavailable` |

### Edge

| Surface | Fix | Test |
| --- | --- | --- |
| Blacklist sync | `connect_any_shard()` in Lua | `blacklist_sync_test.lua` |
| XDP stats read | `xdpstats.ReadRedisAny` | `snapshot_test.go` |

## Architecture snapshot

```
Control plane ──fan-out──► Redis shard 0 (globals, pub/sub hub)
                         ► shard 1..N (budget + replicated globals)

Tracker: budget on shards 1..N; globals via first connected shard when shard 0 nil.
Nginx edge: blacklist sync from first healthy Redis in REDIS_ADDRS.
```

## Recovery

When shard 0 returns after optional-startup outage:

1. **Automatic:** `Shard0CatchupWorker` copies globals from shard 1, validates blacklist parity, publishes `campaigns:update` full-sync.
2. **Manual:** `POST /api/v1/ops/shards/0/catchup` (`shards:write`).
3. **Verify:** `ad_shard0_catchup_last_success_timestamp`; `GET /api/v1/ops/shards` shows shard 0 synced.

```bash
go test -count=1 -run 'TestShard0Nil_CatchupAfterRecovery|TestShard0CatchupWorker_' ./internal/controlplane/
bash scripts/ci/shard0_nil_gate.sh
```

## Operator checks

| Metric | Meaning |
| --- | --- |
| `ad_shard0_client_nil == 1` | Process running without shard 0 client |
| `ad_shard0_catchup_last_success_timestamp` | Last successful shard-0 reconcile (0 = never) |
| `ad_control_fanout_partial_total` | Admin/outbox wrote globals to subset of shards |
| `ad_registry_stale_mode` | Tracker stale-serve |

Full degradation steps: [DEVELOPMENT.md](DEVELOPMENT.md) (shard 0 degradation runbook).

## References

- [ARCHITECTURE.md](ARCHITECTURE.md) section 4.2
- [DEVELOPMENT.md](DEVELOPMENT.md)
- `tests/resilience/shard_outage_fault_test.go`
- `scripts/ci/check_no_shard0_control.sh`

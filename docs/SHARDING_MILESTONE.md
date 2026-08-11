# Sharding Milestone — Shard 0 SPOF Hardening

Status: **P0–P5 complete** — tracker/control survive `rdbs[0]==nil`; metrics, alerts, and runbook in place.
Last regression run: `go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/` → **PASS** (behavioral checks on Redis side effects).

## Goal

Remove operational dependence on Redis **shard 0** as a single point of failure while preserving:

- Hot-path SLA for campaigns on shards 1..N (`/track`, budget Lua, streams).
- Correct fail-closed behavior for campaigns whose `StaticSlot` maps to shard 0.
- Control-plane liveness (no crash loops, no stuck outbox) during shard-0 outage or `REDIS_SHARD0_OPTIONAL_STARTUP=1`.

Shard 0 today is not only a routing bucket — it is the **designated home** for pub/sub control plane, edge blacklist sync, fraud aggregate writes, and several global-key conventions. See `docs/ARCHITECTURE.md` §4.2 and `.cursor/rules/data-layer.mdc` Part I.

---

## Proof harness

| Package | File | Tests |
| --- | --- | --- |
| `internal/controlplane` | `shard0_nil_spof_test.go` | 24 |
| `internal/ingestion` | `shard0_nil_spof_test.go` | 16 |

**Scenario simulated:** `rdbs[0] == nil` (as after `ConnectRedisShards` with `REDIS_SHARD0_OPTIONAL_STARTUP=1`), shards 1..N connected via miniredis.

```bash
go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/
```

Naming: `TestShard0Nil_*` — behavioral regression (Redis GET/EXISTS/XLen/pubsub). Ingestion still has legacy `TestProof_Shard0Nil_*` for P2/P3 degradation paths.

---

## Failure matrix (test-backed)

### P0 — Process crash / CrashLoopBackOff ✅ fixed

| Component | Symptom | Test | Fix |
| --- | --- | --- | --- |
| `domain.SyncWorker` | panic on tick | `TestShard0Nil_SyncWorkerSyncAllNoOpWithoutPanic` | skip nil in `serve.go`; `SyncAll` guard |
| Control readiness | panic in probe | `TestShard0Nil_ReadinessProbeSkipsNilShard0` | `pingConnectedRedisShards` |
| Control shutdown | panic on exit | `TestShard0Nil_GracefulShutdownSkipsNilShard0` | `closeConnectedRedisShards` |
| `SyncUserConsentToRedis` | panic | `TestShard0Nil_SyncUserConsentWritesHealthyShards` | skip nil; **writes verified on shard 1..2** |
| `PurgeUserDataRedis` | panic | `TestShard0Nil_PurgeUserDataRedisHealthyShards` | skip nil; **key deleted on shard 1** |
| Tracker `/health` | panic | `TestShard0Nil_TrackerHealthSkipsNilShard0` | `pingConnectedRedisShards` |
| `BrokerReconcileWorker` | panic | `TestShard0Nil_BrokerReconcileSampleSkipsNilShard0` | skip nil; **3 XADD entries counted** |

**Impact:** With `REDIS_SHARD0_OPTIONAL_STARTUP=1`, tracker and control stay up; budget shards 1..N serve traffic. See Phase 5 metrics/runbook.

### P1 — Outbox / admin mutation stall ✅ fixed (best-effort fan-out)

Previously returned `redis shard 0 is nil` and blocked the outbox worker. Now `forEachConnectedShard` skips nil/error shards, increments `ad_control_shard_fanout_skipped_total{shard,reason}`, and succeeds when ≥1 shard writes.

| Function | Test | Typical outbox / caller |
| --- | --- | --- |
| `syncKeyToAllShards` | `TestShard0Nil_SyncKeyToAllShardsHealthyShards` | brand creatives |
| `publishCampaignControlToAllShards` | `TestShard0Nil_PublishCampaignControlHealthyShards` | `campaigns:update` |
| `publishControlChannelToAllShards` | `TestShard0Nil_PublishControlChannelHealthyShards` | consent channel |
| `syncGlobalConfigToAllShards` | `TestShard0Nil_SyncGlobalConfigHealthyShards` | `UPDATE_SETTINGS` |
| `OutboxWorker.handleUpdateSettings` | `TestShard0Nil_OutboxHandleUpdateSettings` | outbox integration |
| `setNXOnAllShards` | `TestShard0Nil_SetNXOnAllShardsHealthyShards` | region outbox relay |
| `syncGlobalSetMemberToAllShards` | `TestShard0Nil_SyncGlobalSetMemberHealthyShards` | blacklist IP worker |
| `syncGlobalSetReplaceToAllShards` | `TestShard0Nil_SyncGlobalSetReplaceHealthyShards` | emergency breaker IPs |
| All shards nil | `TestShard0Nil_FanoutAllNilShardsFails` | still errors |

**Impact:** Admin mutations and outbox processing continue on shards 1..N during shard-0 outage. **Caveat:** shard 0 does not receive writes until recovery; run `replicateConfigVersionFromPrimary` / full fan-out catch-up after shard 0 is back (no automated job yet).

**Broker fallback:** `publishCampaignUpdate` with `CAMPAIGN_UPDATE_BROKER_FALLBACK=1` still masks Redis fan-out when broker is up (`TestShard0Nil_PublishCampaignUpdateBrokerFallback`).

### P2 — Silent / partial degradation ✅ fixed (tracker globals)

| Component | Behavior | Test | Fix |
| --- | --- | --- | --- |
| `FraudStreamWriter.flushAggregates` | XADD on first connected shard | `TestShard0Nil_FraudStreamAggregateFlushesHealthyShard` | `firstConnectedRedisShard` |
| `BrandCreativeStore` | loads from healthy shard | `TestShard0Nil_BrandCreativeStoreLoadsFromHealthyShard` | `firstConnectedRedis(rdbs)` in `cmd/tracker/main.go` |
| `DealFloorCache` | refreshes from healthy shard | `TestShard0Nil_DealFloorCacheRefreshFromHealthyShard` | same |
| `StartRtbCatalogReloadWatch` | pub/sub on healthy shard | `TestShard0Nil_RtbCatalogReloadWatchNilRDBNoOp` (nil guard) | `firstConnectedRedis(rdbs)` in main |
| `pickSegmentShard` | skips nil slots | `TestShard0Nil_PickSegmentShardSkipsNil` | linear probe from hash |
| Global hash/delete fan-out | writes shards 1..N only | `DeleteGlobalKeySkipsNil`, … | P1; shard 0 catch-up manual |
| Campaigns on slot → shard 0 | `503 shard_unavailable` | Docker fault test | expected |

### P3 — Already resilient (keep behavior, add regression tests)

| Component | Test | Mechanism |
| --- | --- | --- |
| `SettingsWatcher` | `SettingsWatcherPickHealthyShardSkipsNil` | `pickHealthyShard` starts at index 1 |
| `FraudBlacklistFilter` / placement blacklist | `PickLocalGlobalShardSkipsNil` | `pickLocalGlobalShard` |
| `Registry` pub/sub | `RegistryStartWatchShardsNoPanic` | `if rdb == nil { continue }` |
| `readFraudAggForce` | `ReadFraudAggForceSkipsNil` | skip nil in loop |
| Consent wiring | `ConsentStoreUsesHealthyShard`, `FirstConnectedRedisSkipsNil` | `firstConnectedRedis` in `cmd/tracker/main.go` |
| Auth revocation | `MiddlewareRevocationSkipsNil` | `middleware.go` + `identity/revocation.go` skip nil |
| `replicateConfigVersionFromPrimary` | `ReplicateConfigVersionSkipsNilShard0` | `PickHealthyControlShard` |
| `LockoutLimiter` constructor | (code review) | filters nil shards in `NewLockoutLimiter` |
| Tracker shutdown | (code review) | `main.go:667` skips nil on `Close` |
| Budget shards 1..N | `HealthyShardsStillWritable` | independent Redis masters |

### P4 — Static / edge ✅ fixed

| Surface | Fix | Test |
| --- | --- | --- |
| Edge blacklist sync | `connect_any_shard()` tries shards 1..N in order | `blacklist_sync_test.lua` |
| Edge quarantine sub | uses `blacklist_sync.connect_any_shard()` | (shared helper) |
| Edge config sync | uses `connect_any_shard()` for `config:values` | — |
| XDP stats admin read | `xdpstats.ReadRedisAny` | `snapshot_test.go` |
| ML ops reader | already skips nil `rdbs[0]` in loop | — |

---

## Architecture snapshot

```
                    ┌─────────────────────────────────────────┐
                    │           Control plane (cmd/control)    │
                    │  Outbox ──► sync*ToAllShards / publish*            │
                    │           (best-effort; skip nil shard 0)        │
                    │  SyncWorker[0] ──► skipped when nil              │
                    │  Readiness / shutdown ──► skip nil slots           │
                    └──────────────────┬────────────────────────┘
                                       │ fan-out (all shards)
         ┌─────────────────────────────┼─────────────────────────────┐
         ▼                             ▼                             ▼
   Redis shard 0                  shard 1                      shard 2..N
   (global pub/sub,                (budget keys,                (budget keys,
    edge sync source)               replicated globals)           …)
         ▲
         │ edge-blacklist-sync.lua (first healthy shard)
    Nginx edge
```

**Tracker** with optional shard 0:

- Hot path for campaigns on shards 1..N: **works** (with warm registry).
- Campaigns mapped to shard 0: **503** (`shard_unavailable`).
- Background: fraud aggregates **flush to shard 1**; RTB floors/creatives/reload watch use first connected shard; broker reconcile skips nil.

---

## Gap in existing resilience CI

| Test | Covers | Does not cover |
| --- | --- | --- |
| `TestFault_Shard0Outage` | Tracker HTTP 202/503 under stopped `redis-0` container | `REDIS_SHARD0_OPTIONAL_STARTUP`, control plane, outbox, nil-client panics |
| `scripts/ci/check_no_shard0_control.sh` | forbids new `rdbs[0]` literals in control (except `getRDB`) | runtime nil slots in `rdbs` slice |

**Recommendation:** `bash scripts/ci/shard0_nil_gate.sh` in CI `pr-fast` (no Docker). Keep compose drill for end-to-end tracker SLA.

---

## Milestone phases

### Phase 1 — Stop the bleeding (P0) ✅

1. Nil-safe loops in `serve.go` (SyncWorker start, readiness, shutdown) — mirror tracker `main.go:667-669`.
2. Nil-safe `SyncUserConsentToRedis`, `PurgeUserDataRedis`, tracker `handler.go` health.
3. Nil-safe `BrokerReconcileWorker.sample` (skip nil shards; HWM from `firstConnectedRedisShard`).
4. `domain.SyncWorker.SyncAll` no-op when `rdb==nil`.
5. Helpers: `pingConnectedRedisShards`, `closeConnectedRedisShards` (`redis_shard_health.go`).
6. Gate: `TestShard0Nil_*` — behavioral regression (Redis GET/EXISTS/XLen), not panic-oracle only.

**Proof command:**

```bash
go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/
```

### Phase 2 — Outbox fail-soft (P1) ✅

1. Policy: **best-effort fan-out** — log + metric `ad_control_shard_fanout_skipped_total{shard,reason}`, succeed if ≥1 shard written.
2. Applied via `forEachConnectedShard` in `redis_shard_fanout.go` to `sync*ToAllShards`, `publish*ToAllShards`, `setNXOnAllShards` (not budget keys).
3. **Catch-up on recovery:** manual / existing `replicateConfigVersionFromPrimary` pattern; automated full reconcile job still open.
4. Gate: `TestShard0Nil_OutboxHandleUpdateSettings` + fan-out tests with nil shard 0.

**Proof command:** same as Phase 1 (`go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ...`).

### Phase 3 — Tracker global paths (P2) ✅

1. `FraudStreamWriter.flushAggregates`: `firstConnectedRedisShard` for aggregate XADD.
2. `BrandCreativeStore` / `DealFloorCache` / `StartRtbCatalogReloadWatch`: `firstConnectedRedis(rdbs)` in `cmd/tracker/main.go`.
3. `broker_reconcile.go`: HWM via `firstConnectedRedisShard` (P0).
4. `pickSegmentShard`: linear probe skipping nil slots.
5. Gate: `TestShard0Nil_FraudStreamAggregateFlushesHealthyShard`, brand/deal floor behavioral tests.

### Phase 4 — Edge (P4) ✅

1. `edge-blacklist-sync.lua`: `connect_any_shard()` — try every configured Redis address on failure (globals replicated on 1..N).
2. `edge-quarantine-sub.lua` + `edge-config.lua`: reuse `connect_any_shard()`.
3. Admin XDP stats: `xdpstats.ReadRedisAny` in `adminapi_wire.go`.
4. Gate: `deploy/nginx/lua/tests/blacklist_sync_test.lua` (CI `compliance.sh`); `TestReadRedisAnySkipsNilAndFindsSnapshot`.

### Phase 5 — Operational clarity ✅

1. **`REDIS_SHARD0_OPTIONAL_STARTUP`**: documented in `.env.example`, `docs/DEVELOPMENT.md` §Shard 0 degradation, and this file. Applies to tracker **and** control after P0–P4 (no per-binary split).
2. **Prometheus**: `ad_shard0_client_nil` (set in `ConnectRedisShards`); `ad_control_fanout_partial_total{op}` (partial fan-out success); existing `ad_control_shard_fanout_skipped_total{shard,reason}`.
3. **Alerts**: `Shard0ClientNil`, `ControlFanoutPartial` in `deploy/monitoring/prometheus.rules.yaml`.
4. **CI gate**: `scripts/ci/shard0_nil_gate.sh` wired into `scripts/ci/pr_fast.sh`.

---

## Test inventory (complete)

### Control plane (`shard0_nil_spof_test.go`)

| Test | Expected |
| --- | --- |
| `TestShard0Nil_SyncWorkerSyncAllNoOpWithoutPanic` | no panic |
| `TestShard0Nil_ReadinessProbeSkipsNilShard0` | OK |
| `TestShard0Nil_GracefulShutdownSkipsNilShard0` | no panic |
| `TestShard0Nil_SyncUserConsentWritesHealthyShards` | OK on shards 1..2 |
| `TestShard0Nil_PurgeUserDataRedisHealthyShards` | OK |
| `TestShard0Nil_SyncKeyToAllShardsHealthyShards` | OK on shards 1..N |
| `TestShard0Nil_PublishCampaignControlHealthyShards` | OK |
| `TestShard0Nil_PublishControlChannelHealthyShards` | OK (pubsub on shard 2) |
| `TestShard0Nil_SyncGlobalConfigHealthyShards` | OK |
| `TestShard0Nil_OutboxHandleUpdateSettings` | OK |
| `TestShard0Nil_SetNXOnAllShardsHealthyShards` | OK |
| `TestShard0Nil_SyncGlobalSetMemberHealthyShards` | OK |
| `TestShard0Nil_SyncGlobalSetReplaceHealthyShards` | OK |
| `TestShard0Nil_FanoutAllNilShardsFails` | error |
| `TestShard0Nil_HealthyShardsStillWritable` | OK |
| `TestShard0Nil_MiddlewareRevocationSkipsNil` | OK |
| `TestShard0Nil_DeleteGlobalKeySkipsNil` | OK (partial) |
| `TestShard0Nil_SyncGlobalHashFieldSkipsNil` | OK (partial) |
| `TestShard0Nil_ReplicateConfigVersionSkipsNilShard0` | OK |
| `TestShard0Nil_PublishCampaignUpdateBrokerFallback` | OK via broker |
| `TestShard0Nil_GetRDBSingleNilShard` | nil client |

### Ingestion (`shard0_nil_spof_test.go`)

| Test | Expected today |
| --- | --- |
| `TestShard0Nil_TrackerHealthSkipsNilShard0` | OK |
| `TestProof_Shard0Nil_SettingsWatcherPickHealthyShardSkipsNil` | OK |
| `TestProof_Shard0Nil_PickLocalGlobalShardSkipsNil` | OK |
| `TestProof_Shard0Nil_RegistryStartWatchShardsNoPanic` | OK |
| `TestProof_Shard0Nil_ReadFraudAggForceSkipsNil` | OK |
| `TestShard0Nil_FraudStreamAggregateFlushesHealthyShard` | OK (XADD on shard 1) |
| `TestShard0Nil_BrandCreativeStoreLoadsFromHealthyShard` | OK |
| `TestShard0Nil_DealFloorCacheRefreshFromHealthyShard` | OK |
| `TestShard0Nil_PickSegmentShardSkipsNil` | OK |
| `TestProof_Shard0Nil_FirstConnectedRedisSkipsNil` | OK |
| `TestProof_Shard0Nil_BrandCreativeStoreNilRDBNoOp` | stale |
| `TestProof_Shard0Nil_DealFloorCacheNilRDBNoOp` | stale |
| `TestProof_Shard0Nil_RtbCatalogReloadWatchNilRDBNoOp` | no watch |
| `TestProof_Shard0Nil_SegmentPickShardSingleNil` | nil |
| `TestProof_Shard0Nil_ConsentStoreUsesHealthyShard` | OK |
| `TestProof_Shard0Nil_BrokerReconcileSamplePanics` | panic → fixed: `TestShard0Nil_BrokerReconcileSampleSkipsNilShard0` |

---

## Honest bottom line

- **Shard 0 is still a SPOF** only for campaigns whose StaticSlot maps to shard 0 (`503`) and for edge-bpf-sync default write target (`FirstRedisAddr`).
- **`REDIS_SHARD0_OPTIONAL_STARTUP=1` is viable for `cmd/control` and tracker globals** after P0–P2; shard 0 catch-up after recovery is manual.
- **Tracker ingest on shards 1..N is genuinely isolated** for budget traffic; degradation is mostly control/config/fraud sidecars.
- **Broker fallback for campaign updates works** when broker is configured — complements P1 Redis fan-out.
- **Key drift:** during shard-0 outage, globals exist only on shards 1..N; reconcile shard 0 on recovery before re-enabling edge sync from shard 0.

---

## References

- `docs/ARCHITECTURE.md` §4.2 — key placement
- `docs/DEVELOPMENT.md` — shard 0 degradation runbook
- `.cursor/rules/data-layer.mdc` — shard-0 survival rules
- `tests/resilience/shard_outage_fault_test.go` — tracker HTTP drill (Docker)
- `scripts/ci/check_no_shard0_control.sh` — static control-plane lint

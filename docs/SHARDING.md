# Redis sharding and shard 0 runbook

Shard 0: pub/sub control plane, edge blacklist sync source, fraud aggregate writes, global-key conventions. Budget traffic on shards 1..N continues when shard 0 down if `REDIS_SHARD0_OPTIONAL_STARTUP=1` and registry warm. See [ARCHITECTURE.md](ARCHITECTURE.md) §4.2.

## Goals

- Hot-path SLA on shards 1..N (`/track`, budget Lua, streams).
- Fail-closed `503 shard_unavailable` for campaigns on shard 0.
- Control-plane liveness during shard-0 outage (no crash loops; outbox best-effort).

## Proof harness

```bash
go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/
```

Tests: `internal/controlplane/shard0_nil_spof_test.go`, `internal/ingestion/shard0_nil_spof_test.go`. Scenario: `rdbs[0] == nil`, shards 1..N connected.

## Failure matrix (nil shard 0)

| Component | Symptom | Behavior |
| --- | --- | --- |
| Control | SyncWorker / readiness / shutdown | Skip nil; `pingConnectedRedisShards` |
| Control | Consent / purge / outbox | Skip nil; write 1..N; `ad_control_shard_fanout_skipped_total` |
| Tracker | Health / fraud XADD / globals | `firstConnectedRedisShard`; brand/RTB via `firstConnectedRedis` |
| Tracker | Campaign on shard 0 slot | `503 shard_unavailable` |
| Edge | Blacklist / XDP stats | `connect_any_shard()` Lua; `xdpstats.ReadRedisAny` |

## Recovery

1. **Auto:** `Shard0CatchupWorker` copies globals from shard 1, validates blacklist parity, publishes `campaigns:update`.
2. **Manual:** `POST /api/v1/ops/shards/0/catchup` (`shards:write`).
3. **Verify:** `ad_shard0_catchup_last_success_timestamp`; `GET /api/v1/ops/shards`.

```bash
go test -count=1 -run 'TestShard0Nil_CatchupAfterRecovery|TestShard0CatchupWorker_' ./internal/controlplane/
bash scripts/ci/shard0_nil_gate.sh
```

## Operator metrics

| Metric | Meaning |
| --- | --- |
| `ad_shard0_client_nil == 1` | Process without shard 0 client |
| `ad_shard0_catchup_last_success_timestamp` | Last reconcile (0 = never) |
| `ad_control_fanout_partial_total` | Globals written to subset of shards |
| `ad_registry_stale_mode` | Tracker stale-serve |

Full degradation: [DEVELOPMENT.md](DEVELOPMENT.md).

## Hot campaign sub-shards (`BehaviorHighVolumeDebit`)

Campaigns with `behavior_flags` bit `BehaviorHighVolumeDebit` spread budget/fcap heat across four sub-slots while each `EVALSHA` stays on a single Redis master.

### Enable

Set `behavior_flags` on the campaign (ops API / admin). No tracker restart; registry sync picks up the flag. Pair with `LOCAL_QUOTA_MODE=live` for full-skip on eligible traffic.

### Key layout

| Key | Hash tag | Notes |
| --- | --- | --- |
| `budget:quota:{campaign_id}` | `{campaign_id:slot_N}` | `N = debitSubSlot(user_id \| click_id) % 4` |
| `fcap:c:{campaign_id}:u:` | `{campaign_id:slot_N}` | Same sub-slot as quota debit |
| `budget:sync:campaign:*`, migration fence | `{campaign_id}` | Unchanged; not sub-sharded |
| `budget:campaign:*` (non-quota legacy) | `{campaign_id}` | Used when quota mode off |

`resolveDebitShard` may route debits to `spreadHighVolumeShard(campaign_id, subSlot)` across Redis masters 1..N. StaticSlot campaign slot is unchanged; sub-slot picks the physical `rdbs[i]` client.

### Invariants

- Single-shard Lua: all keys in one `EVALSHA` share the sub-slot hash tag.
- Migration: `migration_gen` / `BumpMigrationFences` unchanged; fence keys remain on `{campaign_id}`.
- Budget: `domain.AssertBudgetInvariant` — `TestFault_HighVolumeDebit_subShardBudgetInvariant`.

### Verify

```bash
go test ./internal/ingestion/ -run='DebitSubshard|MigrationFence|HighVolumeDebit' -count=1
go test ./tests/e2e/ -run=Multishard -count=1   # needs Docker
```

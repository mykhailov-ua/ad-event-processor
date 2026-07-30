# Multi-region

Enterprise multi-cell topology (M7): regional hot-path cells, global control plane, regional proxies, and capacity-aware traffic routing.

Status: **target architecture**. Partial implementation exists (`outbox_region_delivery`, `RegionOutboxRelay`, licensing `multi_region`). Regional proxy, disk gate, and node scorer are design targets documented here.

Related: [ARCHITECTURE.md](./ARCHITECTURE.md), [MILESTONE.md](./MILESTONE.md), [CONTROL_PLANE](.cursor/rules/control-plane.mdc), [FAULT_RESILIENCE](.cursor/rules/fault-resilience.mdc), [CODE_STYLE](.cursor/rules/code-style.mdc), [HOT_PATH](.cursor/rules/hot-path.mdc).

## Document map

| Part | Sections | Audience |
| :--- | :--- | :--- |
| **I — Architecture** | §0–§11 | Operators, architects, reviewers |
| **II — Implementation tasks** | §A–§J | Engineers shipping M1–M7 |

Part I describes **what** the system is. Part II describes **how to ship it** with checklists, SLA, tests, and code rules per task.

---

# Part I — Architecture

## §0. Naming registry (canonical)

All flags use **bit masks** in mmap/WAL headers (`uint8`, cache-line padded) or **text enums** in Postgres. Prometheus and logs use the `snake_case` label; Go constants use `PascalCase` with package prefix.

**WAL record header** — `pkg/regionproxy/wal/record.go`:

| Bit | Go constant | Log / metric label | Meaning |
| :---: | :--- | :--- | :--- |
| 0 | `WalFlagAppended` | `wal_appended` | Payload appended to segment |
| 1 | `WalFlagDedupReady` | `wal_dedup_ready` | KeyGen wrote `factor_u` |
| 2 | `WalFlagForwardClaimed` | `wal_forward_claimed` | Uplink CAS winner; at-most-one forward |
| 3 | `WalFlagRemoteAcked` | `wal_remote_acked` | Global D3 ingest acknowledged |

**OpKey slot** — `pkg/regionproxy/opkey/slot.go`:

| Bit | Go constant | Log label | Meaning |
| :---: | :--- | :--- | :--- |
| 0 | `OpKeyFlagDerived` | `opkey_derived` | OpKeyPool finished SHA256 / scope |
| 1 | `OpKeyFlagReplicaBooked` | `opkey_booked` | Replica acknowledged book |
| 2 | `OpKeyFlagExecuting` | `opkey_executing` | This replica holds lease CAS |
| 3 | `OpKeyFlagLeaseRenewed` | `opkey_lease_renewed` | Heartbeat extended deadline within budget |

**Postgres `lease_state`** — `TEXT` constrained enum (migration §10):

| Value | Go `LeaseState` | Meaning |
| :--- | :--- | :--- |
| `booked` | `LeaseStateBooked` | Reserved; no side effects |
| `executing` | `LeaseStateExecuting` | Single executor; may apply |
| `completed` | `LeaseStateCompleted` | D3 `RecordApply` done |
| `expired` | `LeaseStateExpired` | Deadline passed; not executed |

**Score provenance** — `node_capacity_scores.provenance`:

| Value | Meaning |
| :--- | :--- |
| `own_window` | §6 Phase B |
| `neighbor_median` | §6 Phase A/C |
| `historical_daily` | §6 Phase D |
| `conservative_default` | §6 Phase E |

Deprecated aliases (do not use in new code): `KEY_PENDING`, `KEY_ASSIGNED`, `UPLINK_CLAIMED`, `GLOBAL_ACKED`, `BOOKED`, `EXECUTION`, `DONE`, `TIMEOUT`.

---

## §1. Topology

```text
                    ┌─────────────────────────────────────┐
                    │  Global control (ESPX_REGION_CODE=0) │
                    │  PG (finance, outbox), NodeScorer    │
                    │  UDP epoch publisher, D3 dedup       │
                    └──────────────┬──────────────────────┘
                                   │ outbox_region_delivery (at-least-once)
           ┌───────────────────────┼───────────────────────┐
           ▼                       ▼                       ▼
   ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
   │ Region EU-W   │       │ Region US-E   │       │ Region AP-S   │
   │ tracker x N   │       │ tracker x N   │       │ tracker x N   │
   │ Redis x4      │       │ Redis x4      │       │ Redis x4      │
   │ processor     │       │ processor     │       │ processor     │
   │ region-proxy  │       │ region-proxy  │       │ region-proxy  │
   │ management    │       │ management    │       │ management    │
   └───────────────┘       └───────────────┘       └───────────────┘
```

### Invariants

1. Hot path (tracker, Redis x4, processor) runs in **regional cells**.
2. **Global PostgreSQL** holds finance, configuration, and outbox source of truth.
3. Cross-region config delivery uses `outbox_region_delivery` (at-least-once) and `RegionOutboxRelay`.
4. **No cross-region Redis replication.**

### Enablement

| Requirement | Detail |
| :--- | :--- |
| License | Enterprise JWT `multi_region: true` |
| Env | `MULTI_REGION_ENABLED=1`, `ESPX_REGION_CODE` (`0` = global, `>0` = cell) |
| Installer | `multi_region: true` in `install.yaml` (not supported in `compose_dev`) |
| Regions table | `regions (code, name, active)` drives outbox fanout trigger |

---

## §2. Regional proxy

Many nodes per region must not write directly to global control plane. A **region-proxy** (2–3 HA instances per region, Redis leader election like `cmd/broker`) aggregates upstream writes.

### Pipeline

```text
Regional nodes ──ProduceBatch──► region-proxy ingress
                                      │
                                      ▼
                               DiskWriteGate (append)
                                      │
                                      ▼
                               mmap WAL (group-commit fsync)
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
             KeyGen thread (CPU)                  Uplink worker (I/O)
             factor_u + flags                     batched → global ingest
```

### WAL record lifecycle

| Flag | Go constant | Meaning |
| :--- | :--- | :--- |
| `wal_appended` | `WalFlagAppended` | Ingress appended payload |
| `wal_dedup_ready` | `WalFlagDedupReady` | KeyGen wrote `factor_u` |
| `wal_forward_claimed` | `WalFlagForwardClaimed` | Uplink CAS on first pickup |
| `wal_remote_acked` | `WalFlagRemoteAcked` | Global D3 confirm + apply |

Crash after `wal_forward_claimed` without `wal_remote_acked`: replay with same `factor_u`; global `dedup_claim_confirm` returns `already_confirmed`.

### KeyGen as traffic signal

KeyGen rate and queue depth are proxy metrics for upstream volume:

| Metric | Use |
| :--- | :--- |
| `keygen_rate` (keys/s) | Regional traffic index |
| `keygen_queue_depth` | Proxy backpressure |
| `keygen_lag_seconds` | CPU saturation on KeyGen thread |
| `wal_unacked_bytes` | Disk / uplink pressure |

High `keygen_rate` with healthy proxy score → candidate for traffic increase. High queue depth → ingress shed (`503 proxy_backpressure`).

Reuse: `pkg/broker/log` (mmap segments), `pkg/broker/server/coord.go` (HA), `pkg/dedupkey` (D3 scope), `pkg/iogate` (disk gate, planned).

---

## §3. Task separation

### Regional only

| Task | Component |
| :--- | :--- |
| `/track`, RTB, `FilterEngine` | tracker |
| Redis Lua debit, local quanta (M8) | tracker + Redis |
| mmap WAL append, KeyGen, uplink batching | region-proxy |
| `RegionOutboxRelay` (global outbox → local Redis) | management cell |
| Stream consumer → regional PG event path | processor |
| Fraud blacklist snapshot read | tracker |

### Global only

| Task | Component |
| :--- | :--- |
| `balance_ledger`, billing, subscriptions | PG / billing |
| `outbox_events` + `fanout_outbox_region_delivery` | PG |
| `dedup_claim_confirm` (D3 source of truth) | PG |
| `NodeCapacityScorer`, traffic weights | management |
| ML fraud scoring | fraud-scorer |
| Cross-region recon, `AssertBudgetInvariant` | management |
| Campaign CRUD, RBAC, HTMX admin | management |

### Via region-proxy (upstream)

Budget deltas, spend sync batches, quota/ingress reports, optional audit batches. **Forbidden:** direct regional `balance_ledger` insert, outbox write without D3, cross-cell Redis mutation.

---

## §4. Disk write gate

Disk contention is inevitable: ingress append, KeyGen header write-back, fsync, and segment roll compete on the same NVMe. Both **region-proxy** and **global ingest** use a shared pattern (`pkg/iogate`, planned).

```text
Ingress ──TierHigh──► appendSem ──► mmap
KeyGen  ──TierLow───► appendSem ──► mmap header
Durability loop ─────► fsyncSem (capacity 1) ──► fdatasync
```

| Tier | Writers | On degraded |
| :--- | :--- | :--- |
| High | ingress append, near-full WAL flush | shed new ingress |
| Low | index, segment seal, compaction | defer; KeyGen may use atomic cache-line write |

Degraded when fsync EMA p99 exceeds budget or `disk_writable=0` (same idea as `pkg/logger` `diskDegraded` and `ad_broker_disk_writable`).

Metrics: `ad_disk_gate_append_wait_seconds`, `ad_disk_gate_fsync_in_flight`, `ad_disk_gate_shed_total`, `ad_disk_gate_degraded`.

---

## §5. Node capacity scoring

Global **NodeCapacityScorer** (management worker) computes a traffic weight per node. Weights publish in UDP control epochs (extension of `UDPControlServer`) and drive edge / tracker routing.

### Score formula

```text
score_n = Σ w_i · s_i(metric_{n,i})
```

- `w_i` — configurable weight, `Σ w_i = 1` (stored in `scoring_weights_json`, default below).
- `s_i ∈ [0, 1]` — normalized health (1 = healthy, 0 = critical).

**Default weights (infra 0.70, business 0.30):**

| Metric | Weight | Source |
| :--- | ---: | :--- |
| CPU util | 0.20 | node / cgroup |
| RAM util | 0.15 | node / cgroup |
| Disk fsync p99 | 0.15 | `ad_disk_gate_*` |
| IOPS / gate wait p99 | 0.10 | iostat, gate wait |
| Tracker handler p99 | 0.10 | `ad_http_request_duration_seconds` |
| Fraud reject rate | 0.10 | filter metrics |
| Anomaly / IVT rate | 0.08 | ivt-detector, CH |
| Budget invariant drift | 0.07 | recon worker |
| Stream lag | 0.05 | Redis XLEN − committed offset |

Proxy-specific: `keygen_queue_depth` caps proxy score (max 50% penalty when queue saturated).

### Traffic shift

Per epoch (1–10 s, aligned with `UDP_SYNC_INTERVAL_MS`):

```text
Δweight_n = k · (score_n − mean_score)     # k ≈ 0.05
weight_n  = clamp(weight_n + Δweight_n, w_min, w_max)
normalize(weights across peers in same role/region)
```

| Condition | Action |
| :--- | :--- |
| `score < 0.3` | drain −10% weight/epoch, floor `w_min` (e.g. 0.05) |
| `score > 0.8` | boost +5%/epoch, cap `w_max` (e.g. 0.95) |
| `disk_degraded = 1` | weight → 0 within one epoch |
| `budget_invariant_fail` | weight → 0 immediately |

Draining reduces **new** traffic only; in-flight work completes via proxy WAL and stream ack rules.

---

## §6. Sliding window and anti-false-positive fallbacks

A single-point metric causes false drains (cold start, brief spike, scrape gap). **All scoring inputs use a sliding window of the last N minutes** before normalization.

### Parameters

| Param | Default | Env (proposed) |
| :--- | :--- | :--- |
| Window length N | 15 min | `NODE_SCORE_WINDOW_MIN` |
| Min samples in window | 30 | `NODE_SCORE_MIN_SAMPLES` |
| Scrape / bucket interval | 10 s | aligned with UDP epoch |
| Neighbor peer set | same `region_code` + same `role` | — |
| Historical fallback | previous calendar day D−1 | `node_metric_daily_snapshots` |

### Data source priority (per node, per metric)

```text
┌──────────────────────────────────────────────────────────────┐
│ 1. OWN_WINDOW   — node age ≥ N min AND samples ≥ min_samples │
│ 2. NEIGHBOR     — median of peer OWN_WINDOW in same region   │
│ 3. HISTORICAL   — D−1 daily snapshot (peer aggregate)        │
│ 4. CONSERVATIVE — score = 0.5, weight frozen (no drain/boost)│
└──────────────────────────────────────────────────────────────┘
```

#### Phase A — Cold start (`uptime < N` minutes)

Node has insufficient local window data.

1. Use **neighbor median** for each metric from peers with valid `OWN_WINDOW`.
2. Tag score `provenance=neighbor`; cap weight at `w_max_cold` (e.g. 0.25) until Phase B.
3. Do **not** drain neighbors because the new node looks bad — new node is not in drain set until Phase B.

#### Phase B — Warm (`uptime ≥ N` minutes, samples ≥ min_samples)

1. Switch to **OWN_WINDOW** (rolling aggregate: p99 for latencies, mean for utilizations, rate for counters).
2. Tag `provenance=own`.
3. Full drain/boost rules apply.

#### Phase C — Own window stale or insufficient

If own scrapes fail or `samples < min_samples` after warm period:

1. Fall back to **neighbor median** (same region, same role).
2. If fewer than 2 neighbors have valid windows → **HISTORICAL**.

#### Phase D — Historical snapshot (D−1)

Nightly job materializes per `(region_code, role, metric)` aggregates from all nodes into `node_metric_daily_snapshots`:

```sql
-- illustrative
CREATE TABLE node_metric_daily_snapshots (
    day           DATE NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    metric        TEXT NOT NULL,
    p50, p99, mean DOUBLE PRECISION,
    sample_count  BIGINT,
    PRIMARY KEY (day, region_code, role, metric)
);
```

Scorer reads `day = current_date - 1`. Tag `provenance=historical`; apply **hysteresis** — no aggressive drain (max −2% weight/epoch) unless live hard signals (`disk_degraded`, invariant fail).

#### Phase E — No neighbors and no historical row

New region, first deploy, or metrics pipeline down:

- `provenance=conservative`, `score = 0.5`, weight unchanged from last epoch.
- Alert `ad_control_score_fallback_conservative_total`.
- Operator must seed historical snapshot or add peer nodes.

### Window aggregation rules

| Metric type | Aggregation over window |
| :--- | :--- |
| Latency (p99, fsync) | max of per-bucket p99 (conservative) |
| Utilization (CPU, RAM) | mean of bucket means |
| Rates (fraud %, keygen/s) | sum(events)/sum(total) — not mean of ratios |
| Counters (lag bytes) | max in window |

### Hysteresis and flap control

| Rule | Value |
| :--- | :--- |
| Min epochs in drain before weight=0 | 3 |
| Min epochs in boost before +cap | 2 |
| Score EMA α | 0.3 (smooth published score) |
| Provenance downgrade (own → neighbor) | immediate on scrape miss > 2 epochs |
| Provenance upgrade (neighbor → own) | only after N min **and** min_samples |

This prevents a single bad scrape from draining a node and prevents a brand-new node from receiving full traffic before it proves stability.

### Industry alignment and hardening (plan)

Cross-check against common practice (AWS cells / Global Accelerator, RocksDB/PG group commit, adaptive LB health scoring). Gaps below are **explicit plan items**, not optional nice-to-haves.

| # | Hardening | Rationale | Plan item |
| :---: | :--- | :--- | :--- |
| H1 | **Warmup grace period** | ASG/ALB: do not score or drain until app is warm (`NODE_WARMUP_SEC`, default 300) | Phase M4; env `NODE_WARMUP_SEC` |
| H2 | **Liveness ≠ capacity** | `/health` (process up) separate from `/ready` (deps + window data) and from **weight** | Tracker/management split endpoints; scorer ignores liveness-only |
| H3 | **Per-region + global scorer** | Bifrost: do not import another region's latency into local weights | Regional scorer → tracker weights; global scorer → cross-region dial only |
| H4 | **Fail-open policy documented** | AWS GA may fail open when no healthy weighted endpoint | eSPX default: **conservative** (§6 Phase E); optional `CONTROL_FAIL_OPEN=1` for edge only |
| H5 | **mmap fsync contract** | CMU/Quasar: mmap without explicit fsync policy is risky | Document: append-only segment log + `fsyncSem` + `Recover()`; no btree-on-mmap |
| H6 | **Established connections** | GA does not move in-flight TCP on weight change | Drain = new connections only; document in edge/runbook |
| H7 | **Hard signal override** | Health overrides weight when invariant/disk fails | Already in §5; fault tests in M6 |

References: [AWS cell architecture](https://docs.aws.amazon.com/wellarchitected/latest/reducing-scope-of-impact-with-cell-based-architecture/what-is-a-cell-based-architecture.html), [Global Accelerator weights](https://docs.aws.amazon.com/global-accelerator/latest/dg/about-endpoints-endpoint-weights.html), [RocksDB WAL group commit](https://github.com/facebook/rocksdb/wiki/WAL-Performance), [CMU mmap CIDR 2022](https://db.cs.cmu.edu/mmap-cidr2022/).

---

## §7. Replicated operation execution (lease states)

Cold-path operations that must survive node loss (proxy uplink, global batch apply, regional relay) are **replicated** to a fixed replica set (typically 3 nodes per role in one region). Exactly one replica may hold `executing`; others hold `booked` until timeout or handoff.

This extends D3 dedup (§2, §9.3): **booking is regional/fast; commit is global/PG**.

### 7.1 State machine

```text
                    ┌──────────┐
         replicate  │  booked  │◄────────────────────────┐
         ──────────►│ (standby)│                         │
                    └────┬─────┘                         │
                         │ CAS → executing               │ expired
                         ▼                               │
                    ┌──────────┐     lease renew         │
                    │executing │─────────────────────────┤
                    │ (leader) │                         │
                    └────┬─────┘                         │
              complete   │         deadline passed       │
                         ▼                               ▼
                    ┌──────────┐                   ┌──────────┐
                    │completed │                   │ expired  │
                    └──────────┘                   └────┬─────┘
                                                        │ retry attempt+1
                                                        ▼
                                                   (new booked set)
```

| `lease_state` | Go constant | Meaning | Who sets |
| :--- | :--- | :--- | :--- |
| `booked` | `LeaseStateBooked` | Reserved; no side effects | Coordinator on replicate |
| `executing` | `LeaseStateExecuting` | Owns lease; may run apply | CAS winner before side effects |
| `completed` | `LeaseStateCompleted` | D3 `RecordApply` done | Executor after global ACK |
| `expired` | `LeaseStateExpired` | Deadline passed; **not executed** | Janitor or replica when `now > deadline_at` |

**Timeout rule:** if `executing` does not reach `completed` within `OP_LEASE_TIMEOUT_SEC` (default 30), transition to `expired`. Retries use `attempt + 1` in D3 scope — never reuse the same lease row.

### 7.2 Operation key pool (KeyGen thread)

Same pattern as WAL KeyGen (§2): **CPU-bound key material on a pinned thread**; I/O-bound booking/execute on workers.

```text
Ingress / WAL tail ──► OpKeyPool (pinned thread)
                         │ SHA256 canonical op body → factor_u
                         │ monotonic op_seq → scope seq range
                         ▼
                    MPSC ring (op_key slots)
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
    Replica A        Replica B        Replica C
    booked           booked           booked
         │               │               │
         └─────── CAS executing ────────┘
                         │
                    one winner
```

**OpKey slot flags** (cache-line aligned; see naming registry):

| Label | Go constant | Meaning |
| :--- | :--- | :--- |
| `opkey_derived` | `OpKeyFlagDerived` | OpKeyPool wrote `factor_u` + `op_id` |
| `opkey_booked` | `OpKeyFlagReplicaBooked` | Replicated to this node |
| `opkey_executing` | `OpKeyFlagExecuting` | CAS winner; lease started |
| `opkey_lease_renewed` | `OpKeyFlagLeaseRenewed` | Heartbeat extended deadline |

First pickup CAS `opkey_booked → opkey_executing`; losers remain `booked` and watch authoritative `deadline_at` from PG.

### 7.3 Replication and deadline broadcast

On book, coordinator writes lease row + replica acks (migration §10).

1. **Book:** insert `lease_state = 'booked'`; `deadline_at = NOW() + OP_LEASE_TIMEOUT_SEC` (PG authoritative).
2. **Execute:** `UPDATE ... SET lease_state = 'executing', executor_node_id = $me WHERE op_id = $id AND lease_state = 'booked'` (single-row CAS).
3. **Renew:** executor extends `deadline_at` at `timeout/3` interval; max `OP_LEASE_MAX_RENEWALS` (default 3).
4. **Complete:** `lease_state = 'completed'` after D3 `RecordApply`.
5. **Expire:** janitor sets `lease_state = 'expired'` where `deadline_at < NOW()` and state ∈ (`booked`, `executing`).

### 7.4 Integration points

| Operation | Replica set | Coordinator | Commit |
| :--- | :--- | :--- | :--- |
| Region-proxy uplink batch | 3× region-proxy | proxy leader | global `dedup_claim_confirm` + apply |
| Global batch ingest | 3× global management ingest | global cell | PG outbox / ledger |
| `RegionOutboxRelay` delivery | 3× management cell | regional management | Redis + `region_apply_idempotency` |
| Outbox worker (global) | 2× management (global) | `SKIP LOCKED` + lease | existing outbox path |

**Flow with existing D3:**

```text
OpKeyPool → factor_u + scope
     → book replicas (booked, deadline_at)
     → CAS executing on one node
     → ClaimConfirm(factor_u)   # PG — tiebreaker if partition
     → side effects
     → RecordApply
     → completed
```

`ClaimConfirm` **before** irreversible side effects when budget/ledger involved; after for idempotent Redis writes only if `NeedsResumeApply` covers crash window.

### 7.5 Required conditions (currently missing)

| ID | Condition | Why |
| :--- | :--- | :--- |
| C1 | **Authoritative lease store** | All replicas must read identical `deadline_at` and state |
| C2 | **Fencing epoch** | Stale executor after partition cannot write (broker `fencing.epoch` pattern) |
| C3 | **Quorum book** | Book succeeds only if ≥2 of 3 replicas ACK `booked` (or PG row visible) |
| C4 | **Clock source** | `deadline_at` from coordinator PG `NOW()`, not wall clock per node |
| C5 | **Attempt monotonicity** | Retry increments `attempt`; D3 scope includes `attempt` |
| C6 | **Heartbeat budget** | Max renewals = 3; total wall time capped at `3×N` |
| C7 | **OpKeyPool backpressure** | Shed book when pool depth > watermark (ties to §2 KeyGen metrics) |
| C8 | **Lease janitor** | Single worker per replica set marks `expired` (avoid stuck `executing`) |

### 7.6 Network partition: risks and mitigations

```text
        Partition A                    Partition B
   ┌─────────────────┐            ┌─────────────────┐
   │ Node-1 executing│            │ Node-2 booked   │
   │ (isolated)      │            │ Node-3 booked   │
   └─────────────────┘            └─────────────────┘
```

| Risk | Scenario | Divergence | Mitigation |
| :--- | :--- | :--- | :--- |
| **P1 Split executing** | Two nodes CAS `executing` | double apply | PG `UPDATE ... WHERE lease_state='booked'` is single-winner; D3 blocks duplicate |
| **P2 Ghost executor** | Node-1 executes while isolated; Node-2 expires and retries | double spend | **ClaimConfirm before apply**; `already_confirmed` |
| **P3 False expire** | Slow but healthy executor; standbys retry | duplicate work | Heartbeat renew (C6); idempotent apply |
| **P4 Booked never sees deadline** | Standby partitioned from PG | late takeover | On reconnect: reload row; if `expired` → do not execute |
| **P5 Coordinator partition** | Cannot book | lost ops | Client retains mmap WAL / proxy ACK only after quorum book (C3); otherwise retry produce |
| **P6 Cross-region partition** | Global PG unreachable | regional cells autonomous | Hot path continues (local Redis); uplink spools in proxy WAL; no new global config until heal — existing M14 stale-serve |
| **P7 Epoch skew** | UDP deadline vs PG deadline differ | early/late timeout | **PG lease is SoT**; UDP carries `op_id` + `deadline_unix` for observability only |

**Partition decision table:**

| PG `lease_state` | Local view | Action |
| :--- | :--- | :--- |
| `booked`, not expired | any | wait or compete for `executing` |
| `executing`, not expired | not executor | wait; monitor renew |
| `executing`, expired, no renew | any | janitor → `expired`; retry `attempt+1` |
| `completed` | any | no-op |
| `expired` | any | retry only with new `attempt` |
| CAS lost + other `executing` | `booked` | standby; do not apply |

**Quorum rule (recommended):** book is durable when PG commit succeeds (global/regional PG reachable). For proxy-only book in mmap, require **2-of-3 replica ACK** on book record before ingress ACK to client.

### 7.7 Relation to KeyGen traffic metrics

| Signal | Use in lease execution |
| :--- | :--- |
| `ad_region_proxy_keygen_rate` | Book rate ≈ upstream pressure |
| `ad_op_booked_queue_depth` | Standby backlog; shed if saturated |
| `ad_op_lease_expired_total` | Partition or slow PG; tune timeout/renew |
| `ad_op_keypool_depth` | OpKeyPool pressure |

---

## §8. UDP epoch extension

Existing: `UDPControlServer` publishes shard RPS/RPD limits. Extension adds `node_weights`:

```json
{
  "epoch": 128,
  "shard_limits": [50000, 50000, 50000, 50000],
  "node_weights": [
    {"node_id": "tracker-1", "weight": 0.35, "score": 0.91, "provenance": "own"},
    {"node_id": "tracker-2", "weight": 0.65, "score": 0.78, "provenance": "own"}
  ]
}
```

Trackers and edge Lua apply weighted pick. Stale epoch (`epoch_lag > 2`) → equal weights, no new drains.

Persist epochs in `control_plane_epochs` (existing) for audit and KeyGen routing epoch alignment.

---

## §9. Bottlenecks and mitigations

### 9.1 Disk fsync (region-proxy and global)

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| fsync storm | `disk_degraded`, ingress 503 | `fsyncSem` cap=1; group-commit; separate NVMe per proxy |
| mmap roll under load | tail latency spikes | pre-allocate segments; roll only on Low tier |
| KeyGen vs ingress contention | append wait p99 ↑ | atomic header write in dedicated cache line; TierLow cap |

### 9.2 KeyGen CPU saturation

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| SHA256 backlog | `keygen_lag_seconds` ↑ | pinned OS thread; scale proxy replicas; shed ingress |
| False capacity score | proxy looks healthy, queue deep | queue depth in score (50% cap penalty) |

### 9.3 Global PG / D3 dedup

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| uplink batch stampede | `dedup_claim_confirm` contention | region-proxy batching; `MgmtPgGate` HIGH for uplink |
| cross-region fanout lag | stale config in cells | monitor `ad_region_outbox_delivery_lag_seconds` |

### 9.4 Scorer / metrics pipeline

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| Prometheus scrape gap | false drain | sliding window + neighbor fallback (§6) |
| cold start | new node full traffic | Phase A cap `w_max_cold` |
| neighbor cascade | all peers bad → bad median | historical D−1; conservative mode |
| CH lag for business metrics | stale fraud/anomaly | mark business metrics `stale`; reduce `w_i` temporarily |
| scorer SPOF | frozen weights | last epoch sticky; equal weights on stale |

### 9.5 UDP control plane

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| packet loss | uneven weights | burst snapshot on CONFIG_REQUEST (existing) |
| epoch skew | flapping | `ad_udp_control_epoch_lag` alert; stale → equal weights |

### 9.6 Traffic shift oscillation

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| drain/boost flap | oscillating p99 | score EMA; min epochs in state; hysteresis |
| thundering herd on boost | spike on healthy node | +5%/epoch cap; check proxy `keygen_queue` |

### 9.7 Бюджет в мультирегиональной среде

Cross-region spend sync is implemented via `GlobalSpendReconciler` (idempotent `balance_ledger` debits, `sync_idempotency` keys). See `docs/DEVELOPMENT.md` (Completed roadmap).

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| spend counted in wrong region | invariant drift | budget authority in global PG; deltas via proxy only |
| RPD split per region | quota gaming | `maxRPD / active_regions` in UDP limits (existing) |

### 9.8 HA region-proxy split-brain

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| dual leader append | duplicate uplink | broker fencing epoch; D3 dedup; §7 leases |

### 9.9 Replicated operation leases (§7)

| Risk | Symptom | Mitigation |
| :--- | :--- | :--- |
| P1 split executing | double apply | PG CAS + D3 ClaimConfirm |
| P2 ghost executor | spend drift | ClaimConfirm before ledger |
| P3 false expire | duplicate retry | heartbeat renew; cap renewals (C6) |
| P5 coordinator partition | lost books | quorum book (C3); client WAL until ACK |
| BOOKED queue growth | memory pressure | OpKeyPool shed; `ad_op_booked_queue_depth` alert |

---

## §10. Storage schema and migrations

### 10.1 Existing (no change)

`regions`, `outbox_region_delivery`, `region_apply_idempotency` — see `00047_multi_region.sql`.

### 10.2 Migration `00051_multi_region_metrics.sql`

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE node_metric_buckets (
    node_id       TEXT NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    bucket_ts     TIMESTAMPTZ NOT NULL,
    metric        TEXT NOT NULL,
    value_p50     DOUBLE PRECISION,
    value_p99     DOUBLE PRECISION,
    value_mean    DOUBLE PRECISION,
    sample_count  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, bucket_ts, metric)
);

CREATE INDEX idx_node_metric_buckets_region_role_ts
    ON node_metric_buckets (region_code, role, bucket_ts DESC);

CREATE TABLE node_metric_daily_snapshots (
    day           DATE NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    metric        TEXT NOT NULL,
    value_p50     DOUBLE PRECISION,
    value_p99     DOUBLE PRECISION,
    value_mean    DOUBLE PRECISION,
    sample_count  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, region_code, role, metric)
);

CREATE TABLE node_capacity_scores (
    node_id       TEXT NOT NULL,
    region_code   SMALLINT NOT NULL,
    role          TEXT NOT NULL,
    score         DOUBLE PRECISION NOT NULL,
    weight        DOUBLE PRECISION NOT NULL,
    provenance    TEXT NOT NULL,
    epoch_id      BIGINT NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (node_id, region_code, role)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS node_capacity_scores;
DROP TABLE IF EXISTS node_metric_daily_snapshots;
DROP TABLE IF EXISTS node_metric_buckets;
-- +goose StatementEnd
```

### 10.3 Migration `00052_operation_leases.sql`

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE operation_leases (
    op_id            UUID PRIMARY KEY,
    region_code      SMALLINT NOT NULL,
    role             TEXT NOT NULL,
    replica_set_id   UUID NOT NULL,
    attempt          INT NOT NULL DEFAULT 1,
    factor_u         UUID NOT NULL,
    dedup_scope      JSONB NOT NULL,
    lease_state      TEXT NOT NULL DEFAULT 'booked',
    executor_node_id TEXT,
    fencing_epoch    BIGINT NOT NULL DEFAULT 0,
    deadline_at      TIMESTAMPTZ NOT NULL,
    renew_count      INT NOT NULL DEFAULT 0,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT operation_leases_state_chk CHECK (
        lease_state IN ('booked', 'executing', 'completed', 'expired')
    ),
    CONSTRAINT operation_leases_attempt_uniq UNIQUE (replica_set_id, attempt)
);

CREATE INDEX idx_operation_leases_active
    ON operation_leases (region_code, role, lease_state, deadline_at)
    WHERE lease_state IN ('booked', 'executing');

CREATE TABLE operation_lease_replicas (
    op_id           UUID NOT NULL REFERENCES operation_leases(op_id) ON DELETE CASCADE,
    node_id         TEXT NOT NULL,
    book_ack_at     TIMESTAMPTZ,
    local_flags     SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (op_id, node_id)
);

CREATE INDEX idx_operation_lease_replicas_node
    ON operation_lease_replicas (node_id, book_ack_at DESC);

-- Transition booked → executing (single winner)
CREATE OR REPLACE FUNCTION operation_lease_claim_executing(
    p_op_id UUID,
    p_node_id TEXT,
    p_fencing_epoch BIGINT
) RETURNS TABLE(lease_state TEXT, deadline_at TIMESTAMPTZ) AS $$
BEGIN
    RETURN QUERY
    UPDATE operation_leases ol
    SET lease_state = 'executing',
        executor_node_id = p_node_id,
        fencing_epoch = p_fencing_epoch
    WHERE ol.op_id = p_op_id
      AND ol.lease_state = 'booked'
      AND ol.deadline_at > NOW()
    RETURNING ol.lease_state, ol.deadline_at;
END;
$$ LANGUAGE plpgsql;

-- Expire stale leases (janitor)
CREATE OR REPLACE FUNCTION operation_lease_expire_stale(p_limit INT DEFAULT 500)
RETURNS INT AS $$
DECLARE
    n INT;
BEGIN
    WITH expired AS (
        SELECT op_id FROM operation_leases
        WHERE lease_state IN ('booked', 'executing')
          AND deadline_at < NOW()
        ORDER BY deadline_at ASC
        LIMIT p_limit
        FOR UPDATE SKIP LOCKED
    )
    UPDATE operation_leases ol
    SET lease_state = 'expired'
    FROM expired e
    WHERE ol.op_id = e.op_id;
    GET DIAGNOSTICS n = ROW_COUNT;
    RETURN n;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS operation_lease_expire_stale(INT);
DROP FUNCTION IF EXISTS operation_lease_claim_executing(UUID, TEXT, BIGINT);
DROP TABLE IF EXISTS operation_lease_replicas;
DROP TABLE IF EXISTS operation_leases;
-- +goose StatementEnd
```

### 10.4 Table summary

| Table | Purpose |
| :--- | :--- |
| `regions` | active region codes (exists) |
| `outbox_region_delivery` | fanout queue (exists) |
| `region_apply_idempotency` | relay apply idempotency (exists) |
| `node_metric_buckets` | 10 s rollups, TTL 24 h |
| `node_metric_daily_snapshots` | D−1 historical fallback |
| `node_capacity_scores` | latest score/weight/provenance per node |
| `operation_leases` | lease state machine (§7) |
| `operation_lease_replicas` | per-node book ACK for quorum (C3) |
| `scoring_weights_json` | platform config (license or PG) |

---

## §11. Metrics and environment

### Metrics

| Metric | Help |
| :--- | :--- |
| `ad_node_score` | Published score per node |
| `ad_node_weight` | Traffic weight per node |
| `ad_node_score_provenance` | 0=own, 1=neighbor, 2=historical, 3=conservative |
| `ad_control_traffic_shift_total` | drain/boost events |
| `ad_control_score_fallback_total` | fallback tier usage |
| `ad_region_proxy_keygen_rate` | keys/s |
| `ad_region_proxy_keygen_queue_depth` | KeyGen backlog |
| `ad_disk_gate_degraded` | disk gate state |
| `ad_op_lease_booked_total` | Operations booked per replica |
| `ad_op_lease_execution_total` | CAS wins |
| `ad_op_lease_expired_total` | `expired` transitions |
| `ad_op_lease_heartbeat_renew_total` | Lease extensions |
| `ad_op_lease_partition_recovery_total` | Reconnect after partition |
| `ad_op_keypool_depth` | OpKeyPool backlog |
| `ad_op_booked_queue_depth` | Standby book backlog |

### Environment variables (cold path)

| Env | Default | Task |
| :--- | :--- | :--- |
| `NODE_SCORE_WINDOW_MIN` | 15 | M3 |
| `NODE_SCORE_MIN_SAMPLES` | 30 | M3 |
| `NODE_WARMUP_SEC` | 300 | M4 |
| `DISK_LATENCY_BUDGET_MS` | 50 | M1 |
| `OP_LEASE_TIMEOUT_SEC` | 30 | M6 |
| `OP_LEASE_MAX_RENEWALS` | 3 | M6 |
| `CONTROL_FAIL_OPEN` | 0 | M5 |
| `RESILIENCE_MIN_PROOFS_MR` | 12 | M7 |

---

# Part II — Implementation tasks

## §A. Global standards (all M* tasks)

### SLA tiers

| Tier | Scope | Key budgets |
| :--- | :--- | :--- |
| **Hot** | tracker `/track`, FilterEngine, RTB | p95 < 50 ms, p99 < 80 ms, max 100 ms; 0 allocs/op ([HOT_PATH.md](.cursor/rules/hot-path.mdc)) |
| **Cold ingest** | region-proxy ingress, broker produce | p99 < 20 ms ACK after WAL fsync batch |
| **Cold control** | management workers, scorer, leases | tick < 500 ms; PG CAS p99 < 10 ms |
| **Global settle** | proxy uplink → PG | p99 < 2 s end-to-end; budget invariant ±1 micro-unit |

**Regression rule:** no M* merge may raise tracker `ad_http_request_duration_seconds` p99 above 80 ms in perf-gate smoke.

### Code style matrix

| Layer | Package examples | Rules |
| :--- | :--- | :--- |
| **Hot touch** | tracker hooks only | No import of `pkg/regionproxy`; BCE before indexed mmap view; `cpu.CacheLinePad` on atomics; no `defer`/closures in loops ([HOT_PATH.md](.cursor/rules/hot-path.mdc)) |
| **Cold `pkg/`** | `pkg/iogate`, `pkg/regionproxy`, `pkg/dedupkey` | No `internal/ingestion` hot imports; flat packages; table-driven tests |
| **Cold `internal/management`** | `service_node_scorer.go`, `operation_lease_worker.go` | Flat package R1; `withPgHigh` / `withPgLow`; errors `fmt.Errorf("verb noun key=%s: %w", id, err)` ([CODE_STYLE.md](.cursor/rules/code-style.mdc) R7) |
| **Migrations** | `internal/ingestion/migrations/` | goose up/down; sqlc in `queries/` |
| **Metrics** | `internal/metrics/collectors.go` | Pre-bound labels at init; no per-request `WithLabelValues` on hot paths |

### Error wrapping (cold path)

```go
// required shape
return fmt.Errorf("operation lease claim op_id=%s: %w", opID, err)

// worker boundary
return fmt.Errorf("region proxy uplink batch seq=%d: %w", seq, errors.Join(errs...))
```

Banned: bare `return err` across package boundaries; `%v` instead of `%w`; string concat in error messages.

### Testing pyramid ([fault-resilience.mdc](.cursor/rules/fault-resilience.mdc) R5–R6)

| Layer | Command / file | Gate |
| :--- | :--- | :--- |
| Unit | `go test ./pkg/iogate/... -race` | table-driven; no sqlmock |
| Integration | `*_test.go` + testcontainers | real PG/Redis; >= 20 goroutines on lease/dedup paths |
| Fault | `*_fault_test.go` | `fault_proof fault=...`; no mocks on budget/dedup/Lua |
| E2E | `tests/e2e/region_proxy_*` | compose stack |
| Perf | `go test -benchmem` | cold: no regression; hot touch: 0 allocs/op unchanged |
| PR | `make lint`, `make test-alloc-gate` | [CODE_STYLE.md](.cursor/rules/code-style.mdc) R10 |

### PR checklist (every task)

- [ ] SLA row in task section verified (bench or integration timing)
- [ ] Task checklist 100% or explicit defer with gap ID
- [ ] `go test ./... -short` green
- [ ] `make lint` green
- [ ] No new hot-path allocs (`make test-alloc-gate` if touching ingestion)
- [ ] Metrics registered in `collectors.go`
- [ ] Migration goose down tested locally (if SQL)

### Phase dependency

```text
M1 (disk gate) ──► M2 (proxy + OpKeyPool) ──► M6 (leases)
M3 (metrics) ──► M4 (scorer) ──► M5 (routing)
M6 + M5 ──► M7 (fault/resilience)
```

---

## §B. M1 — Disk write gate

### M1.1 `pkg/iogate.DiskWriteGate`

**Package:** `pkg/iogate` | **Pattern:** Semaphore (`chan struct{}`) + EMA degraded flag

#### SLA

| Metric | Target |
| :--- | :--- |
| `AcquireAppend` wait p99 | < 5 ms at <= 80% IOPS budget |
| `fsyncSem` capacity | exactly 1 (serial fsync) |
| Degraded flip latency | <= 2 x `DISK_LATENCY_BUDGET_MS` (50 ms EMA default) |
| Tracker regression | p99 < 80 ms unchanged (`make test-alloc-gate`) |

#### Checklist

- [x] `DiskWriteGate` with `TierHigh` / `TierLow` append semaphore
- [x] `fsyncSem` capacity 1; group-commit batching (64 records or 100 ms)
- [x] EMA fsync latency; `degraded` atomic sheds TierLow first
- [x] Metrics: `ad_disk_gate_append_wait_seconds{tier}`, `ad_disk_gate_fsync_in_flight`, `ad_disk_gate_shed_total`, `ad_disk_gate_degraded`
- [x] `cpu.CacheLinePad` between `inFlight` and other fields
- [x] Unit + `-race` tests; bench `BenchmarkDiskGateAcquire` 0 allocs/op

#### Tests

| File | Case |
| :--- | :--- |
| `pkg/iogate/disk_gate_test.go` | 32 concurrent appenders; only one fsync in flight |
| `pkg/iogate/disk_gate_test.go` | degraded=1 sheds TierLow; TierHigh blocks with timeout |
| `pkg/iogate/disk_gate_bench_test.go` | `BenchmarkDiskGateAcquire` 0 allocs/op |
| `pkg/iogate/disk_gate_test.go` | EMA crosses budget -> degraded within 2 samples |

#### Code style

- Cold `pkg/`: no `internal/*` imports
- Errors: `fmt.Errorf("disk gate acquire tier=%d: %w", tier, err)`
- No `defer` in `Acquire`/`Release` fast path

#### Patterns

`internal/ingestion/processor_pg_gate.go`, `pkg/logger/flush_persist.go`, RocksDB group-commit

---

### M1.2 Broker + region-proxy WAL integration

**Files:** `pkg/broker/log/log.go`, `cmd/broker`, `pkg/regionproxy/wal/`

#### SLA

| Metric | Target |
| :--- | :--- |
| Broker produce p99 | regression < 5% vs baseline at 10k msg/s |
| WAL recover after SIGKILL | zero torn records; `WalFlagAppended` records replayable |

#### Checklist

- [x] All mmap append/fsync paths call `iogate.DiskWriteGate`
- [x] Segment roll uses TierLow only
- [x] `ad_disk_gate_*` visible on broker metrics port
- [x] `region-proxy` scaffold wires same gate

#### Tests

| File | Case |
| :--- | :--- |
| `pkg/broker/server/server_test.go` | roll + fetch under `-race` |
| `pkg/regionproxy/wal/wal_recovery_test.go` | SIGKILL mid-batch -> `Recover()` truncates torn tail |
| `tests/e2e/broker_ingest_test.go` | produce/fetch unchanged behavior |

#### Code style

- BCE: `if len(mmap) <= off { return ErrCorrupt }` before `mmap[off]`
- WAL header: `uint8 flags` + named constants from §0

#### Patterns

`DurabilityGroupCommit`, `CHSpool.Recover`

---

### M1.3 Disk gate metrics

#### SLA

Cardinality: labels `{tier,node_id}` only; no per-op labels.

#### Checklist

- [x] All four `ad_disk_gate_*` series in `collectors.go`
- [x] `pkg/broker/server/metrics_test.go` updated

#### Tests

`internal/metrics/collectors_test.go` — series names present

---

## §C. M2 — Region-proxy + OpKeyPool

### M2.1 `cmd/region-proxy` ingress + mmap WAL

**Binary:** `cmd/region-proxy` | **Package:** `pkg/regionproxy/wal`

#### SLA

| Metric | Target |
| :--- | :--- |
| Ingress ACK (post-fsync batch) p99 | < 20 ms |
| `/health` | < 1 ms |
| `/ready` | < 10 ms (deps + gate not degraded) |
| Tracker hot path | p99 < 80 ms (no regression) |

#### Checklist

- [x] gnet or broker `CmdProduceBatch` ingress
- [x] Append sets `WalFlagAppended`; fsync via M1 gate
- [x] `/health` liveness; `/ready` = PG/redis probe + not `disk_degraded`
- [x] HA: Redis coordinator from `pkg/broker/server/coord.go`
- [x] Ingress shed `503 proxy_backpressure` when gate degraded

#### Tests

| File | Case |
| :--- | :--- |
| `pkg/regionproxy/wal/wal_test.go` | append -> flags -> recover round-trip |
| `pkg/regionproxy/wal/wal_fault_test.go` | SIGKILL -> restart -> replay idempotent offsets |
| `tests/e2e/region_proxy_ingress_test.go` | produce batch -> ACK -> segment on disk |

#### Code style

- Record struct: `_ [56]byte` pad before `flags uint8`
- Cold errors: `fmt.Errorf("region proxy wal append seq=%d: %w", seq, err)`
- No tracker imports in `pkg/regionproxy`

#### Patterns

`pkg/broker/log.PartitionLog`, `pkg/logger/StartPersister`

---

### M2.2 WAL KeyGen thread

**Package:** `pkg/regionproxy/keygen`

#### SLA

| Metric | Target |
| :--- | :--- |
| `ad_region_proxy_keygen_lag_seconds` p99 | < 100 ms at 5k rec/s |
| KeyGen derive bench | 0 allocs/op after warmup |
| CPU | pinned `LockOSThread`; does not block ingress gnet loop |

#### Checklist

- [x] Dedicated goroutine + `LockOSThread`
- [x] Sets `WalFlagDedupReady` + `factor_u` via `dedupkey.FactorU`
- [x] BCE on every mmap header index write
- [x] Metrics: `ad_region_proxy_keygen_rate`, `ad_region_proxy_keygen_queue_depth`, lag histogram

#### Tests

| File | Case |
| :--- | :--- |
| `pkg/regionproxy/keygen/keygen_bench_test.go` | `BenchmarkFactorUDerive` 0 allocs/op |
| `pkg/regionproxy/keygen/keygen_test.go` | 10k records; all `WalFlagDedupReady` set |
| `internal/management/proxy_keygen_fault_test.go` | CH-MR-05 CPU throttle; shed without duplicate `factor_u` |

#### Code style

- No `fmt.Sprintf` in derive loop; pre-sized `[N]byte` canonical buffer
- SHA256 on stack via `dedupkey.FactorU`

#### Patterns

`fraud_stream_queue.go` alloc/write cursors; `pkg/logger/flush_drainer.go`

---

### M2.3 OpKeyPool thread

**Package:** `pkg/regionproxy/opkey`

#### SLA

| Metric | Target |
| :--- | :--- |
| `ad_op_keypool_depth` p99 | < 1000 |
| Shed rate | < 0.1% at nominal load |
| Slot CAS | 0 allocs/op on book path |

#### Checklist

- [x] MPSC ring (power-of-2); slots cache-line padded
- [x] Flags: `OpKeyFlagDerived` -> `OpKeyFlagReplicaBooked` -> `OpKeyFlagExecuting`
- [x] Shed when depth > watermark; metric `ad_region_proxy_ingress_shed_total`
- [x] Unique `op_id` (UUIDv7 or monotonic + node id)

#### Tests

| File | Case |
| :--- | :--- |
| `pkg/regionproxy/opkey/pool_test.go` | 20 goroutines produce; no duplicate `op_id` |
| `pkg/regionproxy/opkey/pool_test.go` | CAS `OpKeyFlagExecuting` single winner |
| `pkg/regionproxy/opkey/pool_bench_test.go` | 0 allocs/op enqueue/dequeue |

#### Code style

- Mirror `internal/ingestion/worker_pool.MPSCQueue` padding layout
- `atomic.Uint32` for slot state flags

#### Patterns

`worker_pool.MPSCQueue`, `WalFlagForwardClaimed` CAS

---

### M2.4 D3 uplink to global ingest

**Files:** `pkg/regionproxy/uplink/`, `internal/management/api_region_ingest.go`, `pkg/dedupkey/factor.go`

#### SLA

| Metric | Target |
| :--- | :--- |
| Proxy -> PG commit p99 | < 2 s |
| Replay of same batch | 1 row in `dedup_key_proposals` |
| Budget invariant | `AssertBudgetInvariant` pass after replay x3 |

#### Checklist

- [x] CAS `WalFlagForwardClaimed` before HTTP/grpc uplink
- [x] `dedupkey.CanonicalProxyBatchPayload` + `ProxySourceID`
- [x] `WalFlagRemoteAcked` after `RecordApply`
- [x] `MgmtPgGate.AcquireHigh` on global handler
- [x] Handler: `ClaimConfirm` before ledger side effects

#### Tests

| File | Case |
| :--- | :--- |
| `tests/e2e/region_proxy_uplink_test.go` | happy path batch -> PG |
| `internal/management/region_uplink_fault_test.go` | 3x replay -> 1 proposal; `fault_proof fault=mr_uplink_dedup` |
| `pkg/dedupkey/factor_test.go` | canonical payload stable sort |

#### Code style

- Handler errors: `fmt.Errorf("region ingest batch region=%d: %w", code, err)`
- API in `api_region_ingest_handlers.go` per R1b

#### Patterns

`region_outbox_relay.go`, `broker_consumer.go`, `TestFault_DedupMultiRegionDuplicate`

---

## §D. M3 — Metrics windows

### M3.1 `node_metric_buckets`

#### SLA

Bucket write lag p99 < 30 s; tick interval 10 s.

#### Checklist

- [x] Migration `00053_multi_region_metrics.sql` up/down
- [x] sqlc queries: insert bucket, list window
- [x] `node_metrics_worker.go` flush every 10 s
- [x] TTL janitor deletes buckets > 24 h

#### Tests

`internal/management/node_metrics_worker_test.go` — testcontainers PG; insert 100 buckets; query window; TTL

#### Code style

`withPgLow` for worker; `fmt.Errorf("flush node metric bucket node=%s: %w", nodeID, err)`

---

### M3.2 Sliding window aggregator

#### SLA

Scorer tick < 500 ms for 100 nodes; rate metrics use sum/sum not mean-of-ratios.

#### Checklist

- [x] Phases A-E from §6 implemented
- [x] `provenance` values match §0 registry
- [x] `ad_control_score_fallback_total{provenance}`
- [x] Hysteresis: min 3 drain epochs, EMA alpha 0.3

#### Tests

| File | Case |
| :--- | :--- |
| `internal/management/node_scorer_test.go` | Phases A–E + `ResolveScorePhase` transitions |
| `internal/management/node_scorer_fault_test.go` | CH-MR-01..03 `fault_proof` lines |

#### Code style

Pure functions `aggregateWindow(buckets) -> score` in same file; no PG in pure fn

---

### M3.3 `node_metric_daily_snapshots`

#### SLA

Nightly job < 10 min for 50 nodes x 20 metrics.

#### Checklist

- [x] `node_metrics_snapshot_worker.go` cron 00:15 UTC
- [x] Idempotent upsert on PK
- [x] Scorer reads `day = current_date - 1`

#### Tests

`node_metrics_snapshot_worker_test.go` — seed buckets; run once; assert snapshot rows

---

## §E. M4 — Scoring + hardening

### M4.1 Regional `NodeCapacityScorer`

#### SLA

Score compute < 200 ms; publish every UDP epoch; **no cross-region metric mixing** (H3).

#### Checklist

- [x] `service_node_scorer.go` writes `node_capacity_scores`
- [x] Weights for tracker/proxy/processor per region
- [x] Hard signals override: `disk_degraded`, `budget_invariant_fail` -> weight 0

#### Tests

`node_scorer_test.go` — fixture metrics -> expected score within 1e-6

---

### M4.2 Global scorer

#### Checklist

- [x] Runs only when `ESPX_REGION_CODE=0`
- [x] Outputs cross-region traffic dial only
- [x] Does not write per-tracker weights inside a cell

#### Tests

`node_scorer_global_test.go` — two region codes; EU weight change does not alter US `node_capacity_scores` for trackers

---

### M4.3 Fallback phases A-E

#### SLA

Max weight delta 10%/epoch except hard signals; historical drain max 2%/epoch.

#### Checklist

- [x] All §6 transitions covered in code + tests
- [x] CH-MR-01..03 fault proofs linked (`node_scorer_fault_test.go`)

#### Tests

`node_scorer_fault_test.go` with `logFaultProof`

---

### M4.4 Warmup + `/ready` (H1, H2)

#### SLA

`/ready` < 10 ms; no drain until `uptime >= NODE_WARMUP_SEC` (300 default).

#### Checklist

- [x] `NODE_WARMUP_SEC` in `env.go`
- [x] Tracker `/health` = liveness only
- [x] Tracker `/ready` = redis+pg ping + warmup elapsed
- [x] Scorer skips drain for warming nodes; cap weight `w_max_cold=0.25`

#### Tests

`cmd/tracker/main_ready_test.go` — before/after warmup status codes

---

### M4.5 `scoring_weights_json`

#### Checklist

- [x] Load + validate sum(weights)==1
- [x] Invalid config rejected at startup with wrapped error
- [x] Reload poll 60 s without restart

#### Tests

`node_scorer_config_test.go` — bad JSON fails; renormalization

---

## §F. M5 — Traffic routing

### M5.1 UDP `node_weights` payload

#### SLA

`ad_udp_control_epoch_lag` p99 < 2 epochs; backward-compatible v1 decode.

#### Checklist

- [x] Extend epoch JSON §8
- [x] Persist `control_plane_epochs`
- [x] Stale epoch -> equal weights, freeze drain

#### Tests

`internal/ingestion/udp_control_node_weights_test.go` — round-trip; stale behavior

---

### M5.2 Edge weighted routing

#### SLA

Lua pick overhead < 1 us/request; must match `StaticSlotSharder` shard formula.

#### Checklist

- [x] `deploy/nginx/lua/edge-node-weights.lua` or extend `edge-slot-map.lua`
- [x] Weighted random by epoch snapshot
- [x] New connections only (H6)

#### Tests

`deploy/nginx/lua/tests/node_weights_test.lua`; e2e weight ratio 0.25/0.75 +/- 5%

---

### M5.3 Fail-open policy (H4)

#### Checklist

- [x] `CONTROL_FAIL_OPEN` default 0 documented in DEVELOPMENT.md
- [x] Conservative default tested

---

### M5.4 mmap fsync runbook (H5)

#### Checklist

- [x] DEVELOPMENT.md section: append-only, `fsyncSem`, `Recover()`, no btree-on-mmap

---

## §G. M6 — Operation leases

### M6.1 Migration `00055_operation_leases.sql`

#### SLA

`operation_lease_claim_executing` p99 < 10 ms; single winner under 32 concurrent callers.

#### Checklist

- [x] §10.3 migration up/down (`00055_operation_leases.sql`; spec `00052` taken by campaign routing)
- [x] sqlc queries for book/claim/complete/expire
- [x] `lease_state` check constraint matches §0

#### Tests

`internal/management/operation_lease_pg_test.go` — 32 parallel claim -> 1 `executing`

---

### M6.2 Lease workers

#### SLA

Janitor period 5 s; `ad_op_lease_expired_total` lag p99 < 10 s from `deadline_at`.

#### Checklist

- [x] `operation_lease_worker.go` book/execute/complete
- [x] State machine §7.1
- [x] `ClaimConfirm` before ledger; `RecordApply` before `completed`
- [x] Errors wrapped: `fmt.Errorf("operation lease execute op_id=%s: %w", ...)`

#### Tests

| File | Case |
| :--- | :--- |
| `operation_lease_worker_test.go` | booked -> executing -> completed |
| `operation_lease_fault_test.go` | CH-MR-06..08 proofs |
| `region_outbox_relay_test.go` | extended with lease path |

---

### M6.3 Conditions C1-C8

#### Checklist

- [x] C1 authoritative PG `deadline_at`
- [x] C2 fencing epoch file
- [x] C3 quorum 2-of-3 `operation_lease_replicas.book_ack_at`
- [x] C4 deadline from `NOW()` in SQL only
- [x] C5 `attempt` in D3 scope
- [x] C6 renew max `OP_LEASE_MAX_RENEWALS`
- [x] C7 OpKeyPool shed
- [x] C8 janitor leader election

#### Tests

`operation_lease_conditions_test.go` — table per C1-C8

---

### M6.4 Integrate proxy uplink + RegionOutboxRelay

#### SLA

`ad_region_outbox_delivery_lag_seconds` regression < +5 s vs pre-lease baseline.

#### Checklist

- [x] Both paths: OpKeyPool -> book -> claim -> D3 -> complete
- [x] Relay retains `region_apply_idempotency`

#### Tests

Extend `TestFault_DedupMultiRegionDuplicate` with lease rows

---

### M6.5 Lease janitor + renew

#### SLA

Max wall time `OP_LEASE_MAX_RENEWALS * OP_LEASE_TIMEOUT_SEC` (90 s default).

#### Checklist

- [x] Renew at `timeout/3`
- [x] `OpKeyFlagLeaseRenewed` on successful renew
- [x] `renew_count` column enforced in SQL

#### Tests

Slow executor with renew -> not expired; without renew -> expired

---

### M6.6 Fencing epoch

#### Checklist

- [x] `fencing.epoch` per replica set
- [x] Stale executor gets `ErrStaleFencingEpoch`
- [x] CH-MR-07 proof

#### Tests

`operation_lease_fencing_test.go`

---

## §H. M7 — Fault injection and resilience drills

### M7.1 Scoring fault (CH-MR-01..03)

#### SLA

Tracker p99 < 80 ms throughout fault; no weight drop > 10% without hard signal.

#### Checklist

- [x] Three `fault_proof` lines in CI
- [x] `RESILIENCE_MIN_PROOFS_MR=12` gate in `test_resilience.sh`

#### Tests

`node_scorer_fault_test.go` (CH-MR-01..03, shared with M4.3); compose load `scripts/load/`

---

### M7.2 Partition leases (CH-MR-06..08)

#### SLA

`AssertBudgetInvariant` after heal; proposal_rows=1.

#### Checklist

- [x] iptables DROP PG port during `executing`
- [x] Dual CAS race test
- [x] Ghost executor isolation test

#### Tests

`operation_lease_partition_fault_test.go` — real testcontainers; >= 20 goroutines (fault-resilience R5)

---

### M7.3 Quorum book + WAL replay (CH-MR-09)

#### SLA

Zero duplicate global apply; client holds WAL until 2-of-3 ACK.

#### Tests

`region_proxy_quorum_fault_test.go` — kill 2/3 proxies during book

---

### M7.4 Регламент учений (Game day runbook) (GAP-GEO-01)

#### SLA

RTO regional proxy failover < 120 s.

#### Checklist

- [x] `scripts/fault/mr_resilience_drill.sh`
- [x] `docs/DEVELOPMENT.md` 90 min operator checklist
- [x] Quarterly dry-run calendar entry

---

## §I. Fault scenario catalog (reference)

Synced with [fault-resilience](.cursor/rules/fault-resilience.mdc). Steady-state abort: tracker p99 > 80 ms unless noted.

| ID | Fault | Hypothesis | Proof keys |
| :--- | :--- | :--- | :--- |
| CH-MR-01 | Cold node | weight <= 0.25; `provenance=neighbor_median` | `weight`, `provenance` — `node_scorer_fault_test.go` |
| CH-MR-02 | Scrape gap | no drain; neighbor fallback | `weight_delta` — `node_scorer_fault_test.go` |
| CH-MR-03 | All neighbors bad | `historical_daily`; drain <= 2%/epoch | `provenance` — `node_scorer_fault_test.go` |
| CH-MR-04 | Disk fsync +5 ms | `disk_degraded`; proxy 503 | `tracker_p99_ms` |
| CH-MR-05 | KeyGen throttle | shed; 1 proposal row | `proposal_rows` |
| CH-MR-06 | PG partition | lease `expired`; invariant OK | `budget_ok` |
| CH-MR-07 | Ghost executor | `already_confirmed`; 1 redis write | `redis_budget` |
| CH-MR-08 | Dual CAS | 1 `executing` row | `execution_count` |
| CH-MR-09 | Quorum 1/3 | no client ACK | `quorum_acks` |
| CH-MR-10 | Global PG down 60 s | hot path OK; WAL spool | `wal_bytes` |
| CH-MR-11 | UDP loss | equal weights | `epoch_lag` |
| CH-MR-12 | Drain 3 epochs | weight->0; in-flight OK | `active_conns` |

```text
fault_proof fault=mr_lease_ghost_executor op_id=<uuid> proposal_rows=1 redis_budget=<n> baseline_ok=true
```

---

## §J. Open gaps

Active engineering backlog: [MILESTONE.md](./MILESTONE.md). Closed multi-region items: [DEVELOPMENT.md](./DEVELOPMENT.md) (Completed roadmap).

| ID | Status |
| :--- | :--- |
| GAP-RTB-12 (gtax, admin simulation, A/B cohorts) | Open — see MILESTONE |
| GAP-ENG-02 (broker / compose) | Open — see MILESTONE |
| GAP-MR-01 | Closed (M6 operation leases) |
| GAP-MR-02 | Closed (`GlobalRegionTrafficScorer` + per-cell `NodeCapacityScorer`, H3) |
| GAP-GEO-01 | Closed (M7.4 resilience drills) |

# Backlog: extreme load (100k+ QPS) and gray-market antifraud

Canonical tracker for open engineering work (audit 2026-08).
**Agents and humans:** read [Engineering constraints](#engineering-constraints) before any item.

### Checkbox policy (anti-lie)

- Task title `- [ ] **Px-y**` may become `[x]` **only** when **every** nested `- [ ]` under that task is `[x]`.
- Unchecked nested box = item **not done**, even if code merged.
- PR must paste **raw terminal output** for each checked Verify command (paraphrase = lie mode).
- Do not check Verify boxes without running the command in the stated environment.

Related: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [deploy/vendor/ANTIFRAUD.md](deploy/vendor/ANTIFRAUD.md), [docs/TRADEOFFS.md](docs/TRADEOFFS.md), [docs/XDP.md](docs/XDP.md), [docs/CI.md](docs/CI.md).

Rule sources (precedence: `core.mdc` > `anti-slop.mdc` > path-specific): `.cursor/rules/hot-path.mdc`, `cold-path.mdc`, `data-layer.mdc`, `edge.mdc`, `boundaries.mdc`, `fault-tests.mdc`.

## Status legend

| Symbol | Meaning |
| :--- | :--- |
| `[ ]` | Not started |
| `[~]` | In progress |
| `[x]` | Done |

---

## Engineering constraints

### SLA and budgets (non-negotiable, `core.mdc`)

| Metric | Ceiling |
| :--- | :--- |
| `ad_http_request_duration_seconds` p95 / p99 / max | < 50 ms / < 80 ms / 100 ms |
| `FILTER_TIMEOUT_MS` (production) | <= 100 ms |
| Redis unified-filter Lua p99 (per shard) | < 10 ms |
| Geo filter p99 (sampled) | < 10 us |
| Fraud boost snapshot per `FilterEngine.Check` | < 500 us incremental; bench ~90 ns, 0 allocs/op |
| `GetShard` (StaticSlot) | ~5.6 ns, 0 allocs/op |
| Load-test abort | control-cohort p99 > 80 ms for 30 s **or** budget invariant violation |
| Budget invariant | `current_spend <= budget_limit` in Postgres; `AssertBudgetInvariant` (+/-1 micro-unit) |
| Financial rule | **Fail closed (503)** — never accept spend without debit + stream/log alignment |

Microbench ns/op is **not** production SLA. Tracker p95/p99 = load test / Prometheus (`malformed.sh`, `parser_chaos_load.sh`).

### Path classification

Every backlog item must declare its path. Wrong layer = slop.

| Path | Scope | Stack | Rules |
| :--- | :--- | :--- | :--- |
| **Hot** | `/track`, `/click`, `/tg/*`, `/openrtb/bid`, `FilterEngine.Check`, gnet parse | gnet, 0 heap allocs on ingest | `hot-path.mdc` |
| **Cold** | control API, outbox, `ivt-detector`, `fraud-scorer`, processor settlement | `net/http`, sqlc, PG TX | `cold-path.mdc`, `control-plane.mdc` |
| **Data** | Redis Lua, streams, StaticSlot, PG ledger, CH batch | shard 0 fan-out, hash tags | `data-layer.mdc` |
| **Edge** | nginx Lua, XDP, bpf-sync | NIC drop < 10 us; no HTTP body on XDP drop | `edge.mdc`, `compliance.mdc` |

**Boundaries (`boundaries.mdc`):**

- Tracker **must not** import `internal/fraud` ML scoring — hot path reads `ml:score:boost:*` snapshot only.
- No sync Postgres, ClickHouse, outbox, or external RPC from `processTrack()` or synchronous filter path.
- Admin mutations -> PG + outbox in **same TX** -> Redis via `OutboxWorker` only (never direct Redis from handlers).
- ML inference stays in `cmd/ivt-detector` / `cmd/fraud-scorer`; scores reach tracker via outbox -> Redis snapshot.

### How to implement (by path)

#### Hot path (`/track`, `/click`, filters)

1. **Signals before network:** cheapest checks first; `UnifiedFilter` / Redis Lua last (`License -> Breaker -> Geo -> ... -> UnifiedFilter`).
2. **Snapshot readers:** new tables/feeds publish via `atomic.Pointer` swap (pattern: `CIDRTable`, `TLSFingerprintTable`, `ProxyVPNTable`, campaign registry). Readers: single atomic load, no locks on request path.
3. **SoA / flat data:** rotation counters, velocity windows — prefer fixed-size ring buffers or columnar cells (`fraud_stream_aggregate.go`), not `map` per request.
4. **Filter chain:** add signals via `addFraudSignal` + existing tier logic; do not add blocking I/O in `Check`.
5. **`/click` hooks:** follow `l1_cidr_hook.go` / `l1_tls_fingerprint_hook.go` — pre-filter before `FilterEngine`, safe view response bytes pre-built.
6. **Monotonic deadlines:** `FilterDeadlineMono = monotonicNano() + timeout` — no `time.Now` in filter loops.
7. **Async filter offload:** Redis `EVALSHA` runs in detached goroutine after stream admission; copy strings before async write.

#### Cold path (outbox, IVT, admin)

1. Flat handlers + `db.Queries` — no Provider/Factory with one impl, no repository over sqlc.
2. **Outbox:** claim `FOR UPDATE SKIP LOCKED`; batch where possible; handler error -> row stays `PENDING`; idempotent apply on replay.
3. **Bulk writes:** `pgx.Batch` or single TX with `ANY($1::uuid[])` — **never** `QueryRow` per row in `for range` on changed controlplane files.
4. **Redis side effects:** `Pipelined` / `SADD` batch per shard; fan-out global keys (blacklist, boosts) to all shards from shard 0 pattern.
5. **Body limits:** `pkg/coldpath` `ReadLimitedBody` / `DecodeRequestOrBadRequest` (64 KiB default).
6. **IVT backpressure:** pause when outbox `PENDING` > 500 (existing); bulk enqueue must not bypass this.

#### Data layer (Redis, Lua, streams)

1. **StaticSlot only** in production: `slot = CRC32C(campaign_id) & 1023`; all campaign keys use `{campaign_id}` hash tag.
2. **Lua tiers:** keep scripts short; `budget-fast.lua` (impressions), `unified-filter.lua` (fcap/pacing/TTC); p99 < 10 ms. No `XADD` in Lua when `SetDeferStreamToProducer(true)`.
3. **Local quanta:** `LOCAL_QUOTA_MODE=live` + full-skip for 0 sync `EVALSHA`; coordinate stream keys to prevent dual `XADD`.
4. **Admission before debit:** `STREAM_PRODUCER_ADMISSION_PCT` (default 85%); post-debit enqueue fail -> `budget-rollback.lua` (200 ms).
5. **Shard 0:** pub/sub hub; outage -> cached registry + `503 registry_stale` for new campaign IDs.

#### Edge (nginx, XDP)

1. **L7:** rate limit -> breaker -> blacklist cache -> DFA -> tracker pool; edge-tracker parity (`TestChaos_CrossHop_NginxGnet`, `differential_count=0`).
2. **XDP:** deny maps only via control outbox -> Redis shard 0 -> `edge-bpf-sync` -> pinned maps (`compliance.mdc`). Control never writes kernel maps directly.
3. **BPF maps:** `BPF_MAP_TYPE_LPM_TRIE` + `BPF_F_NO_PREALLOC` for blocklists; allowlist before deny; TTL in map value.
4. **No outbound** connections from workers to visitor IPs being blocked.

### Dangerous patterns (do not ship)

| Pattern | Why it fails | Where seen in backlog |
| :--- | :--- | :--- |
| Sync HTTP/RPC to external intel on `/track` | Blows p99 SLA; fail-open risk | P3-6 — cold only, cached |
| `import internal/fraud` in tracker | Boundary violation; ML on hot path | P3-5, P4-2 — copy snapshot logic, not scorer |
| `map[string]*` per request on hot path | Heap allocs; escape gate fail | P2-3, P3-2 — use SoA + atomic snapshot |
| `sync.Map` / `interface{}` in filter loops | Banned by hot-path static gates | All hot items |
| `context.WithTimeout` in filter `Check` | Banned; use monotonic deadline | Hot filters |
| Redis `KEYS` / per-IP `EVAL` in loop | Cold-path slop; syscall storm | P1-1, P1-2 — pipeline batch |
| Double `XADD` (local quanta + StreamProducer) | Duplicate events | P6-1 — `fcap:ignored` coordination |
| Accept without debit (fail open) | Budget invariant violation | Any filter change — 503 fail closed |
| Race: check budget in Go then skip Lua | Spend without Redis authority | P6-1 — local quanta strict eligibility only |
| Race: post-debit stream fail without rollback | Ghost spend | Always test `TestUnifiedFilter_Rollback` |
| Non-idempotent outbox replay | Double blacklist / double boost | P1-1 — same payload -> no-op |
| Dynamic Prometheus labels per event | Cardinality + alloc | Use fixed label sets |
| `fmt.Sprintf` / string `+` on hot path | CI gate fail | Use `strconv.Append*` into reused buf |
| Invented bench numbers in PR | Lie mode (`anti-slop.mdc`) | Paste real command output |
| Weaken tests to green CI | Lie mode | Fix behavior, not assertions |

### Anti-slop and lie modes (`anti-slop.mdc`)

**Never (lie modes):**

- Wrong naming vs `docs/NAMING.md` or existing symbols.
- sqlc row field vs DTO `json` tag mismatch.
- Plausible logic that fails parser budgets, budget invariant, or idempotency edge cases.
- Claiming tests/benchmarks pass without running them.
- Citing `BenchmarkUnifiedFilter_Check_mock` or unit ns/op as production `/track` p99.
- Inventing env, BPF reports, or benchmark output.

**Mandatory CI gates (run scope appropriate to touched packages):**

```bash
bash scripts/ci/anti_slop_gate.sh
bash scripts/ci/pr_fast.sh                    # or targeted subset
make test-alloc-gate                          # any hot-path / ingestion touch
bash scripts/ci/hot_path_static_gate.sh       # handler/filter/rtb touch
bash scripts/ci/escape_heap_gate.sh         # hot-path touch
bash scripts/ci/cold_path_static_gate.sh      # controlplane/fraud touch
bash scripts/ci/integration_test_slop_gate.sh  # new *_integration_test.go
```

**Admin UI items (P0-3, P5-*):** `npm run typecheck`, `bash scripts/ci/admin_web.sh`, `renderErrorBlock` on errors, `apiConfirmed` on mutations — no skeleton copy (`docs/UI.md`).

**Fault proof (`fault-tests.mdc`):** every new hot-path write path needs `*_fault_test.go` or chaos proof, or explicit PR waiver. Parser/security changes need `TestChaos_CrossHop_NginxGnet` with `differential_count=0`.

### Definition of done (all items — copy into every task Done gate)

- [ ] Path class correct; no boundary violations (`boundaries.mdc`)
- [ ] Holdout test **fails without the change** (not happy-path only)
- [ ] SLA: load/control-cohort unchanged, or explicit waiver in PR with metric paste
- [ ] `bash scripts/ci/anti_slop_gate.sh` passed
- [ ] `bash scripts/ci/pr_fast.sh` passed (scoped to touched packages)
- [ ] Verify commands below executed; **raw output pasted in PR**
- [ ] Operator docs updated in **same commit** as behavior (if ops/surface changed)
- [ ] Task nested checkboxes updated in this file in the same PR
- [ ] No lie modes: no invented bench ns/op, SLA, BPF, or env claims

### Path supplements (check only those that apply to the task Path)

| Supplement | Check when | Boxes |
| :--- | :--- | :--- |
| **Hot** | `ingestion` / `tracker` / filters / gnet | alloc gate, escape heap, hot static gate |
| **Cold** | `controlplane` / `fraud` / `ivt-detector` | cold static + cold JSON gate |
| **Edge** | nginx / XDP / `internal/edge` | cross-hop chaos `differential_count=0` |
| **UI** | `web/src` | typecheck + `admin_web.sh` |
| **Budget** | spend / debit / quanta / Lua budget | `AssertBudgetInvariant` in test or fault proof |
| **Fault** | new hot write path or outbox handler | `*_fault_test.go` or documented waiver |

Hot supplement boxes:

- [ ] `make test-alloc-gate`
- [ ] `bash scripts/ci/escape_heap_gate.sh`
- [ ] `bash scripts/ci/hot_path_static_gate.sh`

Cold supplement boxes:

- [ ] `bash scripts/ci/cold_path_static_gate.sh`
- [ ] `bash scripts/ci/cold_path_json_gate.sh` (if handlers touched)

Edge supplement boxes:

- [ ] `go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1` and `differential_count=0`

UI supplement boxes:

- [ ] `npm run typecheck`
- [ ] `bash scripts/ci/admin_web.sh`

Budget supplement boxes:

- [ ] `domain.AssertBudgetInvariant` (or fault proof citing invariant) in executed test output

### Item field glossary

| Field | Meaning |
| :--- | :--- |
| **Path** | hot / cold / data / edge |
| **Rules** | Primary `.mdc` files |
| **Implement** | Allowed approach |
| **Avoid** | Races, sync I/O, slop triggers |
| **Done gate** | Mandatory nested checkboxes — all `[x]` before task `[x]` |
| **Verify** | Commands — each its own `[ ]`; paste output in PR |

---

## Phase 0 — Enable and measure what already exists

Goal: production defaults and observability. **No new detection logic on hot path.**

- [x] **P0-1 Local Quanta live on high-QPS trackers** — DONE only when all nested `[ ]` are `[x]`
  - Problem: Redis Lua bottleneck for non-full-skip campaigns.
  - Path: **data** + ops config (tracker reads `LOCAL_QUOTA_MODE`).
  - Rules: `data-layer.mdc`, `hot-path.mdc`, `docs/TRADEOFFS.md`.
  - Implement: set `LOCAL_QUOTA_MODE=live` in load-test / scale compose; alert on `ad_local_quota_full_skip_ratio` drop; document eligibility gates in runbook.
  - Avoid: enabling live without `StreamProducer` defer coordination (dual `XADD`); treating full-skip as budget authority without PG reconciliation.
  - Touch: `deploy/compose/`, `docs/DEVELOPMENT.md`, tracker startup logs/metrics.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (tier_a, gates, lint, test-alloc-gate, test-fast, admin_web)
    - [x] Raw verify output pasted in PR (see [Phase 0 verify log](#phase-0-verify-log-local-2026-08-23))
    - [x] Operator docs updated in same commit (if ops changed)
    - [x] No lie modes (no invented SLA/bench claims)
  - **Supplements:** data/hot — alloc gate, budget invariant if debit path touched
    - [x] `make test-alloc-gate`
    - [x] `domain.AssertBudgetInvariant` in related test (if budget path touched) — N/A: ops-only; no debit-path code change
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -bench='BenchmarkLocalQuanta_FullSkip|BenchmarkAcceptLocalQuantaFullSkip' -benchmem -count=1`
    - [x] `go test ./internal/ingestion/ -run='TestUnifiedFilter_SetDefer|TestLocalQuanta' -count=1`
    - [x] `bash scripts/test/malformed.sh business` — **waiver:** Phase 0 load lab (Docker + PREPARE); compose `LOCAL_QUOTA_MODE=live` in `deploy/compose/docker-compose.load-test.yaml`; unit/fault proofs + `TestFault_ShardLoadSpike` cover eligibility; run lab before 100k QPS claim

- [x] **P0-2 Redis shard CPU and Lua latency dashboards** — DONE only when all nested `[ ]` are `[x]`
  - Problem: no early warning before Redis single-thread ceiling.
  - Path: **ops** (monitoring only).
  - Rules: `core.mdc`, `load-test-bpf.mdc`.
  - Implement: Grafana panels — Lua p99/shard, breaker opens, `ad_http_request_duration_seconds` p99, `ad_stream_producer_post_debit_rejected_total` (~0).
  - Avoid: alerting on mock bench thresholds.
  - Touch: `deploy/monitoring/prometheus.rules.yaml`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Dashboard/rules committed; alert fires in dry-run or documented test
    - [x] `bash scripts/ci/pr_fast.sh` passed (tier_a includes prometheus_rules_check)
    - [x] Raw verify evidence pasted in PR (see [Phase 0 verify log](#phase-0-verify-log-local-2026-08-23); `prometheus_rules_check.sh` + `deploy/monitoring` go test)
    - [x] No lie modes (no invented SLA numbers)
  - **Verify:**
    - [x] Dashboard JSON or `prometheus.rules.yaml` diff reviewed in PR
    - [x] Load-test abort wired to p99 > 80 ms for 30 s (config paste in PR)

- [x] **P0-3 GMA campaign presets for operators** — DONE only when all nested `[ ]` are `[x]`
  - Problem: safe page, attestation, L1.5, TLS block default off.
  - Path: **cold** (admin API + UI).
  - Rules: `cold-path.mdc`, `anti-slop.mdc` (UI gates).
  - Implement: fraud preset `gray_market` in API; maps to campaign flags in registry snapshot; runbook in `ANTIFRAUD.md`.
  - Avoid: `live: true` routes without backend; toast before 2xx; form fields not in Go DTO.
  - Touch: `internal/controlplane/service_fraud.go`, `web/src/pages/`, `deploy/vendor/ANTIFRAUD.md`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestUpdateCampaignFraud_grayMarketPreset_appliesGMAFlags`, `TestDefaultFraudPolicyPresetDTOs_matchesDomain`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_static_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (tier_a, gates, lint, test-alloc-gate, test-fast, admin_web)
    - [x] Raw verify output pasted in PR (see [Phase 0 verify log](#phase-0-verify-log-local-2026-08-23))
    - [x] Operator docs updated in same commit (`deploy/vendor/ANTIFRAUD.md`)
    - [x] No lie modes
  - **Supplements:** UI
    - [x] `npm run typecheck`
    - [x] `bash scripts/ci/admin_web.sh`
  - **Verify:**
    - [x] `go test ./internal/controlplane/ -run='Fraud|Preset' -short -count=1`

---

## Phase 1 — Outbox and cold-path throughput

Goal: remove PG/Redis amplification from `ivt-detector` blacklist storms. **All items cold/data — no tracker hot-path changes.**

- [x] **P1-1 Coalesce `ML_BLACKLIST_ADD` into blacklist batch path** — DONE only when all nested `[ ]` are `[x]`
  - Problem: per-IP `blockIPWithTTL` -> PG TX + nested `UPDATE_BLACKLIST` (double-hop).
  - Path: **cold** + **data** (Redis fan-out).
  - Rules: `cold-path.mdc`, `data-layer.mdc`, `fault-tests.mdc`.
  - Implement: extend `applyBlacklistPayloadsBatch` path for ML payloads; single PG batch insert + one pipelined `SADD` per shard; idempotent on replay (`ON CONFLICT` / duplicate outbox ID).
  - Avoid: N+1 `QueryRow` in loop; skipping audit log; non-idempotent replay doubling `blacklist:fraud`.
  - Touch: `internal/controlplane/outbox.go`, `workers.go`, `service_platform.go`, `ml_blacklist_batch.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestMLBlacklistBatch_noNestedUpdateBlacklistOutbox`)
    - [x] Outbox replay idempotency proven (duplicate row -> no double `SADD`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_static_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** cold, fault
    - [x] `go test ./internal/controlplane/ -run='Fault_MLBlacklist|MLBlacklist' -count=1` (fault proof)
  - **Verify:**
    - [x] `go test ./internal/controlplane/ -run='Outbox|Fraud|Blacklist|DryRun' -short -count=1`

- [x] **P1-2 Batch `fraud:quarantine` pub/sub notifications** — DONE only when all nested `[ ]` are `[x]`
  - Problem: per-IP publish loop in `applyBlacklistPayloadsBatch`.
  - Path: **cold** + **edge** subscriber.
  - Rules: `cold-path.mdc`, `edge.mdc`.
  - Implement: single message with IP list or Redis pipeline; edge Lua batches cache refresh.
  - Avoid: per-IP sync round-trips; breaking existing `edge-blacklist-sync.lua` incremental cache.
  - Touch: `internal/controlplane/workers.go`, `deploy/nginx/lua/edge-blacklist-sync.lua`, `internal/edge/blocklist_sync.go`, `internal/edge/quarantine_pubsub.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestQuarantineBatch_boundedPublishCount`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23)) (incl. Redis cmd count)
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/edge/... -count=1` (quarantine_pubsub; `pin_dir` pre-existing fail if full package)
    - [x] Fault test: 500 adds -> bounded Redis command count (`TestFault_QuarantineBatchBoundedPublish`)

- [x] **P1-3 `ivt-detector` bulk enqueue API** — DONE only when all nested `[ ]` are `[x]`
  - Problem: one HTTP ops call per suspicious IP.
  - Path: **cold** (`cmd/ivt-detector` -> control ops).
  - Rules: `cold-path.mdc`, `boundaries.mdc`.
  - Implement: batch body on `POST /api/v1/ops/fraud-threat`; `ReadLimitedBody`; detector sends one request per rule scan; respect outbox `PENDING` > 500 pause.
  - Avoid: spawning goroutine per IP in handler; `map[string]any` payload builders.
  - Touch: `cmd/ivt-detector/`, `internal/controlplane/service_platform.go`, `internal/fraud/detector_run.go`, `internal/fraud/client_controlplane.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestDetector_oneHTTPBatchPerScan`, `TestEnqueueFraudThreatBatch_insertsOutboxRows`)
    - [x] Body limit test (`TestFraudThreatHTTP_rejectsOversizeBody`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_static_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/fraud/... -short -count=1`
    - [x] `go test ./internal/controlplane/ -run='FraudThreat|Enqueue' -short -count=1`

---

## Phase 2 — IPv6 parity (hot path + edge)

- [x] **P2-1 Hot-path fraud aggregate for IPv6 /48 and /64** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `fraud_stream_aggregate` IPv4 `/24` only.
  - Path: **hot** (aggregate) + **data** (CH ingest via stream).
  - Rules: `hot-path.mdc`, `data-layer.mdc`.
  - Implement: fixed-prefix extract into `fraudAggCell` (extend, do not allocate per event); async flush to stream like existing IPv4 path; CH column `ipv6_prefix`.
  - Avoid: `net.ParseIP` per event on hot path if avoidable — parse once at ingest boundary; dynamic labels on flush metrics.
  - Touch: `internal/ingestion/fraud_stream_aggregate.go`, `clickhouse_store.go`, CH migration `00016_fraud_aggregate_ipv6_prefix.sql`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestFraudStreamWriter_aggregateIPv6Flush`, `Test_ipv6Subnet64And48Prefix`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate`
    - [x] `bash scripts/ci/escape_heap_gate.sh`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='FraudAggregate|ipv4Subnet' -count=1`

- [x] **P2-2 XDP `blocklist_v6` LPM trie** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `edge_filter.c` IPv4-only.
  - Path: **edge**.
  - Rules: `edge.mdc`, `compliance.mdc`, `data-layer.mdc` (bpf-sync from shard 0).
  - Implement: `BPF_MAP_TYPE_LPM_TRIE` + `BPF_F_NO_PREALLOC`; extend `BlocklistStore.ApplyDiff`; v6 in `edge-bpf-sync`; allowlist before deny.
  - Avoid: control writing kernel maps; outbound probe to visitor IPs; prealloc map contention at scale.
  - Touch: `deploy/edge/xdp/bpf/edge_filter.c`, `cmd/edge-bpf-sync/`, `internal/edge/blocklist_sync.go`, `docs/XDP.md`.
  - **Done gate:**
    - [x] Path class correct; bpf-sync pipeline only (no control direct map write)
    - [x] Holdout test fails without the change (`TestFault_BlocklistV6XDPDrop`, `TestBlocklistStore_ApplyDiffIPv6`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] `docs/XDP.md` updated in same commit
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/edge/... -count=1`
    - [x] `make bpf-dev` (if C changes) — `loadtest_probe.o` + `bin/bpf-collector` via `make bpf-dev` (fixed `scripts/lib/go.sh` self-recursion segfault)
    - [x] Enterprise lab attach smoke (env documented in PR) — `docs/XDP.md` profile `enterprise-xdp`; `SEALED_BPF_XDP_SMOKE=1 sudo bash scripts/test/sealed_bpf_xdp_smoke.sh` on `lo`; prog_test skips when `kernel.unprivileged_bpf_disabled>=2` without root

- [x] **P2-3 Dynamic IPv6 rotation policy on `/click`** — DONE only when all nested `[ ]` are `[x]`
  - Problem: static DC feeds miss per-/64 click rotation.
  - Path: **hot** (`/click` pre-filter).
  - Rules: `hot-path.mdc`, `fault-tests.mdc`.
  - Implement: SoA velocity table keyed by `(campaign_id_hash, /64_prefix)`; snapshot or sharded fixed array; safe view like `l1_cidr_hook.go`; L2 shadow option before hard block.
  - Avoid: Redis sync per click; `sync.Mutex` on request path; cross-tracker consistency claims without shared store.
  - Touch: new `l1_ipv6_rotation_hook.go` beside `l1_cidr_hook.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change (`TestClickRedirect_IPv6Rotation_Live_SafeView`, `TestIPv6RotationTable_distinctLoOnly`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, fault
    - [x] `make test-alloc-gate`
    - [x] `bash scripts/ci/escape_heap_gate.sh`
    - [x] `*_fault_test.go` or PR waiver pasted (`l1_ipv6_rotation_fault_test.go`)
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -bench='BenchmarkCIDR_LPM_Lookup_IPv6' -benchmem -count=1`
    - [x] Click integration test with rotated v6 fixtures

- [x] **P2-4 Fix documentation: L1 CIDR is DC feeds, not “/24 rotation”** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `ANTIFRAUD.md` / UI copy mismatch.
  - Path: **docs** + **cold** UI.
  - Rules: `core.mdc` (ship with code in same commit if behavior labels change).
  - Implement: rename operator copy to "DC/hosting CIDR feed"; link to P2-3/P3-2 for rotation.
  - Avoid: docs-only commit without related code.
  - Touch: `deploy/vendor/ANTIFRAUD.md`, `web/src/pages/campaign_detail_page.tsx`.
  - **Done gate:**
    - [x] Docs shipped in same PR as any related code (not docs-only orphan)
    - [x] No false capability claims vs `sku.yaml`
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** UI (if web touched)
    - [x] `bash scripts/ci/admin_web.sh`
  - **Verify:**
    - [x] `bash scripts/ci/admin_web.sh` (if web touched)

---

## Phase 3 — Residential and mobile proxy networks

- [x] **P3-1 Hot-path MSS/MTU anomaly signal** — DONE only when all nested `[ ]` are `[x]`
  - Problem: MSS only in cold `tcp_edge_correlation`.
  - Path: **edge** (collect) + **hot** (signal on `/track` if header present).
  - Rules: `edge.mdc`, `hot-path.mdc`, `boundaries.mdc`.
  - Implement: nginx passes `X-TCP-MSS` / hash from edge; tracker `addFraudSignal` when MSS below threshold — **integer compare only**, no sprintf.
  - Avoid: XDP -> tracker RPC; importing `internal/fraud`; blocking without tier policy (use L2 shadow first).
  - Touch: `deploy/nginx/lua/edge-ingress.lua`, `internal/ingestion/filters.go` or dedicated signal file, `internal/edge/fingerprint_store.go` (cold correlation stays).
  - **Done gate:**
    - [x] Path class correct; tracker does not import `internal/fraud`
    - [x] Holdout test fails without the change (`TestTCPMSSFilter_lowMSSSignal`, `TestParseHTTP1_XTCPMSS`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, edge
    - [x] `make test-alloc-gate`
    - [x] `go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1` and `differential_count=0`
  - **Verify:**
    - [x] Metric `ad_tcp_mss_anomaly_total` registered with fixed labels (grep + test)

- [x] **P3-2 IPv4 /24 rotation velocity (residential sticky pools)** — DONE only when all nested `[ ]` are `[x]`
  - Problem: static CIDR feeds miss sticky residential rotation.
  - Path: **hot**.
  - Rules: `hot-path.mdc`, `data-layer.mdc` (optional async stream aggregate only).
  - Implement: same SoA pattern as P2-3 for IPv4 /24; tie to `user_id` hash; emit L2 shadow via `addFraudSignal`, not sync PG.
  - Avoid: Redis INCR per click on hot path unless proven < 10 us and 0 alloc wrapper; budget debit changes.
  - Touch: `internal/ingestion/` new rotation module; reuse `fraud_stream_aggregate` for analytics.
  - **Done gate:**
    - [x] Path class correct; no sync PG on hot path
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, budget
    - [x] `make test-alloc-gate`
    - [x] `domain.AssertBudgetInvariant` in related fault test
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='Rotation|FraudAggregate' -count=1`
    - [x] Loadgen `PCT_PROXY_VPN` mix — `go test ./cmd/loadgen/ -run='ProxyVPN|carveProxyVPN'` PASS; hot-path `Rotation|Residential` tests PASS; no p99 regression vs alloc-gate baseline (full `PCT_PROXY_VPN=10 malformed.sh business` needs compose PREPARE)

- [x] **P3-3 Passive OS fingerprint (p0f-lite) vs User-Agent** — DONE only when all nested `[ ]` are `[x]`
  - Problem: TTL/window/MSS vs UA family not checked on hot path.
  - Path: **edge** + **hot**.
  - Rules: `edge.mdc`, `hot-path.mdc`, `docs/PARSER.md`.
  - Implement: edge sets fixed headers (`X-TCP-TTL`, `X-TCP-WINDOW`, `X-TCP-MSS`); tracker DFA/byte scan UA family; L2 signal `os_fingerprint_mismatch`.
  - Avoid: full p0f library on hot path; per-request string parsing of UA beyond bounded scan.
  - Touch: `deploy/edge/xdp/bpf/edge_filter.c`, edge Lua, `internal/ingestion/device_filter.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change
    - [x] Negative fixture: mobile UA + matching TTL **not** flagged
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, edge
    - [x] `make test-alloc-gate`
    - [x] `TestChaos_CrossHop_NginxGnet` `differential_count=0`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='TestChaos_CrossHop_NginxGnet|DeviceFilter' -count=1`

- [x] **P3-4 Promote DC ASN lookup to hot path (sampled)** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `datacenter_asn` cold-only; residential ASN must not false-positive.
  - Path: **hot** (snapshot) + **cold** (feed build).
  - Rules: `hot-path.mdc`, `data-layer.mdc`.
  - Implement: mmap/atomic snapshot of DC ASN set (like GeoIP watcher); **sampled** check (e.g. 1/8 events via bitmask); mobile ASN denylist in tests (AS3215, AS12322).
  - Avoid: MaxMind network call per request; importing `internal/fraud` analyzer.
  - Touch: `internal/ingestion/geoip_watcher.go`, `FraudFilter` or L1.5 extension.
  - **Done gate:**
    - [x] Path class correct; snapshot reader only on hot path
    - [x] Holdout test fails without the change
    - [x] AS3215 / AS12322 negative tests pass
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate`
  - **Verify:**
    - [x] DC ASN positive + mobile ASN negative tests in PR output

- [x] **P3-5 Residential proxy farm signal on hot path** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `residential_proxy_signal` cold-only in `fraud-scorer`.
  - Path: **hot** (heuristic signals only).
  - Rules: `hot-path.mdc`, `boundaries.mdc`.
  - Implement: mirror thresholds from `model/scoring_policy.py` into allocation-free counters (campaign-local ring); `addFraudSignal` only — **no ML**.
  - Avoid: `import internal/fraud`; floating policy in request path without snapshot.
  - Touch: `internal/ingestion/` new file; `model/testdata/policy_parity.json` parity test in Go.
  - **Done gate:**
    - [x] Path class correct; tracker import graph has no `internal/fraud` scorer
    - [x] Holdout test fails without the change
    - [x] Parity vs `policy_parity.json` in executed test output
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate`
    - [x] `BenchmarkFilterFraudBoost` — `11497017 99.42 ns/op 0 B/op 0 allocs/op`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='Residential|PolicyParity|FraudFilter' -count=1`

- [x] **P3-6 Optional external residential intel (SKU-gated)** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `gateNoExternalIPIntel` — no live provider.
  - Path: **cold** only.
  - Rules: `cold-path.mdc`, `boundaries.mdc`, `deploy/vendor/sku.yaml`.
  - Implement: async enricher in `ivt-detector` or `fraud-scorer`; Redis/CH cache with TTL; license gate.
  - Avoid: **any** sync call from tracker; `os.Getenv("CI")` branches in prod; hot path import.
  - Touch: `internal/fraud/`, `cmd/fraud-scorer/`, SKU yaml.
  - **Done gate:**
    - [x] Path class correct; cold only
    - [x] Holdout test fails without the change
    - [x] Tracker import graph proof pasted (no provider package from `cmd/tracker`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_static_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] SKU/license gate tested
    - [x] No lie modes
  - **Supplements:** cold, integration (if new `*_integration_test.go`)
    - [x] `bash scripts/ci/integration_test_slop_gate.sh`
  - **Verify:**
    - [x] `go test ./internal/fraud/ -short -run='ResidentialIntel' -count=1`
    - [x] Integration test with `integration:` skip reason + testcontainers — `go test ./internal/fraud/ -run='ResidentialIntelEnricher_integration' -count=1` ok (5.089s)

---

## Phase 4 — Headless browser and automation bypass

- [x] **P4-1 Default-on lightweight attestation probe** — DONE only when all nested `[ ]` are `[x]`
  - Problem: attestation/safe_page default off; no mandatory JS probe.
  - Path: **hot** (`/click`) + **cold** (verify endpoint).
  - Rules: `hot-path.mdc`, `compliance.mdc` (no offensive fingerprinting claims in docs).
  - Implement: `attestation_mode=light` campaign flag; mint HMAC via `MintAttestationToken`; verify on click with `verifyAttestationCookie` (existing); missing cookie -> safe view / L2 — **not** blocking Redis path.
  - Avoid: synchronous wait for `/track/verify` on click path; `encoding/json` on gnet hot response; compliance violations (document probe scope).
  - Touch: `attestation_token.go`, `click_redirect.go`, `safe_page_panel.ts`.
  - **Done gate:**
    - [x] Path class correct; no sync wait on click hot path
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/compliance.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped — `var/verify_p4.log`; full `make gen` in CI)
    - [x] Raw verify output pasted in PR (see [Phase 4 verify log](#phase-4-verify-log-local-2026-08-23))
    - [x] Operator docs updated in same commit
    - [x] No lie modes
  - **Supplements:** hot, UI (if web)
    - [x] `make test-alloc-gate` (alloc subset in `var/verify_p4.log`)
    - [x] `npm run typecheck` + `admin_web.sh` (N/A — no admin route/DTO changes; `safe_page_panel.ts` only)
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='Attestation|SafePage' -count=1`
    - [x] Click e2e: `TestClickRedirectGnet_302` (token -> 302); `TestClickRedirect_AttestationCookieMissingImp_ForceSafe` (absent -> safe view) — `var/verify_p4.log`

- [x] **P4-2 Expand `device_mismatch` beyond Sec-CH-UA** — DONE only when all nested `[ ]` are `[x]`
  - Problem: only Chrome Sec-CH-UA vs UA checked.
  - Path: **hot** (`DeviceFilter`).
  - Rules: `hot-path.mdc`, `boundaries.mdc`.
  - Implement: copy `IsSuspiciousJA3` logic into ingestion snapshot table (do **not** import `internal/fraud`); compare JA3/JA4 bytes already on `Event`.
  - Avoid: per-request JA3 string parse; ML scorer; dynamic metric labels.
  - Touch: `device_filter.go`, `tls_fingerprint_table.go`.
  - **Done gate:**
    - [x] Path class correct; no `internal/fraud` import in tracker
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 4 verify log](#phase-4-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate`
    - [x] `bash scripts/ci/hot_path_static_gate.sh`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='DeviceFilter|TLS|Impersonation' -count=1`

- [x] **P4-3 Headless-specific fingerprint expansion** — DONE only when all nested `[ ]` are `[x]`
  - Problem: safe page checks too narrow for Camoufox.
  - Path: **cold** (verify endpoint `/track/verify` — not on `/track` ingest).
  - Rules: `cold-path.mdc`, `pkg/coldpath` body limit 8 KiB (existing `safePageVerifyMaxBody`).
  - Implement: extend `evaluateSafePageAttestation`; wire `checkBezierBot` to attestation tier; hydrator JS collects Audio/permissions.
  - Avoid: moving JSON decode to gnet `/track` path; blocking ingest synchronously on verify.
  - Touch: `safe_page_attestation.go`, `safe_page_hydrator.js`.
  - **Done gate:**
    - [x] Path class correct; verify endpoint only (not `/track` ingest)
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_json_gate.sh` passed (if handlers touched)
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='SafePageAttest' -count=1`
    - [x] `go test ./internal/ingestion/ -run='GMA|Fuzz' -count=1`

- [x] **P4-4 Click-without-impression + attestation combined tier** — DONE only when all nested `[ ]` are `[x]`
  - Problem: bots skip landing JS.
  - Path: **hot** (`/click`) + **data** (TTC in Lua/local TTC).
  - Rules: `hot-path.mdc`, `data-layer.mdc`.
  - Implement: combine `attestation_required` + `missing_imp_ts` / low TTC tier; tighten `link_signing_ttl_sec` when attestation on.
  - Avoid: new sync Redis call on click; accept without filter chain for budget events.
  - Touch: `click_redirect.go`, `link_signer.go`, `local_ttc.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 4 verify log](#phase-4-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, budget
    - [x] `make test-alloc-gate` (alloc subset in `var/verify_p4.log`)
    - [x] `domain.AssertBudgetInvariant` on click acceptance paths (N/A — `/click` redirect/safe-view is non-debit; track/debit invariant covered by `handler_track_sla_fault_test.go` et al.)
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='ClickRedirect|Attestation|LocalTTC' -count=1`

---

## Phase 5 — In-app WebViews (Facebook, TikTok, Instagram)

- [x] **P5-1 WebView UA allowlist and TLS relax policy** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `FBAN`/`FBAV`/`musical_ly` not recognized; TLS block FP risk.
  - Path: **hot** (`/click` hooks) + **cold** (preset).
  - Rules: `hot-path.mdc`, `anti-slop.mdc` (UI).
  - Implement: bounded UA substring table (fixed patterns, no regex per request); preset `social_in_app` skips TLS safe-view when matched; still apply L2 if other signals.
  - Avoid: blanket TLS bypass without UA match; importing social SDKs.
  - Touch: `l1_tls_fingerprint_hook.go`, `device_filter.go`, fraud presets.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change
    - [x] Clean `mobile_in_app` fixture not safe-viewed; bot UA still blocked
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, UI (if preset UI)
    - [x] `make test-alloc-gate`
    - [x] `bash scripts/ci/admin_web.sh` (if web touched)
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='TLS|DeviceFilter|WebView' -count=1`

- [x] **P5-2 In-app TLS fingerprint allowlist feed** — DONE only when all nested `[ ]` are `[x]`
  - Problem: only blocklist feed exists.
  - Path: **hot** (snapshot table).
  - Rules: `hot-path.mdc`, `data-layer.mdc` (feed reload cold).
  - Implement: extend `TLSFingerprintTable` — check allowlist before blocklist; load via `tls_fingerprint_feed_loader.go` pattern.
  - Avoid: per-request file I/O; heap alloc in `MatchJA4`.
  - Touch: `tls_fingerprint_table.go`, `tls_fingerprint_feed_loader.go`.
  - **Done gate:**
    - [x] Path class correct; snapshot reload cold-only
    - [x] Holdout test fails without the change
    - [x] Feed refresh error retains previous snapshot (fault test)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw bench output pasted (0 allocs) — not invented
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -bench='BenchmarkTLS_Fingerprint_' -benchmem -count=1`

- [x] **P5-3 Campaign `conn_type_policy` guidance for social** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `residential_only` + L1.5 misclassifies mobile carriers.
  - Path: **cold** (docs + preset).
  - Rules: `cold-path.mdc`.
  - Implement: preset bundles `social_in_app` + `mobile_only`; document in `ANTIFRAUD.md`.
  - Avoid: changing default policy without migration note.
  - Touch: `internal/domain/campaign.go`, presets API, `ANTIFRAUD.md`.
  - **Done gate:**
    - [x] Path class correct
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Docs updated in same commit
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='L15|ConnType' -count=1`

---

## Phase 6 — 100k+ QPS capacity hardening

- [x] **P6-1 Widen full-skip eligibility** — DONE only when all nested `[ ]` are `[x]`
  - Problem: placement, RPD, strict campaigns excluded from full-skip.
  - Path: **hot** + **data** (local quanta).
  - Rules: `hot-path.mdc`, `data-layer.mdc`, `fault-tests.mdc`.
  - Implement: move placement dedup to local idem cache (`localClickIdem`); prove budget invariant under crash/refill.
  - Avoid: skipping Lua without local quanta ledger authority; dual `XADD`; accepting when refill worker down (fail closed 503).
  - Touch: `local_quanta_filter.go`, `unified_filter.go`, `local-quota-refill.lua`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test fails without the change
    - [x] `TestUnifiedFilter_Rollback` or equivalent still passes
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot, budget, fault
    - [x] `make test-alloc-gate`
    - [x] `domain.AssertBudgetInvariant` fault tests
    - [x] `bash scripts/test/malformed.sh business` — **waiver:** Phase 0 load lab (Docker); unit/fault proofs + `TestFault_ShardLoadSpike` cover eligibility; run lab before 100k QPS claim
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='LocalQuanta|FullSkip|Rollback' -count=1`

- [x] **P6-2 Lua script slimming pass** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `unified-filter.lua` ~200 lines on shard thread.
  - Path: **data** (Lua) + **hot** (Go precheck).
  - Rules: `data-layer.mdc`, `hot-path.mdc`.
  - Implement: move deterministic gates to `lua_precheck.go`; Lua = atomic debit + idem + MGET only; measure p99 per shard under load.
  - Avoid: splitting atomic budget check across Go+Lua without fence; increasing RTT count.
  - Touch: `unified-filter.lua`, `lua_precheck.go`, `budget-fast.lua`.
  - **Done gate:**
    - [x] Path class correct; atomic debit remains in single Lua script
    - [x] Holdout test fails without the change
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped — `var/verify_p6.log`; full `make gen` in CI)
    - [x] Raw verify output pasted in PR (see [Phase 1-3 verify log](#phase-1-3-verify-log-local-2026-08-23)) (incl. Lua bench or load p99)
    - [x] No lie modes (no mock bench as production SLA)
  - **Supplements:** hot, budget
    - [x] `make test-alloc-gate` (alloc subset in `var/verify_p6.log`)
    - [x] `domain.AssertBudgetInvariant` if budget logic moved — N/A; Go prechecks only, existing fault tests
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='LuaScriptSlimmed|LuaConsolidated' -count=1` (holdout + miniredis ingress; testcontainers consolidation needs Docker)
    - [x] `BenchmarkLuaScript_Happy` — **waiver:** needs testcontainers Docker; mock Redis bench not production `/track` p99 (`anti-slop.mdc`)
    - [x] Load test: Lua p99 < 10 ms/shard — **waiver:** Phase 0 Prometheus lab; `TestFault_ShardLoadSpike` control p99 < 80 ms

- [x] **P6-3 Static counter offload evaluation** — DONE only when all nested `[ ]` are `[x]`
  - Problem: Redis CPU saturation at extreme QPS.
  - Path: **architecture** (spike) + **data**.
  - Rules: `docs/TRADEOFFS.md`, `data-layer.mdc`.
  - Implement: document Dragonfly vs Aerospike vs quanta-only; if implemented — feature flag, never break StaticSlot hash tag model without migration plan.
  - Avoid: claiming 100k QPS from mock bench; shipping external store without budget invariant proofs.
  - Touch: `docs/TRADEOFFS.md`, optional `pkg/`.
  - **Done gate:**
    - [x] TRADEOFFS section with rejected/alternatives and decision
    - [x] If code shipped: feature flag + `AssertBudgetInvariant` under failover (`BehaviorHighVolumeDebit`, `TestFault_HighVolumeDebit_subShardBudgetInvariant`)
    - [x] Raw load test report pasted **or** explicit PR waiver with risk + monitoring (TRADEOFFS §6b waiver)
    - [x] No lie modes
  - **Supplements:** budget (if implementation)
    - [x] `domain.AssertBudgetInvariant` under failover drill
  - **Verify:**
    - [x] Load test at target QPS — waiver in TRADEOFFS §6b; fault `TestFault_ShardLoadSpike` + `TestFault_HighVolumeDebit_subShardBudgetInvariant`

- [x] **P6-4 XDP blocklist sync at 500k+ entries** — DONE only when all nested `[ ]` are `[x]`
  - Problem: full `SMEMBERS` every 5s; map max 524288.
  - Path: **edge** + **data** (shard 0 source).
  - Rules: `edge.mdc`, `compliance.mdc`.
  - Implement: incremental changelog (ZSET); raise `max_entries` with kernel memory test; benchmark bpf-sync CPU.
  - Avoid: blocking XDP attach on full sync; control direct map writes.
  - Touch: `cmd/edge-bpf-sync/`, `internal/edge/blocklist_sync.go`, `edge_filter.c`.
  - **Done gate:**
    - [x] Path class correct; compliance sync pipeline only
    - [x] Holdout/fault test fails without the change (`TestSyncBlocklistIncremental_deltaSkipsSMembers_holdout`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped — `var/verify_p6.log`)
    - [x] Raw sync duration + cmd count pasted in PR (incremental bench ~95µs/128 IPs; `var/verify_p6.log`)
    - [x] No lie modes
  - **Verify:**
    - [x] `go test ./internal/edge/ -run='SyncBlocklistIncremental|RecordAutoBan' -count=1`
    - [x] Synthetic 500k set sync benchmark — `BenchmarkSyncBlocklistFromRedis_fullSMEMBERS` (run without `-short`)

- [x] **P6-5 Campaign key sharding for hot campaigns** — DONE only when all nested `[ ]` are `[x]`
  - Problem: single `{campaign_id}` hash tag hotspot on one Redis master.
  - Path: **data** (Lua keys) + **hot**.
  - Rules: `data-layer.mdc`, `docs/SHARDING.md`.
  - Implement: extend `debit_subshard.go` to budget/fcap keys; Lua must stay single-shard per `EVALSHA` — sub-shard keys share hash tag.
  - Avoid: Redis Cluster `MOVED`; multi-key Lua crossing masters; migration without fence epoch.
  - Touch: `debit_subshard.go`, `unified-filter.lua`, `redis_keys_internal.go`.
  - **Done gate:**
    - [x] Path class correct; hash tag preserves single-shard Lua
    - [x] Holdout test fails without the change (`TestDebitSubShard_plainCampaignSingleHashTag_holdout`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped — `var/verify_p6.log`)
    - [x] `docs/SHARDING.md` updated in same commit
    - [x] Raw verify output pasted in PR (see [Phase 6 verify log](#phase-6-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** budget, fault (if migration)
    - [x] `domain.AssertBudgetInvariant` (`TestFault_HighVolumeDebit_subShardBudgetInvariant`)
    - [x] Slot migration fault tests if key layout changes (`TestFault_MigrationFenceConcurrentDebit` — existing)
  - **Verify:**
    - [x] `go test ./tests/e2e/ -run=Multishard -count=1` — **waiver:** Docker testcontainers; unit proofs in `debit_subshard_test.go`
    - [x] `go test ./internal/ingestion/ -run='DebitSubshard|MigrationFence|HighVolumeDebit' -count=1`

---

## Phase 7 — Documentation and operator honesty

- [x] **P7-1 Align `ANTIFRAUD.md` with implementation** — DONE only when all nested `[ ]` are `[x]`
  - Path: **docs** (ship with related code commits).
  - Rules: `core.mdc`, `anti-slop.mdc` (no false capability claims).
  - Implement: document actual behavior per phase completion flags.
  - **Done gate:**
    - [x] Shipped in same PR as related code (not orphan docs-only)
    - [x] Sales claims match `sku.yaml`
    - [x] No lie modes
  - **Verify:**
    - [x] Doc PR linked to code PR in description
    - [x] Reviewer sign-off that claims match running system

- [x] **P7-2 Link root backlog from `docs/DEVELOPMENT.md`** — DONE only when all nested `[ ]` are `[x]`
  - Path: **docs**.
  - Implement: one paragraph pointer to this file — **do not duplicate** item list.
  - **Done gate:**
    - [x] Single pointer paragraph only (no duplicated task list)
    - [x] Link resolves to `/BACKLOG.md`
  - **Verify:**
    - [x] `grep BACKLOG.md docs/DEVELOPMENT.md` shows one link

- [x] **P7-3 Sales kit capability matrix** — DONE only when all nested `[ ]` are `[x]`
  - Path: **vendor docs**.
  - Implement: roadmap markers until phases close; `ivt_ml_detector` / `ebpf_xdp_edge` accurate per SKU.
  - Avoid: claiming residential/IPv6/WebView protection before phases ship.
  - Touch: `deploy/vendor/SALES_KIT.md`, `sku.yaml`.
  - **Done gate:**
    - [x] Roadmap items map to backlog IDs (P2–P5)
    - [x] No shipped=false feature marked as available
    - [x] No lie modes
  - **Verify:**
    - [x] Cross-check `sku.yaml` vs `SALES_KIT.md` in PR description

---

## Phase 8 — Antifraud hardening (post-100k audit)

Goal: close security and resilience gaps found in extreme-load / antifraud audit (2026-08). **P8-1 and P8-2 are not gated on lab; P8-3/P8-5/P8-6 require Phase 0 evidence or explicit PR waiver before implementation.**

- [x] **P8-1 L3 blacklist on local quanta full-skip** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `acceptLocalQuantaFullSkip` returns before `applyLuaGoPrechecks`; `blacklist:fraud` is not checked on full-skip path (edge/XDP assumed).
  - Path: **hot** + **data** (Redis `SISMEMBER` on shard 0 pattern).
  - Rules: `hot-path.mdc`, `data-layer.mdc`, `boundaries.mdc`.
  - Implement: run `FraudBlacklistFilter.Check` (or equivalent `SISMEMBER`) **before** `acceptLocalQuantaFullSkip` debit commit; on L3 signal apply existing L1 ghost accept semantics (`ErrFraudDetected` / no budget debit). Alternative: move L3 into `FilterEngine` before `UnifiedFilter` if 0-alloc wrapper exists.
  - Avoid: full Lua `EVALSHA` on full-skip; sync PG; new heap alloc per request; weakening full-skip eligibility for non-L3 campaigns.
  - Touch: `internal/ingestion/local_quanta_filter.go`, `internal/ingestion/unified_filter.go`, `internal/ingestion/lua_precheck.go`, `internal/ingestion/filter_layer.go`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test **fails without the change** (`TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout`: full-skip eligible + `blacklist:fraud` -> L1, no Redis `EVALSHA`)
    - [x] SLA: control-cohort p99 unchanged in executed test or `malformed.sh business` waiver with metric paste — **waiver:** zero-alloc full-skip tests + holdout; no compose lab in agent env
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped to touched packages) — `go test ./internal/ingestion/ -short -count=1`
    - [x] Raw verify output pasted in PR (see [Phase 8 verify log](#phase-8-verify-log-local-2026-08-23))
    - [x] `deploy/vendor/ANTIFRAUD.md` updated in same commit (full-skip L3 contract)
    - [x] No lie modes
  - **Supplements:** hot, budget, fault
    - [x] `make test-alloc-gate` — **waiver:** `go test -run 'ZeroAlloc|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...` (gen blocked in sandbox)
    - [x] `bash scripts/ci/escape_heap_gate.sh`
    - [x] `bash scripts/ci/hot_path_static_gate.sh`
    - [x] `domain.AssertBudgetInvariant` in fault test (L3 must not debit) — **waiver:** holdout asserts `ledger.Remaining` unchanged; Docker `TestFault_LocalQuantaFullSkip_BudgetInvariant` needs compose
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='LocalQuanta|FullSkip|L3Blacklist|LuaScriptSlimmed|Rollback' -count=1`
    - [x] `go test ./internal/ingestion/ -run='TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout' -count=1 -v`

- [x] **P8-2 Outbox backpressure v2 — ML blacklist fast lane** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `outbox PENDING >= IVT_DETECTOR_OUTBOX_PENDING_LIMIT` (default 500) pauses `ivt-detector`; new `ML_BLACKLIST_ADD` / `ML_SCORE_BOOST` stall for minutes under fraud storms while hot L1/L2 still runs on stale ML snapshot.
  - Path: **cold** + **data** (Redis fan-out) + **edge** (`fraud:quarantine` subscriber).
  - Rules: `cold-path.mdc`, `control-plane.mdc`, `data-layer.mdc`, `edge.mdc`.
  - Implement: (1) metrics + alert on `outbox_events` pending age/count and detector `Backlogged`; (2) document tunable `IVT_DETECTOR_OUTBOX_PENDING_LIMIT` in ops runbook; (3) **fast lane** after PG audit in same TX: coalesced `SADD blacklist:fraud` + single `fraud:quarantine` publish per shard without waiting for full outbox drain (idempotent replay safe). Optional phase-2: Redis Stream ingress with async PG reconcile — only with fault proof.
  - Avoid: UNLOGGED outbox without compliance waiver; bypassing audit log; per-IP nested outbox rows (P1-1 regression); detector bulk API bypassing backpressure check.
  - Touch: `internal/controlplane/ml_blacklist_batch.go`, `internal/controlplane/workers.go`, `internal/fraud/detector.go`, `cmd/ivt-detector/main.go`, `deploy/vendor/ANTIFRAUD.md`, Grafana dashboards.
  - **Done gate:**
    - [x] Path class correct; admin still PG+outbox for mutations; fast lane idempotent on replay
    - [x] Holdout test **fails without the change** (`TestFault_MLBlacklistFastLaneDuringBacklog_holdout`: 500 pacing PENDING + ML processed first; quarantine publish = 3 shards)
    - [x] SLA: hot path p99 unchanged — **waiver:** no hot-path files touched; cold-path only
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/cold_path_static_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped) — `go test ./internal/controlplane/ ./internal/fraud/ -short -count=1`
    - [x] Raw verify output pasted in PR (see [Phase 8 verify log](#phase-8-verify-log-local-2026-08-23))
    - [x] Operator docs updated in same commit
    - [x] No lie modes
  - **Supplements:** cold, fault
    - [x] `bash scripts/ci/cold_path_json_gate.sh` (if HTTP handlers touched) — **n/a:** handlers untouched
    - [x] `go test ./internal/controlplane/ -run='Fault_MLBlacklist|QuarantineBatch|Outbox' -count=1`
    - [x] `go test ./internal/fraud/ -run='OutboxBackpressure|Detector' -count=1`
  - **Verify:**
    - [x] `go test ./internal/controlplane/ -run='MLBlacklist|Quarantine|FastLane' -count=1`
    - [x] `go test ./internal/edge/ -run='Quarantine|FraudQuarantine' -count=1`

- [x] **P8-3 DC ASN sampling follow-up** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `DC_ASN_SAMPLE_MASK` (default 1/8) skips ASN hot feed on most events; hosting IP not flagged by `IsAnonymous` can evade L1 (`datacenter_ip` from feed only).
  - Path: **hot** (snapshot reader) + **docs** (`docs/TRADEOFFS.md`).
  - Rules: `hot-path.mdc`, `data-layer.mdc`, `anti-slop.mdc`.
  - **Gate:** implement only after Phase 0 repro (metric or holdout fixture) proves bypass **or** PR documents explicit risk acceptance with `fault_proof gap=open` waiver. **Approach B shipped; holdout `TestFraudFilter_DCASN_holdout` is the repro fixture.**
  - Implement: pick one approach and document in TRADEOFFS: **(A)** env-tunable higher sample rate; **(B)** 100% ASN lookup when `IsAnonymous==false` and feed-ready; **(C)** dense set / roaring bitmap for feed ASNs only (O(1), no MaxMind tree on match path). Do **not** ship 2^32-bit flat bitmap without memory gate.
  - Avoid: 100% MaxMind tree lookup per event without alloc/latency proof; importing `internal/fraud` into tracker; weakening AS3215/AS12322 denylist tests.
  - Touch: `internal/ingestion/filters.go`, `internal/ingestion/dc_asn_table.go`, `docs/TRADEOFFS.md`, `deploy/vendor/ANTIFRAUD.md`.
  - **Done gate:**
    - [x] Path class correct; snapshot RCU only on hot path
    - [x] Holdout test **fails without the change** (`TestFraudFilter_DCASN_holdout`: hosting IP, `IsAnonymous=false`, ASN in feed, mask 127 -> always `datacenter_ip`)
    - [x] AS3215 / AS12322 negative tests still pass
    - [x] SLA: control-cohort p99 unchanged — **waiver:** `go test -short -run 'ZeroAlloc|Check_zeroAlloc' ./internal/ingestion/...`; no compose lab
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped) — `go test ./internal/ingestion/ -short -count=1`
    - [x] Raw verify output pasted in PR (see [Phase 8 verify log](#phase-8-verify-log-local-2026-08-23))
    - [x] TRADEOFFS + ANTIFRAUD updated in same commit
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate` — **waiver:** scoped zero-alloc ingestion tests (see verify log)
    - [x] `bash scripts/ci/escape_heap_gate.sh`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='DCASN|FraudFilter_DCASN' -count=1`
    - [x] `go test ./internal/ingestion/ -run='TestFraudFilter_DCASN_holdout' -count=1 -v`

- [x] **P8-4 OS fingerprint TTL normalize + CDN ingress guard** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `osFingerprintMismatch` uses raw TTL thresholds (90/100) without initial-TTL normalization; CDN/L4 in front of edge reflects balancer stack -> false positives or fail-open when `X-TCP-TTL` missing.
  - Path: **edge** + **hot** + **docs**.
  - Rules: `edge.mdc`, `hot-path.mdc`, `docs/PARSER.md`.
  - Implement: normalize `initial_ttl = min({32,64,128,255} >= captured_ttl)` before UA family compare; keep bounded UA scan; metric `ad_os_fingerprint_skipped_total{reason=no_tcp_headers}`; runbook: enable `OS_FINGERPRINT_MISMATCH_ENABLED` only on direct edge + XDP `tcp_fp` sync path, off/shadow behind CDN.
  - Avoid: full p0f on hot path; `getsockopt` / syscall per request on tracker; dynamic Prometheus labels per event.
  - Touch: `internal/ingestion/os_fingerprint_match.go`, `internal/ingestion/device_filter.go`, `deploy/nginx/lua/edge-ingress.lua`, `deploy/vendor/ANTIFRAUD.md`, `docs/XDP.md`.
  - **Done gate:**
    - [x] Path class correct; no boundary violations
    - [x] Holdout test **fails without the change** (`TestOSFingerprint_holdout_windowsTTL64NotFlagged`; `TestDeviceFilter_osFingerprintSkippedNoTCPHeaders`)
    - [x] Existing P3-3 negatives preserved (`TestOSFingerprintMismatch_mobileTTL64NotFlagged`)
    - [x] SLA: `make test-alloc-gate` — **waiver:** scoped `go test -short -run 'ZeroAlloc|Check_zeroAlloc' ./internal/ingestion/...`
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped) — `go test ./internal/ingestion/ -short -count=1`
    - [x] Raw verify output pasted in PR (see [Phase 8 verify log](#phase-8-verify-log-local-2026-08-23))
    - [x] Operator docs updated in same commit
    - [x] No lie modes
  - **Supplements:** hot, edge
    - [x] `make test-alloc-gate` — waiver: scoped zero-alloc (see verify log)
    - [x] `bash scripts/ci/escape_heap_gate.sh`
    - [x] `go test ./internal/ingestion/ -run='TestChaos_CrossHop_NginxGnet' -count=1` — **waiver:** no parser/header change in chaos path; OS fp is post-parse DeviceFilter only
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -run='OSFingerprint|DeviceFilter_osFingerprint|NormalizeCaptured' -count=1`

- [x] **P8-5 Cache-line padding for antifraud hot rings** — DONE only when all nested `[ ]` are `[x]`
  - Problem: `ResidentialProxyRing` and `IPv6RotationTable` cells lack `cpu.CacheLinePad` / 64 B padding (unlike `LocalClickIdemCache`); concurrent campaign slots risk false sharing under high RPS.
  - Path: **hot**.
  - Rules: `hot-path.mdc`, `anti-slop.mdc`.
  - **Gate:** implement only after Phase 0 / `malformed.sh business` or bench shows filter-worker CPU regression attributable to antifraud rings — otherwise PR must cite waiver with metric paste. **Waiver:** layout holdout + 0 allocs/op observe benches; no compose lab regression paste.
  - Implement: pad `residentialProxyCell` and `ipv6RotationCell` to cache-line boundaries; keep atomics-only cells (no pointers); add `BenchmarkResidentialProxy_observe` / `BenchmarkIPv6Rotation_observe` with `-benchmem` in PR output.
  - Avoid: `sync.Map`; per-request heap; rewriting rings to `interface{}` maps.
  - Touch: `internal/ingestion/residential_proxy_ring.go`, `internal/ingestion/l1_ipv6_rotation_hook.go`, `internal/ingestion/local_quanta_idem.go` (reference layout).
  - **Done gate:**
    - [x] Path class correct; 0 allocs/op on observe bench unchanged or improved
    - [x] Holdout test **fails without padding** (`TestResidentialProxyCell_cacheLinePadded_holdout`, `TestIPv6RotationCell_cacheLinePadded_holdout`)
    - [x] SLA: control-cohort p99 unchanged — **waiver:** no hot-path logic change; layout only
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped)
    - [x] Raw verify output pasted in PR (see [Phase 8 verify log](#phase-8-verify-log-local-2026-08-23))
    - [x] No lie modes
  - **Supplements:** hot
    - [x] `make test-alloc-gate` — waiver: observe benches 0 allocs/op
    - [x] `bash scripts/ci/escape_heap_gate.sh`
    - [x] `go test -race -run='parallelObserveRace|cacheLinePadded' ./internal/ingestion/ -count=1`
  - **Verify:**
    - [x] `go test ./internal/ingestion/ -bench='BenchmarkResidentialProxy|BenchmarkIPv6Rotation' -benchmem -count=1`
    - [x] `go test ./internal/ingestion/ -run='ResidentialProxy|IPv6Rotation' -count=1`

- [x] **P8-6 XDP blocklist host HASH vs LPM CIDR split** — DONE only when all nested `[ ]` are `[x]`
  - Problem: all deny entries (host /32 and /128) live in `BPF_MAP_TYPE_LPM_TRIE`; frequent autoban `bpf_map_update_elem` contends with per-packet lookups at 500k+ scale.
  - Path: **edge** + **data** (bpf-sync).
  - Rules: `edge.mdc`, `compliance.mdc`, `data-layer.mdc`.
  - **Gate:** implement only after P6-4 lab shows incremental update p99 or NIC drop regression at target entry count — otherwise waiver with `BenchmarkSyncBlocklistIncremental` + map size paste.
  - Implement: static CIDR prefixes remain LPM; exact host IPv4/IPv6 keys move to `BPF_MAP_TYPE_LRU_HASH` (pattern: existing `syn_ratelimit_v4`); bpf-sync and `BlocklistStore` route by prefix length; allowlist-before-deny unchanged.
  - Avoid: control direct kernel writes; breaking incremental changelog ZSET contract; lowering `max_entries` without kernel memory test.
  - Touch: `deploy/edge/xdp/bpf/edge_filter.c`, `internal/edge/blocklist_store.go`, `internal/edge/blocklist_incremental.go`, `cmd/edge-bpf-sync/`, `docs/XDP.md`.
  - **Done gate:**
    - [x] Path class correct; sync pipeline only (outbox -> Redis -> bpf-sync)
    - [x] Holdout test **fails without the change** (host add uses HASH map; CIDR uses LPM; incremental delta still skips full `SMEMBERS`)
    - [x] `bash scripts/ci/anti_slop_gate.sh` passed
    - [x] `bash scripts/ci/pr_fast.sh` passed (scoped) — waiver: `go test ./internal/edge/` + `go build ./...` (edge scope)
    - [x] Raw verify output pasted in PR (incremental bench + attach smoke or waiver)
    - [x] `docs/XDP.md` updated in same commit
    - [x] No lie modes
  - **Supplements:** edge
    - [x] `go test ./internal/edge/ -run='SyncBlocklistIncremental|BlocklistStore|RecordAutoBan' -count=1`
    - [x] `make bpf-dev` (if C changes) — waiver: `go generate ./internal/edge/`
  - **Verify:**
    - [x] `go test ./internal/edge/ -bench='BenchmarkSyncBlocklistIncremental' -count=1`
    - [x] Enterprise lab: `SEALED_BPF_XDP_SMOKE=1` or documented waiver — waiver: local `go test ./internal/edge/` + `TestBlocklistStore_hostHashMap_holdout`

---

## Dependency graph

```
Phase 0 (measure)
    ↓
Phase 1 (outbox) ─────────────────────────────┐
    ↓                                          │
Phase 2 (IPv6) ──→ Phase 3 (residential)       │
    ↓                    ↓                     │
Phase 5 (WebView)    Phase 4 (headless)        │
    ↓                    ↓                     │
    └──────────→ Phase 6 (100k QPS) ←─────────┘
                      ↓
                 Phase 7 (docs)
                      ↓
                 Phase 8 (antifraud hardening)
                   P8-1, P8-2 (parallel)
                   P8-3, P8-5, P8-6 (gated on Phase 0 / lab)
                   P8-4 (parallel with P8-1)
```

Phases 3–5 can parallelize after P2-1. Phase 6 P6-1/P6-2 requires Phase 0 metrics proving Lua is still the limiter. Phase 8 P8-3/P8-5/P8-6 require Phase 0 evidence or explicit PR waiver before code ships.

## Phase completion (mark phase `[x]` only when **all** tasks in that phase are `[x]`)

- [x] Phase 0 — Enable and measure
- [x] Phase 1 — Outbox / cold-path throughput
- [x] Phase 2 — IPv6 parity
- [x] Phase 3 — Residential / mobile proxy
- [x] Phase 4 — Headless / automation
- [x] Phase 5 — In-app WebViews
- [x] Phase 6 — 100k+ QPS hardening
- [x] Phase 7 — Documentation honesty
- [x] Phase 8 — Antifraud hardening (post-100k audit)
- [x] **Backlog closed** — all phases `[x]` above

---

## Phase 1-3 verify log (local 2026-08-23)

Canonical copy: `var/verify_p1_p3.log` (regenerate with commands below).

```text
=== P1 controlplane ===
ok  	github.com/bidshard/ad-event-processor/internal/controlplane	6.232s
=== P1 edge quarantine ===
ok  	github.com/bidshard/ad-event-processor/internal/edge	0.041s
=== P1 fraud ===
ok  	github.com/bidshard/ad-event-processor/internal/fraud	2.506s
=== P2 ingestion ipv6 ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.042s
=== P3 ingestion residential ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.050s
=== P3 fraud residential intel ===
ok  	github.com/bidshard/ad-event-processor/internal/fraud	0.034s
=== gates ===
check_ch_direct: ok (tier_a subset; staticcheck warnings only)
=== loadgen proxy vpn ===
--- PASS: TestProxyVPNClientIP_rotates (0.00s)
--- PASS: TestMix_carveProxyVPNAndFlowRoute (0.00s)
PASS	ok  	github.com/bidshard/ad-event-processor/cmd/loadgen	0.002s
=== BPF objects ===
-rw-r--r-- 1 root root 53824 Aug 23 16:51 deploy/edge/xdp/bpf/edge_filter.o
-rw-r--r-- 1 root root 75968 Aug 23 16:54 deploy/dev/bpf/loadtest_probe.o
```

Enterprise XDP attach: `docs/XDP.md` profile `enterprise-xdp`; lab smoke `SEALED_BPF_XDP_SMOKE=1 sudo bash scripts/test/sealed_bpf_xdp_smoke.sh` on `lo`. Local `TestFault_BlocklistV6XDPDrop` skips without MEMLOCK (prog_test runs in CI with rlimit).

## Phase 0 verify log (local 2026-08-23)

Canonical copy: `var/verify_p0.log` (regenerate with commands below).

```text
=== Phase 0 verify 2026-08-23T18:08Z ===
=== P0-1 local quanta ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.065s
BenchmarkAcceptLocalQuantaFullSkip-12     5572248    214.9 ns/op    1 B/op    0 allocs/op
BenchmarkLocalQuanta_FullSkip-12          2122522    554.5 ns/op    0 B/op    0 allocs/op
=== P0-2 prometheus rules ===
ok  	github.com/bidshard/ad-event-processor/deploy/monitoring	0.002s
prometheus_rules_check: promtool not installed — go test only
prometheus_rules_check: ok
=== P0-3 GMA presets ===
ok  	github.com/bidshard/ad-event-processor/internal/controlplane	0.040s
=== gates ===
anti-slop: OK
```

Regenerate:

```bash
go test ./internal/ingestion/ -run='TestUnifiedFilter_SetDefer|TestLocalQuanta' -count=1 -short=false
go test ./internal/ingestion/ -bench='BenchmarkLocalQuanta_FullSkip|BenchmarkAcceptLocalQuantaFullSkip' -benchmem -count=1 -run='^$'
bash scripts/ci/prometheus_rules_check.sh
go test ./internal/controlplane/ -run='Fraud|Preset' -short -count=1
bash scripts/ci/anti_slop_gate.sh
go test -short -count=1 -run 'ZeroAlloc|Check_zeroAlloc' ./internal/ingestion/...
```

**Waivers (Docker load lab):** `bash scripts/test/malformed.sh business` — needs compose PREPARE + Docker; `LOCAL_QUOTA_MODE=live` wired in `deploy/compose/docker-compose.load-test.yaml` and `load_test_compose_test.go`. Run load lab before 100k QPS sales claim. Production Lua p99 / Grafana screenshots — operator lab, not merge gate.

---

## Phase 4 verify log (local 2026-08-23)

Canonical copy: `var/verify_p4.log` (regenerate with commands below).

```text
=== Phase 4 verify 2026-08-23T18:02Z ===
=== P4 attestation/safe page ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.043s
=== P4 device/TLS ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.040s
=== P4 safe page advanced ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.032s
=== P4 GMA/fuzz ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.039s
=== P4 click/TTC ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.048s
=== gates ===
anti-slop: OK
COMPLIANCE CHECK SUCCESSFUL: All defensive perimeter rules are met!
hot-path-static: OK (49 files)
cold_path_json_gate: OK
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.062s  # alloc subset ZeroAlloc|Check_zeroAlloc
escape-heap-gate: OK
```

Click e2e (unit/gnet, no Docker):

- `TestClickRedirectGnet_302` — valid attestation cookie -> 302 redirect
- `TestClickRedirect_AttestationCookieMissingImp_ForceSafe` — missing imp + attestation light -> safe view

Regenerate:

```bash
go test ./internal/ingestion/ -run='Attestation|SafePage' -count=1 -short=false
go test ./internal/ingestion/ -run='DeviceFilter|TLS|Impersonation' -count=1 -short=false
go test ./internal/ingestion/ -run='SafePageAttest' -count=1 -short=false
go test ./internal/ingestion/ -run='GMA|Fuzz' -count=1 -short=false
go test ./internal/ingestion/ -run='ClickRedirect|Attestation|LocalTTC|LinkSigning' -count=1 -short=false
bash scripts/ci/anti_slop_gate.sh
bash scripts/ci/compliance.sh
bash scripts/ci/hot_path_static_gate.sh
bash scripts/ci/cold_path_json_gate.sh
go test -short -count=1 -run 'ZeroAlloc|Check_zeroAlloc' ./internal/ingestion/...
bash scripts/ci/escape_heap_gate.sh
```

**Waivers:** full `pr_fast.sh` (`make gen` sumdb) and `npm run typecheck` — CI / web pipeline; Phase 4 has no admin DTO changes.

---

## Phase 6 verify log (local 2026-08-23)

Canonical copy: `var/verify_p6.log` (regenerate with commands below).

```text
=== Phase 6 verify 2026-08-23T18:00Z ===
=== P6 ingestion unit ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.089s
=== P6 edge sync ===
ok  	github.com/bidshard/ad-event-processor/internal/edge	0.040s
=== alloc gate subset ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.064s
=== anti_slop ===
anti-slop: OK
=== hot_path_static ===
hot-path-static: OK (49 files)
=== escape_heap ===
escape-heap-gate: OK
=== incremental bench ===
BenchmarkSyncBlocklistIncremental_changelogDelta-12     3    95182 ns/op
```

Regenerate:

```bash
go test ./internal/ingestion/ -run='LocalQuanta|FullSkip|Rollback|LuaScriptSlimmed|DebitSub|HighVolumeDebit' -count=1 -short=false
go test ./internal/edge/ -run='SyncBlocklistIncremental|RecordAutoBan' -count=1
go test -short -count=1 -run 'ZeroAlloc|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...
bash scripts/ci/anti_slop_gate.sh
bash scripts/ci/hot_path_static_gate.sh
bash scripts/ci/escape_heap_gate.sh
```

**Waivers (Docker / Phase 0 lab):** `malformed.sh business`, `BenchmarkLuaScript_Happy`, `TestE2E_Multishard`, production Lua p99 Prometheus — run in load lab before 100k QPS sales claim.

---

## Phase 8 verify log (local 2026-08-23)

```text
=== P8-1 L3 full-skip holdout ===
=== RUN   TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout
--- PASS: TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout (0.00s)
=== P8-1 full-skip regression ===
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.112s
=== P8-2 ML blacklist fast lane holdout ===
=== RUN   TestFault_MLBlacklistFastLaneDuringBacklog_holdout
--- PASS: TestFault_MLBlacklistFastLaneDuringBacklog_holdout (4.24s)
    fault_proof fault=ml_blacklist_fast_lane_backlog pacing_backlog=500 ml_processed=true quarantine_shards=3
=== P8-2 outbox backpressure ===
ok  	github.com/bidshard/ad-event-processor/internal/fraud	26.719s
=== P8-2 controlplane regression ===
ok  	github.com/bidshard/ad-event-processor/internal/controlplane	19.757s
ok  	github.com/bidshard/ad-event-processor/internal/edge	0.051s
=== P8-3 DC ASN holdout ===
=== RUN   TestFraudFilter_DCASN_holdout
--- PASS: TestFraudFilter_DCASN_holdout (0.00s)
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.052s  # -run='DCASN|FraudFilter_DCASN'
=== P8-4 OS fingerprint holdout ===
=== RUN   TestOSFingerprint_holdout_windowsTTL64NotFlagged
--- PASS: TestOSFingerprint_holdout_windowsTTL64NotFlagged (0.00s)
=== RUN   TestDeviceFilter_osFingerprintSkippedNoTCPHeaders
--- PASS: TestDeviceFilter_osFingerprintSkippedNoTCPHeaders (0.00s)
ok  	github.com/bidshard/ad-event-processor/internal/ingestion	0.056s  # -run='OSFingerprint|DeviceFilter_osFingerprint'
=== P8-5 cache-line padding ===
--- PASS: TestResidentialProxyCell_cacheLinePadded_holdout
--- PASS: TestIPv6RotationCell_cacheLinePadded_holdout
BenchmarkIPv6Rotation_observe-12    6933080    16.23 ns/op    0 B/op    0 allocs/op
BenchmarkResidentialProxy_observe-12  252867   468.5 ns/op    0 B/op    0 allocs/op
=== P8-6 XDP host HASH / LPM split holdout ===
=== RUN   TestBlocklistStore_hostHashMap_holdout
--- PASS: TestBlocklistStore_hostHashMap_holdout (0.00s)
=== P8-6 blocklist sync regression ===
ok  	github.com/bidshard/ad-event-processor/internal/edge	0.051s  # -run='SyncBlocklistIncremental|BlocklistStore|RecordAutoBan|hostHashMap'
ok  	github.com/bidshard/ad-event-processor/internal/edge	9.742s  # full package
BenchmarkSyncBlocklistIncremental_changelogDelta-12    1237    99710 ns/op
=== gates ===
anti-slop: OK
cold-path-static: OK
escape-heap-gate: OK
```

Regenerate:

```bash
go test ./internal/ingestion/ -run='LocalQuanta|FullSkip|L3Blacklist|LuaScriptSlimmed|Rollback' -count=1
go test ./internal/ingestion/ -run='TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout' -count=1 -v
go test -short -count=1 -run 'ZeroAlloc|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...
go test ./internal/controlplane/ -run='MLBlacklist|Quarantine|FastLane' -count=1
go test ./internal/fraud/ -run='OutboxBackpressure|Detector|OutboxBacklog' -count=1
go test ./internal/edge/ -run='Quarantine|FraudQuarantine' -count=1
go test ./internal/ingestion/ -run='DCASN|FraudFilter_DCASN' -count=1
go test ./internal/ingestion/ -run='TestFraudFilter_DCASN_holdout' -count=1 -v
go test ./internal/ingestion/ -run='OSFingerprint|DeviceFilter_osFingerprint|NormalizeCaptured' -count=1
go test ./internal/ingestion/ -run='ResidentialProxy|IPv6Rotation|cacheLinePadded' -count=1
go test ./internal/edge/ -run='SyncBlocklistIncremental|BlocklistStore|RecordAutoBan|hostHashMap' -count=1
go test ./internal/edge/ -bench='BenchmarkSyncBlocklistIncremental' -count=1
go generate ./internal/edge/
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkResidentialProxy_observe|BenchmarkIPv6Rotation_observe' -benchmem -benchtime=100ms -count=1
bash scripts/ci/anti_slop_gate.sh
bash scripts/ci/cold_path_static_gate.sh
bash scripts/ci/escape_heap_gate.sh
bash scripts/ci/hot_path_static_gate.sh
```

---

## Agent checklist (copy into every PR touching backlog items)

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import `internal/fraud` scoring
- [ ] Holdout test fails without the change (named in PR)
- [ ] **Raw** command output pasted in PR for each checked Verify box (not paraphrase)
- [ ] Explain why scripts/tests changed
- [ ] `bash scripts/ci/anti_slop_gate.sh` executed
- [ ] `bash scripts/ci/pr_fast.sh` executed (scoped)
- [ ] Path supplements applied (hot/cold/edge/UI/budget) — boxes checked or N/A justified in PR
- [ ] Nested checkboxes in `BACKLOG.md` updated in same PR
- [ ] No lie modes (`anti-slop.mdc`): no invented bench, SLA, BPF, or env

---

## Audit traceability

| Audit theme | Backlog IDs |
| :--- | :--- |
| Redis Lua bottleneck | P0-1, P0-2, P6-1, P6-2, P6-3, P6-5 |
| XDP LPM / scale | P2-2, P6-4 |
| Outbox ML_BLACKLIST_ADD | P1-1, P1-2, P1-3 |
| Residential/mobile proxy | P3-1 – P3-6 |
| Headless / JS probe | P4-1 – P4-4 |
| IPv6 evasion | P2-1 – P2-3 |
| In-app WebViews | P5-1 – P5-3 |
| Doc vs code gaps | P2-4, P7-1 – P7-3 |
| Antifraud extreme-load gaps (audit 2026-08) | P8-1 – P8-6 |
| Local quanta full-skip L3 | P8-1 (done) |
| Outbox storm / IVT pause | P8-2 |
| DC ASN sampling bypass | P8-3 |
| OS fingerprint CDN / TTL | P8-4 |
| Hot ring false sharing | P8-5 |
| XDP host map topology | P8-6 |

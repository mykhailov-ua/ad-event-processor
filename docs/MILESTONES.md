# Closed Milestones (Archive)

Engineering record of completed milestone tracks. **Open work** lives in [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md). Operator runbooks stay in the linked docs below.

**SKU policy:** self-hosted **track → budget → settlement → admin on a single VPS**. Public docs: **BidShard**. Engineering archive: **ad-event-processor**; no `espx` — [NAMING.md](NAMING.md).

---

## 1. Broker integration and Redis RAM offload — CLOSED (2026-08-09)

**Goal:** disk-backed WAL (`pkg/broker`) as primary ClickHouse ingest; cut Redis stream RAM; zero-loss crash recovery on appliance.

| Phase | Outcome |
| :--- | :--- |
| 0 Lua | Tactical `unified-filter.lua` / `budget-fast.lua` optimizations |
| 1 Producer | Lock-free `BrokerProducer`; 0 B/op; p99 Enqueue &lt; 1 µs |
| 2 CH ← broker | `CH_INGEST_SOURCE=broker` disables Redis `_ch`/`_pg` consumers; E2E parity |
| 3 XTRIM | `RedisStreamTrimmer`; lab peak **3.6 MiB**/shard at 100k RPS |
| 4 Replay | `broker replay` CLI + integrity test |
| 5 UDS | Redis/Postgres unix sockets; dial p50 ~4.8 µs |
| 6 TCP backlog | sysctl + `tcp_syn_drop_gate.sh` (ListenOverflow delta = 0) |
| 7 Adaptive breaker | EWMA per shard; Sentinel scenario G at 30k RPS |
| 8 NOSCRIPT | Reconnect preload + CI/compose drills (p99 &lt; 80 ms) |
| 9 CH spool async | `StartAsyncFlusher(20ms)` + spool stress gates |

**Default appliance config:**

```text
/track → Lua (budget; main XADD skipped via fcap:ignored when broker-primary)
       → BrokerProducer → broker mmap WAL
processor: BrokerConsumerGroup _ch_broker + _pg_broker ON; Redis Stream consumers OFF
```

**Docs:** [DEVELOPMENT.md §7 Broker cutover](DEVELOPMENT.md#broker-cutover-ch_ingest_source), [PEL_DRAIN.md](PEL_DRAIN.md), [BENCHMARKS.md](BENCHMARKS.md) §A.4a / §C.

**Verification:**

```bash
bash scripts/fault/milestone_compose_drill.sh all
bash scripts/perf/redis_ram_proof.sh
```

---

## 2. Parser security (ingress hardening) — CLOSED

**Goal:** reject hostile wire/JSON on hot path; nginx ↔ gnet parity; slow-body DoS closed.

| Phase | Gaps | Topic |
| :--- | :--- | :--- |
| P0 | PS-G01 | HTTP/1 slow-body / incomplete hold |
| P1 | PS-G02–G04, PS-H01 | Chunk extensions, ORTB scan cap, edge parity, pool cap |
| P2 | PS-G05–G13, PS-H02–H03, PS-H06 | TE.TE, protobuf budget, HPACK cap, chaos load |
| P3 | PS-H04–H05 | ORTB literal keys, UTF-8 values |

**Docs:** [PARSER_SECURITY.md](PARSER_SECURITY.md), [EDGE_CASES.md §9](EDGE_CASES.md#9-parser-security-vs-infrastructure-failures).

**Verification:**

```bash
bash scripts/fault/parser_chaos_drill.sh
go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
```

Engineering catalog: git history `.cursor/PARSER_SECURITY_MILESTONE.md` (removed 2026-08-16); gap IDs in table above.

---

## 3. Shard 0 SPOF hardening — CLOSED

**Goal:** tracker and control survive `rdbs[0]==nil` (`REDIS_SHARD0_OPTIONAL_STARTUP=1`); budget shards 1..N keep serving; best-effort outbox fan-out.

| Phase | Fix |
| :--- | :--- |
| P0 | Nil-safe SyncWorker, readiness, shutdown, broker reconcile |
| P1 | `forEachConnectedShard` best-effort fan-out + metrics |
| P2 | Tracker globals via `firstConnectedRedis` (fraud agg, creatives, RTB watch) |
| P4 | Edge Lua `connect_any_shard()`; XDP stats `ReadRedisAny` |
| P5 | Prometheus alerts, `shard0_nil_gate.sh` in `pr_fast.sh` |

**Shard 0 recovery:** automated catch-up via `Shard0CatchupWorker` and `POST /api/v1/ops/shards/0/catchup` — see [SHARDING_MILESTONE.md §Shard 0 recovery](SHARDING_MILESTONE.md#shard-0-recovery-automated-catch-up).

**Docs:** [SHARDING_MILESTONE.md](SHARDING_MILESTONE.md), [DEVELOPMENT.md §Shard 0 degradation](DEVELOPMENT.md#shard-0-degradation-runbook).

**Verification:**

```bash
bash scripts/ci/shard0_nil_gate.sh
go test -count=3 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingestion/
```

---

## 4. Codebase cleanup (CUT) — CLOSED

| Priority | Removed / fixed |
| :--- | :--- |
| P0 | Privacy Sandbox / CTV marketing claims; BPF path in DEVELOPMENT |
| P1 | `deploy/k8s/**`, `deploy/terraform/**`, orphan `deploy/management|payment/`, stale Prom targets |
| P2 | `cmd/tracker-quic`; compose Advanced profiles; [FROZEN_FEATURES.md](FROZEN_FEATURES.md) |

**k8s / k3s deployment path removed (2026-08):** archive tree deleted; installer profile `k8s_k3s` and admin UI option removed. Appliance SKU uses `single_vps` only.

---

## 5. Enterprise feature freeze — CLOSED (policy)

Multi-region (`region-proxy`) and NIC XDP (`edge-xdp`) remain in git under license gates; excluded from `single_vps` installer and pilot SKU.

**Docs:** [FROZEN_FEATURES.md](FROZEN_FEATURES.md), [enterprise/MULTI_REGION.md](enterprise/MULTI_REGION.md), [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md).

---

## 6. Fault scenarios E / F / G — CLOSED

| Scenario | Proof |
| :--- | :--- |
| E NOSCRIPT | `TestFault_NOSCRIPTStorm`, `noscript_compose_drill.sh` |
| F CH spool disk | `TestFault_CHSpoolDiskBlock`, `spool_compose_drill.sh` |
| G Sentinel promotion | `TestFault_SentinelPromotionIsolation`, `sentinel.sh` @ 30k RPS |

CI: `scripts/fault/milestone_gates.sh` at end of `scripts/fault/run.sh`.

---

## 7. `management` → `control` naming — CLOSED (2026-08)

**Goal:** align Prometheus metrics, alerts, and env vars with the `control` binary name.

| Area | Change |
| :--- | :--- |
| Metrics | Primary `ad_control_*`; legacy `ad_management_*` dual-published for one release via `internal/metrics/control_publish.go` |
| Alerts | `control-cold-path-alerts`; expressions use `ad_control_*` |
| Alertmanager | Receiver `control-webhook` (breaking: update custom hooks from `management-webhook`) |
| Env | `CONTROL_PORT` preferred; `MANAGEMENT_PORT` still accepted |

**Verification:**

```bash
go test ./internal/metrics/ -run TestControl -count=1
go test ./internal/config/ -run TestResolveControlPort -count=1
```

---

## 8. September 2026 GA appliance (P0) — CLOSED (2026-08-12)

**Goal:** gray-market on-prem SKU — track → budget → settlement → admin on a single VPS; buyer can integrate without engineering docs.

| Track | Outcome | Docs / UI |
| :--- | :--- | :--- |
| Integration kit | Campaign **Integration** + **CAPI & Postbacks** tabs; inbound S2S, macros sub1–30, traffic-source templates (33), affiliate presets (36) | [TRAFFIC_INTEGRATION.md](TRAFFIC_INTEGRATION.md), `web/e2e/campaign_integration.spec.js` |
| Edge `GET /click` | Default **on** for new installs; Lua gate + shard parity | [TRAFFIC_INTEGRATION.md §5](TRAFFIC_INTEGRATION.md#5-enabling-optional-edge-paths), `scripts/test/edge_click_smoke.sh` |
| Cost Sync + True ROI | `/integrations/cost-sync`, `/reports/true-roi` | [TRAFFIC_INTEGRATION.md §3.2](TRAFFIC_INTEGRATION.md#32-cost-sync-join-keys--true-roi) |
| CAPI | Meta / Google / TikTok adapters, worker, DLQ + retry in campaign UI | [TRAFFIC_INTEGRATION.md §3.1](TRAFFIC_INTEGRATION.md#31-inbound-affiliate-s2s-vs-outbound-capi), `scripts/test/capi_meta_staging.sh` |
| Bulk campaign ops | Multi-select pause / resume / export IDs on `/campaigns` | — |
| Campaign health badge | List + detail from budget / pacing / IVT | — |
| Install + USDT license | Gray on-prem QUICKSTART, sales kit, tier enforcement | [QUICKSTART.md](QUICKSTART.md), [deploy/vendor/SALES_KIT.md](../deploy/vendor/SALES_KIT.md) |
| Ops alerts on overview | Outbox, registry stale, shard circuit, license | `/ops`, overview feed |

**Ops gate before `v1.0.0-pilot-gray` tag (manual):** run `bash scripts/test/capi_meta_staging.sh` on staging with Meta `test_event_code` (dry-run: `CAPI_STAGING_DRY_RUN=1`).

**Verification:**

```bash
node web/scripts/build.mjs
bash scripts/ci/pr_fast.sh
bash scripts/ci/admin_web.sh
bash scripts/ops/verify_redis_topology.sh
```

---

## 9. Competitive parity (P1) — CLOSED (2026-08-12)

| Feature | Outcome |
| :--- | :--- |
| Margin Guard portfolio | `/margin-guard` — breach scan → campaign margin tab |
| Live reports | spend-velocity, daypart, geo-device, source-quality, discrepancy, portfolio, overview + nav |
| Smart Alerts | Rules + CH metrics + webhook; `/integrations/smart-alerts` |
| Domain health + SSL | Worker probes; `/settings/domains` — probe + certbot hook |
| RTB onboarding UI | `/rtb/integration` — profile, validate-bid, shadow-diff |
| Brands & creatives | Campaign **Creative** tab — weighted landings CRUD |
| Safe-page redirect | `safe_page_*` on campaign; hot-path `302` on IVT / blacklist |
| Commercial tiers | `starter`…`enterprise` in `sku.yaml`; license banners in admin |
| Zero-redirect tracking | `bidshard-track.js` + CORS on `POST /track` | [TRAFFIC_INTEGRATION.md §3.3](TRAFFIC_INTEGRATION.md#33-zero-redirect-tracking-lander-pixel) |
| SubID ×30 | Hot path + macros + admin reference; CH typed columns deferred | [TRAFFIC_INTEGRATION.md §2](TRAFFIC_INTEGRATION.md#get-click--click-redirect) |

**CUT (not shipping):** Keitaro-style offer/LP weight automation (§11.6); dedicated `/lp-events` API (§11.7) — use `POST /track` on external landers.

---

## 10. Admin SPA (TypeScript + functional GA) — CLOSED (2026-08-12)

**Stack:** Vanilla JS/TS procedural DOM, **esbuild** bundle, `go:embed` in `cmd/control`. HTMX removed.

| Area | Routes / surfaces |
| :--- | :--- |
| Campaigns | List, detail tabs (overview, stats, config, integration, CAPI, filters, margin, events, creative, Telegram) |
| Customers & billing | `/customers`, `/billing` (ledger, invoices), `/billing/invoices/:id` |
| Reports | Hub + live CH reports + Telegram suite + saved views in hub |
| Dashboards | `/dashboards/:role` — adops, cfo, accountant, fraud |
| Integrations | cost-sync, margin-guard, smart-alerts, supply (`sellers.json`, `ads.txt`) |
| RTB | `/rtb/integration`, `/rtb/deals` (list skeleton) |
| Ops | `/ops` (doctor, outbox, metrics), `/ops/shards`, `/ops/blacklist` |
| Settings | Platform, license apply, `/settings/domains` |

**Tooling:** `web/tsconfig.json` strict; `npm run typecheck`; `web/src/types/api/` DTOs; CI `admin_web.sh` + `admin_bundle_gate.sh`.

**Open UI gaps:** [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md) §1 — detailed DoD, testing, SLA, and perf per feature (§1.0 cross-cutting, §1.5 release gate).

**Docs:** [DEVELOPMENT.md §Admin web UI](DEVELOPMENT.md#admin-web-ui-web).

---

## 11. De-branding (`espx` → ad-event-processor) — CLOSED (2026-08-12)

**Policy:** [NAMING.md](NAMING.md) — public **BidShard**, internal **ad-event-processor**, no `espx` / `ESPX_*`.

| Phase | Outcome |
| :--- | :--- |
| CI gates | `scripts/ci/check_no_espx.sh`, `check_no_espx_core.sh` |
| Surfaces | `web/`, `pkg/branding`, public docs |
| Paths / env | `/etc/ad-event-processor`, `AD_EVENT_PROCESSOR_*` (+ `ESPX_*` read alias one release) |
| Metrics / ingress | `ad_event_processor_edge_*`, `ad_event_processor_native` |
| Go module + CLI | `cmd/ad-event-processor`; `cmd/espx` deleted |

---

## 12. On-prem operability + traffic ingress — CLOSED (2026-08-12)

| Track | Outcome |
| :--- | :--- |
| Shard 0 catch-up | `Shard0CatchupWorker`, `POST /api/v1/ops/shards/0/catchup` | [SHARDING_MILESTONE.md](SHARDING_MILESTONE.md) |
| Redis 4-shard appliance | `redis-4`/`redis-5` behind compose profile; installer emits 4 masters | [ARCHITECTURE.md](ARCHITECTURE.md), `verify_redis_topology.sh` |
| k8s/k3s CUT | Archive tree removed | [CUT_CANDIDATES.md](CUT_CANDIDATES.md) |
| Doctor probes | slot-map parity, listen backlog, license in CLI bundle | `ad-event-processor doctor` |
| Local quanta + Redis SIGKILL | Budget invariant fault test | [EDGE_CASES.md](EDGE_CASES.md) |
| Cold-path JSON limits | `pkg/coldpath` + `cold_path_json_gate.sh` | [COLD_PATH_JSON.md](COLD_PATH_JSON.md) |
| Path-based ingress | Separate `/track`, `/click`, `/openrtb/bid`, `/tg/*`; optional edge expose | [TRAFFIC_INTEGRATION.md](TRAFFIC_INTEGRATION.md), [ARCHITECTURE.md §1.1](ARCHITECTURE.md) |

---

## 13. Single-VPS installer (Tier 0–2) — CLOSED (2026-08)

**Goal:** one-path appliance install without host Go; Caddy ingress; GHCR release images.

| Tier | Outcome |
| :--- | :--- |
| 0 | Docker build runs codegen; `--yes` install; doctor; QUICKSTART |
| 1 | GHCR images; release tarball; `get.sh` bootstrap |
| 2 | Caddy ingress profile; install done UI; kernel preflight WARN |

**Operator doc:** [QUICKSTART.md](QUICKSTART.md). Agent spec removed from `.cursor/INSTALLER.md` (2026-08-16).

---

## 14. Cold-path isolation queue (ISO-01–ISO-18) — CLOSED (2026-08-14)

**Goal:** PG/Redis consistency, outbox ordering, N+1 batching on cold paths without hot-path regression.

All items **Done** (recon adjust, quota repair, outbox ordering, shard-0 catch-up policy, postback in-flight claim, etc.). Engineering queue removed from `.cursor/ISOLATION.md` (2026-08-16).

**Verification:** `make test-integration`; `bash scripts/ci/pr_fast.sh`.

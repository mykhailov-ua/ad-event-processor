# ad-event-processor

Self-hosted ad event stack: high-throughput tracker (`/track`, `/click`), optional in-process OpenRTB exchange (`/openrtb/bid`), admin API and React UI (`:8188`), and async settlement pipeline (Postgres financial truth, ClickHouse analytics). Go module: `ad-event-processor`.

Not a managed SaaS. A fresh clone is sources-only: run codegen, provide env, fetch GeoIP, build BPF if needed (see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)).

---

## Features

### Stack layers

| Layer | Role |
| :--- | :--- |
| Edge (nginx/OpenResty) | L7 ingress, rate limits, blacklist cache, body parsing, shard routing to tracker pool |
| Tracker (`:8181-8184`) | Accept/reject under SLA; `FilterEngine`; Redis budget/dedup; async stream log |
| Processor (`:8186`) | Redis stream consumers -> Postgres, ClickHouse |
| Control (`:8188`) | Admin UI, `/api/v1/*`, outbox workers, billing, reports |
| Optional sidecars | `ivt-detector`, `fraud-scorer`, `edge-xdp`, `edge-bpf-sync`, `region-proxy` |

HTTP **202 on `/track`** means ingest accepted and validated on the hot path. It does **not** mean Postgres or ClickHouse committed the event.

### Hot-path design

On synchronous `/track`, `/click`, `/tg/*`, `/openrtb/bid`:

| Rule | Detail |
| :--- | :--- |
| No sync databases | No Postgres, ClickHouse, outbox, or ML inference on the request thread |
| Config snapshot | Campaign registry in `atomic.Pointer` + Redis pub/sub reload; no per-request PG fetch |
| Redis budget | At most one `EVALSHA` per accepted event (`unified-filter.lua`); zero when local quanta full-skip eligible |
| Stream log | `XADD` async via `StreamProducer`; not in the same Lua script as debit |
| Fail closed | 503 when stream admission or Redis breaker is open |

Sharding: `slot = CRC32C(campaign_id) & 1023`, `shard = slot_table[slot]`. Edge Lua and Go use the same static slot map. Redis: standalone masters + Sentinel (not Cluster); `{campaign_id}` hash tag keeps multi-key Lua on one master.

gnet workers do not block on Redis: `FilterEngine.Check` (including `EVALSHA`) runs in a detached goroutine.

### Traffic endpoints

| Endpoint | Method | Role |
| :--- | :--- | :--- |
| `/click` | GET | 302 redirect with macros; full `FilterEngine`; click budget debit |
| `/track` | POST | S2S postback / conversion ingest; JSON, protobuf, or OpenRTB 3 ingress per campaign format |
| `/openrtb/bid` | POST | OpenRTB 2.x exchange; in-process auction; no full filter chain |
| `/tg/click`, `/tg/impression` | GET | Telegram Mini App traffic |
| `/api/v1/*` | * | Control plane (operators + self-serve API keys) |

Wire policy (nginx <-> gnet parity, chaos-tested): `/track` requires `Content-Length` and rejects chunked `Transfer-Encoding`; `/openrtb/bid` allows chunked with size caps.

Macros: `{campaign_id}`, `{click_id}`, `{sub1}`...`{sub30}`, UTMs. Zero-redirect tracking via `web/src/static/track.js` (POST `/track` with CORS). Traffic wiring, Cost Sync, CAPI, and templates: [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md).

### Filter engine and budgeting

Go-local filters run before Redis (license, geo, schedule, entitlements, fraud signals). `UnifiedFilter` is last.

| Concern | Implementation |
| :--- | :--- |
| Budget / dedup / pacing | Atomic Lua on campaign shard |
| Placement blacklist | In-process TTL cache (5 s); `HEXISTS` on miss |
| Fraud blacklist (L3) | In-process TTL cache; `SISMEMBER` on miss |
| Ingress RPD | `EntitlementsFilter` -> Redis `INCR` |
| ML fraud boost | Go snapshot (`ml:score:boost:*`); async sync via `SettingsWatcher` |
| Local quanta | `LOCAL_QUOTA_MODE=live` can eliminate sync `EVALSHA` for eligible traffic |

Postgres holds financial truth (`current_spend`, `balance_ledger`); Redis budgets are operational limits reconciled async.

### Antifraud

Multi-layer: optional nginx/eBPF perimeter, Go `FilterEngine` on tracker, async IVT rules and ML on cold path.

| Level | Behavior |
| :--- | :--- |
| L1 reject | Hard fraud signal(s); HTTP 403, or silent reject (decoy 202/302) when `silent_reject_enabled` |
| L2 shadow | Accepted with `ShadowEvent=true`; logged to ClickHouse; no budget debit when fraud signals skip unified Lua |
| L3 blacklist | IP on Redis `blacklist:fraud`; edge may 403 before tracker |

Hot-path signals include datacenter IP, low TTC, TLS blocklist (JA3/JA4), device mismatch, TCP MSS anomaly, OS fingerprint mismatch, IPv4 rotation, residential proxy (SKU-gated), attestation missing. Presets: `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app`.

Limits (see [deploy/vendor/ANTIFRAUD.md](deploy/vendor/ANTIFRAUD.md)):

- ML does not run on the tracker; `cmd/fraud-scorer` and `cmd/ivt-detector` are cold-path sidecars only.
- XDP drops known L3/L4 bad IPs and syn-flood patterns, not app-layer residential fraud.
- Behind CDN/ALB without edge TCP fingerprint sync, TCP MSS / TTL signals fail-open or must be disabled.
- Cold-path ML `silent_reject` action adds IP to `blacklist:fraud`; it does not flip `silent_reject_enabled` on campaigns.

Open work: [deploy/vendor/antifraud_backlog.md](deploy/vendor/antifraud_backlog.md).

### OpenRTB / in-process RTB

- Structure-of-Arrays in-memory catalog; `atomic.Pointer` read on hot path.
- Auction: geo-partitioned scan; bid x CTR scoring; first/second price + reserve.
- Modes: `RTB_MODE=off|shadow|live`; shadow uses `RunAuctionEval` without spend.
- OpenRTB 2.6: DFA parse on hot path; in-place bid response write.
- Admin: deals, floors, `validate-bid-request`, `shadow-diff`, reconcile export.

`/track` runs RTB only when `RTB_MODE=shadow|live`. `/openrtb/bid` always runs auction when licensed. OpenRTB 3.0 and multi-imp >1 are not implemented.

### Control plane and admin

Single `cmd/control` modular monolith:

- React admin SPA (`web/`) with role dashboards (buyer, adops, fraud, CFO, operator, campaign) and integration screens under `/integrations/*` (see below).
- ~290 `/api/v1` routes: campaigns, customers, brands/creatives, supply (`sellers.json`, `ads.txt`), billing/invoices, ledger, disputes, margin guard, smart alerts, domains/TLS, flows/landers/offers, team/RBAC, integration schemas, postbacks, RTB admin, recon, reports, self-serve, Telegram, license, ops (DLQ, outbox, shards, doctor).

Outbox: every config mutation + `outbox_events` in the same PG transaction; `OutboxWorker` polls ~20 ms and applies Redis side effects. Tracker never polls outbox.

### Integrations

Traffic ingest, Cost Sync (22 networks, daily campaign-level spend), outbound CAPI (Meta/Google/TikTok + webhook), 82 bundled traffic click schemas plus 77 affiliate templates, and Enterprise Meta/Google platform campaign sync. Full tables, credential fields, and explicit non-goals: [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md).

### Billing and analytics

- Payment webhooks -> `payment_outbox` -> `SettlementHandler` -> `balance_ledger`.
- Invoices, PDF export, tax profiles, self-serve payment intents, Cost Sync for true ROI reports.
- ClickHouse ingest from processor; materialized views for report aggregates. Campaign stats may show `stale=true` when CH lag > 5 min.

### Edge perimeter

| Component | Role |
| :--- | :--- |
| nginx `:8180` / `:443` | Rate limit, circuit breaker, blacklist shared dict, body DFA, proxy to tracker |
| `edge-blacklist-sync.lua` | Redis shard 0 -> nginx cache; changelog + periodic full sync |
| `edge-xdp` + `edge-bpf-sync` | Optional (Enterprise license); BPF map drops at NIC |
| Tarpit | Optional `EDGE_TARPIT_ENABLED`; capped; never on billing paths |

### Multi-region (Enterprise)

`region-proxy` with regional WAL, quorum book/ack, uplink to global control. Regional processor with `MULTI_REGION_ENABLED=1`. Requires Enterprise license (`features.multi_region`).

### Engineering gates (CI)

Hot-path static gates (no `fmt.Sprintf`, no `interface{}` boxing on ingest), allocation gate on `/track`, parser chaos nginx <-> gnet differential count = 0, fault tests for budget invariants and outbox ordering, license red-team verify tiers.

---

## Documentation

| Document | Content |
| :--- | :--- |
| [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md) | Cost Sync, CAPI, bundled schemas, platform sync |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Hot/cold boundary, topology, ports, Redis sharding |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Local stack, codegen, tests, SLA gates |
| [docs/MANAGED_SAAS.md](docs/MANAGED_SAAS.md) | Vendor-hosted isolated compose cells |
| [deploy/vendor/ANTIFRAUD.md](deploy/vendor/ANTIFRAUD.md) | Fraud signals, layers, edge/XDP, cold-path ML |
| [deploy/vendor/antifraud_backlog.md](deploy/vendor/antifraud_backlog.md) | Open antifraud work items |
| [deploy/vendor/sku.yaml](deploy/vendor/sku.yaml) | License SKU limits and feature flags |
| `.cursor/rules/` | Engineering constraints (SLA, hot path, CI) |

---

## Services and ports

| Binary | Port | Role |
| :--- | :---: | :--- |
| nginx edge | 8180, 443 | Ingress, Lua filters, proxy to tracker |
| `tracker` | 8181-8184 | `/track`, `/click`, `/openrtb/bid`, `/tg/*` |
| `processor` | 8186 | Redis stream consumers -> Postgres, ClickHouse |
| `control` | 8188 | Admin UI, `/api/v1/*`, outbox workers |
| payment webhooks | 8187 | Balance top-up hooks |

Metrics: tracker `9101-9104`, processor `9106`, control `9108`. Postgres `5430`, Redis masters `6479-6482`, ClickHouse `9000` (compose defaults).

---

## SLA budgets (CI / load test)

| Surface | Target |
| :--- | :--- |
| Tracker handler | p95 < 50 ms, p99 < 80 ms (abort load test if p99 > 80 ms for 30 s) |
| Filter deadline | `FILTER_TIMEOUT_MS` <= 100 ms (production) |
| Redis unified-filter Lua | p99 < 10 ms per shard |
| Hot path ingest | Zero heap allocations on `/track` (`make test-alloc-gate`) |
| RTB `RunAuction` | p99 < 15 us; candidate scan p99 < 500 (microbench scope) |

Details: `.cursor/rules/core.mdc`, `.cursor/rules/hot-path.mdc`.

---

## Repository bootstrap

Sources-only clone: run codegen, provide env, build BPF if needed, fetch GeoIP per DEVELOPMENT.md.

**One command (dev appliance):**

```bash
bash scripts/install/appliance_bootstrap.sh
```

Default profile `ingest-only` (~4 GB RAM). Full stack with ClickHouse:

```bash
bash scripts/install/appliance_bootstrap.sh --profile full
```

Manual steps (equivalent):

```bash
make gen
make proto
cp .env.example .env
bash scripts/dev/stack.sh full
```

License file default path: `var/license.jwt`. Issue JWT: `go run ./cmd/license-issue` (see `deploy/vendor/KEYS.md`).

---

## Compose profiles

| Profile | Services |
| :--- | :--- |
| `single-vps` / `full` | Tracker, processor, control, Postgres, Redis x4, ClickHouse |
| `ingest-only` | Same without ClickHouse (lower RAM) |
| `infra` | Datastores only for local Go binaries |
| `analytics-ml` | Adds `fraud-scorer`, `ivt-detector` |

---

## Licensing

Monthly Ed25519 JWT; limits by peak RPS and host count per `deploy/vendor/sku.yaml`. Enforcement in `internal/licensing`. No per-campaign or per-event caps in SKU (`max_active_campaigns: 0`, `max_events_per_month: 0` = unlimited in license schema).

| SKU | Peak RPS | Hosts | IVT detector | ML boost | OpenRTB | eBPF XDP | Multi-region |
| :--- | ---: | ---: | :---: | :---: | :---: | :---: | :---: |
| Starter | 10k | 1 | no | no | no | no | no |
| Pro | 25k | 1 | yes | no | no | no | no |
| Scale | 75k | 3 | yes | yes | yes | no | no |
| Network | 150k | 10 | yes | yes | yes | no | yes |
| Enterprise | custom | 99 | yes | yes | yes | yes | yes |
| Pilot | 5k | 1 | no | no | no | no | no |

Full fields: [deploy/vendor/sku.yaml](deploy/vendor/sku.yaml).

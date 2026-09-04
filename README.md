# ad-event-processor

Self-hosted server software for HTTP traffic measurement, event ingestion, campaign routing, and spend accounting. Operators install and run the stack on infrastructure they control. The vendor supplies binaries, install scripts, and an offline license token; the vendor does not host operator traffic or store operator campaign data in the default delivery model.

Go module: `ad-event-processor`. A fresh clone is sources-only: run codegen, provide env, fetch GeoIP, build BPF if needed (see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)).

---

## Product scope

### What the software is

| Category | Description |
| :--- | :--- |
| Traffic routing | HTTP redirects (`/click`) with macro substitution; optional programmatic bid endpoint (`/openrtb/bid`) |
| Event ingestion | Server-to-server postbacks and browser events (`/track`); Telegram Mini App endpoints (`/tg/*`) |
| Campaign configuration | Budgets, schedules, geo rules, flows, and integration hooks via admin API |
| Spend enforcement | Redis-backed debit/dedup on the ingest path; Postgres as financial ledger on the cold path |
| Analytics | ClickHouse for event storage and report queries (optional profile) |
| Traffic quality rules | Rule-based filters on the ingest path; batch scoring workers on the cold path (license-gated) |

### What the software is not

| Exclusion | Detail |
| :--- | :--- |
| Hosted SaaS | No multi-tenant cloud operated by the vendor |
| Data processor (default model) | Vendor does not receive or store operator production events on vendor systems |
| Ad network or traffic seller | No traffic acquisition; operator wires sources to their own endpoints |
| Guaranteed outcome service | No SLA on ROI, fraud elimination, or conversion rates |
| Penetration or access-bypass tooling | No features whose primary purpose is unauthorized access to third-party systems |

### Operator responsibilities

The licensee configures traffic sources, legal bases for data collection, hosting, backups, TLS, and integrations. Misuse of dual-use infrastructure (including offenses under applicable computer-crime law) is the operator's responsibility. See [deploy/vendor/PUBLIC_OFFER.md](deploy/vendor/PUBLIC_OFFER.md).

---

## Architecture

### Process map

| Process | Port(s) | Role |
| :--- | :---: | :--- |
| nginx / OpenResty edge | 8180, 443 | L7 ingress, rate limits, blacklist cache, body policy, shard proxy |
| `tracker` | 8181-8184 | Ingest accept/reject; filter chain; async event log |
| `processor` | 8186 | Stream consumers to Postgres and ClickHouse |
| `control` | 8188 | Admin HTTP API (`/api/v1/*`), outbox workers, billing hooks |
| payment webhooks | 8187 | Balance top-up callbacks |
| Optional sidecars | — | `ivt-detector`, `fraud-scorer`, `edge-xdp`, `edge-bpf-sync`, `region-proxy` |

Metrics: tracker `9101-9104` (compose), processor/control on service HTTP ports (`/metrics`), edge `8180/edge/metrics`.

### Hot path constraints

On synchronous `/track`, `/click`, `/tg/*`, `/openrtb/bid`:

| Rule | Detail |
| :--- | :--- |
| No sync databases | No Postgres, ClickHouse, outbox, or ML inference on the request thread |
| Config snapshot | Campaign registry in `atomic.Pointer` + Redis pub/sub reload |
| Redis budget | At most one `EVALSHA` per accepted event; zero when local quanta full-skip eligible |
| Stream log | Async `XADD` via `StreamProducer`; not inside the debit Lua script |
| Fail closed | HTTP 503 when stream admission or Redis breaker is open |

HTTP **202 on `/track`** means the event passed hot-path validation. It does not assert Postgres or ClickHouse persistence.

Sharding: `slot = CRC32C(campaign_id) & 1023`. Edge Lua and Go share the same static slot map. Redis uses standalone masters + Sentinel; `{campaign_id}` hash tag keeps Lua keys on one master.

Thread model: gnet epoll enqueues to `PinnedWorkerPool`; synchronous `FilterEngine.Check` (including `EVALSHA`) runs on pinned workers (`cmd/tracker/doc.go`).

### HTTP surfaces

| Endpoint | Method | Role |
| :--- | :--- | :--- |
| `/click` | GET | Redirect with macros; full filter chain; click budget debit |
| `/track` | POST | S2S / browser events; JSON, protobuf, or OpenRTB 3 ingress per campaign |
| `/openrtb/bid` | POST | OpenRTB 2.x auction in-process; no full filter chain |
| `/tg/click`, `/tg/impression` | GET | Telegram Mini App |
| `/api/v1/*` | * | Control plane (operator and self-serve API keys) |

Wire policy: `/track` requires `Content-Length` and rejects chunked `Transfer-Encoding` (nginx/gnet parity tests). Macros: `{campaign_id}`, `{click_id}`, `{sub1}`-`{sub30}`, UTMs. Browser snippet: `web/src/static/track.js` (POST `/track` with CORS).

Integration catalog: [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md).

### Filter and budget pipeline

Go-local filters run before Redis (license, geo, schedule, entitlements, rule-based signals). `UnifiedFilter` is last.

| Concern | Implementation |
| :--- | :--- |
| Budget / dedup / pacing | Lua on campaign shard |
| Blacklists | In-process TTL cache; Redis on miss |
| Ingress RPD | Redis `INCR` via entitlements filter |
| ML score overlay | Go snapshot from Redis; written by `fraud-scorer` sidecar (not per-request inference) |

Postgres holds financial truth; Redis budgets are operational limits reconciled asynchronously.

### Traffic quality (rule-based + batch)

Layers: optional nginx/eBPF perimeter, Go filters on tracker, batch IVT rules and ML on cold path.

| Mode | Behavior |
| :--- | :--- |
| Hard reject | HTTP 403, or silent accept (decoy 202/302) when `silent_reject_enabled` on campaign |
| Shadow | Accepted with analytics flag; no budget debit when fraud signals skip unified Lua |
| Blacklist | Redis `blacklist:fraud`; optional XDP drop at NIC (Enterprise license) |

Limits documented in [deploy/vendor/ANTIFRAUD.md](deploy/vendor/ANTIFRAUD.md): ML and IVT run in sidecars, not on every `/track`; CDN without edge TCP sync may require disabling some fingerprint signals.

### OpenRTB

In-memory catalog; `RunAuction` on hot path. Modes: `RTB_MODE=off|shadow|live`. OpenRTB 3.0 and multi-imp auctions with `imp` count > 1 are not implemented. License: Scale tier and above (`openrtb_engine` / `rtb_live`).

### Control plane

Single `cmd/control` process: admin API, outbox workers, billing. Outbox applies config mutations to Redis in the same Postgres transaction as the admin write. Tracker does not poll outbox.

Admin UI: React SPA in `web/` (build embeds into `internal/controlplane/admin_static_stub/`). Until a release includes `web/dist`, operators may use HTTP API only.

### Edge (optional)

nginx circuit breaker, generational blacklist sync, shard routing. `edge-xdp` + `edge-bpf-sync` require Enterprise license and suitable kernel/BTF.

### Multi-region (license-gated)

`region-proxy`, regional WAL, slot migration: Network / Enterprise SKUs (`multi_region`, `slot_migration`).

---

## Engineering verification

| Surface | Target / gate |
| :--- | :--- |
| Tracker handler | p95 < 50 ms, p99 < 80 ms (load test abort if p99 > 80 ms for 30 s) |
| Filter deadline | `FILTER_TIMEOUT_MS` <= 100 ms (production) |
| Redis unified-filter Lua | p99 < 10 ms per shard |
| Hot path alloc | Zero heap allocs on `/track` (`make test-alloc-gate`) |
| Parser parity | nginx vs gnet `differential_count = 0` |

Details: `.cursor/rules/core.mdc`, `.cursor/rules/hot-path.mdc`.

---

## Bootstrap

```bash
bash scripts/install/appliance_bootstrap.sh              # dev; default ingest-only
bash scripts/install/appliance_bootstrap.sh --profile full
```

Production install entrypoint:

```bash
bash scripts/install/ad-event-processor-install.sh --accept-eula up
```

Manual equivalent: `make gen`, `make proto`, `cp .env.example .env`, `bash scripts/dev/stack/stack.sh full`. License file default: `var/license.jwt`. Issue JWT: `go run ./cmd/license-issue` ([deploy/vendor/KEYS.md](deploy/vendor/KEYS.md)).

### Compose profiles

| Profile | Services |
| :--- | :--- |
| `single-vps` / `full` | Tracker, processor, control, Postgres, Redis x4, ClickHouse |
| `minimal` | Reduced Redis shard count (~6 GB RAM) |
| `ingest-only` | Without ClickHouse |
| `analytics-ml` | Adds `fraud-scorer`, `ivt-detector` |

---

## Licensing

Monthly Ed25519 JWT on disk; offline verification; no license-server ping. Limits: peak RPS and host activations per [deploy/vendor/sku.yaml](deploy/vendor/sku.yaml). Schema values `max_active_campaigns: 0` and `max_events_per_month: 0` mean no license cap on those dimensions.

| SKU | USDT/mo | Peak RPS | Hosts | IVT | ML boost | OpenRTB | Multi-region | eBPF XDP |
| :--- | ---: | ---: | ---: | :---: | :---: | :---: | :---: | :---: |
| Starter | 129 | 10k | 1 | no | no | no | no | no |
| Pro | 329 | 25k | 1 | yes | no | no | no | no |
| Scale | 649 | 75k | 3 | yes | yes | yes | no | no |
| Network | 1,199 | 150k | 10 | yes | yes | yes | yes | no |
| Enterprise | 2,500+ | custom | 99 | yes | yes | yes | yes | yes |
| Pilot | 0 | 5k | 1 | no | no | no | no | no |

Apply JWT: Admin Settings, `license-apply`, or `POST /api/v1/license/apply` (no restart). Commercial detail: [deploy/vendor/SALES.md](deploy/vendor/SALES.md).

---

## Documentation

| Path | Content |
| :--- | :--- |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Hot/cold boundary, topology |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Bootstrap, codegen, tests |
| [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md) | Cost sync, CAPI, templates |
| [deploy/DEPLOY.md](deploy/DEPLOY.md) | Compose, nginx, Redis, CH, edge |
| [cmd/CMD.md](cmd/CMD.md) | Binaries and ports |
| [deploy/vendor/MARKETING.md](deploy/vendor/MARKETING.md) | Product specification (buyer-facing) |
| [deploy/vendor/PUBLIC_OFFER.md](deploy/vendor/PUBLIC_OFFER.md) | License terms template |
| `.cursor/rules/` | Engineering constraints |

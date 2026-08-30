# internal

Application logic. Flat domain packages — **not** layered `dto/usecase/repositories` trees. `controlplane` is the admin composition root and must shrink over time; domain code lives in sibling packages.

Cross-ref: `.cursor/rules/boundaries.mdc`, `.cursor/rules/hot-path.mdc`, `.cursor/rules/cold-path.mdc`, `.cursor/rules/modular-monolith.mdc`.

---

## Hot / cold boundary

This is the primary constraint when editing `internal/`.

```
HOT  — tracker request thread: ingest, filter, stream producers, rtb auction, openrtb parse
COLD — control, processor settlement, payment, reports, fraud ML, outbox workers
```

| Surface | Allowed sync I/O | Forbidden |
| :--- | :--- | :--- |
| `/track`, `/click`, `/tg/*` | 0–1 Redis `EVALSHA`; in-memory snapshots | Postgres, CH, outbox, ML infer, external HTTP |
| `/openrtb/bid` | In-process auction | Full `FilterEngine` chain |
| `/api/v1/*` | Postgres, Redis via outbox worker | — |

**HTTP 202 on `/track`** = accepted on hot path. **Not** PG/CH committed.

---

## Hot path packages

### `ingest/`

gnet HTTP ingress, handlers, filter engine bundles, landing/OpenRTB glue.

| Concern | Location / pattern |
| :--- | :--- |
| HTTP FSM | `httpingress/` — no `net/http` per request |
| Handlers | `handler*.go`, `track_core.go`, `click_redirect.go` |
| Filter engine | `filter_engine_bundle.go` (engine types — LOC drain ongoing) |
| Fraud signals | `filter_layer.go`, `filters.go`, `fraudTrackOutcome` |
| Parsers | DFA JSON, protobuf (vtproto), OpenRTB 3 FSM |

**Development rules:**

1. **Run cheap filters before Redis.** License → geo → schedule → fraud snapshot → `UnifiedFilter` last.
2. **Never block gnet epoll on Redis.** Tier A enqueues to `PinnedWorkerPool`; synchronous `FilterEngine.Check` (incl. `EVALSHA`) runs on Tier B pinned workers only (`hot-path.mdc` **Tracker thread model**). Copy `Accept`/`Origin` via `string()`; response via `cloneAsyncWriteBytes`.
3. **At most one `EVALSHA` per accepted event.** Zero when `LOCAL_QUOTA_MODE=live` full-skip eligible.
4. **Reserve stream admission before debit.** `TryReserve` → Lua → async `XADD`.
5. **No heap allocs in inner loops.** Pools, stack buffers; `unsafe.String` only over `OffloadHTTPPin`/arena on Tier B pinned worker.
6. **No `internal/fraud` import.** Read `ml:score:boost:*` from `SettingsWatcher` snapshot only.
7. **No `internal/controlplane` import.** Hot path must not see admin handlers.

**Banned APIs on request path:** `fmt.Sprintf`, `interface{}`, `json.Marshal`, `context.With*`, `defer` in inner loops, dynamic Prometheus labels per event.

**Test:**

```bash
make test-alloc-gate
bash scripts/ci/static/hot_path_static.sh
bash scripts/ci/static/escape_heap.sh
go test ./internal/ingest/ -run TestChaos_CrossHop_NginxGnet -count=1
go test ./internal/ingest/ -run 'TestStreamProducer|TestUnifiedFilter' -count=1
```

---

### `filter/`

Go-local filters and unified-filter helpers. Partial extraction from ingest mega-bundles.

Subpackages: `unified/`, `netintel/`.

**Rule:** Placement and fraud blacklist use in-process TTL cache (5 s) — `HEXISTS`/`SISMEMBER` on miss only.

---

### `stream/`

`StreamProducer`, `BrokerProducer`, admission, CH sink wiring.

| Invariant | Test |
| :--- | :--- |
| Post-debit reject → rollback Lua | `TestUnifiedFilter_RollbackDebit_LocalQuanta` |
| No dual XADD (Lua + Go) | `TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix` |
| Admission race | `TestStreamProducerAdmissionRaceWithoutReserve` |

Subpackages: `broker/`, `auditlog/`, etc.

---

### `track/`

Thin track/click helpers, IP rotation tables. Residual landing bodies may still live in `ingest/` during LOC drain.

---

### `rtb/` + `openrtb/`

| Package | Role |
| :--- | :--- |
| `rtb/` | In-process `RunAuction`, SoA catalog (`catalog_shard.go`), budget CAS |
| `openrtb/` | OpenRTB 2.x parse, in-place bid response write |

**Rules:**

- Catalog via `atomic.Pointer` — single load per auction.
- Scan cap p99 < 500 candidates (microbench scope).
- `/openrtb/bid` does **not** run full fraud filter chain.

---

### `domain/`

Shared hot types, sharding (`StaticSlotSharder`, `sharding_amd64.s`), budget invariants.

```
slot = CRC32C(campaign_id) & 1023
```

**Test:** `go test ./internal/domain/ -run Sharding -count=1`

---

### `edge/`

BPF map types, Redis blocklist sync into pinned maps, XDP helpers. Generated bpf2go under `bpf_edge_bpf*.go` — do not hand-edit.

Parallel L7 path lives in `deploy/nginx/lua/` (not this package): generational `blacklist_cache`, `edge_config` ASN mirror, `slot_map` / `node_weights` version-last writes. `perimeter_blacklist_cache.go` mirrors full-sync L7 semantics for Go unit tests only.

**Test:**

```bash
go test ./internal/edge/... -short -count=1
bash scripts/test/edge/lua_tests.sh compliance
```

## Control plane composition

### `controlplane/`

`Service`, bridges (`*_bridge.go` ≤ 200 lines), outbox shell, route registration.

| Rule | Detail |
| :--- | :--- |
| Bridges wire only | No SQL, no business rules in bridge files |
| No god growth | New admin logic → `internal/<domain>/handlers.go` first |
| Port methods ≤ 12 | Split `Host`/`Effects` before adding methods |

Key bridges: `campaign_bridge.go`, `fraudadmin_bridge.go`, `outbox_bridge.go`, `licensing_bridge.go`, …

### `control/`

`cmd/control` bootstrap — `deps.go` constructs `Service`, starts workers.

---

## Admin domain packages

Each owns HTTP handlers + store for one API surface. Registered from `controlplane/adminapi_wire.go`.

| Package | API prefix / role |
| :--- | :--- |
| `campaign` | `/api/v1/campaigns/*`, runtime, wizard, experiments |
| `flow` | Flows, landers, offers |
| `brand` | Brands, creatives |
| `supply` | sellers.json, ads.txt |
| `fraudadmin` | `/api/v1/fraud/*`, presets, ML ops |
| `billingadmin` | `/api/v1/billing/*` |
| `opsadmin` | `/api/v1/ops/*`, audit export |
| `platformadmin` | Customers, team, domains |
| `settingsadmin` | Blacklist TTL, emergency breaker |
| `licensingadmin` | License status/apply |
| `rtbadmin` | RTB deals, floors |
| `dashboardadmin` | Role dashboards |
| `reports` | Report catalog, CH queries |
| `reports/fraud` | Fraud/IVT/silent-reject reports |
| `reportjob` | Async export jobs |
| `smartalerts` | Alert rules + worker |
| `telegram` | Bots, Mini App, postbacks |
| `postback` | Postback config |
| `costsync` | Cost Sync credentials |
| `marginguard` | Margin policies |
| `platformsync` | Meta/Google platform campaigns |
| `integrationschema` | Custom schemas |
| `traffictemplates` | Generated templates |
| `migrationsource` | Keitaro/Binom import |
| `shardadmin` | Shard lease admin |
| `privacyadmin` | Consent admin |
| `governance` | Quotas, spend caps |
| `reconciliation` | Recon windows, spend sync |
| `outbox` | Poll loop, event dispatch |
| `automation` | Automation hooks |
| `nodeadmin` | Node scoring |

**Cold path rules for all handlers:**

- Read body via `pkg/coldpath.ReadLimitedBody` (64 KiB default).
- No `KEYS`, `FLUSHALL` in production Go.
- No O(N) DB loops — batch with `ANY($1::uuid[])`.
- Admin mutation + outbox row in **one transaction**.

**Test:** `go test ./internal/<pkg>/ -short -count=1`, `bash scripts/ci/static/cold_path_static.sh`.

---

## Data and infrastructure packages

| Package | Role |
| :--- | :--- |
| `clickhouse` | Store, migrations — canonical DDL sync with `deploy/clickhouse/init.sql` |
| `database` | Postgres pool |
| `payment` | Webhooks, payment outbox |
| `ledger` | `balance_ledger` |
| `identity` | Auth |
| `notify` | Slack/Telegram |
| `broker` | In-process broker server |
| `regionproxy` | Multi-region logic |
| `config` | Env parsing — single source for all binaries |
| `licensing` | JWT, HWID, MCK, entitlements |
| `fraud` | ML scoring (**cold only**) |
| `openapi` | OpenAPI document assembly |
| `doctor` | Health probes |
| `testutil` | testcontainers helpers |

---

## Package ceilings

| Ceiling | Limit |
| :--- | :--- |
| Prod `.go` files per package root | 40 (excl. `db/`, `*_test.go`) |
| LOC per package root | ~8000 |
| Single file | ~500 lines |
| Bridge file | 200 lines |
| Port (`Host`, `Effects`) | 12 methods |

Enforced by `scripts/ci/static/package_size_gate.sh`.

---

## Import matrix (hard)

```
cmd/*           → internal/*, pkg/*
internal/ingest → domain, pkg/*, rtb, openrtb — NOT controlplane, NOT fraud scoring
pkg/*           → stdlib, pkg/* only — NOT internal/*
controlplane    → domain packages, bridges only
```

---

## `doc.go` policy

Every production package root needs:

```go
// Package foo does …
package foo
```

Edit in the same PR when adding a package. No route lists in `doc.go` (breaks block parsers).

---

## Verification tiers

| Tier | When | Command |
| :--- | :--- | :--- |
| Fast | Every PR | `bash scripts/ci/pr_fast.sh` |
| Integration | Redis/PG/CH behavior | `make test-integration` |
| Fault | Budget, outbox, broker | `make test-fault` |
| Alloc | Hot path edits | `make test-alloc-gate` |

Never claim hot-path wiring from mocks alone — cite tier (`boundaries.mdc`).

---

## Common mistakes

1. **Importing `controlplane` from `ingest`** — boundary violation.
2. **Second Redis call per event** — sequential RTT on hot path.
3. **Sync `XADD` in Lua** — dual-write bug; use deferred stream producer.
4. **Handler SQL in `controlplane`** — move to domain `store.go` / sqlc.
5. **God bridge > 200 lines** — split by subdomain bridge file.
6. **Per-request PG campaign load** — use registry snapshot.
7. **ML on tracker** — sidecar only; snapshot read async.

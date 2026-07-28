# Technology milestone backlog

Open engineering work not yet implemented. Closed items: [DEVELOPMENT.md — Completed roadmap](./DEVELOPMENT.md#completed-roadmap).

**Out of scope:** buyer/finance HTMX dashboards (GAP-PROD-01), Grafana-style queue monitoring UI (GAP-OPS-04). HTMX in management is for errors and cold-path admin flows only, not product dashboards.

**Related:** [ARCHITECTURE.md](./ARCHITECTURE.md), [MULTI_REGION.md](./MULTI_REGION.md), [DEVELOPMENT.md](./DEVELOPMENT.md), `.cursor/rules/code-style.mdc`, `.cursor/rules/hot-path.mdc`, `.cursor/rules/chaos.mdc`, `.cursor/rules/rtb.mdc`.

---

## Document map

| Section | Contents |
| :--- | :--- |
| [Global standards](#global-standards) | SLA tiers, code style matrix, testing pyramid, PR checklist (all milestones) |
| [Milestones](#milestones) | Per-gap problem, deliverables, DoD, SLA, patterns, style, tests |
| [Deferred (UI)](#deferred-ui) | Intentionally excluded work |
| [Closed reference](#closed-reference) | Recently shipped items |

### Recommended order

```text
(deferred UI only: GAP-PROD-01, GAP-OPS-04)
```

---

## Global standards

Apply to **every** milestone unless a gap overrides a row explicitly.

### SLA tiers

| Tier | Scope | Key budgets |
| :--- | :--- | :--- |
| **Hot** | tracker `/track`, `FilterEngine`, RTB `RunAuction` | p95 < 50 ms, p99 < 80 ms, max 100 ms; **0 allocs/op** on touched paths |
| **Cold ingest** | region-proxy, broker produce | p99 < 20 ms ACK after WAL fsync batch |
| **Cold control** | management workers, payment webhooks, CHQuery | worker tick < 500 ms; PG CAS p99 < 10 ms |
| **Global settle** | proxy uplink, payment settlement | p99 < 2 s end-to-end; `AssertBudgetInvariant` +/-1 micro-unit |
| **Edge** | nginx Lua | pick / policy overhead < 1 us/request where on request path |

**Regression rule:** no milestone merge may raise tracker `ad_http_request_duration_seconds` p99 above 80 ms in perf-gate smoke.

### Code style matrix

| Layer | Packages | Rules |
| :--- | :--- | :--- |
| **Hot touch** | `internal/ingestion`, `internal/rtb` | `.cursor/rules/hot-path.mdc`, `.cursor/rules/espx.mdc`: no `defer`/closures in loops, no `interface{}` on request path, BCE before indexed access, pre-bound metrics |
| **Cold service** | `internal/management`, `internal/payment`, `internal/adminapi` | Flat package R1; `fmt.Errorf("verb noun key=%s: %w", id, err)`; `withPgHigh` / `withPgLow` for workers |
| **Cold `pkg/`** | `pkg/*` | No `internal/ingestion` imports; table-driven tests |
| **Edge Lua** | `deploy/nginx/lua/` | ASCII comments; no dynamic labels; match `StaticSlotSharder` routing |
| **Migrations** | `internal/*/migrations/` | goose up/down tested locally |
| **Metrics** | `internal/metrics/collectors.go` | Pre-bound labels at init; register in same PR |

### Testing pyramid

| Layer | Command / pattern | Gate |
| :--- | :--- | :--- |
| Unit | `go test ./<pkg>/... -race` | Table-driven; no sqlmock on budget/dedup/Lua paths |
| Integration | `*_test.go` + testcontainers | Real PG/Redis/CH where I/O involved |
| Chaos | `*_chaos_test.go` | `chaos_proof fault=<name> ...`; >= 20 goroutines on balance paths when applicable |
| E2E | `tests/e2e/*` | Compose stack for cross-service gaps |
| Perf | `go test -benchmem` | Hot touch: 0 allocs/op unchanged; run `make test-alloc-gate` |
| PR | `make lint`, `go test ./... -short` | `.cursor/rules/code-style.mdc` R10 |

### PR checklist (every milestone)

- [ ] SLA row for this gap verified (bench, integration timing, or load smoke)
- [ ] Definition of done 100% or explicit defer with gap ID
- [ ] `make lint` green; `make test-alloc-gate` if hot path touched
- [ ] New metrics in `collectors.go` when observability is in deliverables
- [ ] Migration goose down tested (if SQL)
- [ ] No new ignored `json.Unmarshal` / `uuid.Parse` on cold path (R8.6)

---

## Milestones

---

### GAP-RTB-11 — Pre-auction gates (daypart + frequency-cap)

| | |
| :--- | :--- |
| **Area** | RTB / hot path |
| **Priority** | P1 |
| **SLA tier** | Hot |

#### Problem

Daypart and frequency-cap run inside unified Redis filter or after candidate ranking. Invalid traffic still enters `RunAuction`, wasting scan budget and polluting metrics.

#### Deliverables

1. Daypart bitmask on campaign/deal metadata; reject before `RunAuction`.
2. Frequency-cap pre-check before auction scan (snapshot or read-only Redis).
3. `NoBidReason` mapping for each reject; metrics pre-bound at init.

#### Definition of done

- [x] Daypart reject occurs before first candidate scan; `BenchmarkRunAuction` and `/track` alloc gate unchanged on happy path
- [x] Frequency-cap reject does not call unified-filter Lua for the cap check alone
- [x] Every new reject maps to `NoBidReason` and `filterRejectKind` / metric label
- [x] Table-driven tests cover: in-window, out-of-window, cap exceeded, cap OK, missing metadata fail-open policy documented
- [x] `make test-alloc-gate` green; perf-gate smoke p99 < 80 ms
- [x] `.cursor/rules/rtb.mdc` updated if auction order changes

**Shipped:** pre-auction daypart bitmask + flat-hash fcap snapshot in `rankCandidates`; ~37 ns/op `BenchmarkAuction` (0 allocs/op).

---

### GAP-RTB-12 — Platform ops (remaining scope)

| | |
| :--- | :--- |
| **Area** | RTB / control plane |
| **Priority** | P2 |
| **SLA tier** | Cold control + global settle |

**Done:** Cross-region spend sync (`GlobalSpendReconciler`, region-proxy producer). CTV gtax settlement, admin dry-run, A/B cohort fanout. See Completed roadmap.

Ship as **three sub-milestones** (may be separate PRs). Each sub-milestone must satisfy global PR checklist.

**Shipped (12a–12c):** `ApplyCTVSettlement` gRPC + `ctv_gtax_settlements`; `?dry_run=1` on pause/resume/blacklist; `experiment_cohorts` + registry cohort snapshot + `UPDATE_COHORT_SNAPSHOT` outbox.

---

#### GAP-RTB-12a — CTV gtax

##### Problem

CTV inventory lacks tax calculation hooks in settlement and billing.

##### Deliverables

1. Tax profile extension for CTV line items in `billing` schema.
2. Settlement hook applies gtax before `balance_ledger` commit.
3. Idempotent tax rows keyed by event/settlement ID.

##### Definition of done

- [x] goose migration up/down; sqlc regenerated
- [x] `AssertBudgetInvariant` passes in integration test with tax debit
- [x] Outbox or settlement path documented in handler godoc
- [x] No hot-path import from `internal/billing`

##### SLA

| Metric | Target |
| :--- | :--- |
| Settlement handler p99 | < 500 ms (cold) |
| Ledger correctness | +/-1 micro-unit after replay x3 |

##### Patterns

- `internal/payment/provider_stripe.go` — provider webhook idempotency
- `internal/management/recon_*.go` — ledger reconciliation
- `internal/billing/` — invoice generation

##### Code style

- Flat `internal/billing` / `internal/payment`; DTOs with `json` tags at HTTP boundary only
- Money: micro-units `int64`; `billing_money.go` conventions in management

##### Testing

| Layer | Requirement |
| :--- | :--- |
| Integration | testcontainers PG; tax row idempotency |
| Chaos | `chaos_proof fault=gtax_settlement_replay` with `proposal_rows=1` |

##### Touch

`internal/billing/`, `internal/payment/`, `internal/management/` settlement handlers.

---

#### GAP-RTB-12b — Admin simulation (dry-run)

##### Problem

Admin mutations (pause, budget change, blacklist) have no dry-run mode; operators cannot preview side effects.

##### Deliverables

1. `?dry_run=1` or `X-Dry-Run: 1` on selected `/api/v1` mutation routes.
2. Transaction rolled back or shadow validation only; no outbox enqueue.
3. Response DTO lists would-change fields.

##### Definition of done

- [x] Documented route list in handler godoc
- [x] Integration test: dry-run produces zero outbox rows and zero Redis writes
- [x] Live run unchanged when header absent
- [x] RBAC unchanged; audit log records dry-run flag

##### SLA

| Metric | Target |
| :--- | :--- |
| Dry-run handler p99 | < 200 ms (same order as live mutation) |

##### Patterns

- `internal/management/api_*.go` — handler registration
- `internal/management/outbox_handlers.go` — side effect boundary
- `pkg/coldpath` — pagination/JSON helpers

##### Code style

- Handler decodes request DTO; service accepts explicit `dryRun bool`
- Errors: `writeServiceError` / `mapServiceError` only at HTTP boundary

##### Testing

| Layer | Requirement |
| :--- | :--- |
| Integration | PG testcontainer; before/after row counts |
| Unit | Table-driven validation errors for dry-run payload |

##### Touch

`internal/management/api_*.go`, `handler_*.go`, selected `service_*.go`.

---

#### GAP-RTB-12c — A/B cohorts

##### Problem

No cohort assignment or global sync for experiment flags across regions.

##### Deliverables

1. Cohort assignment store (PG) + stable hash by `user_id` / device id.
2. Config fanout via existing outbox + `RegionOutboxRelay` for multi-region.
3. Tracker reads cohort snapshot from registry (atomic pointer swap).

##### Definition of done

- [x] Cohort config visible on tracker registry reload without restart
- [x] Multi-region: `outbox_region_delivery` path tested with `MultiRegionCell()`
- [x] No per-request PG read on `/track`
- [x] Migration + sqlc queries

##### SLA

| Metric | Target |
| :--- | :--- |
| Registry reload | < 100 ms cold; 0 allocs on read path |
| Fanout lag | `ad_region_outbox_delivery_lag_seconds` < 5 s p99 in smoke |

##### Patterns

- `internal/ingestion/registry.go` — snapshot publish
- `internal/management/region_outbox_relay.go` — regional apply
- `internal/campaignmodel` — experiment fields without tags

##### Code style

- Hot read: `atomic.Pointer` snapshot; no `sync.Map` on `/track`
- Cold write: transactional outbox in one PG transaction

##### Testing

| Layer | Requirement |
| :--- | :--- |
| Integration | Fanout to Redis shard; registry contains cohort |
| Chaos | Optional CH-MR-style proof if cross-region |

##### Touch

`internal/management/`, `internal/ingestion/registry.go`, migrations.

---

### GAP-OPS-03 — ClickHouse query governance

| | |
| :--- | :--- |
| **Area** | Operations / data layer |
| **Priority** | P3 |
| **SLA tier** | Cold control |

#### Problem

Direct ClickHouse driver calls bypass `internal/database.CHQuery`. Timeouts, concurrency limits, and query audit are inconsistent.

#### Deliverables

1. Inventory and eliminate direct `conn.Query` outside `CHQuery` in `internal/`.
2. Extend `CHQuery` with timeout, max concurrency, and structured slog on slow queries.
3. CI or `scripts/ci/` guard for new raw CH access.

#### Definition of done

- [x] Zero production `Query`/`Exec` on CH conn in `internal/` outside `CHQuery` wrapper (allowlist file if exceptions exist)
- [x] All admin, IVT, forecast paths use `CHQuery`
- [x] `CHQuery` exposes metrics: `ad_ch_query_duration_seconds`, `ad_ch_query_rejected_total`
- [x] Document timeout env vars in `.env.example`
- [x] `scripts/ci/check_ch_direct.sh` guard for new raw CH access

#### SLA

| Metric | Target |
| :--- | :--- |
| CHQuery wait p99 | < 30 s (admin); < 1500 ms (forecast path, existing budget) |
| Rejected (gate full) | Fail fast < 10 ms; HTTP 503 sanitized body |

#### Patterns

- `internal/database/ch_query.go` (or equivalent `CHQuery`)
- `internal/management/service_forecast.go` — `forecastCHQueryTimeout`
- `internal/ivtdetector/*_rule.go` — inject `*database.CHQuery`

#### Code style

- Cold path: `context.WithTimeout` per query; `%w` error wrap
- No CH queries from `internal/ingestion` hot packages

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Unit | Gate semaphore: max concurrency enforced |
| Integration | testcontainers ClickHouse; slow query logs at threshold |
| Lint | `scripts/ci/check_ch_direct.sh` or golangci custom linter |

#### Touch

`internal/database/`, `internal/management/`, `internal/ivtdetector/`, `internal/adminapi/`.

---

### GAP-PROD-03 — OpenAPI specification

| | |
| :--- | :--- |
| **Area** | API contract |
| **Priority** | P3 |
| **SLA tier** | N/A (documentation) |

#### Problem

`/api/v1` JSON surfaces are documented only in godoc. No OpenAPI 3 machine-readable contract.

#### Deliverables

1. `docs/openapi/openapi.yaml` (or generated) covering management + adminapi JSON routes.
2. CI drift check (spec vs routes or contract tests).
3. Link from `DEVELOPMENT.md`.

#### Definition of done

- [x] Every documented path has request/response schema matching actual JSON (`snake_case`)
- [x] CI fails on unlisted `/api/v1` route or schema mismatch
- [x] No HTMX `/admin/*` HTML routes in spec
- [x] Security schemes: `X-Admin-API-Key`, session cookie documented

#### SLA

| Metric | Target |
| :--- | :--- |
| CI spec check | < 2 min |
| Spec coverage | 100% of `/api/v1` JSON (excluding 501 stubs) |

#### Patterns

- Handler godoc today on `api_*.go`, `adminapi/*_handlers.go`
- `pkg/coldpath` response envelopes

#### Code style

- OpenAPI file: YAML; no business logic in spec
- Optional codegen: do not duplicate DTOs; generate from annotations only if single source of truth is maintained

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Contract | `tests/contract/openapi_test.go` — sample requests match schema |
| CI | `make openapi-lint` or equivalent in Makefile |

#### Touch

`docs/openapi/`, `internal/management/api_register.go`, `internal/adminapi/`, `.github/workflows/`.

---

### GAP-PAY-01 — Cryptocurrency payment gateway

| | |
| :--- | :--- |
| **Area** | Payments |
| **Priority** | P5 |
| **SLA tier** | Cold control + global settle |

#### Problem

`cmd/payment` supports Stripe only. Enterprise customers need on-chain or custodial crypto settlement.

#### Deliverables

1. `PaymentProvider` interface; Stripe and crypto implementations.
2. Webhook HMAC verification; idempotency in `payment` schema migrations.
3. Outbox -> management settlement unchanged; `AssertBudgetInvariant` on credit path.
4. Env-gated compose profile for sandbox provider.

#### Definition of done

- [x] `CREATE SCHEMA payment` migration for crypto-specific tables if needed
- [x] Webhook replay x3 creates one ledger row
- [x] Secrets only via env `Secret` type in config
- [x] Service boundary: no payment imports in `internal/ingestion`
- [x] Chaos proof: `chaos_proof fault=crypto_webhook_replay proposal_rows=1`

#### SLA

| Metric | Target |
| :--- | :--- |
| Webhook handler p99 | < 500 ms |
| Settlement end-to-end | p99 < 2 s to `balance_ledger` visible |

#### Patterns

- `internal/payment/provider_stripe.go`
- `internal/payment/webhook_*.go`
- `internal/management/outbox_handlers.go` — settlement consumer

#### Code style

- Flat `internal/payment`; gRPC in `pb/`
- PCI: no card/crypto PII in logs; structured `slog` with payment intent ID only

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Integration | testcontainers PG; webhook signature vectors |
| Chaos | Duplicate webhook delivery; budget invariant |
| Unit | Provider interface table tests with recorded fixtures |

#### Touch

`internal/payment/`, `cmd/payment/`, `api/payment.proto`, `deploy/` compose profile.

---

### GAP-CMP-01 — Edge tarpit + compliance matrix

| | |
| :--- | :--- |
| **Area** | Compliance / edge |
| **Priority** | P4 |
| **SLA tier** | Edge |

#### Problem

`edge-tarpit.lua` is optional and partial. Compliance control mapping is not maintained.

#### Deliverables

1. Production profile defaults for tarpit thresholds; metrics `espx_edge_tarpit_*` wired to Prometheus.
2. `docs/COMPLIANCE_MATRIX.md` — control ID, implementation file, test proof (no HTMX UI).
3. Integration test or chaos script with `EDGE_TARPIT_ENABLED=1`.

#### Definition of done

- [x] Tarpit disabled by default in dev; enabled documented for production edge profile
- [x] Compliance matrix lists every control in `.cursor/rules/compliance.mdc` with file + test reference
- [x] No tracker Go changes required for default tarpit path
- [x] Lua unit test in `deploy/nginx/lua/tests/`

#### SLA

| Metric | Target |
| :--- | :--- |
| Tarpit delay path | Adds configured delay only; normal requests unaffected |
| Lua overhead | < 1 us when tarpit off |

#### Patterns

- `deploy/nginx/lua/edge-tarpit.lua`, `edge-metrics.lua`
- `deploy/nginx/lua/tests/node_weights_test.lua` — luarocks test style

#### Code style

- Lua: ASCII comments; English identifiers
- Metrics via `ngx.shared.DICT`; no per-IP labels in Prometheus

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Lua | `*.lua` tests under `deploy/nginx/lua/tests/` |
| Integration | curl against edge with oversized header; expect delay metric increment |
| Chaos | Optional: `chaos_proof fault=edge_tarpit_triggered` |

#### Touch

`deploy/nginx/lua/`, `docs/COMPLIANCE_MATRIX.md`, `deploy/monitoring/`, `.cursor/rules/compliance.mdc`.

---

### GAP-ENG-02 — Broker and region-proxy in local compose

| | |
| :--- | :--- |
| **Area** | Engineering / dev UX |
| **Priority** | P3 |
| **SLA tier** | Cold ingest (smoke) |

#### Problem

`cmd/broker` lives only in `deploy/broker/docker-compose.yaml`. `region-proxy` needs `deploy/multi-region/docker-compose.yaml` overlay. One-command local lab is fragmented.

#### Deliverables

1. Compose profile `multi-region` (or `tools`) in root stack: broker optional, `region-proxy` service.
2. `scripts/local-dev/dev_stack.sh` target: `dev_stack.sh multi-region up`.
3. `DEVELOPMENT.md` section: env matrix for global vs regional processor.

#### Definition of done

- [x] `docker compose --profile multi-region up -d region-proxy` succeeds on clean clone (documented build step)
- [x] `tests/e2e/region_proxy_*` runnable against compose profile (documented in DEVELOPMENT)
- [x] `.env.example` documents `REGION_PROXY_ADDR`, `GLOBAL_INGEST_URL`
- [x] No change to default profile services (trackers start unchanged)

#### SLA

| Metric | Target |
| :--- | :--- |
| region-proxy `/health` | < 1 ms |
| region-proxy `/ready` | < 10 ms |
| ProduceBatch smoke | p99 < 20 ms post-fsync (single node) |

#### Patterns

- `deploy/multi-region/docker-compose.yaml` (existing overlay)
- `deploy/broker/docker-compose.yaml` — HA lab reference
- `Dockerfile` — `/broker`, `/region-proxy` binaries

#### Code style

- YAML only in deliverable; no Go unless healthcheck script needed
- Document ports: 9093 proxy, broker 9093 conflict noted in docs

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Smoke | `scripts/local-dev/dev_stack.sh status` includes region-proxy when profile on |
| E2E | `go test ./tests/e2e/... -run RegionProxy` against compose |

#### Touch

`docker-compose.yaml`, `deploy/multi-region/`, `scripts/local-dev/`, `docs/DEVELOPMENT.md`, `.env.example`.

---

### GAP-ENG-03 — Vendor telemetry

| | |
| :--- | :--- |
| **Area** | Engineering / observability |
| **Priority** | P4 |
| **SLA tier** | Cold control |

#### Problem

Vendor health (MaxMind, Stripe, SMTP, Telegram) is opt-in and fragmented.

#### Deliverables

1. `VendorProbe` interface + registry in `pkg/` or `internal/config/`.
2. Cold worker ticks probes; exports `ad_vendor_probe_success`, `ad_vendor_probe_latency_seconds`.
3. `VENDOR_TELEMETRY_ENABLED` default on in production profile.

#### Definition of done

- [x] At least MaxMind, Stripe, notifier providers instrumented
- [x] Probe failure does not block hot path (management worker only)
- [x] Metrics registered in `collectors.go` with bounded label set (`vendor` enum)
- [x] Document env in `.env.example`

#### SLA

| Metric | Target |
| :--- | :--- |
| Probe tick period | 60 s default; single probe < 5 s timeout |
| Worker tick | < 500 ms total per cycle |

#### Patterns

- `internal/management/service.go` — `startWorker` pattern
- `internal/ingestion/geoip_updater.go` — external API poll
- `internal/metrics/collectors.go`

#### Code style

- Cold path only; no imports from `internal/ingestion` in probe package
- Errors: log once per interval; counter `ad_vendor_probe_errors_total`

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Unit | Mock HTTP server; probe success/fail transitions |
| Integration | Optional: skip if no network in CI |

#### Touch

`internal/management/` or new `internal/telemetry/`, `internal/metrics/collectors.go`, `cmd/management/main.go`.

---

### GAP-DB-03 — Weighted processor gates (multi-instance)

| | |
| :--- | :--- |
| **Area** | Database / processor |
| **Priority** | P3 |
| **SLA tier** | Cold control |

#### Problem

`ProcessorPgGate` / `ProcessorChGate` limit concurrency inside one process. Multiple processor replicas do not share weighted scheduling by lag or load.

#### Deliverables

1. Per-instance weight published to Redis or UDP epoch (reuse node-weight pattern).
2. Stream consumer assigns shard read cadence or batch size by weight.
3. Floor weight prevents starvation; drain on hard signals (PG gate wait p99).

#### Definition of done

- [x] Two processor replicas in compose: slow one receives less work within 3 epochs
- [x] `AssertBudgetInvariant` holds under weighted drain (weight affects read cadence only; store/debit path unchanged)
- [x] No hot-path tracker changes
- [x] Metrics: `ad_processor_weight`, `ad_processor_stream_lag_seconds` per instance

#### SLA

| Metric | Target |
| :--- | :--- |
| Weight update epoch | 1-10 s aligned with `UDP_SYNC_INTERVAL_MS` |
| Lag fairness | p99 lag ratio between replicas < 3:1 after 5 min steady load |
| PG gate wait p99 | < 50 ms per process (unchanged) |

#### Patterns

- `internal/ingestion/processor_pg_gate.go`, `processor_ch_gate.go`
- `internal/management/service_node_scorer.go` — weight publish
- `deploy/nginx/lua/edge-node-weights.lua` — weighted pick reference
- `internal/ingestion/stream_consumer.go`

#### Code style

- Weights in `atomic` or snapshot; no per-shard `sync.Map` on consume loop
- Config: `PROCESSOR_WEIGHT_FLOOR`, `PROCESSOR_WEIGHT_CEIL` in `env.go`

#### Testing

| Layer | Requirement |
| :--- | :--- |
| Integration | Two consumers, artificial slow PG on one instance |
| Chaos | `chaos_proof fault=processor_weight_drain` with `lag_ratio` key |
| Bench | No regression on single-instance consume path |

#### Touch

`internal/ingestion/processor_*_gate.go`, `stream_consumer.go`, `cmd/processor/main.go`, optional `internal/management/` UDP extension.

**Note:** Distinct from GAP-DB-01/02 (`pkg/iogate`, completed).

---

## Deferred (UI)

| ID | Reason deferred |
| :--- | :--- |
| GAP-PROD-01 | Buyer and finance HTMX dashboards; `internal/adminapi/` returns 501 |
| GAP-OPS-04 | Unified DLQ/spool dashboard; use Prometheus/Grafana until product UI is in scope |

---

## Closed reference

| ID | Summary |
| :--- | :--- |
| GAP-GEO-01 | Game days M7.4 — `scripts/chaos-drills/mr_game_day.sh` |
| GAP-GEO-02 | Postgres failover — `pg_failover.go` |
| GAP-MR-01/03 | Operation leases, quorum book |
| GAP-MR-02 | Global vs regional scorer (H3) |
| GAP-RTB-10 | VAST 4.2 + creative auction |
| GAP-RTB-11 | Pre-auction gates (daypart + frequency-cap) |
| GAP-RTB-12 (slice) | Cross-region spend sync + region-proxy producer |
| GAP-RTB-12a | CTV gtax settlement (`ApplyCTVSettlement`, billing CTV tax profile) |
| GAP-RTB-12b | Admin dry-run (`?dry_run=1` / `X-Dry-Run: 1` on pause/resume/blacklist) |
| GAP-RTB-12c | A/B cohorts (PG store, outbox fanout, registry snapshot) |
| GAP-DATA-01 | PII hash before CH |
| GAP-DB-01/02 | Disk gate, `iogate`, WAL |
| GAP-ENG-01 | Management domain registry |
| GAP-OPS-03 | ClickHouse query governance (`CHQuery`, CI allowlist) |
| GAP-ENG-02 | Broker and region-proxy in local compose (`multi-region` profile) |
| GAP-DB-03 | Weighted processor gates (multi-instance stream cadence) |
| GAP-CMP-01 | Edge tarpit + compliance matrix |
| GAP-ENG-03 | Vendor telemetry probes (MaxMind, Stripe, notifier) |
| GAP-PAY-01 | Cryptocurrency payment gateway (Stripe + crypto providers, hold worker, webhook HMAC) |
| GAP-PROD-03 | OpenAPI 3 spec for `/api/v1` JSON + CI drift gate |

Full evidence table: [DEVELOPMENT.md — Completed roadmap](./DEVELOPMENT.md#completed-roadmap).

# eSPX engineering milestone

Единый бэклог: все открытые GAP'ы, невыполненные задачи и критерии приёмки. Закрытые пункты: [docs/DEVELOPMENT.md — Completed roadmap](docs/DEVELOPMENT.md#completed-roadmap).

**Порядок:** P01 = самый сложный → P49 = самый простой. Стабильные ID (`GAP-*`) не меняются; **P** — только приоритет исполнения.

**Связанные документы:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md), [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md), `.cursor/rules/*.mdc`, `.cursor/GAP_SPECS.md` (детальные SQL-планы).

**Отложено (UI):** GAP-PROD-01, GAP-OPS-04 — вне этого списка до GAP-PROD-02.

---

## Содержание

| Раздел | Содержание |
| :--- | :--- |
| [Глобальный контракт](#глобальный-контракт) | SLA, syscall, zero-alloc, кэш, потоки, disk I/O, стиль, тесты, запрет комментариев |
| [Сводная таблица](#сводная-таблица-p01p49) | Все 49 задач |
| [Спецификации](#спецификации) | DoD по каждому P |
| [Отложено / отменено](#отложено-и-отменено) | Deferred, cancelled |
| [Закрыто](#закрыто-справочник) | Shipped reference |

---

## Глобальный контракт

Применяется к **каждому** P, если в задаче не указано иное.

### SLA tiers

| Tier | Scope | p95 | p99 | p999 / ceiling | Прочее |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Hot** | `/track`, `FilterEngine`, RTB `RunAuction` | < 50 ms | < 80 ms | max 100 ms | 0 allocs/op на затронутых путях |
| **Hot micro** | Geo filter (sampled), pacing/segment read, `GetShard` | — | geo < 10 µs; auction < 15 µs | — | Redis Lua p99 < 10 ms/shard |
| **Cold ingest** | region-proxy, broker produce | — | < 20 ms ACK | — | после WAL fsync batch |
| **Cold control** | management workers, webhooks, `CHQuery` | — | worker tick < 500 ms | — | PG CAS p99 < 10 ms |
| **Global settle** | settlement, ledger, payment credit | — | p99 < 2 s | — | `AssertBudgetInvariant` ±1 micro-unit |
| **Edge** | nginx Lua | — | — | — | overhead < 1 µs когда feature off |
| **N/A** | docs, CI, git hygiene | — | — | — | CI job < 8 min |

**Регрессия:** merge не поднимает `ad_http_request_duration_seconds` p99 выше 80 ms в perf-gate smoke. Production: `FILTER_TIMEOUT_MS` ≤ 100.

### Syscall и I/O overhead

| Layer | Правило |
| :--- | :--- |
| Hot path | gnet/epoll; без blocking read/write в request loop; Redis timeout = monotonic deadline |
| Cold HTTP | `context.WithTimeout` на внешние вызовы; connection pool, не per-request dial |
| Disk append | `pkg/iogate`: `appendSem` (ёмкость N) + `fsyncSem` (1) — запись и fsync разделены; hot path не ждёт disk |
| CH / PG | `CHQuery` gate (concurrency + timeout); `withPgHigh` / `withPgLow` / `PgPoolSettle` — pool tiering |
| Edge | `ngx.shared.DICT`; без per-IP Prometheus labels |

### Zero-allocation (hot path)

Запрещено на `/track`: `defer` в циклах, closures, `interface{}` boxing, `sync.Map`, `fmt.Sprintf`, string `+`, dynamic metric labels, `context.WithValue`, `json.Marshal` на reject.

Обязательно: vtproto pools, byte-slice parse, BCE (`if len(buf) <= i { return ErrMalformed }`), pre-bound metrics, `filterRejectSpecs` table, `atomic.Pointer` snapshots.

Проверка: `make test-alloc-gate`; `go test -benchmem` на затронутых bench; `go build -gcflags="-m"` — нет `CALL runtime.panicIndex` без early abort.

### Data-oriented design vs idiomatic Go

| Layer | Подход |
| :--- | :--- |
| Hot (`internal/ingestion`, `internal/rtb`) | SoA, flat slices, bitmasks, presort buckets; typed enums (`NoBidReason`, `filterRejectKind`); reject без `error` в ядре |
| Cold (`internal/management`, `payment`, `auth`) | Идиоматичный Go: sentinels, `%w`, `errors.Is`; handler → service → store; flat package R1 |
| Shared | `internal/campaignmodel` — без тегов; mapping только на I/O границе |

Запрещено: entity/usecase/repository слои, nested packages, дубли struct на одну таблицу, reflection mappers.

### Кэш, false sharing, потоки

- Contended atomics: `_ [56]byte` или `cpu.CacheLinePad` между полями разных cores.
- `PinnedWorkerPool`: dispatch по `campaign_id` hash; queue padding 64 byte.
- Config/registry: `atomic.Pointer` swap; один load вне tight loop.
- CPU-bound на hot path: запрещён; cold workers — отдельные goroutine с tick/batch.
- Settlement: pinned lanes (`crc32(campaign_id) % N`), single consumer per lane — порядок per campaign.
- `GOMAXPROCS` / `PinnedWorkerPool` size из env или `runtime.NumCPU()`; `GOMEMLIMIT` 90% RAM при autotune.

### Disk I/O (семафоры)

Паттерн `pkg/iogate/disk_gate.go`:

- `appendSem chan struct{}` — лимит параллельных append.
- `fsyncSem chan struct{}` — один fsync в момент времени.
- Producer enqueue → append under `appendSem` → async fsync under `fsyncSem`.
- Broker WAL, CH spool, log evacuation — только через gate; метрика `ad_disk_gate_wait_seconds`.

### Паттерны (cold path)

Transactional outbox; `FOR UPDATE SKIP LOCKED`; idempotency keys (`ON CONFLICT DO NOTHING`); pool tiering; pre-bound Prometheus labels; `fmt.Errorf("verb noun key=%s: %w", id, err)`.

### Обработка ошибок

- Hot: sentinels + `classifyFilterErr` на границе; infra → counter, без log storm.
- Cold: `mapServiceError` + `writeServiceError`; `pgx.ErrNoRows` только для not found.
- Workers: return `error` → retry; permanent → mark + alert.
- Запрещено: `_ = json.Unmarshal` без ветки; mask all DB errors as 404.

### Запрет комментариев в коде

- **Никаких** `//`, `/* */`, godoc в `internal/`, `cmd/`, `pkg/`, hand-written SQL.
- Исключения: `//go:` (`embed`, `noinline`, …) и `//nolint:` с краткой причиной.
- В коде **запрещены** ссылки на `GAP-*`, `P##`, backlog, scenario catalogs.
- CI: `scripts/ci/check_comments.sh`; цель: `STRICT_NO_COMMENTS=1` (P39).

### Терминология тестов

Использовать: **fault**, **resilience**, **edge**. Не использовать: chaos, robust, game-day (в именах тестов и скриптов).

Proof line в output: `fault_proof fault=<name> key=value`.

### Пирамида тестов

| Layer | Gate |
| :--- | :--- |
| Unit | `go test ./<pkg>/... -race`; table-driven |
| Integration | testcontainers PG/Redis/CH |
| Fault | `*_fault_test.go`; `fault_proof`; ≥20 goroutines на ledger paths |
| E2E | `tests/e2e/*` |
| Perf | `make test-alloc-gate` если hot path |
| SQL | `EXPLAIN (ANALYZE, BUFFERS)` в `TestExplainAudit_*` |
| Contract | `tests/contract/openapi_test.go` |

### PR checklist (каждый P)

- [ ] DoD 100% или явный defer в PR body
- [ ] SLA проверен (bench / integration timing / load smoke)
- [ ] `make lint`; `make check-local`
- [ ] `make test-alloc-gate` если hot path
- [ ] Новые метрики в `internal/metrics/collectors.go`
- [ ] Goose up/down если schema
- [ ] Runbook если DoD требует docs
- [ ] Нет комментариев кроме `//go:` / `//nolint:`

---

## Сводная таблица (P01→P49)

| P | ID | Задача | Сложность | CI |
| :---: | :--- | :--- | :---: | :--- |
| P01 | GAP-HYG-27 | Symmetric control replication | Very high | full-test |
| P02 | GAP-HYG-30 | PG volume meter / drift audit / pinned settlement | Very high | full-test |
| P03 | GAP-HYG-28 | Modular monolith deploy | High | — |
| P04 | GAP-PROD-11 | RBAC & field masking | High | — |
| P05 | GAP-HYG-03 | Wire `adminapi.RegisterRoutes` | High | full-test |
| P06 | GAP-HYG-04 | Remove HTMX/HTML UI | High | full-test |
| P07 | GAP-PROD-02 | Bundled SPA | High | — |
| P08 | GAP-BIZ-04 | Margin guard & revenue share | High | — |
| P09 | GAP-OPS-05 | Zero-DevOps (`espx doctor`) | High | — |
| P10 | GAP-HYG-22 | Scripts hygiene P0 | High | perf-gate |
| P11 | GAP-HYG-25 | Dead code pass | High | openapi |
| P12 | GAP-PROD-03 | Vendor SKU + operator plans YAML | Medium–high | — |
| P13 | GAP-PROD-06 | License protection hardening | Medium–high | — |
| P14 | GAP-BIZ-01 | Smart pacing (VPP) | Medium–high | — |
| P15 | GAP-HYG-06 | H1 single-writer guard (gtax) | Medium–high | full-test |
| P16 | GAP-OPS-06 | Embedded lite dashboard | Medium | — |
| P17 | GAP-BIZ-02 | Bid shading / floor optimizer | Medium | — |
| P18 | GAP-BIZ-03 | Smart retargeting segments | Medium | — |
| P19 | GAP-PROD-04 | License heartbeat policy | Medium | — |
| P20 | GAP-DATA-02 | Operator data security hardening | Medium | — |
| P21 | GAP-PROD-05 | Optional ClickHouse profile | Medium | — |
| P22 | GAP-PROD-08 | Opt-in product telemetry | Medium | — |
| P23 | GAP-HYG-26 | Coldpath helpers | Medium | full-test |
| P24 | GAP-HYG-18 | Rename legacy fault terminology | Medium | — |
| P25 | GAP-PROD-07 | Deploy profiles | Medium | — |
| P26 | GAP-HYG-05 | Filter timeout HTTP contract | Medium | full-test |
| P27 | GAP-SUP-01 | Redacted debug bundle | Medium | — |
| P28 | GAP-PROD-09 | SPA feedback + diagnostic bundle | Medium | — |
| P29 | GAP-HYG-07 | Repo docs migration | Medium | lint |
| P30 | GAP-HYG-09 | CI Tier A | Medium | lint |
| P31 | GAP-HYG-10 | 501 stub routes | Medium | openapi |
| P32 | GAP-HYG-08 | Remove legacy taxonomy refs | Medium | Tier A |
| P33 | GAP-HYG-12 | `check_comments` baseline | Medium | lint |
| P34 | GAP-HYG-21 | fmt vs slog audit | Medium | — |
| P35 | GAP-HYG-20 | Receivers and `_ =` | Medium | — |
| P36 | GAP-HYG-19 | Ingestion file renames | Medium | — |
| P37 | GAP-HYG-15 | `domains.go` registry | Low | full-test |
| P38 | GAP-HYG-14 | golangci-lint 10 issues | Low | lint |
| P39 | GAP-HYG-16 | Comment purge + zero-comment CI | Low | Tier A |
| P40 | GAP-HYG-13 | Skip sqlc in check_comments | Low | lint |
| P41 | GAP-HYG-29 | Broker ops hygiene | Low | — |
| P42 | GAP-HYG-23 | Dockerfile layout | Low | — |
| P43 | GAP-HYG-24 | Scripts deletion pass | Low | — |
| P44 | GAP-HYG-11 | `git filter-repo` root binaries | Low | — |
| P45 | GAP-PROD-10 | Community vs Pro split | Low | — |
| P46 | GAP-HYG-17 | Git history soft-squash (optional) | Low | — |
| P47 | GAP-HYG-31 | Self-hosted paradigm (residual code) | Low | — |
| P48 | GAP-PROD-12 | Brand boundary (BidShard UI vs neutral runtime) | Low | lint |
| P49 | GAP-ML-01 | Fraud ML platform plumbing | Medium–low | full-test |

---

## Спецификации

Каждая задача: метаданные → **Проблема** → **Решение** → SLA/паттерны/тесты (где применимо) → **Definition of done**.

---

### P01 — GAP-HYG-27 — Symmetric control replication

| | |
| :--- | :--- |
| **Область** | Control plane / Redis / registry |
| **Сложность** | Very high |
| **Зависимости** | Стабильный `StaticSlotSharder`; outbox publisher |
| **Блокирует CI** | `full-test` |
| **Touch** | `internal/management/outbox_handlers.go`, `internal/ingestion/registry.go`, `internal/auth`, `scripts/ci/check_no_shard0_control.sh` |

#### Проблема

Сегодня control-plane сигналы (pub/sub `campaigns:update`, auth lockout, pg_failover notify, outbox Redis publish) завязаны на **shard 0**. При падении shard 0:

- Трекеры на shards 1–3 продолжают обслуживать `/track` по slot routing, но **не получают** обновления каталога кампаний — stale registry до ручного вмешательства.
- Lockout и failover-сигналы не доходят до остальных шардов — дыра в безопасности и ops.
- Оператор вынужден держать shard 0 как скрытый SPOF, хотя data-plane уже шардирован по `crc32(campaign_id)`.

Политика stale-serve при недоступном registry уже есть; задача — **симметричная репликация control**, без переноса budget/filter keys на shard 0.

#### Решение

1. **Fan-out publisher:** при любом control-событии (активация кампании, entitlement reload, cohort update) management публикует `campaigns:update` (и bump `campaign_epoch`) на **все** Redis connections из pool, не только `rdbs[0]`.
2. **Per-shard epoch:** ключ `campaign_epoch` на каждом shard; bump через `INCR`/`SET` атомарно при activation. Tracker сравнивает локальный epoch с shard-local значением — reload только при расхождении.
3. **Shard-local subscribe:** `registry.go` подписывается на pub/sub **своего** shard connection (или poll с `REGISTRY_POLL_MS` если pub/sub недоступен на read replica topology). Hot path по-прежнему читает `atomic.Pointer` snapshot — 0 allocs.
4. **Убрать hardcode `rdbs[0]`:** auth lockout keys — либо replicate write на все shards, либо вынести в shard-agnostic store (PG + short TTL cache). CI guard `check_no_shard0_control.sh` ловит регрессии.
5. **Observability:** `ad_control_fanout_lag_seconds{shard}` — время от PG commit outbox до epoch bump на shard; `ad_registry_epoch` gauge per process.

Hot path **не меняется:** routing остаётся `StaticSlotSharder`; только cold reload path.

#### Паттерны

Multi-shard fan-out; atomic epoch versioning; stale-serve unchanged; cold-only I/O.

#### SLA

| Метрика | Target |
| :--- | :--- |
| Registry reload (cold) | p99 < 100 ms |
| Fanout lag (PG commit → epoch on all shards) | p99 < 5 s |
| Hot `/track` | p99 < 80 ms, 0 allocs delta |

#### Стиль кода

Cold path: idiomatic Go, `%w` errors в publisher. Hot path: без новых imports из management; snapshot swap через `atomic.Pointer`. Запрещены комментарии в коде.

#### Тестирование

- `TestFault_Shard0PubsubDown`: shard 0 pub/sub down, shards 1–3 продолжают ingest; registry eventually consistent после fan-out с другого publisher path.
- `fault_proof fault=symmetric_control_epoch shards=4`
- Integration: activation → epoch bump на всех 4 shards в testcontainers.

#### Definition of done

**Redis fan-out**
- [x] `campaigns:update` пишется на **все** shard connections
- [x] `campaign_epoch` на каждом shard; bump atomic per activation
- [x] `registry.go` — shard-local pub/sub или poll `REGISTRY_POLL_MS`

**Remove shard-0 hardcode**
- [x] `rg 'rdbs\[0\]' internal/` — zero для control-plane (lockout, pg_failover, outbox redis)
- [x] `internal/auth` lockout keys replicated или shard-agnostic

**CI / metrics / tests**
- [x] `scripts/ci/check_no_shard0_control.sh` green
- [x] `ad_control_fanout_lag_seconds` per shard; `ad_registry_epoch` gauge
- [x] `TestFault_Shard0PubsubDown` — shards 1–3 continue
- [x] `fault_proof fault=symmetric_control_epoch shards=4`

---

### P02 — GAP-HYG-30 — PG volume meter / drift audit / pinned settlement

| | |
| :--- | :--- |
| **Область** | Billing / settlement / reconciliation |
| **Сложность** | Very high |
| **Зависимости** | Outbox; processor stream consumer |
| **Блокирует CI** | `full-test` |
| **Touch** | `internal/management/` settlement workers, `internal/billing/`, `cmd/billing`, `cmd/processor` |

#### Проблема

Три связанные боли:

1. **Volume meter из ClickHouse:** биллинг читает CH для `accepted_events` — CH может отставать, быть выключен (`CH_ENABLED=0`) или расходиться с PG source of truth. Инвойсы строятся на аналитике, а не на authoritative ledger.
2. **Drift Redis ↔ PG ↔ CH:** spend keys в Redis, `balance_ledger` в PG, `campaign_stats` в CH — без автоматического audit оператор узнаёт о расхождении только когда бюджет «ломается» или клиент жалуется.
3. **Settlement contention:** `SettlementWorker` и admin API делят один PG pool; batch settlement блокирует read-heavy API; порядок событий per campaign не гарантирован при параллельных goroutines.

Нарушение budget invariant (`current_spend <= budget_limit`) — hard stop для продакшена.

#### Решение

1. **PG-authority volume meter:** worker агрегирует `COUNT(*)` из `events` WHERE `status='accepted'` по calendar month → `billing.usage_meters`. `GenerateInvoice` читает **только** PG. CH path удалён или за `VOLUME_METER_SOURCE=pg`.
2. **Pinned settlement lanes:** `lane = crc32(campaign_id) % N` (`SETTLEMENT_LANES` = GOMAXPROCS). Каждая lane — один consumer goroutine, buffered channel, strict per-campaign ordering. Batch: 100 ms **или** 1000 events. `XACK` Redis stream только после PG `COMMIT`.
3. **Pool tiering:** `PgPoolSettle` (max conns ≤ lanes+2) — только settlement/recon. `PgPoolRead` — API, exports, CH triggers. Admin не голодает settlement и наоборот.
4. **ReconWorker (5 min):** три audit: (A) Redis spend vs ledger+lag, (B) `campaign_stats` vs CH MV ±0.01%, (C) sample 100 customers/hour invariant. Drift → metric + optional `ForceRefillFromPG`; critical → outbox `FORCE_PAUSE`.
5. **LedgerInvariantWorker (24 h):** full scan `customers.balance` vs `SUM(balance_ledger)`.
6. **Schema:** partial index `events(created_at) WHERE status='accepted'`; `sync_idempotency` unique `(event_id, campaign_id)`.

Disk I/O settlement path — через существующий pool gate; hot path не трогаем.

#### Паттерны

Pinned consumer; transactional batch; idempotency `ON CONFLICT DO NOTHING`; pool tiering; outbox side effects; `AssertBudgetInvariant`.

#### SLA

| Метрика | Target |
| :--- | :--- |
| Recon tick | p99 < 500 ms |
| Settlement commit | p99 < 500 ms |
| Global settle end-to-end | p99 < 2 s |
| Meter rollup query | index scan, no seq scan >1M rows |

#### Стиль кода

Workers: `errors.Join` на batch failures; structured slog с `campaign_id`. Settlement: без `defer` в inner batch loop где hot-adjacent. Money: `int64` micro-units only.

#### Тестирование

- `go test ./internal/management/... -run 'Settlement|Recon|VolumeMeter'`
- `EXPLAIN_AUDIT=1 go test ./internal/billing/... -run Explain`
- `TestFault_ReconDriftRefill` + `AssertBudgetInvariant`
- `fault_proof fault=recon_drift_within_band`
- `fault_proof fault=settlement_replay proposal_rows=1` — replay x3, одна ledger row
- Runbook `docs/runbooks/RECONCILIATION_AND_SETTLEMENT.md` синхронизирован

#### Definition of done

**Volume meter (PG authority)**
- [x] Worker агрегирует из `events` / settlement counters — **no** `CHQuery` в meter path
- [x] `billing.usage_meters` upsert для `accepted_events`, calendar month
- [x] `cmd/billing` `GenerateInvoice` читает только PG `usage_meters`
- [x] Legacy CH `VolumeMeterWorker` removed или `VOLUME_METER_SOURCE=pg`

**Pinned settlement**
- [x] `SettlementWorker`: `lane = crc32(campaign_id) % N` (`SETTLEMENT_LANES` default = GOMAXPROCS)
- [x] Per-lane buffered channel; single goroutine per lane
- [x] Batch: 100 ms или 1000 events
- [x] `XACK` только после PG `COMMIT`

**Pool isolation**
- [x] `PgPoolSettle`: max conns ≤ lanes + 2
- [x] `PgPoolRead`: admin API, exports, CH handlers
- [x] `PG_POOL_SETTLE_MAX_CONNS` в `.env.example`

**ReconWorker (5 min)**
- [x] Audit A: Redis spend vs `SUM(balance_ledger)` + stream lag
- [x] Audit B: `campaign_stats` vs CH hourly MV (tolerance 0.01%)
- [x] Audit C: sample 100 customers/hour ledger invariant
- [x] Drift → `ad_recon_drift_micro`; optional `ForceRefillFromPG`
- [x] Invariant fail → outbox `FORCE_PAUSE`

**LedgerInvariantWorker (24 h)**
- [x] Full scan customers vs ledger; notifier alert on mismatch

**Schema**
- [x] Index `events_created_at_status_idx` WHERE `status = 'accepted'`
- [x] `sync_idempotency` unique `(event_id, campaign_id)`

**Metrics**
- [x] `ad_recon_drift_micro{campaign_id}` (bounded cardinality)
- [x] `ad_settlement_lag_seconds`, `ad_settlement_lane_depth{lane}`, `ad_volume_meter_rows_total`

**Verify**
- [x] `go test ./internal/management/... -run 'Settlement|Recon|VolumeMeter'`
- [x] `EXPLAIN_AUDIT=1 go test ./internal/billing/... -run Explain`
- [x] `AssertBudgetInvariant` after `TestFault_ReconDriftRefill`
- [x] `fault_proof fault=recon_drift_within_band`
- [x] `fault_proof fault=settlement_replay proposal_rows=1`
- [x] `docs/runbooks/RECONCILIATION_AND_SETTLEMENT.md` matches code

**SQL:** см. `.cursor/GAP_SPECS.md` § SQL — GAP-HYG-30.

---

### P03 — GAP-HYG-28 — Modular monolith deploy

| | |
| :--- | :--- |
| **Область** | Deploy / compose / binary layout |
| **Сложность** | High |
| **Зависимости** | P25 (profiles) частично пересекается |
| **Touch** | `docker-compose.yaml`, `cmd/management`, `docs/SELF_HOSTED.md` |

#### Проблема

Сейчас self-hosted стек предполагает **много отдельных контейнеров** (management, auth, payment, billing, notifier) даже на одном VPS. Для оператора с 1–2 CPU это:

- Лишний overhead на сеть между контейнерами и дублирование workers.
- Сложная матрица «что включать» для ingest-only vs full network operator.
- Риск **двойной регистрации workers** при масштабировании tracker (margin-guard, cost-sync стартуют в каждой реплике).

Нужна модель **modular monolith**: один `control` binary/profile для cold path, отдельно `tracker` и `processor` для hot path.

#### Решение

1. **Compose profile `single_vps`:** сервис `control` = management+auth+payment+billing+notifier в одном процессе (или одном контейнере с subcommands — решение в PR, но workers единожды).
2. **Профили возможностей:** `ingest_only` (tracker+processor+redis+pg, без payment/billing), `network_operator` (+ payment), `analytics_ml` (+ CH, fraud-scorer, ivt-detector).
3. **Worker ownership:** margin-guard, cost-sync, volume meter, recon — **только** в control; tracker replicas не поднимают cold workers.
4. **Документация:** матрица `ESPX_REGION_CODE=0|>0` — global vs regional processor, какие сервисы обязательны.
5. **Smoke:** каждый profile поднимается одной командой compose; health endpoints зелёные.

#### Паттерны

Compose profiles; single-writer workers; env-gated service list.

#### SLA

N/A (deploy correctness). Smoke: all `/health` < 2 s after `up`.

#### Тестирование

- `docker compose --profile ingest_only up -d` — payment/billing absent, tracker healthy
- `docker compose --profile network_operator up -d` — payment healthy
- Manual matrix в `docs/SELF_HOSTED.md`

#### Definition of done

- [ ] Profile `single_vps`: `control`, `tracker`, `processor`
- [ ] Profiles `ingest_only`, `network_operator`, `analytics_ml` в compose + `docs/SELF_HOSTED.md`
- [ ] `ESPX_REGION_CODE=0|>0` matrix в `docs/DEVELOPMENT.md`
- [ ] margin-guard / cost-sync только в control binary
- [ ] `docker compose --profile ingest_only up -d` — payment/billing absent
- [ ] `docker compose --profile network_operator up -d` — payment healthy

---

### P04 — GAP-PROD-11 — RBAC & field masking

| | |
| :--- | :--- |
| **Область** | AuthZ / admin API / self-hosted multi-tenant |
| **Сложность** | High |
| **Touch** | `internal/management/authz/`, DTO scrubbing, `deploy/operator/roles.yaml` |

#### Проблема

Self-hosted install часто обслуживает **несколько ролей** на одном кластере: оператор платформы, buyer (видит spend, не видит креативы), finance (баланс без targeting), support (read-only masked). Сейчас:

- Permissions есть частично, но **нет field-level masking** — `target_url`, `creative_payload` утекают в JSON для ролей с `campaigns:read`.
- Audit log не фиксирует, что мутация была с masked-ролью.
- Self-serve API key routes не ограничены `ScopeCustomer` жёстко.

Для enterprise и white-label это блокер compliance и доверия между участниками одного инстанса.

#### Решение

1. **Policy engine:** `EffectivePermissions(userID) → set[string]`; кэш snapshot на login и `POST /api/v1/ops/roles/reload`. `MaskLevel(full|masked)` выводится из набора permissions.
2. **DTO scrubbing в service layer:** `CampaignResponse.Scrub(level)` обнуляет чувствительные поля **до** JSON encode — не regex в handler.
3. **Scopes:** `ScopeGlobal`, `ScopeCustomer`, `ScopeTeam` на role assignment; self-serve routes всегда `ScopeCustomer` из API key metadata.
4. **Config:** `deploy/operator/roles.yaml` + hot reload; до YAML — SQL seed в runbook.
5. **Audit:** `admin_audit_log.is_masked=true` для мутаций masked-ролей; `GET /api/v1/audit` показывает флаг.
6. **Schema:** permissions seed `campaigns:read:masked`, `campaigns:write:masked`, `campaigns:pause`.

AuthZ check — O(1) из in-memory snapshot; без PG на каждый request.

#### Паттерны

Policy snapshot; scrub at service boundary; scoped RBAC; audit trail.

#### SLA

| Метрика | Target |
| :--- | :--- |
| AuthZ check (cached) | p99 < 5 ms |
| Scrub overhead | < 1 µs per DTO field (cold) |

#### Тестирование

- Table-driven: 20 cases permission × route × field
- Integration: buyer role → `GET /campaigns/{id}` без `target_url`
- `fault_proof fault=rbac_mask_enforced`
- OpenAPI: 403 для insufficient permission

#### Definition of done

**Schema**
- [ ] `admin_audit_log.is_masked BOOLEAN NOT NULL DEFAULT false`
- [ ] Permissions: `campaigns:read:masked`, `campaigns:write:masked`, `campaigns:pause`

**Policy**
- [ ] `authz/policy.go` — `EffectivePermissions(userID) → set`
- [ ] `MaskLevel(full|masked)` from permission set
- [ ] Snapshot refresh on login + `POST /api/v1/ops/roles/reload`

**DTO scrubbing**
- [ ] `CampaignResponse.Scrub(level)` zeroes `target_url`, `creative_payload`, `referrer_filter`
- [ ] Scrub in service layer, not handler string replace
- [ ] Integration: buyer role → `target_url` absent

**Scopes / config / audit**
- [ ] `ScopeGlobal`, `ScopeCustomer`, `ScopeTeam`
- [ ] `deploy/operator/roles.yaml` + reload endpoint
- [ ] Mutations by masked roles → `is_masked=true` in audit log

**Verify**
- [ ] Table-driven: 20 cases permission × route × field
- [ ] `fault_proof fault=rbac_mask_enforced`
- [ ] OpenAPI documents 403

---

### P05 — GAP-HYG-03 — Wire `adminapi.RegisterRoutes`

| | |
| :--- | :--- |
| **Область** | HTTP routing / OpenAPI drift |
| **Сложность** | High |
| **Блокирует CI** | `full-test` |
| **Touch** | `cmd/management/main.go`, `internal/adminapi/`, `internal/management/handler_api.go` |

#### Проблема

Маршруты `/api/v1` зарегистрированы **дважды**: в `internal/management/handler_api.go` и в `internal/adminapi/`. Последствия:

- OpenAPI generator не знает полный catalog — drift между spec и runtime.
- Дублированные handler bodies расходятся при правках (один путь исправили, второй нет).
- `adminapi` package существует, но не является single owner — dead code риск (см. P11).

SPA (P07) и contract tests требуют **один источник правды** для маршрутов.

#### Решение

1. `cmd/management/main.go` монтирует API **исключительно** через `adminapi.RegisterRoutes(mux, deps)`.
2. Удалить дублирующие `HandleFunc` из `handler_api.go`, `handler_selfserve.go` — thin delegate или move.
3. `internal/adminapi/register.go` — exported route list для OpenAPI generator.
4. `make openapi-gen` + `make openapi-lint` — zero unlisted `/api/v1` paths.
5. Security schemes без изменений: `X-Admin-API-Key`, session cookie, `X-Consent-Signature`.
6. Дубликаты **удалить**, не комментировать.

#### Паттерны

Single HTTP owner; handler → service → store; contract-first OpenAPI.

#### SLA

Cold HTTP handler p99 < 200 ms (unchanged).

#### Тестирование

- `go test ./internal/adminapi/... ./cmd/management/... -short`
- `tests/contract/openapi_test.go` per route prefix
- `curl` smoke `/api/v1/health` → 200

#### Definition of done

- [ ] `cmd/management/main.go` mounts exclusively via `adminapi.RegisterRoutes(mux, deps)`
- [ ] No duplicate `HandleFunc` in `handler_api.go` for paths in adminapi
- [ ] `register.go` — single route catalog for OpenAPI
- [ ] Duplicate handler bodies deleted (not commented out)
- [ ] `make openapi-gen` + `make openapi-lint` green
- [ ] `go test ./internal/adminapi/... ./cmd/management/... -short`
- [ ] `tests/contract/openapi_test.go` per route prefix

---

### P06 — GAP-HYG-04 — Remove HTMX/HTML UI

| | |
| :--- | :--- |
| **Область** | Management / payment cold path |
| **Сложность** | High |
| **Блокирует CI** | `full-test` |
| **Зависимости** | P07 (SPA) для полного UX; interim placeholder допустим |
| **Touch** | `internal/management/htmx_*.go`, `handler_billing.go`, `internal/payment/*_html*` |

#### Проблема

Legacy **server-rendered HTMX** (`/admin/*`, billing HTML fragments, payment checkout HTML) противоречит self-hosted модели:

- Два контракта: JSON `/api/v1` для API и HTML для UI — двойная поддержка, разный error handling.
- SPA (P07) не может быть единственным UI, пока management отдаёт `text/html` на success paths.
- CI не может гарантировать JSON-only: `check_no_html_success.sh` падает.
- HTMX coupling мешает RBAC/masking (P04) — поля утекают в HTML templates.

Документация `SELF_HOSTED.md` уже объявляет HTMX deprecated; код отстаёт.

#### Решение

1. **Удалить** все `htmx_*.go`, HTML templates в `handler_billing.go`, `payment/*_html*.go`.
2. **Контракт API:** каждый `/api/v1/*` success → `Content-Type: application/json`; errors только через `writeServiceError` / `pkg/coldpath`.
3. **`/admin/*`:** `410 Gone` с JSON body или полное удаление из mux — breaking change документировать.
4. **Interim до P07:** embed placeholder `index.html` **или** JSON 404 на `GET /` со ссылкой на docs.
5. **CI:** `check_no_html_success.sh`; OpenAPI без `/admin/*` HTML routes.
6. Убрать `text/template` imports из management/payment (кроме минимальных error pages если останутся).

#### Паттерны

JSON-only cold path; breaking change с версией в release notes.

#### SLA

N/A.

#### Тестирование

- `rg 'htmx|text/html' internal/management internal/payment` — zero on success paths
- Contract: 20 sample routes never return `text/html`
- `make openapi-lint`

#### Definition of done

**Delete**
- [ ] All `htmx_*.go` removed
- [ ] `handler_billing.go` HTML templates removed
- [ ] `internal/payment/*_html*.go` removed
- [ ] No `text/template` in management/payment (except error pages if any)

**API**
- [ ] Every `/api/v1/*` success → `Content-Type: application/json`
- [ ] Errors via `writeServiceError` / `pkg/coldpath` only
- [ ] `/admin/*` → `410 Gone` or removed

**CI / interim**
- [ ] `scripts/ci/check_no_html_success.sh` green
- [ ] OpenAPI zero `/admin/*` HTML routes
- [ ] Placeholder `index.html` embed OR JSON 404 on `GET /` — documented in `SELF_HOSTED.md`

**Verify**
- [ ] `rg 'htmx|text/html' internal/management internal/payment` — no success-path matches
- [ ] Contract test: 20 API routes never return `text/html`

---

### P07 — GAP-PROD-02 — Bundled SPA

| | |
| :--- | :--- |
| **Область** | Frontend / management embed |
| **Сложность** | High |
| **Зависимости** | P05, P06 |
| **Touch** | `web/admin/`, `cmd/management/embed.go`, `internal/management/static/` |

#### Проблема

После удаления HTMX (P06) оператору нужен **встроенный UI** без отдельного nginx для frontend. Требования:

- Один бинарь management отдаёт API + static assets + SPA fallback.
- Auth через существующие session/API key endpoints — без нового OAuth flow.
- CSP и отсутствие secrets в JS bundle.
- Self-hosted на air-gapped сети — UI должен собираться в CI и embed'иться, не тянуть CDN runtime.

#### Решение

1. **Monorepo frontend:** `web/admin/` — Vite + React (или существующий stack из PR).
2. **Build pipeline:** `make ui-build` → `internal/management/static/dist/`.
3. **Embed:** `//go:embed dist/*` в `cmd/management/embed.go`.
4. **Routing в mux:** `/api/v1/*` → API handlers; `/assets/*` → cached static (immutable hash filenames); `/*` → `index.html` SPA fallback; исключения `/metrics`, `/health`.
5. **Auth:** SPA использует cookie session или `X-Admin-API-Key` per OpenAPI; login page бьёт в существующие auth endpoints.
6. **Security:** CSP в `deploy/management/production.env`; scan bundle на `sk_live`, passwords.
7. **Brand (P48):** visible strings из `GET /api/v1/meta` или `VITE_BRAND_*`.

#### Паттерны

Embedded static; SPA fallback; API-first.

#### SLA

| Метрика | Target |
| :--- | :--- |
| Static shell (`index.html`, assets) | p99 < 50 ms |

#### Тестирование

- E2E: `go test ./tests/e2e/... -run SPA`
- `rg 'sk_live|password' dist/` — zero
- Lighthouse/security scan optional

#### Definition of done

- [ ] `web/admin/` Vite/React → `make ui-build` → `internal/management/static/dist/`
- [ ] `//go:embed dist/*` in `cmd/management/embed.go`
- [ ] Routing: `/api/v1/*` API; `/assets/*` cached static; `/*` → `index.html` (except `/metrics`, `/health`)
- [ ] Auth: session cookie or API key per OpenAPI
- [ ] CSP in `deploy/management/production.env`
- [ ] `rg 'sk_live|password' dist/` — zero
- [ ] E2E: `go test ./tests/e2e/... -run SPA`

---

### P08 — GAP-BIZ-04 — Margin guard & revenue share

| | |
| :--- | :--- |
| **Область** | Billing / RTB economics |
| **Сложность** | High |
| **Touch** | `balance_ledger`, settlement, `MarginGuardWorker`, management API |

#### Проблема

В RTB и publisher payout модели **расходы на закупку трафика** (`rtb_cost`) могут превысить **выручку от рекламодателя** (`advertiser_spend`) из-за floor errors, latency в stats или fraud. Без автоматического guard:

- Оператор теряет margin незаметно до конца месяца.
- `AssertBudgetInvariant` не ловит multi-leg economics — только customer balance vs ledger sum.
- Нет API для ops «почему кампания paused по margin».

Нужен rolling guard + multi-leg ledger + revenue share types.

#### Решение

1. **Ledger types:** `publisher_payout`, `operator_margin`, `rtb_cost` в `balance_ledger.type`.
2. **Settlement:** multi-leg rows в **одной транзакции** где применимо — atomic debit/credit.
3. **MarginGuardWorker:** tick 5 min; per campaign rolling 1h: если `rtb_cost_sum > revenue_sum * (1 + threshold)` → outbox `FORCE_PAUSE`.
4. **API:** `GET /api/v1/campaigns/{id}/margin` — PG aggregates для dashboard.
5. **Metrics:** guard decisions, pause count — pre-bound labels.

Worker cold-only; hot path не импортирует billing.

#### Паттерны

Multi-leg ledger; outbox `FORCE_PAUSE`; rolling window aggregate in PG.

#### SLA

Guard tick p99 < 500 ms.

#### Тестирование

- `AssertBudgetInvariant` with multi-leg rows
- `fault_proof fault=margin_guard_pause campaign_id=...`
- Integration: artificial rtb_cost spike → pause

#### Definition of done

- [ ] `balance_ledger.type`: `publisher_payout`, `operator_margin`, `rtb_cost`
- [ ] Settlement writes multi-leg rows in one txn
- [ ] `MarginGuardWorker` every 5 min; rolling 1h compare
- [ ] `rtb_cost > revenue * (1 + threshold)` → `FORCE_PAUSE`
- [ ] `GET /api/v1/campaigns/{id}/margin`
- [ ] `AssertBudgetInvariant` with multi-leg rows
- [ ] `fault_proof fault=margin_guard_pause campaign_id=...`

---

### P09 — GAP-OPS-05 — Zero-DevOps (`espx doctor`)

| | |
| :--- | :--- |
| **Область** | Ops CLI / onboarding |
| **Сложность** | High |
| **Зависимости** | P27 (bundle subcommand) |
| **Touch** | `cmd/espx` or `cmd/management` subcommand, `pkg/doctor/` |

#### Проблема

Self-hosted оператор на VPS без dedicated SRE не знает, **готов ли хост** к eSPX: sysctl, file descriptors, Redis latency, CH optional, disk iogate, TLS PG, XDP/bpf для edge. Сейчас проверки размазаны по README, runbooks и ручным curl.

Нужен один CLI с exit codes для automation и checklist MVSS из `DATA_SECURITY.md`.

#### Решение

1. **`espx doctor`** (или subcommand management binary): probes с `--only=redis,sysctl` filter.
2. **Exit codes:** 0 all pass; 1 warnings; 2 failures — для systemd/ansible.
3. **Probes:** kernel bpf/XDP; sysctl (`fs.file-max` ≥1M, `somaxconn` ≥4096); Redis PING each shard p99 <10ms; CH insert 1 row <500ms (skip if `CH_ENABLED=0`); disk iogate smoke < `DISK_LATENCY_BUDGET_MS`; TLS PG when `ESPX_PROFILE=production`.
4. **Autotune on start:** `GOMEMLIMIT` 90% RAM if unset; `PinnedWorkerPool` from `NumCPU()`.
5. **`espx doctor --checklist`:** MVSS rows pass/fail table.
6. **`espx doctor bundle`:** ссылка на P27 redacted tarball.

Пакет `pkg/doctor` — table-driven probes, mockable в unit tests.

#### Паттерны

Probe registry; fail-fast exit codes; autotune env.

#### SLA

Full doctor run < 60 s on reference VPS.

#### Тестирование

- `go test ./pkg/doctor/...`
- Manual on clean Ubuntu VPS
- Each probe isolated with `--only`

#### Definition of done

- [ ] `espx doctor` subcommand; `--only=redis,sysctl`
- [ ] Exit 0/1/2 = pass/warn/fail
- [ ] Probes: kernel XDP/bpf, sysctl, redis PING p99 < 10 ms, CH insert < 500 ms, disk iogate smoke, TLS PG in production
- [ ] Autotune: `GOMEMLIMIT` 90% RAM; `PinnedWorkerPool` from `NumCPU()`
- [ ] `espx doctor --checklist` MVSS from `DATA_SECURITY.md`
- [ ] `espx doctor bundle --out=...` (links P27)
- [ ] `go test ./pkg/doctor/...`

---

### P10 — GAP-HYG-22 — Scripts hygiene P0

| | |
| :--- | :--- |
| **Область** | Repo scripts / CI entrypoints |
| **Сложность** | High |
| **Блокирует CI** | `perf-gate` |
| **Touch** | `scripts/`, `scripts/README.md`, compose refs |

#### Проблема

После реструктуризации `scripts/` часть путей **битая**: README ссылается на несуществующие папки, `management-domain-coverage` не в `scripts/ci/`, perf-gate и edge scripts не проходят smoke на clean clone. Новый разработчик не может запустить `make check-local` без археологии.

#### Решение

1. Аудит всех ссылок в README, compose, Makefile → реальные пути в `scripts/{lib,ci,dev,load,perf,fault,edge,deploy,test}/`.
2. `scripts/README.md` — index: folder → purpose → entry command.
3. Перенести `management-domain-coverage` → `scripts/ci/management_domain_coverage.sh`.
4. Smoke: `dev_preflight.sh`, `perf_gate_run.sh` smoke mode, `edge_phase0.sh` (или documented skip).

#### SLA

perf-gate smoke p99 < 80 ms unchanged after path fixes.

#### Definition of done

- [ ] `scripts/edge/`, `redis/`, `perf/`, `dev/` — all refs resolve
- [ ] `scripts/README.md` index
- [ ] `management-domain-coverage` → `scripts/ci/management_domain_coverage.sh`
- [ ] `dev_preflight.sh`, `perf_gate_run.sh` smoke, `edge_phase0.sh` exit 0

---

### P11 — GAP-HYG-25 — Dead code pass

| | |
| :--- | :--- |
| **Область** | Repo hygiene |
| **Сложность** | High |
| **Зависимости** | P05 (adminapi wired or deleted) |
| **Блокирует CI** | `openapi` |

#### Проблема

Мёртвый код создаёт ложное покрытие и путает ownership: `pkg/broker/partition` без импортёров, `adminapi` дублирует management, Dockerfile ссылается на несуществующие `postback-sender`/`log-shipper`, OpenAPI описывает 501 routes как live.

Каждый лишний пакет — потенциальный import cycle и lint noise.

#### Решение

1. Удалить `pkg/broker/partition`; обновить импортёров или удалить вместе.
2. После P05: `adminapi` — единственный HTTP catalog **или** package deleted.
3. `postback-sender`, `log-shipper` — реализовать в `cmd/` **или** убрать из Dockerfile/compose.
4. OpenAPI sync: только routes с non-501 implementation.

#### Definition of done

- [ ] `pkg/broker/partition` deleted
- [ ] `adminapi` wired (P05) OR package deleted
- [ ] `postback-sender`, `log-shipper` — implement or remove from Dockerfile
- [ ] OpenAPI lists only non-501 routes
- [ ] `go test ./... -short`; `make openapi-lint`

---

### P12 — GAP-PROD-03 — Vendor SKU + operator plans YAML

| | |
| :--- | :--- |
| **Область** | Licensing / entitlements / billing plans |
| **Сложность** | Medium–high |
| **Touch** | `deploy/vendor/sku.yaml`, `deploy/operator/plans.yaml`, `cmd/espx-license` |

#### Проблема

Два уровня конфигурации смешаны:

- **Vendor** подписывает SKU JWT (features, limits) — оператор не должен менять claims.
- **Operator** назначает plans клиентам (`subscription_plans`, `customer_subscriptions`) на своём инстансе.

Сейчас entitlements частично в PG seed, частично в env — нет reload без migration, нет dry-run, tracker узнаёт о лимитах с задержкой.

#### Решение

1. **Vendor:** `deploy/vendor/sku.yaml` schema; `cmd/espx-license` читает SKU → signs Ed25519 JWT (`sku`, `features`, `limits`, `valid_until`, `customer_name`).
2. **Operator:** `deploy/operator/plans.yaml` → upsert `billing.subscription_plans` + `assignments[]` → `customer_subscriptions`.
3. **`POST /api/v1/ops/plans/reload`:** RBAC `ops:write`, `?dry_run=1` preview diff.
4. **Fan-out:** после reload — outbox или `campaigns:update` на все shards; tracker `Effective()` в пределах `SyncEntitlements` interval.
5. **Guard:** operator API не может PATCH vendor JWT fields → 403.

#### SLA

Reload p99 < 500 ms.

#### Definition of done

- [ ] `deploy/vendor/sku.yaml` schema documented
- [ ] `cmd/espx-license` signs JWT Ed25519: `sku`, `features`, `limits`, `valid_until`
- [ ] `deploy/operator/plans.yaml` → `billing.subscription_plans` + `customer_subscriptions`
- [ ] `POST /api/v1/ops/plans/reload` — RBAC `ops:write`; dry-run param
- [ ] Entitlements fan-out → tracker `Effective()` within sync interval
- [ ] Operator cannot PATCH vendor JWT fields (403)
- [ ] Integration: YAML → PG → registry snapshot

---

### P13 — GAP-PROD-06 — License protection hardening

| | |
| :--- | :--- |
| **Область** | Licensing / anti-piracy |
| **Сложность** | Medium–high |
| **Touch** | `vendor.license_activations`, heartbeat handler, license server |

#### Проблема

License key можно **клонировать на несколько инстансов** без привязки к fingerprint. Heartbeat не биндит machine identity; replay атаки не детектятся; JWT на диске живёт слишком долго при компрометации.

Vendor нужен audit trail activations и revoke queue при `same_key_many_fingerprints`.

#### Решение

1. **Schema:** `vendor.license_activations` с `UNIQUE(license_key, fingerprint)`; `max_activations` в JWT или column.
2. **First heartbeat binds fingerprint;** mismatch → 403 + audit log.
3. **JWT refresh:** successful heartbeat → new JWT `exp` ≤ 72h; atomic write file.
4. **Server-side:** flag `same_key_many_fingerprints` → webhook/manual revoke queue.
5. **Payload hygiene:** heartbeat не содержит campaign/domain/IP — contract test.

VerifyJWT hot path < 1 ms — local Ed25519 verify only.

#### Definition of done

- [ ] Migration `vendor.license_activations` `UNIQUE(license_key, fingerprint)`
- [ ] First heartbeat binds fingerprint; mismatch → 403 + audit
- [ ] JWT `exp` ≤ 72h; atomic file update
- [ ] Server flags `same_key_many_fingerprints` → revoke queue
- [ ] Heartbeat payload test — no campaign/domain/IP fields
- [ ] Unit: Ed25519 vectors; integration: clone fingerprint denied
- [ ] `fault_proof fault=license_heartbeat_replay`

---

### P14 — GAP-BIZ-01 — Smart pacing (VPP)

| | |
| :--- | :--- |
| **Область** | Budget / delivery optimization |
| **Сложность** | Medium–high |
| **Touch** | `PacingControllerWorker`, `FilterEngine`, Redis `campaign:{id}:pacing` |

#### Проблема

Равномерный spend в течение дня не соответствует реальному трафику: утренний пик, ночной спад. Без VPP кампания исчерпывает daily budget к полудню или недотрачивает ночью.

CH имеет hourly distribution; hot path не может делать CH query per request.

#### Решение

1. **Cold worker** `PacingControllerWorker` каждые 15 min: CH query 7d hourly distribution per campaign → `target_hourly_share[]` (sum=1.0).
2. **Write** `pacing_ratio` [0,1] в Redis `campaign:{id}:pacing`.
3. **Hot path:** `FilterEngine` читает ratio из **atomic snapshot** (как fraud boost); stochastic throttle когда over hourly curve.
4. **API:** `PATCH /api/v1/campaigns/{id}` `pacing_mode: vpp|off`.
5. **Bench:** `BenchmarkPacingRead` — 0 allocs/op; no CH import in ingestion.

#### SLA

| Метрика | Target |
| :--- | :--- |
| Hot read | < 100 ns |
| Worker tick | p99 < 500 ms |
| Hot path | 0 allocs/op |

#### Definition of done

- [ ] `PacingControllerWorker` tick 15 min
- [ ] CH query 7d hourly distribution → `target_hourly_share[]` sum 1.0
- [ ] Redis `campaign:{id}:pacing` ratio [0,1]
- [ ] `FilterEngine` reads ratio from atomic snapshot; stochastic throttle
- [ ] `BenchmarkPacingRead` — 0 allocs/op
- [ ] `PATCH /api/v1/campaigns/{id}` `pacing_mode: vpp|off`
- [ ] No CH import in `internal/ingestion`

---

### P15 — GAP-HYG-06 — H1 single-writer guard (gtax)

| | |
| :--- | :--- |
| **Область** | Settlement invariant H1 |
| **Сложность** | Medium–high |
| **Блокирует CI** | `full-test` |
| **Touch** | `internal/management/service_gtax.go`, processor settlement |

#### Проблема

`service_gtax.go` в management **напрямую пишет** spend / `balance_ledger` для gtax flows. Это нарушает **single-writer invariant**: только processor/settlement worker должен дебетить spend per campaign. При replay или race возможны duplicate ledger rows и budget drift.

Тест `TestH1_UpdateSpendSingleWriter` сейчас красный — блокер `full-test`.

#### Решение

1. **Management** только enqueue outbox `APPLY_GTV_SETTLEMENT` (или equivalent) с idempotency key.
2. **Processor/settlement worker** — единственный writer `UpdateSpend` / ledger debit для gtax.
3. SQL: `INSERT ... ON CONFLICT (idempotency_hash) DO NOTHING`.
4. Exception allowlist только в `single_writer_guard_test.go` с explicit PR sign-off.
5. `AssertBudgetInvariant` после gtax batch.

#### Паттерны

Outbox; single writer per campaign spend; idempotency hash.

#### SLA

Global settle p99 < 2 s.

#### Definition of done

- [ ] `UpdateSpend` / ledger debit for gtax only from processor/settlement worker
- [ ] `service_gtax.go` enqueues outbox `APPLY_GTV_SETTLEMENT` — no direct PG spend
- [ ] `TestH1_UpdateSpendSingleWriter` green
- [ ] `fault_proof fault=gtax_settlement_replay proposal_rows=1` — replay x3, one row
- [ ] `AssertBudgetInvariant` after gtax batch
- [ ] SQL: `ON CONFLICT (idempotency_hash) DO NOTHING`


### P16 — GAP-OPS-06 — Embedded lite dashboard

| | |
| :--- | :--- |
| **Область** | Ops UI / metrics retention |
| **Сложность** | Medium |
| **Зависимости** | P07 (SPA), P02 (drift metrics) |

#### Проблема

Self-hosted оператор без Grafana не видит сводку: lag processor, recon drift, RPS, состояние redis/pg/ch. Prometheus на `/metrics` не хранит 24h history. GAP-OPS-04 (queue UI) отложен — нужен минимальный встроенный dashboard в SPA.

#### Решение

API summary + downsampled metrics; scraper каждые 15s; PG `ops.metric_samples` или SQLite для `single_vps`; SPA topology cards с red badge при drift.

#### SLA

Dashboard API p99 < 200 ms (10k samples).

#### Definition of done

- [ ] `GET /api/v1/ops/dashboard/summary` — health, drift, RPS
- [ ] `GET /api/v1/ops/dashboard/metrics?range=24h`
- [ ] Scraper: `/metrics` every 15 s
- [ ] `ops.metric_samples` migration + retention > 24h delete
- [ ] SPA dashboard page (P07): topology cards; red badge on `ad_recon_drift_micro > 0`
- [ ] Integration: insert sample → API returns point

---

### P17 — GAP-BIZ-02 — Bid shading / floor optimizer

| | |
| :--- | :--- |
| **Область** | RTB yield |
| **Сложность** | Medium |

#### Проблема

Floor задаётся вручную; win rate по bucket'ам не используется. Оператор либо недозарабатывает, либо переплачивает за inventory.

#### Решение

Weekly `FloorOptimizerWorker` на CH; suggestions в PG; dry-run apply без outbox; live apply → `RELOAD_RTB_CATALOG`. Tracker читает catalog через registry reload.

#### SLA

CH query p99 < 1.5 s.

#### Definition of done

- [ ] `FloorOptimizerWorker` weekly per placement
- [ ] CH win rate by `floor_micro` buckets → `rtb_floor_suggestions`
- [ ] `POST /api/v1/rtb/floors/apply?dry_run=1` — zero outbox when dry-run
- [ ] Apply → `RELOAD_RTB_CATALOG` outbox
- [ ] CH testcontainer; `fault_proof fault=floor_optimizer_dry_run outbox_rows=0`

---

### P18 — GAP-BIZ-03 — Smart retargeting segments

| | |
| :--- | :--- |
| **Область** | Audience / processor + filter |
| **Сложность** | Medium |

#### Проблема

Retargeting требует знания conversion на hot path. PG per request недопустим.

#### Решение

Processor пишет Redis Bloom на conversion; `FilterEngine` читает segment include/exclude из snapshot; 0 allocs; optional PG export table cold-only.

#### SLA

Hot check < 500 ns; `BenchmarkSegmentCheck` 0 allocs/op.

#### Definition of done

- [ ] Processor on conversion: `BF.ADD segment:{id} user_hash`
- [ ] Campaign config: `segment_id`, `ttl_hours`
- [ ] `FilterEngine` checks `segment_exclude` / `segment_include` from snapshot
- [ ] `BenchmarkSegmentCheck` — 0 allocs/op
- [ ] Optional `segment_members` PG table — export only
- [ ] Integration: conversion → subsequent `/track` excluded; TTL expiry

---

### P19 — GAP-PROD-04 — License heartbeat policy

| | |
| :--- | :--- |
| **Область** | Licensing FSM |
| **Сложность** | Medium |
| **Зависимости** | P13 |

#### Проблема

JWT validity и heartbeat offline — разные оси. Нет UX для air-gap maintenance. Hot path не должен делать network.

#### Решение

State machine ACTIVE→OFFLINE_WARN→OFFLINE_GRACE→EXPIRED; cold heartbeat goroutine; SPA banner; metrics; EXPIRED блокирует `/track`.

#### SLA

Heartbeat client < 30 s; zero hot-path network.

#### Definition of done

- [ ] `ESPX_LICENSE_REFRESH_INTERVAL` default 24h; grace/renew env documented
- [ ] States: `ACTIVE` → `OFFLINE_WARN` → `OFFLINE_GRACE` → `EXPIRED`
- [ ] `EXPIRED` blocks ingest `license_expired` after grace
- [ ] SPA banner; `ad_license_offline_days`, `ad_license_state` metrics
- [ ] Unit table: all transition sequences
- [ ] Integration: license server down → warn → grace → 403 `/track`

---

### P20 — GAP-DATA-02 — Operator data security hardening

| | |
| :--- | :--- |
| **Область** | Retention / TLS / MVSS |
| **Сложность** | Medium |

#### Проблема

`events` растёт без retention; production TLS не единым overlay; MVSS checklist не автоматизирован в doctor.

#### Решение

Batched retention worker; optional hash-at-insert; production compose TLS; doctor checklist rows.

#### SLA

Retention < 5 min per 1M rows.

#### Definition of done

- [ ] `EventsRetentionWorker` — `EVENTS_RETENTION_DAYS` default 90; batch 10k + sleep 100ms
- [ ] `ad_events_retention_deleted_total`
- [ ] Optional `EVENTS_HASH_IP_AT_INSERT=1`
- [ ] Compose production: PG `sslmode=verify-full`, Redis TLS
- [ ] `espx doctor --checklist` MVSS rows
- [ ] Integration: old events deleted; TLS smoke

---

### P21 — GAP-PROD-05 — Optional ClickHouse profile

| | |
| :--- | :--- |
| **Область** | Deploy degradation |
| **Сложность** | Medium |

#### Проблема

Ingest-only install не нужен CH; processor не должен падать; stats API должен явно маркировать stale PG fallback.

#### Решение

`CH_ENABLED=0`; compose profiles; stats `stale: true`; degradation matrix в SELF_HOSTED.md.

#### Definition of done

- [ ] `CH_ENABLED=0` — processor skips CH consumer; health OK
- [ ] `analytics_ml` profile: CH + fraud-scorer + ivt-detector
- [ ] `ingest_only` — no CH; processor clean start
- [ ] Stats API: `stale: true`, `source: "pg"` when CH unavailable
- [ ] `docs/SELF_HOSTED.md` degradation matrix

---

### P22 — GAP-PROD-08 — Opt-in product telemetry

| | |
| :--- | :--- |
| **Область** | Vendor pulse |
| **Сложность** | Medium |

#### Проблема

Vendor нужны aggregate signals без PII; air-gap требует opt-in 0 по умолчанию с доказуемым отсутствием outbound HTTP.

#### Решение

Hourly pulse в management only; schema_v1.json; validator rejects forbidden fields; separate telemetry URL.

#### SLA

Pulse upload p99 < 5 s; 0 hot-path allocs.

#### Definition of done

- [ ] `ESPX_TELEMETRY_OPT_IN=0` default
- [ ] `internal/telemetry/pulse.go` — hourly management goroutine only
- [ ] Counters: `accepted_events`, `rejected_events`, `rps_peak`
- [ ] `docs/telemetry/schema_v1.json`; separate `ESPX_TELEMETRY_URL`
- [ ] Payload validator — reject campaign_id/domain/ip
- [ ] Opt-in 0: no outbound HTTP in 1h soak

---

### P23 — GAP-HYG-26 — Coldpath helpers

| | |
| :--- | :--- |
| **Область** | `pkg/coldpath` |
| **Сложность** | Medium |
| **Блокирует CI** | `full-test` |

#### Проблема

Дублирование pagination, UUID parse, JSON decode, migrations в management/adminapi — drift risk.

#### Решение

Extract `pkg/coldpath/`; ≥3 call sites per helper; unified `LogFaultProof`; ingestion не импортирует management-coupled helpers.

#### Definition of done

- [ ] `ApplyTrackedSchemaMigrations`, `ParsePathUUID`, `DecodeRequestOrBadRequest`, `Paginate`, `httpresponse.WriteGRPCError`
- [ ] ≥3 call sites per helper moved from management
- [ ] Single `internal/testutil/fault_proof.go` — `LogFaultProof`
- [ ] `internal/ingestion` does not import coldpath helpers with management types
- [ ] `go test ./pkg/coldpath/...`

---

### P24 — GAP-HYG-18 — Rename legacy fault terminology

| | |
| :--- | :--- |
| **Область** | Naming hygiene |
| **Сложность** | Medium |
| **Статус** | Renames done; verification pending |

#### Проблема

Legacy `chaos_*` names в тестах и скриптах конфликтуют с политикой fault/resilience terminology.

#### Решение

Переименовано в fault/resilience; CI guard `check_no_chaos_refs.sh`; осталось прогнать test suite и drill script.

#### Definition of done

- [x] `*_chaos_test.go` → `*_fault_test.go`
- [x] `TestChaos_*` → `TestFault_*`
- [x] `redis_chaos.go` → `redis_fault.go`
- [x] `chaos_proof` → `fault_proof`
- [x] `scripts/resilience-drills/`; `sentinel-resilience.yaml`
- [x] `scripts/ci/check_no_chaos_refs.sh`
- [ ] `go test ./... -run Fault -short` passes
- [ ] `bash scripts/resilience-drills/test_resilience.sh` passes

---

### P25 — GAP-PROD-07 — Deploy profiles

| | |
| :--- | :--- |
| **Область** | Compose onboarding |
| **Сложность** | Medium |

#### Проблема

Нет единой matrix profile → services → ports → env для ingest-only vs full vs ML.

#### Решение

Formalize compose profiles; README matrix; `espx doctor --profile`; smoke scripts.

#### Definition of done

- [ ] Compose profiles: `ingest_only`, `network_operator`, `analytics_ml`
- [ ] Matrix in `README.md`: profile → services → ports → env
- [ ] `espx doctor --profile <name>` validates containers
- [ ] Three compose smoke scripts in `scripts/local-dev/`

---

### P26 — GAP-HYG-05 — Filter timeout HTTP contract

| | |
| :--- | :--- |
| **Область** | Hot path HTTP |
| **Сложность** | Medium |
| **Блокирует CI** | `full-test` |

#### Проблема

Тесты ожидают 504 для filter timeout; production отдаёт 503 — ломает CI и мониторинг.

#### Решение

`classifyFilterErr`: `ErrFilterTimeout` → 504 only; document in ARCHITECTURE; zero alloc delta.

#### SLA

Hot p99 < 80 ms unchanged; 0 allocs delta.

#### Definition of done

- [ ] `ErrFilterTimeout` → HTTP **504** (not 503)
- [ ] `TestClassifyFilterErr_HandlerMatrix` — timeout row 504
- [ ] `TestAdsPacketHandler_FilterErrors`, `TestFault_HandlerRejectMatrix`, `TestProcessTrack_filterTimeout` — 504
- [ ] `make test-alloc-gate` unchanged
- [ ] `perf_gate_run.sh` smoke p99 < 80 ms
- [ ] `docs/ARCHITECTURE.md` filter error table: 504 for timeout

---

### P27 — GAP-SUP-01 — Redacted debug bundle

| | |
| :--- | :--- |
| **Область** | Support |
| **Сложность** | Medium |
| **Зависимости** | P09 |

#### Проблема

Ручной сбор profiles/logs приводит к утечке secrets (license, URLs, IPs, PII_SALT).

#### Решение

`POST /api/v1/ops/support/bundle` — streaming tar.gz с redaction pipeline и golden tests.

#### SLA

Generation < 30 s.

#### Definition of done

- [ ] `POST /api/v1/ops/support/bundle` — RBAC `ops:write`; max 50 MB; timeout 30 s
- [ ] Contents: `version.json`, `goroutine.pprof`, `heap.pprof`, `logs/redacted.log`, `config/sanitized.env`
- [ ] Redaction: URLs, IPs, `target_url`, `creative`, `license_key`, `PII_SALT_HEX`
- [ ] Golden `testdata/bundle_no_secrets.golden`
- [ ] `go test ... -run BundleRedaction`

---

### P28 — GAP-PROD-09 — SPA feedback + diagnostic bundle

| | |
| :--- | :--- |
| **Область** | Product support |
| **Сложность** | Medium |
| **Зависимости** | P07, P27 |

#### Проблема

Нет in-UI пути отправить feedback vendor с optional diagnostic attach.

#### Решение

SPA form + rate-limited `POST /api/v1/support/feedback`; server-side bundle attach; `support.feedback` table.

#### Definition of done

- [ ] SPA form: message + optional bundle checkbox
- [ ] `POST /api/v1/support/feedback` — rate limited
- [ ] Bundle via P27 when checkbox set
- [ ] Migration `support.feedback`
- [ ] Redaction tests on attached bundle

---

### P29 — GAP-HYG-07 — Repo docs migration

| | |
| :--- | :--- |
| **Область** | Docs layout |
| **Сложность** | Medium |
| **Блокирует CI** | `lint` |

#### Проблема

`docs/` перегружен; broken links; client vs agent docs смешаны.

#### Решение

RESTRUCTURE §7 layout; root MILESTONE.md canonical; move MULTI_REGION; `check_docs_layout.sh`.

#### Definition of done

- [ ] `docs/` layout per RESTRUCTURE §7
- [ ] Legacy `docs/MILESTONE.md` superseded by root `MILESTONE.md`
- [ ] `docs/MULTI_REGION.md` → `.cursor/MULTI_REGION.md` (or runbooks)
- [ ] `rg 'docs/MILESTONE'` — zero broken links
- [ ] `scripts/ci/check_docs_layout.sh` green

---

### P30 — GAP-HYG-09 — CI Tier A

| | |
| :--- | :--- |
| **Область** | CI / lefthook |
| **Сложность** | Medium |
| **Блокирует CI** | `lint` |

#### Проблема

Hygiene checks не единым gate — regressions до full-test.

#### Решение

Tier A в ci.yaml + lefthook; `make check-local` на fresh clone; document in CI_GATES.md.

#### SLA

Lint job < 8 min.

#### Definition of done

- [ ] `ci.yaml` lint: `check_docs_layout`, `check_no_milestone_refs`, `check_no_html_success`, `find_obvious_comments` with `FIND_OBVIOUS_COMMENTS_FAIL=1`
- [ ] `lefthook.yaml` mirrors Tier A
- [ ] Documented in `.cursor/CI_GATES.md`
- [ ] Fresh clone: `make check-local` runs Tier A

---

### P31 — GAP-HYG-10 — 501 stub routes

| | |
| :--- | :--- |
| **Область** | API honesty |
| **Сложность** | Medium |
| **Блокирует CI** | `openapi` |

#### Проблема

Routes возвращают 200 empty body или не реализованы — SPA ломается молча.

#### Решение

Single `stub_routes.go` 501 JSON или unregister; OpenAPI `x-implementation-status: stub`.

#### Definition of done

- [ ] Unimplemented routes not registered OR `stub_routes.go` single 501 JSON source
- [ ] `go test ./tests/contract/... -run Stub`
- [ ] OpenAPI `x-implementation-status: stub` or path removed
- [ ] `make openapi-lint`; no 200 empty body for unimplemented

---

### P32 — GAP-HYG-08 — Remove legacy taxonomy refs

| | |
| :--- | :--- |
| **Область** | Naming |
| **Сложность** | Medium |
| **Блокирует CI** | Tier A |

#### Проблема

`M14`, `M7.4` refs в go/sh/compliance — obsolete taxonomy.

#### Решение

Zero `M[0-9]+` in hand-written code; compliance → `RULE-*`; CI guard.

#### Definition of done

- [ ] `rg '\bM[0-9]+' --glob '*.go' --glob '*.sh'` — zero in hand-written code
- [x] `wire_m14_*` → `campaign_update_fanout_fault_test.go`
- [ ] Compliance matrix IDs use `RULE-*` not `M*`
- [ ] `scripts/ci/check_no_milestone_refs.sh` clean
- [ ] `docs/DEVELOPMENT.md` fault injection section — neutral names

---

### P33 — GAP-HYG-12 — `check_comments` baseline

| | |
| :--- | :--- |
| **Область** | Comment policy |
| **Сложность** | Medium |
| **Блокирует CI** | `lint` |

#### Проблема

73 R9.1 violations блокируют весь lint job.

#### Решение

Fix all violations без ослабления правил; path to P39 zero-comment mode.

#### Definition of done

- [ ] All 73 R9.1 violations fixed (32 non-ASCII, 29 unicode dash, 12 banned words)
- [ ] `bash scripts/ci/check_comments.sh` exits 0
- [ ] No rule relaxations without `CI_GATES.md` update

---

### P34 — GAP-HYG-21 — fmt vs slog audit

| | |
| :--- | :--- |
| **Область** | Logging boundaries |
| **Сложность** | Medium |

#### Проблема

Services логируют напрямую — нарушение log-at-boundary rule.

#### Решение

Services return errors; handlers/workers slog; zero fmt.Print; consistent writeServiceError attrs.

#### Definition of done

- [ ] Services return errors only; no `slog` in `service_*.go` (except lifecycle)
- [ ] Handlers/workers: structured attrs `campaign_id`, `customer_id`, `err`
- [ ] `rg 'fmt\.Print' internal/management internal/adminapi` — zero
- [ ] `writeServiceError` consistent attr keys per `code-style.mdc`

---

### P35 — GAP-HYG-20 — Receivers and `_ =`

| | |
| :--- | :--- |
| **Область** | Error handling |
| **Сложность** | Medium |

#### Проблема

Silent `_ = json.Unmarshal` — corruption risk; inconsistent receivers в flat package.

#### Решение

Explicit error branches on touch; receiver rename convention in outbox/handlers/adminapi.

#### Definition of done

- [ ] `outbox_*.go` — no `_ = json.Unmarshal` without error branch
- [ ] `handler_*.go`, `adminapi/*_handlers.go` — receivers renamed on touch
- [ ] `rg '_ = (json\.Unmarshal|w\.Write)' internal/management internal/adminapi` — zero in touched files

---

### P36 — GAP-HYG-19 — Ingestion file renames

| | |
| :--- | :--- |
| **Область** | `internal/ingestion` navigation |
| **Сложность** | Medium |

#### Проблема

Имена файлов inconsistent: `http1_*` без prefix, `*_legacy.go` дублируют canonical, `openrtb26_*` разбросаны — сложно навигировать flat package и понять ownership. Новые контрибьюторы путают hot path entry points.

#### Решение

Только `git mv` без logic changes: `http1_*` → `handler_http1_*`; merge/delete legacy; contiguous `openrtb26_*`. `make test-alloc-gate` должен быть bit-identical.

#### Тестирование

`make test-alloc-gate` unchanged; `go test ./internal/ingestion/... -short`.

#### Definition of done

- [ ] `git mv` only — no logic changes
- [ ] `http1_*` → `handler_http1_*`
- [ ] `*_legacy.go` merged or deleted
- [ ] `openrtb26_*` contiguous
- [ ] `make test-alloc-gate` unchanged

---

### P37 — GAP-HYG-15 — `domains.go` registry

| | |
| :--- | :--- |
| **Область** | `internal/management` coverage |
| **Сложность** | Low |
| **Блокирует CI** | `full-test` |

#### Проблема

Flat `package management` содержит сотни файлов; `domains.go` registry не покрывает новые production files (`dry_run.go`, `service_cohorts.go`, `service_gtax.go`, `vendor_telemetry.go`). Тест `TestDomainRegistry_AllProductionFilesMapped` красный — блокер full-test.

#### Решение

Добавить missing mappings в `domains.go`; сохранить или поднять threshold `make management-domain-coverage`.

#### Definition of done

- [ ] `domains.go` maps: `dry_run.go`, `service_cohorts.go`, `service_gtax.go`, `vendor_telemetry.go`
- [ ] `TestDomainRegistry_AllProductionFilesMapped` green
- [ ] `make management-domain-coverage` ≥ previous threshold

---

### P38 — GAP-HYG-14 — golangci-lint 10 issues

| | |
| :--- | :--- |
| **Область** | Lint gate |
| **Сложность** | Low |
| **Блокирует CI** | `lint` |

#### Проблема

`make lint` падает на 10 конкретных issues: `ineffassign` в quorum batch, `errcheck` в bpf-collector, 8× `unused` в rtb/ivt/management/bpf. Блокирует все PR до fix.

#### Решение

Fix each reported issue: remove dead assignments, handle errors, delete or use unused symbols. No nolint без причины.

#### Definition of done

- [ ] `ineffassign` in `quorum_batch.go`
- [ ] `errcheck` in `cmd/bpf-collector`
- [ ] 8× `unused` removed in rtb/ivt/management/bpf
- [ ] `make lint` exits 0

---

### P39 — GAP-HYG-16 — Comment purge + zero-comment CI

| | |
| :--- | :--- |
| **Область** | Zero-comment policy |
| **Сложность** | Low |
| **Блокирует CI** | Tier A |

#### Проблема

Кодовая база содержит godoc, inline comments, backlog refs в comments. Политика R9.3: self-documenting code only; `STRICT_NO_COMMENTS=1` должен быть green в CI.

#### Решение

Удалить все non-directive comments из `internal/`, `cmd/`, `pkg/`. Enable `STRICT_NO_COMMENTS=1` и `FIND_OBVIOUS_COMMENTS_FAIL=1` в ci.yaml lint job.

#### Definition of done

- [ ] All non-directive comments deleted from `internal/`, `cmd/`, `pkg/`
- [ ] `rg 'GAP-|P[0-9]{2}|\bM[0-9]' internal/ pkg/ cmd/ tests/` — zero in comments
- [ ] Forbidden words in comments: backlog IDs, legacy taxonomy tags
- [ ] `STRICT_NO_COMMENTS=1` in `ci.yaml` lint job
- [ ] `FIND_OBVIOUS_COMMENTS_FAIL=1` → 0 hits
- [ ] Both `check_comments.sh` modes exit 0

---

### P40 — GAP-HYG-13 — Skip sqlc in check_comments

| | |
| :--- | :--- |
| **Область** | CI false positives |
| **Сложность** | Low |
| **Блокирует CI** | `lint` |

#### Проблема

`check_comments.sh` сканирует generated `internal/ingestion/sqlc/` — false positives (4 hits). Hand-written SQL в `queries/` должен сканироваться.

#### Решение

Skip list в `check_comments.sh` для generated dirs per `CI_GATES.md`; document false-positive count.

#### Definition of done

- [ ] `check_comments.sh` skips `internal/ingestion/sqlc/` and generated dirs per `CI_GATES.md`
- [ ] Still scans hand-written `queries/*.sql`
- [ ] False-positive count documented

---

### P41 — GAP-HYG-29 — Broker ops hygiene

| | |
| :--- | :--- |
| **Область** | `deploy/broker` |
| **Сложность** | Low |

#### Проблема

Broker HA lab compose имеет typo flags; `BROKER_*` env не в `.env.example`; port 9093 conflict с region-proxy не документирован; health endpoint не проверен.

#### Решение

Fix compose command/env; document all vars; README с HA lab и port conflict; curl health smoke.

#### Definition of done

- [ ] `deploy/broker/docker-compose.yaml` — valid command/env
- [ ] `.env.example` all `BROKER_*`
- [ ] `deploy/broker/README.md` — HA lab, port 9093 conflict
- [ ] `curl http://127.0.0.1:8084/health` OK in HA profile

---

### P42 — GAP-HYG-23 — Dockerfile layout

| | |
| :--- | :--- |
| **Область** | `deploy/docker` |
| **Сложность** | Low |

#### Проблема

Несколько Dockerfile без matrix «file → targets → ports»; edge-xdp build steps не documented; optional root multi-stage не verified.

#### Решение

`deploy/docker/README.md` matrix; edge-xdp manual steps; optional root Dockerfile builds tracker/management/processor.

#### Definition of done

- [ ] `deploy/docker/README.md` — Dockerfile → targets → ports
- [ ] edge-xdp manual build steps
- [ ] Optional root `Dockerfile` multi-stage `tracker`, `management`, `processor` green

---

### P43 — GAP-HYG-24 — Scripts deletion pass

| | |
| :--- | :--- |
| **Область** | `scripts/` cleanup |
| **Сложность** | Low |

#### Проблема

`scripts/log-evacuation/` obsolete; `run_broker_load` thin wrapper дублирует README; broken refs после restructure.

#### Решение

Remove or merge log-evacuation; delete wrapper; verify zero broken refs with rg.

#### Definition of done

- [ ] `scripts/log-evacuation/` removed or merged
- [ ] `run_broker_load` wrapper deleted; canonical in README
- [x] `run_operator_drill.sh` / `run_malformed_traffic.sh` replace legacy entrypoints
- [ ] `rg 'log-evacuation|run_broker_load'` — zero broken refs

---

### P44 — GAP-HYG-11 — `git filter-repo` root binaries

| | |
| :--- | :--- |
| **Область** | Git history hygiene |
| **Сложность** | Low |

#### Проблема

Root-level committed binaries (`processor`, `payment`, etc.) раздувают clone size >1MB blobs; `.gitignore` не полный.

#### Решение

`git filter-repo` после archive branch; update `.gitignore`; contributor re-clone procedure.

#### Definition of done

- [ ] Root binaries removed from history via `git filter-repo`
- [ ] Archive branch `archive/pre-filter-repo` pushed before rewrite
- [ ] `.gitignore` includes `/*.exe`, `/*-xdp`, named binaries
- [ ] Contributor re-clone procedure documented
- [ ] `git rev-list --objects --all | rg 'processor$'` — no blob >1MB at root

---

### P45 — GAP-PROD-10 — Community vs Pro split

| | |
| :--- | :--- |
| **Область** | Distribution / licensing |
| **Сложность** | Low |

#### Проблема

Public repo scope не формализован: что open (docs, OpenAPI) vs Pro (ingest source, ML models CDN). CI не собирает stripped multi-arch binaries для release.

#### Решение

Update `LICENSE_COMMERCE.md` § Distribution; CI builds linux/amd64+arm64 stripped; document models CDN license requirement.

#### Definition of done

- [ ] `docs/LICENSE_COMMERCE.md` § Distribution final policy
- [ ] CI builds `linux/amd64`, `linux/arm64` stripped binaries
- [ ] Public repo: docs, OpenAPI, demos — no Pro ingest source
- [ ] License required for `models/` CDN download documented

---

### P46 — GAP-HYG-17 — Git history soft-squash (optional)

| | |
| :--- | :--- |
| **Область** | Git history |
| **Сложность** | Low |
| **Статус** | Optional — requires product decision |

#### Проблема

Длинная нелинейная history затрудняет onboarding; optional squash после explicit decision.

#### Решение

Archive branch + git bundle before squash; CONTRIBUTING notes linear history.

#### Definition of done

- [ ] `archive/pre-squash-YYYYMMDD` branch + `git bundle create`
- [ ] Squash only after explicit product decision
- [ ] `CONTRIBUTING.md` notes linear history

---

### P47 — GAP-HYG-31 — Self-hosted paradigm (residual code)

| | |
| :--- | :--- |
| **Область** | Paradigm alignment |
| **Сложность** | Low |
| **Статус** | Docs shipped; code gaps in P06, P02 |

#### Проблема

`SELF_HOSTED.md` и ARCHITECTURE описывают JSON-only + SPA model, но код всё ещё содержит HTMX (P06) и CH-based metering (P02).

#### Решение

Close P06 and P02 — code catches up to documented paradigm.

#### Definition of done

- [x] `SELF_HOSTED.md`, ARCHITECTURE V/O/A documented
- [ ] GAP-HYG-04 code (P06)
- [ ] GAP-HYG-30 code (P02)

---

### P48 — GAP-PROD-12 — Brand boundary (BidShard UI vs neutral runtime)

| | |
| :--- | :--- |
| **Область** | Branding / white-label |
| **Сложность** | Low |
| **Зависимости** | P07, P09 |
| **Блокирует CI** | `lint` |

#### Проблема

Codename `espx`/`ESPX_*` размазан по env, metrics, ingress schema, Docker volumes, CLI. Product brand — **BidShard** (`bidshard.com`). Brand должен быть только в UI/marketing; runtime, APIs, operator config — neutral (`ADSTACK_*`, `ad_*` metrics, `native_v1` wire).

Hot path (`internal/ingestion/`) не должен содержать brand strings.

#### Решение

1. **`pkg/branding`:** `ProductName`, `VendorName`, `SiteURL`, `SupportEmail` — overridable `BRAND_*`; no import in ingestion.
2. **`GET /api/v1/meta`:** unauthenticated JSON для SPA/external UIs.
3. **SPA/CLI:** visible strings from meta; released binary `bidshard`; `espx` shim deprecation once.
4. **Env dual-read one release:** `ESPX_*` → `ADSTACK_*`; `espx_native` → `native_v1`; compose volumes `platform_*`.
5. **Metrics rename:** `espx_*` → `ad_*` (one release duplicate or static rename).
6. **CI:** `check_brand_boundary.sh` — bidshard only in allowlist dirs; zero espx in ingestion.

#### Паттерны

`pkg/branding` for UI strings; `ADSTACK_*` env; `ad_*` metrics.

#### Definition of done

- [ ] `pkg/branding` with `BRAND_*` overrides; no import in `internal/ingestion/`
- [ ] `GET /api/v1/meta` — unauthenticated JSON
- [ ] SPA strings from meta or `VITE_BRAND_*`
- [ ] Released binary `bidshard`; `espx` shim deprecation once
- [ ] Env aliases dual-read; `.env.example` target names only
- [ ] Metrics `espx_*` → `ad_*`; `IsNativeV1Ingress()`
- [ ] `scripts/ci/check_brand_boundary.sh` in lint/Tier A
- [ ] `rg -i 'bidshard' internal/ cmd/ pkg/ --glob '!pkg/branding/**'` — zero

---

### P49 — GAP-ML-01 — Fraud ML platform plumbing

| | |
| :--- | :--- |
| **Область** | ML cold path (offline/train) |
| **Сложность** | Medium–low |
| **Блокирует CI** | `full-test` |

#### Проблема

Runtime inference shipped (`internal/fraudscoring`, `cmd/fraud-scorer`, `ml_features_1m`, shadow scores). Offline ML — bootstrap stub на synthetic data. Нет:

- Shared feature contract Go ↔ Python (drift risk).
- Artifact validation gate перед `ml_model_versions` → SYNCING.
- Export/eval pipeline до появления labeled data.
- Real `fit` path с time-based split.

Risk: unvalidated model в prod; train/prod vector mismatch.

**Boundary:** Go = serving + orchestration; Python = export/train/eval; tracker **must not** import `internal/fraudscoring`.

#### Решение

**Phase A (pre-label):**
- `feature_spec.go` + `ml/feature_spec.py` — 7 dims, golden fixtures; CI cross-lang agreement.
- `cmd/ml-validate` — NFeatures==7, fixture scores, metadata hash.
- Python skeletons: `gen_fixtures.py`, `train.py validate`, `export_features.py`, `evaluate.py`.
- Optional `cmd/ml-replay` for ops.
- Prometheus `ml_shadow_score`; admin JSON model version/hash.

**Phase B (proxy labels):**
- Documented export/eval; weekly shadow precision runbook.

**Phase C (labeled training):**
- `ml/train.py fit` — parquet in, time-based split, `model.txt` + `metadata.json`.
- Labels contract documented; manual/scheduled fit + validate + shard sync.
- Isolation Forest defer until `fraudscoring_onnx` tested.

#### SLA

`ml-validate` < 5 s; export 7d < 5 min dev CH; inference benches unchanged.

#### Тестирование

- `go test ./internal/fraudscoring/... ./cmd/ml-validate/...`
- `python3 ml/train.py validate` after bootstrap
- `TestTrackerDepGraphExcludesFraudScoringRuntime`
- ML stays cold path only

#### Definition of done

**Phase A**
- [ ] `feature_spec.go` + `ml/feature_spec.py` — 7 dims, golden fixtures
- [ ] CI: Go `ToVector()` ≡ Python `row_to_vector()`
- [ ] `cmd/ml-validate` — NFeatures==7, fixture scores, metadata hash
- [ ] `ml/gen_fixtures.py`, `ml/train.py validate`, `ml/export_features.py`, `ml/evaluate.py`
- [ ] Optional `cmd/ml-replay`
- [ ] Prometheus `ml_shadow_score`; admin JSON model version/hash

**Phase B**
- [ ] Export/eval documented; weekly shadow precision runbook

**Phase C**
- [ ] `ml/train.py fit` — time-based split; `model.txt` + `metadata.json`
- [ ] Labels contract documented
- [ ] Manual/scheduled fit + validate + shard sync

**Out of scope:** feature store, AutoML, hot-path per-request model.

---

## Отложено и отменено

### Отложено (UI)

| ID | Задача | Условие снятия |
| :--- | :--- | :--- |
| **GAP-PROD-01** | Buyer/finance dashboards (external React SPA) | После P07 (GAP-PROD-02) |
| **GAP-OPS-04** | Queue monitoring UI (DLQ, spool) | P16 dashboard или external Grafana |

---

#### GAP-PROD-01 — Buyer/finance dashboards

| | |
| :--- | :--- |
| **Область** | External SPA / finance UX |
| **Приоритет** | Deferred |

##### Проблема

Buyer и finance роли нуждаются в dashboards: campaign stats, balance, spend forecast, invoice history. Server-rendered HTMX для этого deprecated (P06). `internal/adminapi/` reporting routes частично 501. Внешний frontend должен потреблять только JSON `/api/v1` — но dedicated buyer/finance UI не bundled в management SPA (это operator admin UI в P07).

##### Решение

Отдельное React SPA (или module в partner portal) потребляет:
- `GET /api/v1/campaigns/{id}/stats`
- Balance и forecast endpoints
- RBAC/masking из P04 для buyer role

Без server-rendered HTML. Запускается после P07 когда JSON API contract стабилен.

##### Definition of done (когда в scope)

- [ ] External React SPA: `/api/v1/campaigns/{id}/stats`, balance, forecast
- [ ] No server-rendered HTML
- [ ] RBAC buyer role integration tests

---

#### GAP-OPS-04 — Queue monitoring UI

| | |
| :--- | :--- |
| **Область** | Ops visibility |
| **Приоритет** | Deferred |

##### Проблема

Оператор не видит DLQ depth, broker spool size, stream lag в UI — только raw Prometheus или redis CLI. P16 даёт lite dashboard, но без dedicated queue panels. Full Grafana-style UI out of scope для self-hosted v1.

##### Решение

Либо panels в P16 embedded dashboard (DLQ depth, spool bytes, `ad_processor_stream_lag_seconds`), либо external Grafana dashboard JSON в `deploy/monitoring/`. Не отдельный HTMX server UI.

##### Definition of done (когда в scope)

- [ ] DLQ depth visible in P16 dashboard or Grafana
- [ ] Spool size and stream lag cards with alert thresholds documented

### Отменено

| ID | Причина |
| :--- | :--- |
| **GAP-HYG-01** | Split `internal/ingestion/` — conflicts flat package R1 |
| **GAP-HYG-02** | Split `internal/management/` — conflicts flat package R1 |

#### GAP-HYG-01 / GAP-HYG-02 — Package splits (cancelled)

**Проблема (историческая):** Предлагалось разбить flat packages на nested subpackages (`filter/`, `fraud/`, …) для навигации.

**Почему отменено:** Конфликт с `code-style.mdc` R1 (flat service packages), import cycles, hot-path coupling. Decouple через `internal/campaignmodel` и file-prefix navigation (R2) вместо subpackages.

**Решение:** Не выполнять. Использовать `campaignmodel` shared types и filename tags.

### Shipped (справочно)

| ID | Summary |
| :--- | :--- |
| GAP-DATA-01 | PII hash before CH (`pkg/piihash/`) |
| GAP-RTB-10 | VAST 4.2 + creative auction |
| GAP-RTB-11 | Pre-auction gates (daypart + fcap) |
| GAP-RTB-12a–c | CTV gtax, admin dry-run, A/B cohorts |
| GAP-OPS-03 | CHQuery governance |
| GAP-ENG-02 | Broker + region-proxy compose |
| GAP-DB-01/02/03 | Disk gate (`iogate`), WAL, weighted processor gates |
| GAP-CMP-01 | Edge tarpit + compliance matrix |
| GAP-ENG-03 | Vendor telemetry probes |
| GAP-PAY-01 | Crypto payment gateway |
| GAP-PROD-03 (OpenAPI) | OpenAPI 3 + CI drift (distinct from P12 SKU YAML) |

Полная таблица: [docs/DEVELOPMENT.md — Completed roadmap](docs/DEVELOPMENT.md#completed-roadmap).

---

## Закрыто (справочник)

См. [Отложено и отменено → Shipped](#shipped-справочно).

---

## Рекомендуемые PR-срезы

```text
PR-CI-A   P37–P40 (HYG-15,14,16,13)     # lint green baseline
PR-CI-B   P33 (HYG-12)                  # check_comments
PR-CI-C   P26 + P15 (HYG-05, HYG-06)    # full-test ingestion/management
PR-CI-D   P29 + P30 (HYG-07, HYG-09)    # docs + CI Tier A
PR-CI-E   P10 (HYG-22)                  # scripts hygiene
PR-API    P05 + P06 (HYG-03, HYG-04)    # adminapi + HTMX removal
PR-UI     P07 (PROD-02)                 # bundled SPA (after P05/P06)
PR-CORE   P02 (HYG-30)                  # settlement / recon / meter
PR-CTRL   P01 (HYG-27)                  # symmetric control (last arch piece)
PR-TERM   P24 (HYG-18)                  # fault rename verification
```

---

## CI job map (текущие блокеры)

| Job | Failing step | Minimum fix |
| :--- | :--- | :--- |
| `lint` | `check_comments.sh` | P33, P40 |
| `lint` | `make lint` | P38 |
| `full-test` | tests | P26, P15, P37 (+ P33–P38) |
| `resilience` | Docker stack | separate from hygiene list |
| `terraform-validate` | — | OK |

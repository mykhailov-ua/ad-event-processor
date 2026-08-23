# Целевая структура BidShard

Канонический макет воркспейса **bidshard**. Организация по смыслу (заимствована из `lead-intent-processor`)

---

## 1. Семантические паттерны
Паттерны из lead-intent-processor для внедрения в ad-event-processor:

| Паттерн | Пример lead-intent | Роль / смысл |
| :--- | :--- | :--- |
| **Пакеты по этапам** | `ingest` -> `filter` -> `pipeline` -> `scoring` -> `sink` | Один пакет = один этап конвейера с минимальным публичным API |
| **Точка композиции** | `internal/app/deps.go` | Единое место сборки зависимостей. Тонкие бинарные файлы |
| **Разделение warm/cold** | `warmpath/` vs `coldpath/` | Работа во время запроса vs асинхронное обогащение / батчинг |
| **Словарь домена** | `entity/`, `model/` | Типы данных и инварианты без привязки к протоколам и БД |
| **Плагины источников** | `internal/sources/<name>/` | Опциональные интеграции под общим реестром |
| **Адаптеры вывода** | `sink/` | Унифицированный интерфейс для разных хранилищ (Mongo, JSONL, webhooks) |
| **Вложенные продуктовые слои**| `internal/crm/{app,admin,store,webhook}` | Изолированный субпродукт со своими слоями |
| **Тонкий cmd** | 4 бинарника | `main` только парсит флаги и вызывает `app.Run` |
| **Конфиг на границе** | `config/` + `config/env/` | Шаблоны env-переменных лежат рядом с кодом |
| **Папки данных** | `data/`, `fixtures/`, `testdata/` | Рабочие и тестовые данные отделены от `internal/` |
| **Комментарии** | `entity/doc.go`, doc-комментарии полей | Только **почему** и **инварианты** для экспортируемых контрактов |

### Антипаттерны ad-event, которые нужно устранить:
- God struct (`controlplane.Service` на 40+ доменов).
- Плоский пакет на 500 файлов (`internal/controlplane/*`).
- Дублирование логики (например, fraud размазан по ingestion, controlplane и fraud).
- Мусор в корне репозитория (бинарники, `escape_heap_gate.txt`).
- Несоответствие `ARCHITECTURE.md` реальной структуре папок `internal/`.

---

## 2. Структура воркспейса (верхний уровень)
```
bidshard/
├── README.md                      # Карта экосистемы
├── deploy/
│   └── geoip/                     # Общие артефакты MaxMind
├── var/                           # Локальные временные файлы (в gitignore)
│
├── lead-intent-processor/         # Сбор лидов + CRM
└── ad-event-processor/            # Рекламный процессор событий
```

**Правила:**
- В корне `bidshard/` нет общего модуля Go.
- Общие доки лежат в корне, специфичные для продуктов — в их `docs/`.
- Файлы лицензий и секреты в git не комитятся, используется `var/`.

---

## 3. Структура lead-intent-processor (референс)
```
lead-intent-processor/
├── README.md
├── Makefile
├── go.mod
├── pyproject.toml                 # Сайдкар Telethon
├── requirements*.txt
├── cmd/                           # Только тонкие точки входа
├── config/
│   ├── env/                       # Шаблоны env (без секретов)
│   └── caddy/
├── internal/
│   ├── app/                       # Инициализация зависимостей и запуск CLI
│   ├── pipeline/                  # Воркеры, очереди, процессинг
│   ├── ingest/ -> filter/ -> scoring/ -> enrich/ -> dedup/  # Этапы
│   ├── warmpath/                  # Асинхронный анализ Gemini
│   ├── coldpath/                  # Отчеты, аудит
│   ├── sources/                   # Плагины (forum, reddit и др.)
│   ├── sink/                      # Вывод (Mongo, JSONL, webhooks)
│   ├── entity/                    # Доменная модель
│   └── crm/                       # Изолированный CRM-слой
├── pkg/                           # Шаринг без импорта internal
│   └── bpfenv/
└── docker-compose*.yaml
```

---

## 4. Целевая структура ad-event-processor
Цель: повторить разделение на этапы, домены и единую точку композиции из lead-intent, сохранив SLA горячего пути (hot-path).

### 4.1 Корень репозитория
```
ad-event-processor/
├── README.md                      # Быстрый запуск для разработки
├── Makefile | Taskfile.yaml
├── go.mod
├── buf.yaml | sqlc.yaml
├── docker-compose.yaml            # Включает конфиги из deploy/compose/
├── .env.example
├── api/                           # Только Proto-файлы
├── cmd/                           # Точки входа
├── config/
│   └── env/                       # Шаблоны конфигурации
├── internal/                      # Бизнес-логика
├── pkg/                           # Общие библиотеки (без импортов internal/)
├── web/                           # Панель администратора
├── model/                         # Python ML (вне hot path Go)
├── deploy/                        # Настройки деплоя
├── scripts/                       # Скрипты автоматизации
├── docs/                          # Документация
├── tests/                         # Интеграционные и e2e тесты
├── fixtures/ | testdata/          # Тестовые фикстуры
└── data/                          # Локальные данные (в gitignore)
```
**В корне запрещены:** скомпилированные бинарники, `escape_heap_gate.txt`, ad-hoc `.env` файлы (кроме `.env.example`).

### 4.2 cmd/ (бинарные файлы по ролям)
Каждая точка `cmd/<name>/main.go` только парсит флаги, загружает конфиг и вызывает `internal/app` или `OpenModule`.

- **Hot path:** `tracker`, `campaign-shard`
- **Cold ingest:** `processor`, `broker`, `log-shipper`, `log-compactor`, `log-evacuator`
- **Control plane:** `control`, `admin`, `postback-sender`, `dlq`
- **Fraud / ML:** `fraud-scorer`, `ivt-detector`, `ml-validate`, `ml-replay`
- **Edge:** `edge-xdp`, `edge-xdp-fault`, `edge-bpf-sync`, `region-proxy`
- **Licensing / vendor:** `license-issue`, `license-asset-seal`, `trial-registry`, `vendor-trial-bot`
- **Install / ops:** `installer`, `alertmanager-telegram`, `migrate-cold-path`
- **Dev / CI:** `loadgen`, `load-report`, `perf-gate`, `bpf-collector`, `patch-vtproto-hotpath`
- **CLI:** `ad-event-processor`

### 4.3 internal/ (целевая структура)
```
internal/
├── app/                           # Композиция (control, tracker, processor, deps)
├── config/                        # Типизированный конфиг Go
│
├─── HOT PATH (tracker :8181-8184) ───────────────────────────────────
├── ingest/                        # Выделен из ingestion/. 0-alloc контракт. HTTP1/2, parse, cors
├── filter/                        # Фильтры кликов/трекинга (license -> geo -> unified Lua -> localquanta)
├── stream/                        # Запись событий в шину (producer, admission, rollback)
├── track/                         # Обработчики /track, /click, /tg/*
├── registry/                      # Кэш кампаний в памяти
├── rtb/                           # Аукцион, каталог, бюджеты
├── openrtb/                       # Парсинг и биддинг OpenRTB
│
├─── EDGE ────────────────────────────────────────────────────────────
├── edge/                          # eBPF, XDP, блокировки
│
├─── COLD PATH DOMAINS (выделены из controlplane/) ─────────────────────
├── admin/                         # Роутинг, авторизация и middleware админки
├── campaign/                      # Управление кампаниями, лимитами и pacing
├── customer/ | team/ | publisher/ # Домены пользователей и сущностей
├── commercial/ | selfserve/       # Бизнес-правила рекламодателей
├── billing/ | payment/ | ledger/  # Финансы, счета, проводки
├── identity/ | notify/            # Сессии, уведомления
├── fraud/                         # Антифрод (score, admin, detector, outbox)
├── reports/ | ops/                # Отчеты ClickHouse, DLQ, инциденты
├── telegram/ | postback/          # Интеграции и отправка постбэков
├── licensing/ | recon/            # Лицензирование, сверка затрат
├── rtbadmin/ | platform/          # Настройки RTB полов, шаблоны, здоровье доменов
├── forecasting/ | export/         # Прогнозирование, экспорт данных
├── integrations/                  # Costsync и внешние интеграции
├── outbox/                        # Общая очередь транзакционных событий
├── authz/                         # Авторизация (RBAC)
│
├─── PROCESSING & ANALYTICS ──────────────────────────────────────────
├── processor/                     # Обработчик событий холодного пути
├── logpipeline/                   # Обработка логов
├── clickhouse/                    # Клиент ClickHouse
│
├─── SHARED DOMAIN & DATA ──────────────────────────────────────────────
├── domain/                        # Общие доменные типы, генерация sqlc, инварианты
├── database/                      # Пулы подключений Postgres, Redis, CH
├── dedup/ | metrics/ | telemetry/ # Дедупликация, метрики, трассировка
├── installer/ | trialregistry/    # Установщик, триалы
└── testutil/                      # Хелперы для тестов (запрещены в hot path)
```
После миграции пакеты `internal/controlplane/` и `internal/ingestion/` полностью удаляются.

### 4.4 pkg/ (вспомогательные библиотеки)
Библиотеки общего назначения. **Запрещено импортировать internal/ из pkg/**.
Сюда входят: `broker`, `regionproxy`, `coldpath`, `logger`, `lifecycle`, `money`, `netaddr`, `gnetutil`, `clientip`, `dedupkey`, `pgfailover`, `piihash`, `faultproof`.

### 4.5 deploy/
```
deploy/
├── compose/                       # docker-compose.yaml и варианты для тестов
├── profiles/                      # Профили запуска (appliance, enterprise, tools)
├── docker/                        # Docker-файлы компонентов
├── nginx/ | edge/ | redis/ ...    # Конфиги инфраструктуры
└── vendor/                        # Шаблоны лицензий
```

---

## 5. Карта миграции ad-event-processor
Перенос файлов без изменения логики:

### 5.1 controlplane -> доменные пакеты
- `campaign_*`, `service_campaign*` -> `internal/campaign/`
- `customers_*`, `service_customers*` -> `internal/customer/`
- `team_*`, `rbac*` -> `internal/team/` + `internal/authz/`
- `publisher_*` -> `internal/publisher/`
- `commercial_*`, `buyer_*` -> `internal/commercial/`
- `selfserve_*` -> `internal/selfserve/`
- `billing_*`, `service_crypto_billing*` -> `internal/billing/`
- `fraud_*`, `service_fraud*`, `ml_blacklist*` -> `internal/fraud/admin/`
- `reports_*` -> `internal/reports/`
- `ops_*` -> `internal/ops/`
- `tg_*` -> `internal/telegram/`
- `recon_*`, `global_spend_*` -> `internal/recon/`
- `rtb_*` (admin) -> `internal/rtbadmin/`
- `platform_*`, `domain_health_*`, `flow_*`, `template_*` -> `internal/platform/`
- `forecast_*`, `margin_*`, `smart_alerts_*` -> `internal/forecasting/`
- `export_*` -> `internal/export/`
- `integration_schema_*`, `cost_sync_*` -> `internal/integrations/`
- `outbox*.go` (core worker) -> `internal/outbox/`
- `register.go`, `handler.go`, `middleware.go`, `http_*` -> `internal/admin/`
- `serve.go` -> `internal/app/control.go`

### 5.2 ingestion -> пакеты горячего пути (hot path)
- `handler*.go`, `http1_*`, `http2_*` -> `internal/ingest/`
- `filter*`, `unified_*`, `local_quanta*` -> `internal/filter/`
- `broker_*`, `producer*`, `stream_*` -> `internal/stream/`
- `track_*`, `click_*`, `tg_click*` -> `internal/track/`
- `registry_*`, `segment_*`, `settings_*` -> `internal/registry/`
- `rtb_*` -> `internal/rtb/`
- `openrtb_*` -> `internal/openrtb/`
- `processor.go`, `ch_*`, `settlement_*` -> `internal/processor/`

### 5.3 Антифрод
- `internal/fraud/*` -> `internal/fraud/score/`
- `internal/controlplane/fraud_*` -> `internal/fraud/admin/`
- `internal/ingestion/fraud_*`, `filter_fraud_boost*` -> `internal/filter/fraudboost/`

---

## 6. Правила пакетов
- **Максимальный размер:** До ~40 файлов `.go` или ~8 тыс. строк кода на пакет. При превышении — разделять.
- **Один домен — один пакет:** Не плодить префиксы вроде `service_*` в мега-папках.
- **Интерфейс инициализации (`module.go`):** Каждый холодный домен экспортирует `OpenModule(ctx, cfg) (*Module, error)` и метод `Close()`.
- **Ограничения горячего пути:** Никаких `context.WithTimeout`, записей в PG/CH и импортов `internal/fraud/score` из `ingest`.
- **Запрещенные директории:** Никакой лишней церемонии вида `dto/`, `usecase/`, `providers/`, `repositories/`.
- **Каждый домен обязан содержать `doc.go`** с описанием назначения, инвариантов и зоны ответственности пакета.

### Направление зависимостей:
```
cmd -> app -> {admin, campaign, fraud/admin, ...}
                -> outbox, database, domain
tracker cmd -> app -> ingest -> filter -> stream -> track
                       -> domain, registry, rtb (только чтение снапшотов)
processor cmd -> app -> processor -> fraud/score, clickhouse
```
**Запрещено:** `ingest` -> `campaign` (из hot-path в cold admin), `fraud/score` -> `ingest`, `pkg/*` -> `internal/*`.

---

## 7. Стандарт комментариев
Комментарии должны описывать **почему** код работает именно так, его инварианты и граничные случаи, а не пересказывать его суть.

```go
// Package campaign manages creation, editing, and pacing of campaign budget limits.
// Changes enqueue outbox events in the same Postgres transaction.
// Budget invariant: current spend must not exceed the budget limit (see domain.AssertBudgetInvariant).
package campaign
```

```go
// SpendMicros deducts micro-units from the campaign budget on the shard master via Lua.
// Returns ErrBudgetExceeded if the budget limit would be exceeded.
// Guarantees atomicity—no partial deduction without rollback.
func (s *Service) SpendMicros(...)
```

---

## 8. Шаги миграции
1. Выделение `internal/app/`, `internal/admin/` и `internal/outbox/`.
2. Выделение `internal/telegram/`.
3. Выделение `internal/reports/` и `internal/export/`.
4. Выделение `internal/fraud/admin/` и перенос `internal/fraud` -> `fraud/score`.
5. Выделение `internal/campaign/`, `customer/`, `team/`.
6. Выделение `internal/ops/`, `recon/`, `billing/`.
7. Разделение `ingestion/` -> `ingest/`, `filter/`, `stream/`, `track/`.
8. Слияние `rtb` / `openrtb`.
9. Удаление пустых директорий `controlplane/` и `ingestion/`.
10. Обновление правил анализаторов, индексов доки и CI гейтов.

**Проверка после каждого шага:**
```bash
go build ./cmd/control ./cmd/tracker ./cmd/processor
go test ./internal/<измененный_пакет>/... -short -count=1
```

---

## 9. Успешность миграции (критерии)
- Удалены `controlplane/` и монолитный `ingestion/`.
- `go test ./internal/campaign/...` проходит без сборки тяжелых зависимостей (fraud, reports, ops).
- Tracker импортирует только разрешенные hot-path пакеты.
- В корне репозитория чисто (нет бинарников, логов, конфигов окружения).
- Описанная архитектура полностью соответствует реальным пакетам в `internal/`.

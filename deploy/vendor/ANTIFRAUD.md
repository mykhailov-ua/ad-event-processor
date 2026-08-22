# Антифрод BidShard (краткий обзор)

Внутренний документ для вендора и пресейла. Технические детали: [ARCHITECTURE.md](../../docs/ARCHITECTURE.md), [XDP.md](../../docs/XDP.md). Лимиты по SKU: [sku.yaml](./sku.yaml), [SALES_KIT.md](./SALES_KIT.md).

## Как устроено

Антифрод разделён на **горячий путь** (каждый клик/импрессия, p99 < 80 ms) и **холодный путь** (аналитика, ML, блокировки с задержкой минут).

| Слой | Где | Задержка |
| :--- | :--- | :--- |
| Периметр (edge / XDP) | Nginx Lua, опционально eBPF | микросекунды |
| Сигналы и скоринг | Tracker `FilterEngine` | наносекунды–миллисекунды |
| Правила IVT + ML | `ivt-detector`, `fraud-scorer` | 5–15 мин (батч) |
| Оператор | Admin UI, API | по запросу |

**Важно:** LightGBM/ONNX **не** вызывается на каждый `/track`. Модель считает батчами; на трекер попадает только снимок `ml:score:boost:{campaign_id}` в Redis.

## Три уровня реакции

| Уровень | Условие | Поведение |
| :--- | :--- | :--- |
| **L1 — отказ** | 2+ сильных сигнала **или** L3 blacklist | `/track` → **202** (тихий accept), бюджет не списывается; `/click` → **204** |
| **L2 — shadow** | 1 сильный / слабый сигнал / tier suspect–block | Событие принято, помечено `shadow_event`; в отчётах и CH видно как подозрительное |
| **L3 — blacklist** | IP в `blacklist:fraud` | Сильный сигнал `l3_blocklist`; на edge может быть **403** до трекера |

**Ghost IVT** (`ghost_ivt_enabled` на кампании): при ML-угрозе включается через outbox; подозрительный трафик принимается «призрачно» (202/204), без списания и без постбеков CAPI.

## Сигналы на горячем пути

Накапливаются в `fraud_score` (0–100) и `fraud_reason`:

| Код | Сигнал | Вес | Сила |
| :--- | :--- | ---: | :--- |
| `datacenter_ip` | DC / анонимный IP (GeoIP) | 45 | L1 |
| `low_ttc` | Time-to-click ниже порога | 45 | L1 |
| `tls_blocklist` | JA3/JA4 в блок-листе отпечатков | 45 | L1 |
| `l3_blocklist` | IP в fraud blacklist | 100 | L3 |
| `missing_imp_ts` | Клик без метки импрессии | 35 | L2 |
| `device_mismatch` | Несовпадение device / TLS / UA | 35 | L2 |

Пороги tier на кампании: **pass → suspect → ivt → block** (по умолчанию 30 / 60 / 80 / 100). Пресеты: `conservative`, `balanced`, `aggressive`.

ML boost: `fraud_score += ml:score:boost` (снимок из Redis, ~90 ns на проверку).

## Цепочка фильтров `/track`

```
License → Emergency Breaker → Geo → Schedule → Segment → VPP → Fraud (сигналы) → Device → Consent → UnifiedFilter (Redis Lua)
```

Связанные проверки в той же цепочке:

- **Geo** — страна / таргетинг (не IVT, но отсекает до Redis).
- **VPP** — smart pacing по ratio из Redis.
- **FraudBlacklistFilter** — `blacklist:fraud`.
- **FraudFilter** — анонимный / DC IP.
- **DeviceFilter** — TLS fingerprint, device mismatch.
- **Local TTC** — low TTC / missing impression timestamp (в т.ч. local quanta full-skip).
- **UnifiedFilter (Lua)** — бюджет, dedup, fcap, pacing, TTC в Redis.

## Периметр (до трекера)

**Nginx / OpenResty** (`access-check.lua`, `edge-fraud-tier.lua`):

- IP blacklist (`blacklist:manual`, `blacklist:auto`, `blacklist:fraud`) — sync из Redis shard 0.
- Circuit breaker и rate limit на edge.
- Блок по fraud tier score на edge (403/429).
- DFA разбор тела на edge.

**Опционально Enterprise — XDP/eBPF** (`ebpf_xdp_edge` в JWT):

- Drop на NIC по blacklist до userspace.
- SYN fingerprint ringbuf → корреляция с CH (`tcp_edge_correlation`).

## Защита клика и лендинга (GMA)

Флаги на кампании (в реестре трекера):

| Фича | Назначение |
| :--- | :--- |
| `l1_cidr_block_enabled` | Блок /24 ротации на L1 |
| `l15_proxy_vpn_block_enabled` | Proxy/VPN feed (LPM таблица) |
| `tls_fingerprint_block_enabled` | JA3/JA4 blocklist + impersonation (Chrome UA vs подозрительный JA3) |
| `safe_page_enabled` | Safe page вместо лендинга при срабатывании |
| `attestation_enabled` | HMAC attestation token на клике |
| `link_signing_enabled` | Подписанные ссылки с TTL |
| `conn_type_policy` | Политика типа соединения |

## Холодный путь: IVT detector

Сервис `cmd/ivt-detector` (compose profile `analytics-ml`). Сканирует ClickHouse `ml_features_1m` / клики каждые ~5 мин.

**Правила (все → outbox → Redis / PG):**

| Правило | Что ловит |
| :--- | :--- |
| `high_click_to_imp_ratio` | Аномальный CTR IP |
| `shared_fingerprint_cluster` | Кластер общих отпечатков |
| `campaign_ctr_spike` | CTR spike по кампании + IP |
| `interval_bot` | Бот с фиксированным интервалом кликов |
| `datacenter_asn` | Трафик с DC ASN |
| `tcp_edge_correlation` | Расхождение TCP fingerprint edge vs CH |
| `fraud_scoring_rule` | LightGBM/ONNX batch scoring |

**Действия через outbox** (`POST /api/v1/ops/fraud-threat`):

- `ML_SCORE_BOOST` → Redis `ml:score:boost:{campaign_id}`
- `ML_GHOST_IVT` → `ghost_ivt_enabled=true` на кампании
- `ML_BLACKLIST_ADD` → IP в `blacklist:fraud` (TTL)
- `UPDATE_BLACKLIST` — ручной / auto blacklist

Пауза детектора при `outbox PENDING > 500`.

## Холодный путь: fraud-scorer

`cmd/fraud-scorer` — LightGBM (`var/fraudscore/artifacts/model.txt`) или ONNX.

- Батч до 1000 строк (`FRAUD_SCORING_BATCH_SIZE`).
- Политика: residential proxy signal, structural fraud signal, tier decision.
- Ручные метки: `ml_manual_labels` (label 0/1 по `ip_hash`).
- Override скоринга: `ApplyFraudScoringOverride` (ops).

Обучение: `model/`, CI gate `scripts/ci/fraudtrain.sh`.

## OpenRTB

- `RTB_PREBID_IVT=true` — отсев DC/proxy IP **до** аукциона.
- Полный `FilterEngine` на `/openrtb/bid` **не** запускается (отдельный exchange path).

## Margin Guard (смежная фича)

Не IVT, но защита ROI: политики `margin_guard_policies` — пауза кампании при ROI ниже пола, zero-conversion streak, cost/revenue threshold. SKU: `margin_guard: true` на всех платных тирах.

## Админка и API

| Surface | Что даёт |
| :--- | :--- |
| Campaign → Fraud | Пресет, пороги, ghost IVT |
| `GET /api/v1/fraud/presets` | Список пресетов |
| `GET /api/v1/fraud/decisions` | Explain-by-`ip_hash` (audit:read, rate limit) |
| `GET /api/v1/fraud/integrations` | CAPI/postback health по кампаниям |
| `PATCH /api/v1/campaigns/{id}/fraud` | Конфиг антифрода |
| Fraud dashboard | KPI ghost IVT, ML health |
| Telegram fraud report | UI отчёт (`report_telegram_fraud_page`) |
| ML manual labels API | Разметка для обучения |

## Аналитика и потоки

- Подозрительные события → `ad:fraud:stream` → ClickHouse `fraud_events`.
- PII в CH только как `ip_hash` / `ua_hash` (HighwayHash + salt).
- CAPI/postback **не** шлётся для `ghost_event`, `shadow_event`, событий с `fraud_reason`.

## Лицензирование (SKU)

| SKU | `ivt_ml_detector` | `ml_fraud_boost` | eBPF edge |
| :--- | :---: | :---: | :---: |
| pilot / starter / pro | нет | нет | нет |
| scale / network | да | да | нет |
| enterprise | да | да | да |

Pilot: OpenRTB и ML выключены; базовые L1/L2 сигналы на трекере работают.

## Что говорить покупателю

1. **Многослойно:** edge → in-memory сигналы → Redis Lua → batch ML.
2. **Без латентности ML на клике:** скоринг асинхронный, boost — снимок в памяти.
3. **Ghost IVT:** режим «принимаем, но не платим и не репортим» — без палевных 403 ботам.
4. **Прозрачность:** explain API и fraud dashboard для оператора.
5. **Enterprise:** XDP drop на NIC + multi-region (отдельно от appliance SKU).

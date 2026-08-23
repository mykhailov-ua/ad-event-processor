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
| `device_mismatch` | Sec-CH-UA vs UA; Chrome UA vs подозрительный JA3/JA4 (impersonation) | 35 | L2 |

Пороги tier на кампании: **pass → suspect → ivt → block** (по умолчанию 30 / 60 / 80 / 100). Пресеты: `conservative`, `balanced`, `aggressive`, `gray_market`, `social_in_app`.

### Пресет `social_in_app` (Facebook / TikTok / Instagram WebView)

`PATCH /api/v1/campaigns/{id}/fraud` с `{"preset":"social_in_app"}`:

| Действие | Поле кампании | Значение |
| :--- | :--- | :--- |
| Balanced fraud tier | `fraud_threshold_*` | 30 / 60 / 80 / 100 (как `balanced`) |
| In-app WebView relax | `social_in_app_enabled` | `true` |
| L1.5 proxy/VPN | `l15_proxy_vpn_block_enabled` | `true` |
| TLS JA3/JA4 block | `tls_fingerprint_block_enabled` | `true` |
| Mobile carriers only | `conn_type_policy` | `mobile_only` |

**TLS safe-view на `/click`:** при `social_in_app_enabled` и UA с маркерами `FBAN`, `FBAV`, `musical_ly` или `Instagram` TLS blocklist **не** переводит клик в safe-view (типичный in-app TLS). Бот UA без маркеров остаётся на TLS safe-view. L2 attestation и другие сигналы **не** отключаются.

**Allowlist feed:** оператор может добавить `ja3_allowlist.txt` / `ja4_allowlist.txt` рядом с `ja3_blocklist.txt`; allowlist проверяется **до** blocklist (in-app клиенты с известным JA3).

UI: Campaign → Fraud → **Social in-app**; Configuration → conn type и GMA-чекбоксы обновляются после сохранения.

### Пресет `gray_market` (операторский runbook)

`PATCH /api/v1/campaigns/{id}/fraud` с `{"preset":"gray_market"}`:

| Действие | Поле кампании | Значение |
| :--- | :--- | :--- |
| Агрессивные fraud tier | `fraud_threshold_*` | 20 / 45 / 65 / 85 (как `aggressive`) |
| Safe page | `safe_page_enabled` | `true` |
| L2 attestation | `attestation_enabled` | `true` (требует safe page) |
| L1.5 proxy/VPN | `l15_proxy_vpn_block_enabled` | `true` |
| TLS JA3/JA4 block | `tls_fingerprint_block_enabled` | `true` |
| L1 DC/hosting CIDR feed | `l1_cidr_block_enabled` | `true` |
| Подписанные ссылки | `link_signing_enabled` | `true` |

**Не задаёт:** `safe_page_url` — оператор указывает URL в Configuration после пресета. Outbox `UPDATE_CAMPAIGN_FRAUD` обновляет registry snapshot на трекерах.

UI: Campaign → Fraud → кнопка **Gray market (GMA)**; Configuration → GMA чекбоксы синхронизируются после сохранения.

ML boost: `fraud_score += ml:score:boost` (снимок из Redis, ~90 ns на проверку).

## Цепочка фильтров `/track`

```
License → License RPS → Emergency Breaker → Geo → Schedule → Segment → VPP
  → Fraud (GeoIP DC) → Residential proxy ring → TCP MSS anomaly → Device
  → Consent → Entitlements (ingress RPD) → UnifiedFilter (Lua / local quanta)
```

Связанные проверки в той же цепочке:

- **Geo** — страна / таргетинг (не IVT, но отсекает до Redis).
- **Residential proxy ring** — L2 `residential_proxy` (hot, campaign-local; `RESIDENTIAL_PROXY_HOT_ENABLED`).
- **TCP MSS** — L2 при аномальном MSS (`TCP_MSS_ANOMALY_ENABLED`).
- **VPP** — smart pacing по ratio из Redis.
- **FraudBlacklistFilter** — внутри Fraud filter path: `blacklist:fraud`.
- **FraudFilter** — анонимный / DC IP (GeoIP + sampled DC ASN).
- **DeviceFilter** — TLS impersonation, device mismatch (Sec-CH-UA, JA3).
- **EntitlementsFilter** — ingress `MaxRequestsPerDay` (INCR до UnifiedFilter).
- **Placement blacklist** — в UnifiedFilter (Go + cache); Lua **не** делает `HEXISTS` на hot path.
- **Local TTC** — low TTC / missing impression timestamp (в т.ч. local quanta full-skip).
- **UnifiedFilter** — бюджет, dedup, fcap, pacing; `LOCAL_QUOTA_MODE=live` → local quanta full-skip без sync `EVALSHA` когда eligible.

### Local quanta full-skip (capacity)

При `LOCAL_QUOTA_MODE=live`, credit в `LocalQuantaLedger`, и eligible click/impression:

- Dedup клика — `localClickIdem` (in-memory), async `SET NX` в stream worker.
- Placement blacklist и ingress RPD — **до** full-skip (Go), не в Lua.
- Stream: `LocalQuantaStreamPublisher` или defer через `StreamProducer` (`fcap:ignored` coordination).
- Кампании в strict-mode hysteresis, freq/pacing/even/TTC-fail-closed без Go substitute — **не** full-skip.

High-volume campaigns (`BehaviorHighVolumeDebit`): budget/fcap keys с `{campaign_id:slot_N}` hash tag (4 sub-slots) — см. [SHARDING.md](../../docs/SHARDING.md).

## Периметр (до трекера)

**Nginx / OpenResty** (`access-check.lua`, `edge-fraud-tier.lua`):

- IP blacklist (`blacklist:manual`, `blacklist:auto`, `blacklist:fraud`) — sync из Redis shard 0.
- Circuit breaker и rate limit на edge.
- Блок по fraud tier score на edge (403/429).
- DFA разбор тела на edge.

**Опционально Enterprise — XDP/eBPF** (`ebpf_xdp_edge` в JWT):

- Drop на NIC по blacklist до userspace (LPM trie, до ~786k v4 entries per map после BPF rebuild).
- Incremental sync: ZSET `blacklist:changelog:add` / `remove` на shard 0 между полными SMEMBERS (5 min); control-plane fanout пишет changelog при `blacklist:*` updates.
- SYN fingerprint ringbuf → корреляция с CH (`tcp_edge_correlation`).

## Защита клика и лендинга (GMA)

Флаги на кампании (в реестре трекера):

| Фича | Назначение |
| :--- | :--- |
| `l1_cidr_block_enabled` | L1 safe-view по статическим DC/hosting CIDR (AWS, GCP, Azure, Tor). **Не** детектор ротации /24 или /64 — для ротации см. ниже. |
| `l15_proxy_vpn_block_enabled` | Proxy/VPN feed (LPM таблица) |
| `tls_fingerprint_block_enabled` | JA3/JA4 blocklist + allowlist feed (`ja3_allowlist.txt` / `ja4_allowlist.txt` before blocklist) + impersonation (Chrome UA vs подозрительный JA3) |
| `safe_page_enabled` | Safe page вместо лендинга при срабатывании |
| `attestation_enabled` | HMAC attestation token на клике |
| `attestation_mode` | `off` / `light` (L2 + safe view, FilterEngine runs) / `strict` (JS probe stub, no Redis on first click) |
| `/track/verify` attestation | WebRTC, timezone, WebGL vendor/renderer, viewport, canvas/audio, permissions, bezier bot (behavior tier) |
| Attestation + TTC combined | `attestation_mode` on + `missing_imp_ts` / `low_ttc` on `/click` -> force safe view; local TTC fail-closed when attestation on; link TTL capped at 300s |
| `link_signing_enabled` | Подписанные ссылки с TTL |
| `conn_type_policy` | Политика типа соединения |

**Ротация IP (отдельно от L1 CIDR feed):**

| Механизм | Где | Назначение |
| :--- | :--- | :--- |
| IPv6 /64 rotation velocity | Tracker env `IPV6_ROTATION_MODE` (`shadow` / `live`), `IPV6_ROTATION_THRESHOLD` | Динамическая ротация host-адресов внутри одного /64 на `/click` (L1 safe-view в `live`; в `shadow` — L2 сигнал без блока). |
| IPv4 /24 sticky rotation | Tracker env `IPV4_ROTATION_MODE` (`shadow` / `live`), `IPV4_ROTATION_THRESHOLD`; ключ `(campaign, user_id, /24)` | Velocity по /24 для residential sticky pools на `/click` (L1 safe-view в `live`; в `shadow` — L2 `ipv4_rotation`, не sync PG). |
| OS fingerprint (p0f-lite) | Tracker env `OS_FINGERPRINT_MISMATCH_ENABLED`; edge headers `X-TCP-TTL`, `X-TCP-WINDOW`, `X-TCP-MSS` | L2 `os_fingerprint_mismatch` когда TTL/window не совпадают с UA family (bounded scan). |
| DC ASN hot (sampled) | `DC_ASN_HOT_ENABLED`, feed `dc_asn.txt`, `GEOIP_ASN_DB_PATH` | Snapshot DC ASN set + MaxMind ASN lookup; L2 `datacenter_ip` на 1/8 событий (mask `DC_ASN_SAMPLE_MASK`, default 7). Mobile AS3215/AS12322 denylist. |
| Residential proxy farm (hot) | `RESIDENTIAL_PROXY_HOT_ENABLED`, `RESIDENTIAL_PROXY_WINDOW` | Campaign-local ring; L2 `residential_proxy` when thresholds match `model/scoring_policy.py` (no ML on hot path). |
| External residential intel (cold, SKU) | `EXTERNAL_RESIDENTIAL_INTEL_ENABLED`, `EXTERNAL_RESIDENTIAL_INTEL_URL`, JWT `external_residential_intel` | `ivt-detector` async enricher: provider lookup, Redis/CH cache TTL, append `external_residential.txt` for L1.5 feed reload. **No sync call from tracker.** |

L1 CIDR feed и rotation velocity используют общий флаг кампании `l1_cidr_block_enabled` для включения L1 safe-view на `/click`.

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
| `external_residential_intel` | Cold enricher: external provider -> Redis/CH cache + L1.5 feed (`external_residential.txt`) |
| `fraud_scoring_rule` | LightGBM/ONNX batch scoring |

**Действия через outbox** (`POST /api/v1/ops/fraud-threat`):

- Одиночный body: `action`, `ip`, `campaign_id`, `score`, `boost`, `ttl_seconds`
- Bulk body: `items[]` (до 500 строк); один PG batch insert в outbox

- `ML_SCORE_BOOST` → Redis `ml:score:boost:{campaign_id}`
- `ML_GHOST_IVT` → `ghost_ivt_enabled=true` на кампании
- `ML_BLACKLIST_ADD` → IP в `blacklist:fraud` (TTL); worker coalesce: один PG TX + synthetic `UPDATE_BLACKLIST` только для Redis (без nested outbox rows); один `fraud:quarantine` publish на shard с JSON `{"ips":[...]}`
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

| SKU | `ivt_ml_detector` | `ml_fraud_boost` | `external_residential_intel` | eBPF edge |
| :--- | :---: | :---: | :---: | :---: |
| pilot / starter / pro | нет | нет | нет | нет |
| scale / network | да | да | да | нет |
| enterprise | да | да | да | да |

Pilot: OpenRTB и ML выключены; базовые L1/L2 сигналы на трекере работают. External residential intel — только Scale+ JWT (`external_residential_intel: true`); cold enricher, не sync на `/track`.

## Что говорить покупателю

1. **Многослойно:** edge → in-memory сигналы → Redis Lua → batch ML.
2. **Без латентности ML на клике:** скоринг асинхронный, boost — снимок в памяти.
3. **Ghost IVT:** режим «принимаем, но не платим и не репортим» — без палевных 403 ботам.
4. **Прозрачность:** explain API и fraud dashboard для оператора.
5. **Enterprise:** XDP drop на NIC + multi-region (отдельно от appliance SKU).

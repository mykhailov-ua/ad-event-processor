# ML & Fraud Admin Roadmap

Документ описывает P0/P1 задачи по ML-стеку (`model/`, `internal/fraud/`, fraud UI).
Приоритет: **GUI и автономность buyer** → прозрачность решений → офлайн-качество модели.

Связанные поверхности:

| Слой | Путь |
|------|------|
| Offline train/eval | `model/` |
| Production inference | `cmd/fraud-scorer`, `internal/fraud/` |
| Admin API | `internal/controlplane/adminapi/` |
| Buyer UI | `web/src/pages/`, `web/src/components/` |
| CH tables | `ml_features_1m`, `ml_shadow_scores`, `fraud_events` |

Cold-path SLA (admin handlers): p95 handler &lt; 500 ms; CH lookup с hard timeout; без unbounded fan-out.

---

## P0 — Sales & autonomy (2 спринта)

### P0-1 · Campaign Fraud Controls (presets + per-campaign thresholds)

**Цель:** buyer сам настраивает чувствительность fraud-tier без env vars и тикета в support.

**Контекст:** логика уже в `internal/controlplane/service_fraud.go` (`GetCampaignFraudConfig`, `UpdateCampaignFraudConfig`, audit + outbox), но **нет HTTP routes и UI**.

#### Checklist

- [x] `GET /api/v1/campaigns/{id}/fraud` — read config
- [x] `PATCH /api/v1/campaigns/{id}/fraud` — partial update (thresholds, `ghost_ivt_enabled`)
- [x] Presets `conservative` / `balanced` / `aggressive` (маппинг на default thresholds из `internal/domain`)
- [x] `CampaignFraudSection` в `web/src/pages/campaign_detail_page.tsx` (новая вкладка **Fraud**)
- [x] Confirm flow (`confirm_catalog.ts`) для PATCH
- [x] Handler tests + e2e smoke (`campaign_detail` fraud tab)
- [x] Route catalog entry в `adminapi/register.go`

#### Definition of Done

1. Buyer с `campaigns:read` видит текущие пороги и ghost IVT toggle.
2. Buyer с `campaigns:write` меняет пороги или применяет preset; ответ — актуальный `CampaignFraudConfigDTO`.
3. Невалидный порядок (`pass > suspect` и т.д.) → `400` с typed error.
4. Изменение пишется в audit (`UPDATE_CAMPAIGN_FRAUD`) и outbox (`UPDATE_CAMPAIGN_FRAUD`).
5. `fraud-scorer` подхватывает новые пороги в течение `campaignConfigCacheTTL` (60 s, `fraud_scoring_rule.go`).
6. UI показывает human-readable band table (pass / suspect / ivt / block).

#### План реализации

1. **API** — `internal/controlplane/adminapi/fraud_handlers.go`:
   - интерфейс `CampaignFraudService` → `controlplane.Service`
   - `registerFraudRoutes` в `CampaignsHTTP` или отдельный `FraudHTTP`
   - auth: `campaigns:read` / `campaigns:write`; проверка customer scope через campaign ownership (как в `postbacks_handlers.go`).
2. **Presets** — константы в `service_fraud.go` или `domain/campaign.go`; PATCH body: `{ "preset": "balanced" }` **или** explicit thresholds.
3. **Web** — `web/src/helpers/fraud_api.ts`, компонент `campaign_fraud_section.tsx`, слайдеры 0–100 с валидацией порядка на клиенте.
4. **Tests** — `service_fraud_test.go` (есть) + `fraud_handlers_test.go` HTTP round-trip.

#### Endpoints & permissions

| Method | Path | Permission | Body / response |
|--------|------|------------|-----------------|
| `GET` | `/api/v1/campaigns/{id}/fraud` | `campaigns:read` | `CampaignFraudConfigDTO` |
| `PATCH` | `/api/v1/campaigns/{id}/fraud` | `campaigns:write` | `CampaignFraudConfigUpdate` → `CampaignFraudConfigDTO` |

#### Performance

| Метрика | Target |
|---------|--------|
| GET/PATCH p95 | &lt; 100 ms (single PG tx) |
| PG queries per request | 1 read или 1 tx (get-for-update + update + outbox) |
| Payload | &lt; 4 KB |

#### Риски ресурсов

| Риск | Где | Mitigation |
|------|-----|------------|
| Лишний round-trip campaign + fraud | UI загружает campaign и fraud отдельно | Один tab-scoped fetch; не poll чаще 30 s |
| N+1 при bulk preset | Не делать «apply to all campaigns» в P0 | Отложить; если понадобится — один `UPDATE … WHERE customer_id` |
| Незакрытый `rows` | Новый handler | `defer rows.Close()` по convention; pool через `pgxpool` |

---

### P0-2 · Manual labeling UI (buyer-scoped)

**Цель:** нетехнический пользователь помечает IP как fraud/legit; петля обратной связи без ops.

**Контекст:** ops API уже есть (`POST /api/v1/ops/ml-model/labels`, perm `shards:write`) — **недоступен buyer**. Таблица `ml_manual_labels`, upsert по `ip_hash`.

#### Checklist

- [x] `GET /api/v1/fraud/labels?customer_id=&limit=` — buyer-scoped list
- [x] `POST /api/v1/fraud/labels` — buyer-scoped upsert (`source=admin_ui`)
- [x] Опционально: `POST /api/v1/fraud/labels/bulk` (CSV ≤ 500 rows, sync)
- [x] Форма в `FraudBody` (`role_dashboard_page.tsx`): ip_hash, label, reason
- [x] Кнопки fraud/legit в таблице `recent_labels`
- [x] Confirm для POST (`confirm_registry.ts`)
- [x] Миграция: `customer_id UUID` на `ml_manual_labels` (nullable → backfill → NOT NULL для новых)
- [x] Unit + handler tests

#### Definition of Done

1. Buyer с `audit:read` видит свои labels (не global ops list).
2. Buyer с `audit:write` (или новый `fraud:write`) создаёт/обновляет label; `ip_hash` — 32 hex.
3. Duplicate `ip_hash` → upsert, не 409.
4. Fraud dashboard `labels_pending` считает **только labels customer** (сейчас global count — баг).
5. Ops endpoint `/ops/ml-model/labels` остаётся для platform ops (backward compatible).

#### План реализации

1. **Migration** `internal/ingestion/migrations/000NN_ml_manual_labels_customer.sql` — колонка `customer_id`, index `(customer_id, created_at DESC)`.
2. **Service** — `ListMLManualLabelsForCustomer`, `UpsertMLManualLabel` в `service_fraud.go`.
3. **Handlers** — `fraud_labels_handlers.go`, perm `audit:read` / `audit:write` (совпадает с fraud dashboard).
4. **Web** — `fraud_labels_panel.tsx`, валидация hex на клиенте.
5. **Export path** — `model/features_export.py` LEFT JOIN с customer filter (follow-up в P1 если нужно).

#### Endpoints & permissions

| Method | Path | Permission | Notes |
|--------|------|------------|-------|
| `GET` | `/api/v1/fraud/labels` | `audit:read` | Query: `customer_id` (required for buyer), `limit` default 50 max 100 |
| `POST` | `/api/v1/fraud/labels` | `audit:write` | `{ ip_hash, label, reason }` |
| `POST` | `/api/v1/fraud/labels/bulk` | `audit:write` | `{ rows: [...] }`, max 500 |

#### Performance

| Метрика | Target |
|---------|--------|
| POST p95 | &lt; 150 ms |
| GET p95 | &lt; 200 ms |
| Bulk | ≤ 500 rows, single PG tx, p95 &lt; 2 s |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| N+1 при bulk insert | `COPY` или multi-row `INSERT` в одной tx, не loop Exec |
| Global list без LIMIT | Hard cap 100 (как `mlManualLabelsListLimit` в ops) |
| Connection leak | Один `pgxpool` conn на request; `defer rows.Close()` |

---

### P0-3 · Trust & Health tile (ML status, plain language)

**Цель:** buyer видит, что защита работает; метрики без jargon; явный disclaimer про proxy-eval.

**Контекст:** `GET /api/v1/dashboards/fraud` отдаёт ML version, precision, drift, но:
- tier thresholds **захардкожены** 30/60/80 в `service_role_dashboards.go`, не per-campaign;
- precision/recall читаются из **файла** `var/fraudscore/shadow_eval_report.json` (может быть stale);
- нет `eval_generated_at`.

#### Checklist

- [x] Расширить `FraudDashboardDTO`: `ml_eval_generated_at`, `ml_eval_status`, `ml_label_method`, `ml_shards_consistent`
- [x] Убрать misleading global tier constants; показать «defaults» + ссылку на campaign fraud tab
- [x] UI: статус-бейджи `healthy` / `drift_detected` / `eval_stale` / `eval_unavailable`
- [x] Disclaimer: «Precision estimated on proxy labels, not human audit»
- [x] `FraudKpiTiles` — показать eval age («updated 2d ago» → warning if &gt; 24h)
- [x] Tests: dashboard DTO serialization

#### Definition of Done

1. Dashboard показывает ML version, artifact hash (truncated), precision/recall **с датой eval**.
2. Если report старше 48 h → `eval_stale=true`, UI warning (не silent stale).
3. `ml_label_method=proxy` всегда виден рядом с precision.
4. Drift flag с пояснением («traffic mix changed &gt;30% vs training»).
5. Нет claim «accuracy» — только «shadow precision (proxy)».

#### План реализации

1. **Backend** — `fraudMLSnapshot()` читает полный JSON report (`generated_at`, `label_method`); добавить в `FraudDashboardDTO`.
2. **Stale detection** — compare `generated_at` vs `now()`; threshold 48 h configurable via env `FRAUD_EVAL_STALE_HOURS`.
3. **Web** — обновить `FraudBody`, `fraud_kpi_tiles.tsx`; иконки StatusBadge.
4. **Ops cross-read** — optional: merge `GetMLModelStatus().Redis.ShardsConsistent` into dashboard (один PG + file read, без extra CH).

#### Endpoints

Без новых routes — расширение `GET /api/v1/dashboards/fraud`.

#### Performance

| Метрика | Target |
|---------|--------|
| Dashboard p95 | &lt; 500 ms (текущий budget) |
| Extra I/O | +1 file stat (`os.Stat` report mtime); не читать report &gt;1 раз |
| PG queries | Не добавлять; reuse `fraudMLSnapshot` |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| File read на каждый dashboard load | Cache report parsed JSON 60 s in-process (`sync.Map` + TTL) |
| Двойной CH для geo + eval | Geo уже один CH query; eval — file only в P0 |

---

### P0-4 · Explainability: «Why blocked?» IP lookup

**Цель:** прозрачность — tier, score, сигналы (ML / residential proxy / structural / FP-guard).

**Контекст:** `ml_shadow_scores` хранит только `ip_hash, score, model_name, created_at` — **нет signal flags**. Policy replay возможен в Go на `ml_features_1m` raw row.

#### Checklist

- [x] `GET /api/v1/fraud/decisions?ip_hash=&campaign_id=&hours=24`
- [x] Response: `TierDecision` + raw features snapshot + `model_score` + `evaluated_at`
- [x] Policy replay через `fraud.DecideWithCampaign()` (Go), не Python в request path
- [x] UI: search form на fraud dashboard + result card
- [x] Rate limit: 30 req/min per customer
- [x] Handler test с CH mock / integration test
- [x] Документировать: «decision as of last scorer window»

#### Definition of Done

1. Valid `ip_hash` (32 hex) → 200 с breakdown или 404 если нет данных в окне.
2. Response fields: `tier`, `score`, `ml_probability`, `adjusted_probability`, `residential_proxy`, `structural_fraud`, `fp_guard_applied`, `features` (16-dim names), `campaign_thresholds`.
3. p95 &lt; 500 ms при наличии CH index hit.
4. Нет утечки raw IP — только `ip_hash`.
5. CH connection закрывается через shared pool / context timeout.

#### План реализации

1. **CH query** (single round-trip):
   ```sql
   SELECT window_start, campaign_id, events, clicks, spend_micro, budget_limit_micro,
          unique_users, unique_uas, score, model_name, created_at
   FROM ml_features_1m AS f
   LEFT JOIN (
     SELECT ip_hash, argMax(score, created_at) AS score, argMax(model_name, created_at) AS model_name,
            max(created_at) AS created_at
     FROM ml_shadow_scores
     WHERE ip_hash = {ip} AND created_at >= now() - INTERVAL {hours} HOUR
     GROUP BY ip_hash
   ) AS s USING (ip_hash)
   WHERE f.ip_hash = {ip} AND f.window_start >= now() - INTERVAL {hours} HOUR
   ORDER BY f.window_start DESC LIMIT 1
   ```
2. **Go service** — `ExplainFraudDecision(ctx, customerID, ipHash, campaignID, hours)`:
   - verify campaign belongs to customer;
   - `FeatureRow` → optional live `Scorer.ScoreBatch` если shadow score отсутствует (guard: только если artifact loaded, иначе ml_probability=0 + flag `score_missing`).
3. **Handler** — `fraud_handlers.go`, perm `audit:read`.
4. **Web** — `fraud_decision_lookup.tsx`.

#### Endpoints & permissions

| Method | Path | Permission |
|--------|------|------------|
| `GET` | `/api/v1/fraud/decisions` | `audit:read` |

Query params: `ip_hash` (required), `campaign_id` (optional), `hours` (default 24, max 168).

#### Performance

| Метрика | Target |
|---------|--------|
| p95 latency | &lt; 500 ms |
| CH queries | **1** per request |
| PG queries | ≤ 2 (campaign ownership + fraud config) |
| ML inference | 0–1 row `ScoreBatch`; skip if shadow score present |

#### Риски ресурсов

| Риск | Severity | Mitigation |
|------|----------|------------|
| **N+1** campaign verify + config + CH | Medium | Объединить PG в один query с JOIN campaigns |
| **CH connection leak** | High | `context.WithTimeout(5s)`; use `database.CHQuery` pool, не `clickhouse.Open` per request |
| **Live scorer load** | Medium | Prefer shadow score; live score только fallback, feature flag `FRAUD_EXPLAIN_LIVE_SCORE=false` default |
| **Unbounded scan** | High | `hours` cap 168; `LIMIT 1`; partition prune по `window_start` |

---

### P0-5 · Integration health on Fraud dashboard

**Цель:** buyer видит статус postback/CAPI без перехода в каждую кампанию.

**Контекст:** `CampaignPostbackSection` + `postback_api.ts` уже есть per-campaign. Fraud dashboard — read-only aggregate.

#### Checklist

- [x] `GET /api/v1/fraud/integrations?customer_id=` — aggregate postback health
- [x] Response: per campaign `{ campaign_id, name, provider, configured, last_success_at, dlq_count, last_error }`
- [x] UI subsection «Integrations» в fraud dashboard со ссылками на campaign postback tab
- [x] Tests: handler с stub postback reader

#### Definition of Done

1. Список campaigns customer с postback status (configured / failing / idle).
2. `dlq_count` из существующего DLQ store (не новый источник).
3. p95 &lt; 500 ms для ≤ 50 campaigns.
4. Ссылка «Fix» ведёт на `/campaigns/{id}#postbacks`.

#### План реализации

1. **Backend** — один SQL:
   ```sql
   SELECT c.id, c.name, p.provider, p.url_template, …
   FROM campaigns c
   LEFT JOIN postback_configs p ON …
   WHERE c.customer_id = $1 AND c.deleted_at IS NULL
   ```
   DLQ counts: `SELECT campaign_id, count(*) … GROUP BY campaign_id` — **второй query**, не per-campaign loop.
2. **Optional** last success из postback delivery log / Redis — если нет таблицы, P0 показывает только configured + dlq_count.
3. **Web** — таблица в `FraudBody`.

#### Endpoints

| Method | Path | Permission |
|--------|------|------------|
| `GET` | `/api/v1/fraud/integrations` | `audit:read` |

#### Performance

| Метрика | Target |
|---------|--------|
| PG queries | **≤ 2** (campaigns + dlq aggregate) |
| p95 | &lt; 500 ms |
| No per-campaign HTTP | Запрет fan-out к postback test endpoint |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| **N+1** fetch postback per campaign | Batch SQL + GROUP BY dlq |
| **N+1** test postback on page load | Read-only status only; test остаётся manual action |
| Redis/socket per campaign | Не читать Redis в P0 aggregate; только PG |

---

### P0-6 · Ops ML Model page (platform)

**Цель:** ops видит version lifecycle, shard sync, feature importance без curl.

**Контекст:** `GET /api/v1/ops/ml-model` реализован (`ops_handlers.go`); UI отсутствует.

#### Checklist

- [x] `web/src/pages/ops_ml_model_page.tsx`
- [x] Nav entry (ops-only perm `shards:read`)
- [x] Таблицы: active/syncing version, redis shard consistency, shard sync phases
- [x] Top-5 feature importance bar chart (из `importance[]`)
- [x] Link to labels (`GET /ops/ml-model/labels`)
- [x] e2e: page renders with fixture API

#### Definition of Done

1. Ops page загружает один `GET /ops/ml-model` — без secondary fan-out.
2. Показывает `drift_detected`, precision, recall, `importance`.
3. Shard sync table: все rows из `ml_shard_sync_state`.
4. Poll interval ≥ 30 s (не realtime websocket).

#### План реализации

1. `web/src/helpers/ops_ml_api.ts` — wrapper.
2. Page + route in `app_routes.tsx`.
3. Reuse `StatusBadge`, `data-table` patterns.

#### Endpoints

| Method | Path | Permission |
|--------|------|------------|
| `GET` | `/api/v1/ops/ml-model` | `shards:read` |
| `GET` | `/api/v1/ops/ml-model/labels` | `shards:read` |

#### Performance

| Метрика | Target |
|---------|--------|
| Single fetch | 1 HTTP request on load |
| Backend p95 | &lt; 1 s (2 PG + N redis pipelines, N = shard count ≤ 8) |

#### Риски ресурсов

| Риск | Where | Mitigation |
|------|-------|------------|
| **N redis round-trips** | `readMLModelRedis` loops `rdbs` | Already pipelined per shard; N bounded by shard count; document as OK |
| **Double fetch** version + labels | UI | Labels — lazy load on tab click, не on mount |
| File read eval report | `GetMLModelStatus` | Same 60 s cache as P0-3 |

---

## P1 — Autonomy & model quality (следующие 2 спринта)

### P1-1 · Scheduled shadow eval (live metrics)

**Цель:** precision/drift не зависят от ручного запуска `model/evaluate.py` и stale JSON file.

#### Checklist

- [x] CronJob / worker: `python model/evaluate.py` каждые 6 h
- [x] Пишет `var/fraudscore/shadow_eval_report.json` + upsert `ml_eval_reports` (PG)
- [x] `fraudMLSnapshot()` читает PG first, file fallback
- [x] Prometheus gauges: `fraud_ml_shadow_precision`, `fraud_ml_drift_detected`
- [x] Alert rule: eval stale &gt; 12 h
- [x] `evaluate.py`: explicit `client.close()` в `finally`
- [x] Deploy: `model/Dockerfile` + compose/cron manifest

#### Definition of Done

1. Eval runs on schedule без human intervention.
2. Dashboard `eval_generated_at` &lt; 12 h в prod.
3. CH client закрывается после каждого run (no leaked HTTP keep-alive in long-running worker).
4. Failed eval → `status=error` в report, не silent skip.

#### План реализации

1. PG table `ml_eval_reports (id, generated_at, precision, recall, drift_json, status)`.
2. Patch `evaluate.py` / `ch_client.py` — context manager `with ch_client():`.
3. Control plane reader prefers PG.
4. `deploy/compose` CronJob or `controlplane` ticker calling eval binary.

#### Performance

| Метрика | Target |
|---------|--------|
| Eval runtime | &lt; 5 min (CH queries bounded by `FRAUD_EVAL_HOURS`) |
| CH connections | 1 client per run, closed in `finally` |
| CH query count | 2 (`shadow_precision` + `drift`) |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| **Unclosed CH HTTP session** | `clickhouse_connect` client `.close()` in `finally`; test with repeated runs |
| Concurrent eval runs | Advisory lock / k8s `concurrencyPolicy: Forbid` |
| Large CH scan | `hours` default 168; readonly user; query timeout 120 s |

---

### P1-2 · Threshold impact preview

**Цель:** buyer видит «сколько IP затронет новый preset» до сохранения.

#### Checklist

- [x] `POST /api/v1/campaigns/{id}/fraud/preview`
- [x] Body: proposed thresholds или preset
- [x] Response: `{ affected_ips_7d, by_tier: { suspect, ivt, block }, sample_size }`
- [x] UI: preview panel в `CampaignFraudSection` before save
- [x] CH single query с replay thresholds in SQL (approx) или Go batch on sample

#### Definition of Done

1. Preview не персистит изменения.
2. p95 &lt; 3 s using ≤ 10k row sample or CH aggregation.
3. Disclaimer: «estimate based on last 7d shadow scores».

#### План реализации

1. **Fast path** — CH only:
   ```sql
   SELECT countIf(score >= {suspect} AND score < {ivt}) AS suspect, …
   FROM ml_shadow_scores
   WHERE campaign_id = {id} AND created_at >= now() - 7d
   ```
   Requires adding `campaign_id` to `ml_shadow_scores` (migration) **или** join via `ml_features_1m`.
2. **Join path** (no schema change): join shadow + features on `ip_hash` — one query, risk slower.
3. Handler: perm `campaigns:read`.

#### Endpoints

| Method | Path | Permission |
|--------|------|------------|
| `POST` | `/api/v1/campaigns/{id}/fraud/preview` | `campaigns:read` |

#### Performance

| Метрика | Target |
|---------|--------|
| p95 | &lt; 3 s |
| CH queries | **1** |
| Row scan cap | SAMPLE or `LIMIT 10000` subquery |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| Full table scan `ml_shadow_scores` | Partition prune `created_at`; `campaign_id` column (recommended migration) |
| CH connection per preview | Shared pool; 10 s timeout |
| User spam preview | Rate limit 10/min per campaign |

---

### P1-3 · False-positive override API + UI

**Цель:** buyer снимает ошибочный block/boost без ops.

**Контекст:** `ApplyFraudScoringOverride` в `service_fraud.go` (clear boost, remove blacklist) — **не exposed**.

#### Checklist

- [x] `POST /api/v1/fraud/overrides` — `{ campaign_id?, ip? }`
- [x] UI action из explain panel: «Mark false positive»
- [x] Audit: `FRAUD_REMOVE_FALSE_POSITIVE`, `FRAUD_CLEAR_BOOST`
- [x] Confirm level: `elevated`
- [x] Tests

#### Definition of Done

1. At least one of `campaign_id` or `ip` required.
2. Outbox events `ML_SCORE_BOOST` / `UPDATE_BLACKLIST` created.
3. p95 &lt; 300 ms.

#### Endpoints

| Method | Path | Permission |
|--------|------|------------|
| `POST` | `/api/v1/fraud/overrides` | `audit:write` |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| Double outbox on retry | Idempotent blacklist delete; client confirm + disable button |
| N+1 redis in worker | Existing outbox worker batching — не менять в handler |

---

### P1-4 · Mixed eval report (proxy + manual labels)

**Цель:** прозрачность качества модели для power users / ops.

#### Checklist

- [x] Extend `model/shadow_precision.py` — `run_audited_precision()` join `ml_manual_labels`
- [x] Report sections: `proxy_metrics`, `audited_metrics` (separate)
- [x] `GET /api/v1/ops/ml-model/eval` — latest report from PG
- [x] Ops ML page tab «Eval quality»
- [x] CI test `test_shadow_precision.py` with fixture labels

#### Definition of Done

1. Report never presents proxy precision as «accuracy».
2. `audited_metrics.labeled_rows` shown even when 0.
3. Eval job (P1-1) writes both metric blocks.

#### Performance

| Метрика | Target |
|---------|--------|
| Extra CH query | +1 (manual label IPs join) |
| PG | 1 read labels export or join in CH via federated query |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| CH + PG double connection in eval | Sequential: PG export labels to temp set, single CH `IN` query |
| Small audited sample misleading | Show `confidence=low` when `labeled_rows < 30` |

---

### P1-5 · Policy presets platform store

**Цель:** presets не в env vars; ops может менять default bands без redeploy.

#### Checklist

- [x] Table `fraud_policy_presets (name, pass, suspect, ivt, block, updated_at)`
- [x] Seed: conservative / balanced / aggressive
- [x] `GET /api/v1/fraud/presets` (public read)
- [x] `PATCH /api/v1/ops/fraud/presets/{name}` (ops write)
- [x] Campaign PATCH preset resolves from DB
- [x] Parity: presets reflected in `model/policy_config.py` only for offline default docs

#### Definition of Done

1. Preset change applies to new campaign selections immediately.
2. Existing campaigns unchanged until explicit PATCH.
3. Audit on preset change.

#### Performance

| Метрика | Target |
|---------|--------|
| GET presets | p95 &lt; 50 ms, cache 5 min in-process |
| PG | 1 query, ≤ 10 rows |

#### Риски ресурсов

| Риск | Mitigation |
|------|------------|
| Cache invalidation | TTL 5 min; fraud-scorer uses campaign-level thresholds primarily |
| Drift vs `metadata.json` policy | Document: presets affect tiers, not ML calibration |

---

## Сводка: риски N+1 и соединений по задачам

| Task | N+1 risk | Connection/socket risk | Priority fix |
|------|----------|------------------------|--------------|
| P0-1 Campaign fraud | Low | Low (PG pool) | — |
| P0-2 Labels | Medium (bulk) | Low | Multi-row insert |
| P0-3 Trust tile | Low | Low (file cache) | In-process TTL cache |
| P0-4 Explain | **High** | **High (CH)** | Single CH query + timeout |
| P0-5 Integrations | **High** if per-campaign | Medium | 2 batch SQL max |
| P0-6 Ops ML | Medium (redis loop) | Low | Lazy labels tab |
| P1-1 Scheduled eval | Low | **High (CH)** | `client.close()` in `finally` |
| P1-2 Preview | **High** (scan) | **High (CH)** | `campaign_id` on shadow scores |
| P1-3 Override | Low | Low (outbox) | — |
| P1-4 Mixed eval | Medium | Medium | Sequential PG→CH |
| P1-5 Presets | Low | Low | In-process cache |

---

## Порядок выполнения

```
Sprint A: P0-1 → P0-3 → P0-2
Sprint B: P0-4 → P0-5 → P0-6
Sprint C: P1-1 → P1-2
Sprint D: P1-3 → P1-4 → P1-5
```

**Не включено** (низкий sales ROI, отдельный backlog): Python DOD `FeatureBatch`, `ScoreBatchSoA` в Go, расширение feature space, iforest cleanup.

---

## Verification gates

Каждая задача перед merge:

- [ ] `scripts/ci/admin_web.sh` green (UI tasks)
- [ ] `bash scripts/ci/fraudtrain.sh` green (если затронут `model/`)
- [ ] Handler tests + `go test ./internal/controlplane/...`
- [ ] Нет новых per-request `clickhouse.Open` / `pgx.Connect`
- [ ] `register.go` catalog обновлён для новых routes

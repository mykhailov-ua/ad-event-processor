# Vendor ops runbook (enterprise tracker)

Operator reference for enterprise tracker surfaces shipped in the control/tracker stack. Engineering tone; concrete env vars, tables, and metric names. Antifraud and ingest-sink detail: [ANTIFRAUD.md](ANTIFRAUD.md). Architecture: [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md).

---

## Postback inbound auth

Affiliate S2S conversion postbacks hit POST `/track` with `event_type=conversion`. Optional per-customer verification runs in `internal/postback/inbound` before filter debit.

### Config source (Postgres)

Migration `internal/ingest/migrations/00142_team_scope.sql` adds customer columns:

| Column | Type | Default | Role |
| :--- | :--- | :--- | :--- |
| `postback_inbound_ip_allowlist` | `TEXT[]` | `{}` | CIDR or single-IP strings; empty disables IP check |
| `postback_inbound_secret_encrypted` | `BYTEA` | null | HMAC secret (encrypted at rest); null disables signature check |

Tracker loads config via `internal/ingest/postback_inbound_bridge.go` -> `internal/postback/inbound.Store` (default refresh interval 60s). Verification runs only when at least one of allowlist or secret is configured (`inbound.Configured`).

Admin rotate path: `PATCH /api/v1/customers/{id}/postback-auth` (`settings:write`). Secrets are never returned on read.

### Checks

| Check | When active | Failure |
| :--- | :--- | :--- |
| IP allowlist | `cardinality(postback_inbound_ip_allowlist) > 0` | `403` body `POSTBACK_AUTH_FAILED` |
| HMAC-SHA256 | `postback_inbound_secret_encrypted` set | same |

Client IP resolution order: `X-Real-IP`, first hop of `X-Forwarded-For`, then `RemoteAddr`.

Signature verification (see also slug `postback_inbound_hmac` in [ANTIFRAUD.md](ANTIFRAUD.md)):

- Header `X-Postback-Signature: sha256=<hex>` or query param `sig` (without `sha256=` prefix on query; header may include prefix).
- Canonical string: `METHOD|PATH|sorted_query|body_sha256_hex` (uppercase method; query sorted `key=value` joined with `&`, excluding `sig`; body hash is SHA-256 of raw request body).

Hook location: `internal/ingest/handler.go` on `eventType == "conversion"` before `FilterEngine.Check`.

### Dev kill-switch

| Env | Value | Effect |
| :--- | :--- | :--- |
| `POSTBACK_INBOUND_AUTH_DISABLED` | `1` | Skips all inbound checks (dev only; do not set in production) |

When disabled, configured customers accept conversions without IP or HMAC verification.

### Operator steps

1. Set allowlist and/or rotate secret on the customer row (admin API when wired; direct PG until then).
2. Confirm affiliate source IP matches allowlist (or terminate TLS at edge that sets trusted `X-Real-IP`).
3. Recompute affiliate signature on the exact body bytes received; query param `sig` is excluded from the sorted query string.
4. On `403`, grep tracker logs for `POSTBACK_AUTH_FAILED`; verify inbound store refresh (default 60s).

### Verify

```bash
go test ./internal/postback/inbound/ -short -count=1
```

---

## Postback DLQ auto-replay

Failed outbound postbacks land in `postback_dlq`. A replay worker in `internal/postback/replay` re-enqueues `SEND_POSTBACK` outbox events on an exponential schedule. Manual retry remains via Ops DLQ inbox (`POST /api/v1/ops/dlq/inbox/{id}/retry`).

### Schema (migration `00143_postback_dlq_replay.sql`)

| Column | Default | Role |
| :--- | :--- | :--- |
| `replay_count` | `0` | Incremented on each auto-replay |
| `max_replay_count` | `5` | Row excluded from auto-replay when `replay_count >= max_replay_count` |
| `next_retry_at` | null | Eligibility timestamp for FAILED rows |

### Worker behavior

- Binary: `cmd/postback-sender` starts `replay.Start(ctx, pool, 60*time.Second)` alongside the main postback worker.
- Poll query: `ListPostbackDLQForAutoReplay` selects `status='FAILED'`, `replay_count < max_replay_count`, `next_retry_at <= now()`.
- Batch size: 20 rows per tick (`defaultReplayBatch`).
- On replay: same-txn `CreateOutboxEvent` (`SEND_POSTBACK`) + `UpdatePostbackDLQReplay` (`status='RETRIED'`, `last_error='auto-replay'`).
- Backoff after replay N (uses `replay_count` after increment): 5m, 15m, 1h, 6h, 24h (caps at last interval).

### Env

| Env | Default | Effect |
| :--- | :--- | :--- |
| `POSTBACK_AUTO_REPLAY_ENABLED` | enabled when unset | Set `false` or `0` to disable worker ticks |

### Metrics

| Metric | Meaning |
| :--- | :--- |
| `ad_postback_dlq_auto_replay_total` | DLQ rows auto-replayed |
| `ad_postback_dlq_total` | Rows moved to DLQ after dispatch exhaustion |

### Operator steps

1. Ensure `postback-sender` is running and scraping metrics on `:9119` (default).
2. When DLQ grows, inspect `last_error` on `postback_dlq` rows; fix partner URL/auth before expecting auto-replay to succeed.
3. First auto-replay tick can occur within 5 minutes of `FAILED` (initial `next_retry_at`).
4. Delivered posts are idempotent via dispatch slot / `DELIVERED` state; auto-replay must not duplicate HTTP when already delivered.

### Verify

```bash
go test ./internal/postback/replay/ -short -count=1
go test ./internal/postback/ -short -run TestPostback_DLQRetry_afterFailedDispatch_holdout -count=1
```

---

## Postback outbound HMAC signing

When `postback_configs.signing_secret_encrypted` is set (migration `00144_postback_signing_secret.sql`), the postback sender decrypts the secret and signs rendered GET URLs.

### URL template macro

- Include `{sig}` in `postback_configs.url_template`.
- After macro expansion, `internal/postback/signing.SignGETURL` computes `HMAC-SHA256(secret, canonical_url_without_sig)` as lowercase hex.
- `{sig}` is replaced in the final URL. If the rendered URL ends with `sig=` (no value), the hex is appended.

Canonical URL: parsed URL with `sig` query param removed, then `RawQuery` re-encoded (`internal/postback/signing/sign.go`).

Receiver verification contract: slug `postback_outbound_signing` in [ANTIFRAUD.md](ANTIFRAUD.md).

### Verify

```bash
go test ./internal/postback/signing/ -short -count=1
```

---

## Budget failover modes on `/click`

Campaign fields: `budget_failover_mode`, `fallback_click_url` (OpenAPI / `campaigns` table). Normalized in `internal/domain/budget_failover.go`.

| Mode | `/click` when unified filter returns budget reject | Budget debit |
| :--- | :--- | :--- |
| `none` (default) | `402 Payment Required` | No debit on reject path |
| `fallback_url` | `302` to `fallback_click_url` (HTTPS required) | **No debit** on failover redirect |
| `flow_next` | `302` to next flow landing from waterfall selection (`selectFlowLandingWithClickCaps`) | **No debit** on failover redirect |

Implementation: `internal/ingest/budget_failover.go`, triggered from `landing_bundle.go` when `filterRejectBudget` after filter check.

Invariant: failover redirect must not run Lua budget debit. Holdout: `TestBudgetFailover_redirectsNotDebits_holdout`, `TestBudgetFailover_noneReturns402_holdout`.

### Operator steps

1. PATCH campaign with `budget_failover_mode` and `fallback_click_url` (HTTPS required for `fallback_url`).
2. For `flow_next`, configure flow routing (`flow_routing_mode` waterfall) and offer caps before relying on failover.

### Verify

```bash
go test ./internal/ingest/ -short -run TestBudgetFailover -count=1
```

---

## Flow routing modes

Per-flow field `flow_routing_mode` on `flows` table (`internal/flow/store.go`). Normalized in `internal/domain/flow_routing.go`. Selection: `internal/filter/flow_routing.go` `SelectSnapshot`.

| Mode | Offer selection | When no eligible offer |
| :--- | :--- | :--- |
| `weighted` (default) | Hash-weighted pick among non-capped offers | No selection (`404` on `/click` path) |
| `waterfall` | Lowest `offer_priority` first; skips capped / excluded offers | No selection |
| `waterfall_then_landing` | Same waterfall offer pass | Falls back to selected lander URL (static landing) |

Path and lander selection remain weighted by path `weight` and lander `weight` before offer logic. Offer click caps (`cap_daily` / `cap_total` on flow paths) mark offers ineligible in the snapshot.

`flow_next` budget failover reuses waterfall selection with click-cap exclusion (`budget_failover.go`).

### Verify

```bash
go test ./internal/filter/ -short -run 'SelectSnapshot_waterfall|offer_cap' -count=1
```

---

## CH ingest observability

Processor-side proof bundle for event_ts to ClickHouse insert lag and Redis stream backlog.

### Metrics

| Metric | Type | Source | Operator use |
| :--- | :--- | :--- | :--- |
| `ad_ch_ingest_lag_seconds` | Histogram | `internal/stream/clickhouse_store.go` `observeCHIngestLag` | Lag from event `CreatedAt` to successful CH batch insert. Alert when p99 sustained above SLA (start with 60s warning, 120s critical on steady traffic). |
| `ad_processor_stream_xlen` | Gauge `{shard}` | `cmd/processor/main.go` readiness probe (`XLEN` on ingest stream per Redis shard) | Per-shard pending stream length (spec alias `ad_stream_pending_len`). Alert when any `shard` label > 10k for 5m or growth exceeds consumer drain (tune to `REDIS_STREAM_MAXLEN` / `STREAM_MAX_LEN`, default 10000). |
| `ad_ch_single_row_inserts_total` | Counter | ClickHouse poison-pill split path | Spike during load proof indicates batching regression |
| `ad_processor_stream_lag_seconds` | Gauge | Processor PEL lag per instance | `ProcessorStreamLagHigh`: max > 120s for 2m (`PROCESSOR_STREAM_LAG_MAX_SEC` default) |
| `ad_ch_spool_segments` | Gauge | mmap spool when CH down | `ClickHouseSpoolPressure`: >= 6 segments for 2m |
| `ad_processor_stream_backpressure_active` | Gauge | Consumer paused PEL growth during CH outage | `ProcessorStreamBackpressureActive`: == 1 for 5m |

Related (not CH main stream): `ad_fraud_stream_pending` is fraud-stream queue depth, not `ad:events` ingest.

`CH_INGEST_SOURCE=broker` disables Redis `_ch` consumer; use broker lag/divergence metrics instead (`ad_broker_ingest_divergence_high` in shadow).

### Optional load proof

Not in default `pr_fast`.

```bash
LOAD_CH_PROOF=1 bash scripts/test/load/ch_ingest_spike.sh
```

Env knobs: `CH_LAG_P99_MAX` (default 120s), `CH_SINGLE_ROW_MAX_DELTA` (default 50), `BASE_RATE`, `DURATION`.

### Operator steps

1. Rising `ad_processor_stream_lag_seconds`: scale processor workers, restore CH connectivity, check `/ready` on processor (`:8186`).
2. Non-zero `ad_ch_single_row_inserts_total` delta under load: inspect processor logs for poison rows; CH batch split is expected occasionally, sustained growth is not.
3. `ad_ch_ingest_lag_seconds` p99 above 120s with healthy CH: check processor CPU, batch sizes, and Redis PEL depth before tuning CH.

### Verify

```bash
go test ./internal/stream/ -short -run ClickHouse -count=1
```

---

## Bulk-clone lock

`POST /api/v1/campaigns/bulk-clone` clones up to 50 campaigns per request. Concurrent bulk-clone for the same customer is serialized to avoid deadlocks and duplicate name races.

| Surface | Detail |
| :--- | :--- |
| Lock | `pg_advisory_lock(hashtext('bulk_clone:' || customer_id))` |
| Code | `internal/campaign/bulk_clone_lock.go` (`WithBulkCloneCustomerLock`) |
| Idempotency | Per-source key via `BulkCloneIdempotencyKey(bulk_key, source_campaign_id)` |
| Limit | `BulkCloneCampaignMaxSync = 50` sources per HTTP request |

### Operator steps

1. Require `idempotency_key` on bulk-clone requests; retries with the same key return the same cloned IDs.
2. If bulk-clone hangs, check for long-running transactions holding the advisory lock on the same `customer_id`.
3. Parallel bulk-clone across **different** customers is allowed; same customer queues behind the lock.

### Verify

```bash
go test ./internal/campaign/ -short -run TestBulkClone -count=1
```

---

## Team scope

Team ownership filters for admin campaign list and mutation gates. Package: `internal/teamscope`.

### Schema (migration `00142_team_scope.sql`)

| Object | Role |
| :--- | :--- |
| `teams` table | `id`, `customer_id`, `name` |
| `customers.team_enforce_ownership` | `BOOLEAN NOT NULL DEFAULT FALSE` |
| `users.team_id` | FK to `teams` (assigned via member APIs) |

### Enforcement

| Actor | List scope | Mutation |
| :--- | :--- | :--- |
| `MB` (media buyer) | `owner_user_id = self` always | `AssertCampaignAccess` owner match |
| `B` (buyer) | Same as MB when `team_enforce_ownership=true` on customer | Same when enforced |
| `TL` (team lead) | Campaigns owned by any user with same `users.team_id` | Owner must be on TL team |
| Custom masked `ScopeTeam` | MB-style owner filter when `team_enforce_ownership=true` | Owner match when enforced |
| Admin / full read | No owner filter | Unchanged |

Global override: `TEAM_ENFORCE_OWNERSHIP_DEFAULT=true` treats all customers as `team_enforce_ownership=true` without per-row PG read.

Client `owner_user_id` query override is ignored for team-scoped actors without `campaigns:write` (`AllowOwnerQueryOverride`).

Hot path (`/track`, `/click`) does not import `teamscope`.

### APIs

- `GET /api/v1/team/teams`, `POST /api/v1/team/teams` (`team:read` / `team:write`)
- Bulk pause/resume: ownership assert for MB/B when `team_enforce_ownership` or role is MB/B

### Verify

```bash
go test ./internal/teamscope/ -short -count=1
go test ./internal/campaign/ -short -run TestAssertCampaignAccess -count=1
```

# Tracker Enterprise Backend — Technical Specification

Status: **spec only** (not implemented).  
Owner: tracker hot path + control plane cold path.  
Scope: **backend only** — no `web/` changes in this delivery.  
Context: due diligence gaps for **media buying teams** and **CPA networks** (see conversation baseline, March 2026).

Related (partially shipped): `RBAC_ROLE_CONSTRUCTOR_SPEC.md` (role matrix API/UI); this spec closes **enforcement and runtime** gaps that constructor alone does not fix.

Canonical rules: `hot-path.mdc`, `cold-path.mdc`, `control-plane.mdc`, `traffic.mdc`, `data-layer.mdc`, `modular-monolith.mdc`.

---

## 1. Purpose

Close backend gaps that block enterprise sales for:

1. **Media buying teams** — real tenant isolation (not only `MB` + `owner_user_id`), economics masking, multi-user admin safety under concurrent mutations.
2. **CPA / affiliate networks** — postback durability and auth contracts networks expect; cap-aware routing comparable to smartlink/TDS expectations.

### 1.1 In scope (backend)

| Domain | Deliverable |
| :--- | :--- |
| Team isolation | `ScopeTeam` enforcement, masked-role mutation deny, economics redaction on reports/API |
| Ingest observability | Metrics + load-tier proof hooks for spike → CH path (not marketing SLA claims) |
| Admin concurrency | Idempotent bulk paths, bounded report queries, documented lock order |
| Outbound postbacks | DLQ auto-replay, dispatch state machine fix, optional circuit breaker |
| Inbound postback auth | Per-campaign/customer IP allowlist + shared-secret verification on S2S `/track` |
| Outbound postback signing | HMAC query signing for webhook/S2S templates |
| Caps & routing | Offer click caps, waterfall fallback chain, budget-exhausted redirect policy |

### 1.2 Non-goals

| Item | Reason |
| :--- | :--- |
| Admin UI (`web/`) | Explicit operator constraint for this spec |
| Full white-label CPA portal | Separate product surface |
| Per-click ML smartlink | `trafficoptimizer` stays cold-path; no scorer on `/click` |
| Replacing Keitaro/Binom feature parity | Scope bounded to listed gaps |
| RBAC constructor v1.1 (`auth.*` PG tables) | Optional phase; not blocking team isolation v1 |
| Hot-path Postgres / CH on `/track` accept | Architecture invariant (`architecture.mdc`) |

---

## 2. Current state (honest baseline)

### 2.1 Media buying teams

| Area | Today | Gap |
| :--- | :--- | :--- |
| MB isolation | `owner_user_id` filter + `AssertMediaBuyerCampaignAccess` | Only role with ownership enforcement |
| TL visibility | Full **customer** campaign list | No “team group” subset; `ScopeTeam` unused in list filters |
| B (masked) | `ScrubCampaignFields` on campaign DTO | No ownership filter; can pause others’ campaigns; ROI reports leak economics |
| Masked mutations | `IsMaskedMutation` used in audit metadata only | Budget/URL PATCH not rejected server-side |
| Role constructor | `internal/access/` + `deploy/operator/roles.yaml` | Compiles permissions; does not add team filters |
| Multi-user admin | `FOR UPDATE` on lifecycle, `PostgresGate`, outbox `SKIP LOCKED` | No soak proof; bulk clone + heavy reports unbounded |

### 2.2 Ingest → ClickHouse

| Area | Today | Gap |
| :--- | :--- | :--- |
| Buffering | Redis Stream / broker → processor batch (50k / 10s) → `PrepareBatch` + `async_insert` | No CI/load-tier e2e CH row proof |
| Admission | `TryReserve` + 503 overload | Documented; spike demo needs operator metrics |
| Poison path | Binary split → `ad_ch_single_row_inserts_total` | Ops metric exists; no merge gate on spike runs |

### 2.3 CPA postbacks

| Area | Today | Gap |
| :--- | :--- | :--- |
| Queue | Postgres `outbox_events` → `cmd/postback-sender` | No Redis outbound queue (by design) |
| Retry | 5 attempts, exp backoff, 5xx/502 retry | No circuit breaker; no auto DLQ replay |
| DLQ | `postback_dlq` + manual admin/ops retry | `FAILED` dispatch row may block re-dispatch |
| Idempotency | `customer\|click_id\|event_type` hash | OK for outbound |
| Inbound auth | Fraud/datacenter signals | No affiliate IP allowlist; no HMAC on `/track` S2S |
| Outbound signing | Bearer + encrypted tokens | No HMAC URL signing for affiliate templates |

### 2.4 Caps & smart routing

| Area | Today | Gap |
| :--- | :--- | :--- |
| Campaign budget | Unified-filter Lua debit; 402 on exhaust | No failover redirect to backup offer/campaign |
| Daily pacing | EVEN mode in Lua | ASAP burns early (documented) |
| Flow rotator | Weighted lander/offer select | No waterfall; conversion caps only (~30s snapshot) |
| Offer caps | PG conversion counts | No click caps; not real-time |
| Fast click tier | `light` / `redirect_only` skips budget debit | Tradeoff not configurable per campaign |

---

## 3. Goals and success criteria

### 3.1 Functional (team)

1. User with role `B` or custom masked role sees **only owned campaigns** when `team_enforce_ownership=true` on customer (or globally via env).
2. User with `campaigns:read:masked` gets `403` on `PATCH` changing `budget_limit`, `target_url`, `daily_budget`.
3. `GET /api/v1/reports/true-roi` and buyer portfolio endpoints return **redacted economics** for masked snapshots (no raw `revenue_micro` / `true_profit_micro`).
4. `TL` with `team:read` can list campaigns for **team members only** when `users.team_id` is set (not entire customer).
5. `POST /api/v1/campaigns/bulk-clone` with 50 IDs completes without deadlock under 10 concurrent operators (integration tier).

### 3.2 Functional (CPA)

1. Inbound S2S conversion with wrong source IP → `403` when allowlist configured.
2. Inbound S2S without valid `X-Postback-Signature` (or query `sig`) → `403` when secret configured.
3. Advertiser 502 → postback retried → lands in DLQ → **auto-replayed** within 15 minutes without operator click.
4. DLQ manual retry succeeds after prior `FAILED` dispatch (state machine fixed).
5. Campaign with exhausted budget and `fallback_click_url` configured → `/click` returns **302** to fallback (not 402) when policy `budget_failover_mode=fallback_url`.
6. Flow with capped offers tries **waterfall chain** before static landing URL.

### 3.3 Non-functional

| Requirement | Target |
| :--- | :--- |
| Hot path added latency (team filters) | 0 extra PG round-trips on `/track` / `/click` |
| Team list filter | +0–1 PG query on cold admin list (acceptable) |
| Postback auto-replay | At-least-once; no duplicate HTTP when idempotency key `DELIVERED` |
| Offer click cap check | Redis `INCR` p99 < 1 ms per shard (sampled) |
| Config rollout | Feature flags per customer where noted; defaults preserve today behavior |

### 3.4 Definition of done

- [ ] All acceptance scenarios in section 12 pass on T1+ stack (`curl :8188/health`, tracker `:8080/health`).
- [ ] Holdout tests cited in section 13 green in `make test-fast`.
- [ ] Integration/fault rows in section 13 run in `make test-integration` / `make test-fault` where marked.
- [ ] OpenAPI updated for new admin fields and postback auth knobs.
- [ ] `deploy/vendor/VENDOR_OPS_RUNBOOK.md` section for postback auth + DLQ replay (same PR as code).
- [ ] No `web/` diff in PRs claiming this spec (UI follows separately).

---

## 4. Concepts

### 4.1 Team isolation model

```
customer
  └── team (optional team_id on users)
        └── user (owner_user_id on campaigns)
```

| Layer | Enforcement |
| :--- | :--- |
| **Ownership** | `campaigns.owner_user_id` — canonical row owner |
| **Team** | `users.team_id` — TL sees union of campaigns owned by users in same team |
| **Customer** | `users.customer_id` — M/U/TL default boundary |
| **Mask** | `authz.MaskLevel` from permissions — field + report redaction |

`ScopeTeam` in roles YAML becomes **active**: list/query handlers call `TeamScopeFilter` when actor scope is `team` and role is not `MB` (MB keeps owner-only semantics).

### 4.2 Postback auth planes

| Plane | Direction | Mechanism |
| :--- | :--- | :--- |
| **Inbound S2S** | Affiliate → `/track` | Optional IP CIDR allowlist + HMAC-SHA256 over canonical query/body |
| **Outbound** | Tracker → advertiser | Existing retry/DLQ + optional `signing_secret` → `sig=` macro |
| **Durability** | Worker | Auto DLQ replay worker + fixed dispatch FSM |

### 4.3 Cap failover modes (campaign)

| Mode | `/click` when budget exhausted |
| :--- | :--- |
| `reject` (default) | 402 — today |
| `fallback_url` | 302 to `campaigns.fallback_click_url` if set |
| `flow_next` | Delegate to flow waterfall (section 6.4) |

Env: `BUDGET_FAILOVER_DEFAULT=reject` for backward compatibility.

### 4.4 Offer routing modes (flow)

| Mode | Behavior |
| :--- | :--- |
| `weighted` (default) | Today — hash weighted select |
| `waterfall` | Ordered `offer_priority`; skip capped; first eligible wins |
| `waterfall_then_landing` | Then static landing URL; then 404 |

---

## 5. Permission and config surface (backend)

No new permissions required for team isolation v1 beyond existing `team:*`, `campaigns:*`. Admin mutation of new fields uses existing `campaigns:write` / `settings:write`.

### 5.1 New customer-level settings (PG)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `team_enforce_ownership` | bool | `false` | When true, roles with only `campaigns:read:masked` get MB-style owner filter |
| `postback_inbound_ip_allowlist` | text[] | `{}` | CIDR strings; empty = disabled |
| `postback_inbound_secret` | bytea (encrypted) | null | HMAC secret; null = disabled |

Stored in `customers` or `customer_settings` jsonb — prefer dedicated columns + migration for auditability.

### 5.2 New campaign-level fields (PG)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `fallback_click_url` | text | null | Budget failover target |
| `budget_failover_mode` | enum | `reject` | `reject`, `fallback_url`, `flow_next` |
| `click_filter_budget_policy` | enum | `inherit` | `inherit`, `full`, `light_skip_debit`, `redirect_skip_debit` |

### 5.3 New flow offer fields (PG)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `offer_priority` | int | 0 | Waterfall order (lower first) |
| `cap_clicks_daily` | int | 0 | 0 = unlimited |
| `cap_clicks_total` | int | 0 | 0 = unlimited |

Click caps enforced in Redis: `{offer_id}:clicks:daily:{date}`, `{offer_id}:clicks:total`.

---

## 6. Server implementation

### 6.1 Package layout (`modular-monolith.mdc`)

```
internal/teamscope/          # team_id filters, list resolvers (NEW)
  doc.go
  filter.go
  filter_test.go

internal/campaign/mask/      # mutation deny + report scrub helpers (NEW or extend runtime/ops.go)
  enforce.go
  enforce_test.go

internal/postback/inbound/   # S2S auth for /track conversion path (NEW)
  doc.go
  verify.go
  verify_test.go

internal/postback/replay/    # DLQ auto-replay worker (NEW)
  doc.go
  worker.go
  worker_test.go

internal/postback/signing/   # outbound HMAC URL signing (NEW)
  doc.go
  sign.go

internal/filter/offer_click_cap.go   # Redis click caps (extend filter/)
internal/filter/flow_waterfall.go    # waterfall select (extend flow_routing.go)

internal/ingest/budget_failover.go # 402 vs 302 policy on /click (extend landing_bundle.go)

internal/controlplane/teamscope_bridge.go   # wire only
internal/controlplane/postback_replay_bridge.go
```

Forbidden: logic in `controlplane` god files beyond bridges; hot path must not import `postback/replay` or admin packages.

### 6.2 Team isolation (P0)

#### 6.2.1 `TeamScopeFilter`

Central resolver used by:

- `ListCampaigns` / `ListCampaignsFiltered`
- `ListScopedCampaignIDs`
- `GetBuyerPortfolio` (fix missing owner filter)
- Report CH query builder (`reportOwnerUserFilter` extension)

```go
// internal/teamscope/filter.go
func ResolveCampaignListFilter(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (ListFilter, error)
```

| Actor | Filter |
| :--- | :--- |
| `MB` | `owner_user_id = actor` (unchanged) |
| `B` + `team_enforce_ownership` | same as MB |
| `B` without flag | today (masked, all customer) — deprecate default to enforce after one release |
| `TL` + `team_id` set | `owner_user_id IN (SELECT id FROM users WHERE team_id = $team)` |
| `TL` without `team_id` | customer-wide (today) |
| `M`/`U`/`A` | customer-wide |

#### 6.2.2 Masked mutation enforcement

Before campaign PATCH/PUT handlers apply changes:

```go
if authz.IsMaskedMutation(ctx) && mask.TouchesProtectedFields(patch) {
    return 403 FORBIDDEN
}
```

Protected fields: `budget_limit`, `daily_budget`, `target_url`, `creative_payload`, `referrer_filter`, `pacing_mode` spend knobs.

Holdout: `TestMaskedMutation_rejectsBudgetPatch_holdout` — invert `if` → test must fail.

#### 6.2.3 Economics redaction

| Endpoint | Rule |
| :--- | :--- |
| `GET /api/v1/reports/true-roi` | If `snap.Mask == MaskMasked`, zero or omit profit/revenue columns |
| `GET /api/v1/dashboards/buyer-portfolio` | Apply `TeamScopeFilter` + mask KPIs |
| Campaign list `margin_breach` | Omit for masked snapshot |

Export CSV: reuse `ExportProfileBuyerSummary`; extend for true-roi job specs.

#### 6.2.4 Pause scope for `B`

`POST bulk-action` pause/resume: call `AssertMediaBuyerCampaignAccess` for **all** team-scoped roles when `team_enforce_ownership` or role is `MB`/`B`.

#### 6.2.5 Schema migration

```sql
-- migrations/XXXX_team_scope.sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS team_id UUID NULL REFERENCES teams(id);
CREATE TABLE IF NOT EXISTS teams (
  id UUID PRIMARY KEY,
  customer_id UUID NOT NULL REFERENCES customers(id),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);
```

Invite/patch member APIs accept `team_id` (backend only; UI later). `GET /api/v1/team/teams` list for assignment.

### 6.3 Ingest observability (P1)

Not a feature — **operator proof bundle**:

| Metric / gate | Action |
| :--- | :--- |
| `ad_ch_ingest_lag_seconds` | New histogram: event_ts → CH insert (processor) |
| `ad_stream_pending_len` | Per-shard XLEN alert threshold doc |
| `scripts/test/load/ch_ingest_spike.sh` | Optional tier: N RPS 60s, assert lag p99 < X, `ad_ch_single_row_inserts_total` delta < Y |
| CI | Gate **optional** (`LOAD_CH_PROOF=1`); never in default `pr_fast` |

No change to batch sizes in v1 unless load proof fails.

### 6.4 Admin concurrency (P1)

| Surface | Mitigation |
| :--- | :--- |
| `POST /api/v1/campaigns/bulk-clone` | `pg_advisory_xact_lock(hashtext('bulk_clone:'||customer_id))`; idempotency key required |
| Heavy reports | Enforce `max_date_range_days` server-side (default 90); `400` if exceeded |
| Campaign PATCH | Keep `FOR UPDATE`; document lock order in `internal/campaign/lifecycle.go` comment |

Integration: `TestBulkClone_concurrentTen_noDeadlock` with 10 parallel httptest + real PG (testcontainers).

### 6.5 Postback durability (P0)

#### 6.5.1 Dispatch FSM fix

States: `PENDING` → `IN_FLIGHT` → `DELIVERED` | `FAILED`.

| Transition | Rule |
| :--- | :--- |
| DLQ retry | Reset `FAILED` → `PENDING` or new row; `resolveDispatchSlot` must accept retry |
| Success | `DELIVERED` immutable |

Holdout: `TestPostback_DLQRetry_afterFailedDispatch_holdout`.

#### 6.5.2 Auto-replay worker

New daemon tick in `cmd/postback-sender` or sidecar goroutine:

1. `SELECT * FROM postback_dlq WHERE status='FAILED' AND next_retry_at <= now() AND replay_count < max`
2. Exponential schedule: 5m, 15m, 1h, 6h, 24h (configurable)
3. Re-enqueue outbox `SEND_POSTBACK` with same payload hash
4. Metric: `ad_postback_dlq_auto_replay_total`

Manual retry remains; auto is additive.

#### 6.5.3 Circuit breaker (P1)

Per host: after N consecutive failures in 5m, open circuit 10m; metric `ad_postback_circuit_open_total`; fail fast to DLQ with reason `circuit_open`.

### 6.6 Postback auth (P0–P1)

#### 6.6.1 Inbound verification (`/track` S2S)

Hook in conversion ingest path **before** filter debit (or immediately after parse): `internal/postback/inbound/verify.go`.

| Check | Config source |
| :--- | :--- |
| IP allowlist | `customer.postback_inbound_ip_allowlist` |
| HMAC | Header `X-Postback-Signature: sha256=<hex>` over `method|path|sorted_query|body_sha256` |

Failure: `403` with `POSTBACK_AUTH_FAILED` (not silent accept).

Env kill-switch: `POSTBACK_INBOUND_AUTH_DISABLED=1` (dev only; CI holdout ensures prod path fails closed when configured).

#### 6.6.2 Outbound signing (P1)

When `postback_configs.signing_secret` set:

- Macro `{sig}` = `HMAC-SHA256(secret, canonical_url_without_sig)`
- Document in `deploy/vendor/ANTIFRAUD.md` slug `postback_outbound_signing`

### 6.7 Caps & routing (P1–P2)

#### 6.7.1 Offer click caps (Redis)

On `/click` after offer select:

1. `INCR` daily + total keys with TTL
2. If over cap, treat offer as ineligible (same as conversion cap skip)
3. Snapshot refresh: include click counts in flow snapshot builder (poll PG + Redis merge)

#### 6.7.2 Flow waterfall

`flow_routing_mode` on flow (enum `weighted` | `waterfall`). Implementation in `SelectSnapshot`:

- Sort offers by `offer_priority`
- First offer passing geo/device + **both** conversion and click caps wins

#### 6.7.3 Budget failover on `/click`

In `landing_bundle.go` after filter returns code `3` (budget):

```go
switch campaign.BudgetFailoverMode {
case FallbackURL:
    return 302 fallback_click_url
case FlowNext:
    return selectWaterfallOffer(...)
default:
    return 402
}
```

**Invariant:** failover path must not debit budget (no Lua spend on rejected-then-redirect).

#### 6.7.4 Fast tier budget policy

Campaign field `click_filter_budget_policy`:

| Value | Behavior |
| :--- | :--- |
| `full` | Always run unified filter debit |
| `light_skip_debit` | Today `light` behavior |
| `redirect_skip_debit` | Today `redirect_only` |

Global tier from license/env remains; per-campaign override for operators who need caps on fast paths.

---

## 7. HTTP API (backend additions)

Prefix `/api/v1`. Session auth unchanged. No new routes required for core behavior; extensions below.

| Method | Path | Permission | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/team/teams` | `team:read` | List teams for customer |
| `POST` | `/api/v1/team/teams` | `team:write` | Create team |
| `PATCH` | `/api/v1/customers/{id}/postback-auth` | `settings:write` | IP allowlist + inbound secret rotate |
| `PATCH` | `/api/v1/campaigns/{id}` | `campaigns:write` | New fields: `fallback_click_url`, `budget_failover_mode`, `click_filter_budget_policy` |
| `PATCH` | `/api/v1/flows/{id}/offers/{oid}` | `campaigns:write` | `offer_priority`, `cap_clicks_*` |
| `GET` | `/api/v1/postbacks/dlq` | `postbacks:read` | Include `next_retry_at`, `replay_count` |
| `POST` | `/api/v1/postbacks/dlq/{id}/retry` | `postbacks:write` | Fixed FSM (section 6.5.1) |

OpenAPI: `api/openapi/paths/team_scope.yaml`, extend `campaigns.yaml`, `postbacks` paths.

---

## 8. Security

| Threat | Mitigation |
| :--- | :--- |
| TL escalates to customer-wide via API param | Ignore client `owner_user_id` override for `ScopeTeam` without `campaigns:write` |
| Inbound auth bypass | Verify before debit; config required per customer; no global disable in prod compose |
| Secret leakage in audit | `postback_inbound_secret` never returned; rotate via PATCH with empty read |
| Failover URL open redirect | Validate URL scheme `https` only (or allowlist host suffix per customer) |
| Auto-replay storm | `replay_count` cap + circuit breaker per host |
| Hot-path PG for team filter | Team membership cached in Redis `team:members:{team_id}` TTL 60s; invalidation on member PATCH |

---

## 9. Acceptance scenarios

### 9.1 Team isolation

| ID | Scenario | Expected |
| :--- | :--- | :--- |
| TI-1 | `B` + `team_enforce_ownership=true`, list campaigns | Only owned rows |
| TI-2 | `B` PATCH `budget_limit` on owned campaign | `403` masked mutation |
| TI-3 | `TL` with `team_id`, list campaigns | Only campaigns owned by users in team |
| TI-4 | `MB` GET teammate’s campaign by ID | `403` |
| TI-5 | Masked `true-roi` report | No `true_profit_micro` in JSON |
| TI-6 | `B` bulk pause on foreign campaign | `403` when enforce ownership |

### 9.2 Postbacks

| ID | Scenario | Expected |
| :--- | :--- | :--- |
| PB-1 | S2S `/track` from non-allowlisted IP | `403` |
| PB-2 | Valid IP, bad HMAC | `403` |
| PB-3 | Advertiser 502 × 5 → DLQ → wait auto-replay | Second HTTP attempt within 15m |
| PB-4 | Manual DLQ retry after FAILED dispatch | `200` delivery; dispatch row consistent |
| PB-5 | Outbound URL with `{sig}` | Receiver can verify HMAC |

### 9.3 Caps & routing

| ID | Scenario | Expected |
| :--- | :--- | :--- |
| CR-1 | Budget exhausted, `budget_failover_mode=fallback_url` | `302` to configured URL |
| CR-2 | All offers click-capped, waterfall mode | Next offer in priority order |
| CR-3 | All offers capped, `waterfall_then_landing` | Static landing URL |
| CR-4 | `click_filter_budget_policy=full` on `redirect_only` tier | Budget Lua runs (latency tradeoff accepted) |

### 9.4 Concurrency

| ID | Scenario | Expected |
| :--- | :--- | :--- |
| CC-1 | 10 parallel bulk-clone same customer | All complete; no deadlock |
| CC-2 | Report query 365d range | `400` exceeds max window |

---

## 10. Testing and CI

| Tier | Command | Scope |
| :--- | :--- | :--- |
| Unit | `go test ./internal/teamscope/ -short -count=1` | filters |
| Unit | `go test ./internal/campaign/mask/ -short -count=1` | mutation deny |
| Unit | `go test ./internal/postback/inbound/ -short -count=1` | HMAC/IP |
| Unit | `go test ./internal/postback/ -short -run 'DLQ|Retry|Replay' -count=1` | FSM + replay |
| Holdout | `TestMaskedMutation_rejectsBudgetPatch_holdout` | mutation gate |
| Holdout | `TestPostback_DLQRetry_afterFailedDispatch_holdout` | DLQ path |
| Holdout | `TestBudgetFailover_redirectsNotDebits_holdout` | no debit on 302 failover |
| Integration | `make test-integration` + `team_scope_integration_test.go` | PG team_id |
| Integration | `postbacks_inbound_auth_integration_test.go` | httptest + miniredis |
| Fault | `make test-fault` | broker unchanged; postback replay optional tier |
| Load | `LOAD_CH_PROOF=1 scripts/test/load/ch_ingest_spike.sh` | optional |
| Static | `bash scripts/ci/static/team_scope_gate.sh` | masked endpoints list uses `TeamScopeFilter` |

Do not claim CPA postback durability from mock HTTP only (`anti-slop.mdc`).

---

## 11. Rollout plan

| Phase | Deliverable | Depends on |
| :--- | :--- | :--- |
| **P0a** | Team schema (`teams`, `users.team_id`), `TeamScopeFilter`, masked mutation deny, portfolio fix | — |
| **P0b** | Postback FSM fix + auto-replay worker | — |
| **P0c** | Inbound postback IP + HMAC verification | customer settings migration |
| **P1a** | Economics report redaction; TL team list; pause scope for `B` | P0a |
| **P1b** | Outbound HMAC signing; circuit breaker | P0b |
| **P1c** | Bulk-clone lock + report date window | — |
| **P2a** | Offer click caps (Redis) + waterfall routing | — |
| **P2b** | Budget failover modes on `/click` | P2a optional |
| **P2c** | Per-campaign fast-tier budget policy | — |
| **P3** | CH ingest lag metric + optional load script | ops |
| **P4** | (Optional) `team_enforce_ownership` default true; `B` breaking change announcement | P1a |

Feature flags:

| Flag | Default |
| :--- | :--- |
| `TEAM_ENFORCE_OWNERSHIP_DEFAULT` | `false` |
| `POSTBACK_AUTO_REPLAY_ENABLED` | `true` |
| `POSTBACK_INBOUND_AUTH_REQUIRED` | `false` (per-customer config) |
| `BUDGET_FAILOVER_ENABLED` | `true` |

---

## 12. Documentation updates (same PR as each phase)

| File | Change |
| :--- | :--- |
| `deploy/vendor/VENDOR_OPS_RUNBOOK.md` | Postback auth, DLQ auto-replay, failover modes |
| `deploy/vendor/ANTIFRAUD.md` | Inbound HMAC slug, outbound signing |
| `docs/ARCHITECTURE.md` | Team scope diagram, budget failover branch |
| `.cursor/rules/control-plane.mdc` | `ScopeTeam` enforcement note |
| `.cursor/rules/traffic.mdc` | Inbound postback auth hook |
| `api/openapi/openapi.yaml` | New paths and fields |

---

## 13. Open decisions (operator sign-off)

| # | Question | Recommendation |
| :--- | :--- | :--- |
| D1 | Default `team_enforce_ownership` for new customers? | **false** v1; opt-in per customer |
| D2 | TL without `team_id` sees customer-wide or deny? | **Customer-wide** (today) with warning in audit |
| D3 | Budget failover charges spend? | **No** — redirect only when debit would fail |
| D4 | Auto-replay max attempts after DLQ? | **5** over 24h then terminal |
| D5 | Inbound HMAC on all `/track` or conversion-only? | **Conversion events only** v1 |
| D6 | Waterfall cross-campaign spillover? | **v1 no** — same flow only |
| D7 | Click caps in Redis vs PG? | **Redis INCR** hot path; PG reconcile nightly |

---

## 14. References

| Artifact | Path |
| :--- | :--- |
| RBAC constructor (shipped/partial) | `RBAC_ROLE_CONSTRUCTOR_SPEC.md`, `internal/access/` |
| MB owner filter | `internal/campaign/misc_helpers.go` (`AssertMediaBuyerCampaignAccess`) |
| Mask scrub | `internal/campaign/runtime/ops.go` (`ScrubCampaignFields`) |
| Unified filter / budget | `internal/filter/unified/unified-filter.lua` |
| Flow routing | `internal/filter/flow_routing.go` |
| Click path | `internal/ingest/landing_bundle.go` |
| Postback worker | `internal/postback/postback_sender_worker.go` |
| Outbox enqueue | `internal/postback/conversion_outbox.go` |
| CH batching | `internal/stream/clickhouse_store.go`, `internal/stream/processor.go` |
| Architecture | `docs/ARCHITECTURE.md` |

---

*End of specification.*

# Arbitrage / Keitaro-gap closure backlog

Product positioning: **ad-event-processor** is an event/budget/fraud processor with tracking, not a 1:1 Keitaro clone. This backlog closes **operational gaps** that make arbitrage teams reject the stack on a sales call, without regressing hot-path invariants (`architecture.mdc`, `hot-path.mdc`).

**Status baseline (2026-09):** backlog **closed** for P0–P5 shipped waves; hot path production-grade (gnet, Redis Lua, async CH). Optional/deferred items remain under each epic **Not done** subsection (XDP TCP opt emit, alloc-gate paste, fault-tier CH backpressure).

---

## Cross-cutting SLA (non-negotiable)

These ceilings apply to **all** items below. New work must not violate them.

| Surface | Metric | Ceiling | Abort / fail |
| :--- | :--- | :--- | :--- |
| Tracker `/track`, `/click` | `ad_http_request_duration_seconds` p95 | < 50 ms | Load-test abort if p99 > 80 ms for 30 s (`core.mdc`) |
| Tracker p99 | hard ceiling | < 100 ms | |
| Filter deadline | `FILTER_TIMEOUT_MS` (prod) | <= 100 ms | |
| Redis unified-filter Lua | p99 per shard | < 10 ms | |
| Geo filter | p99 (sampled) | < 10 us | |
| Hot path Postgres | synchronous per request | **0** (except stale-registry grace; see P2-REGISTRY-STALE) | |
| Hot path ClickHouse | synchronous per request | **0** | |
| Sync Redis per accept | `EVALSHA` count | <= 1 (0 with local-quanta full-skip) | |
| Postgres budget | `current_spend <= budget_limit` | +/-1 micro-unit | `AssertBudgetInvariant` on fault tests |
| Admin API body | max size | 64 KiB | `pkg/coldpath.DefaultMaxBody` |
| Domain TLS issuance (P0/P1) | park + cert ready | <= 60 s p95 per hostname (operator SLO; not hot path) | |
| Postback dispatch | success rate (healthy config) | >= 99% over 24 h (excluding partner 4xx) | DLQ growth alert |

**Verification tiers** (cite in every PR):

| Tier | Command | Proves |
| :--- | :--- | :--- |
| Fast | `bash scripts/ci/pr_fast.sh` | Static + unit + holdouts |
| Integration | `make test-integration` | Redis/PG/CH testcontainers |
| Fault | `make test-fault` | Budget invariant, broker cutover |
| Hot alloc | `make test-alloc-gate` | Changed ingest/filter files |
| Admin web | `bash scripts/ci/admin/web.sh` | OpenAPI + TS + UI gates |
| Load | `malformed.sh` / `parser_chaos_load.sh` | Tracker p99 under traffic |

---

## Priority matrix

| Priority | Theme | Outcome for arbitrage buyer |
| :--- | :--- | :--- |
| **P0** | Finish half-shipped integrations + SSL + RBAC UI | "Postbacks and domains work like a real tracker" |
| **P1** | Ops at scale (bulk domains, TDS UX, presets, observability) | "50 campaigns/day without SSH" |
| **P2** | Hot-path footguns + latency tradeoffs | "20k RPS without PG surprises; optional fast click" |
| **P3** | Fraud/review positioning + incremental threat-intel signals; **conversion FP collateral** (CGNAT, relay, WebView) | Honest limits; tighten gaps vs residential headless crawlers; measurable FP budget |
| **P4** | Research-tier passive/runtime probes | Not a sales promise; lab + corpus only |
| **P5** | Perimeter & commercial intelligence protection (AppSec) | Anti-scraping, Sybil/coordinated probe defense, zone parity |

---

## P0 — must ship before Keitaro comparison pitch

**Wave status:** closed (2026-09).

### P0-STATUS-MAPPING-INGEST

**Status:** closed (2026-09).

**Problem:** `ApplySchema` sets `campaigns.status_integration_schema_id`, but `MapAffiliateStatus` in `internal/integrationschema/schema.go` is **test-only**. Runtime mapping uses manual `PUT /api/v1/campaigns/{id}/conversion-mappings` via `ConversionPayoutApplier` (`internal/stream/conversion_payout.go`).

**User stories:**

- Buyer applies affiliate network preset; inbound `status` / `affiliate_status` / `conversion_status` on `POST /track` maps to `goal_name` + `payout_micro` without hand-editing mappings.
- Re-apply preset updates mappings idempotently (same external status -> same row).

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| Load schema | On conversion ingest (processor batch) and/or tracker accept path: resolve `status_integration_schema_id` -> embedded YAML via `integrationschema` catalog |
| Map | Call `MapAffiliateStatus`; on miss: reject with `conversion_reject` code or pass-through per campaign flag |
| Persist | Optionally auto-upsert `campaign_conversion_mappings` on apply (transaction with `status_integration_schema_id` update) |
| Wire keys | Accept `status`, `affiliate_status`, `conversion_status` (existing); document optional alias `lead_status` -> same mapper |

**Frontend scope (`web/`):**

| Task | Path |
| :--- | :--- |
| Apply preset -> mappings | `integrations_affiliate_presets.tsx` today is **read-only**; add "Apply to campaign" -> existing integration apply API |
| Campaign ops | `campaign_ops_panel.tsx` / `use_campaign_ops_panel_workspace.ts`: show linked `status_integration_schema_name`; button "Sync from preset" |
| Editor integrations | Surface `status_integration_schema_name` from campaign DTO (`types_bundle.go`) |

**OpenAPI:** extend apply-integration response with `mappings_applied_count`; document reject behavior on unknown external status.

**DoD:**

- [x] `MapAffiliateStatus` called from production path (processor batch schema fallback + apply sync via `integrationschema`)
- [x] Holdout: conversion with `status=sale` + preset maps payout; unknown status -> documented reject or default
- [x] Apply schema populates or refreshes `campaign_conversion_mappings` without manual PUT
- [x] UI: preset apply shows success count or field-level 400
- [x] `go test ./internal/stream/ -run ConversionPayout -count=1`
- [x] `go test ./internal/integrationschema/ -count=1`
- [x] Integration test: processor batch with mapped payout + `AssertBudgetInvariant` unaffected

**SLA:** mapping lookup in processor batch: **< 1 ms p99 per event** (in-memory schema cache; no per-event PG).

**Dependencies:** none.

---

### P0-GOOGLE-OFFLINE-CONVERSIONS

**Status:** closed (2026-09).

**Problem:** `internal/postback/provider_google.go` posts to placeholder `customers/default/offlineUserDataJobs:run`. Not production Google Ads API.

**User stories:**

- Buyer sends offline conversion with `gclid` + value; worker creates/runs OfflineUserDataJob or uses UploadClickConversions (API version pinned in config).
- Invalid customer ID / OAuth errors surface in DLQ with actionable message.

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| Auth | Service account or OAuth refresh token from encrypted postback config |
| Flow | Create job -> add operations -> run (or single-shot upload API per Google doc) |
| Config | `customer_id`, `conversion_action_id`, `developer_token` in postback config DTO |
| Idempotency | Reuse `ResolveEventID` + `postback_dispatches` hash |

**Frontend scope:**

| Task | Path |
| :--- | :--- |
| Provider form | `postback_config_form.tsx`: Google-specific fields (customer ID, conversion action, auth) |
| Validation | Client C2 hints; server S4 required |

**DoD:**

- [x] No hardcoded `customers/default` in production path
- [x] `go test ./internal/postback/ -run Google -count=1` with httptest mock of full job lifecycle
- [x] Staging gate: `bash scripts/ci/static/capi_staging.sh` extended or sibling `google_offline_staging.sh`
- [x] DLQ row shows Google API error body (truncated, no secrets)
- [x] OpenAPI postback config schema updated

**SLA:** dispatch latency same as other providers (async worker); **no** Google HTTP on tracker hot path.

**Dependencies:** none.

---

### P0-WILDCARD-SSL-DNS01

**Status:** closed (2026-09).

**Problem:** `SetupDomainSSL` shells out to per-host certbot/Caddy (`domain_health_handlers.go`). No `*.domain` DNS-01 / wildcard. Arbitrage teams burn domains in bulk.

**User stories:**

- Operator adds tracking domain pool with wildcard cert `*.trk.example.com` in < 60 s without SSH.
- New subdomain automatically TLS-allowed via Caddy on-demand + `IsTLSAllowed` gate.

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| DNS-01 | ACME DNS-01 via Cloudflare API (`cloudflare_client.go`); store cert metadata in PG |
| Wildcard | Issue `*.zone` + apex optional; renew worker (control-plane tick or cron job doc) |
| API | `POST /api/v1/ops/domains/wildcard-ssl` (zone_id, pool_id); bulk status in domain health |
| TLS gate | `IsTLSAllowed` matches pool members + wildcard SAN |

**Frontend scope:**

| Task | Path |
| :--- | :--- |
| Wildcard wizard | `domains_directory.tsx` + `use_domains_page_workspace.ts`: zone picker, pool, progress, renew status |
| Bulk table | Columns: SSL expiry, ACME state, Cloudflare proxied |

**DoD:**

- [x] DNS-01 issuance tested against Cloudflare sandbox or recorded httptest
- [x] Renew path documented; metric `ad_event_processor_domain_ssl_renew_total`
- [x] No SSH required for standard deploy (`DOMAIN_SSL_SETUP_ENABLED` path documented)
- [x] `go test ./internal/platformadmin/domains/ -run Domain -count=1`
- [x] UI: ErrorBlock on failure; no fake empty table

**SLA:** issuance p95 < 60 s (ACME + DNS propagation); hot path unchanged.

**Dependencies:** `CLOUDFLARE_*` configured (`domain_park.go`).

---

### P0-UI-PERMISSION-GATE

**Status:** closed (2026-09).

**Problem:** Server RBAC is real (`RequirePermission`, `roles.yaml`), but **no React route-level guard** (`control-plane.mdc` Known gaps). Deep links show shell; security is API-only.

**User stories:**

- Media buyer opens `/ops/dlq` -> forbidden page, not empty chrome.
- Nav hide is UX only; route guard matches OpenAPI `x-permissions`.

**Frontend scope:**

| Task | Path |
| :--- | :--- |
| `PermissionGate` | New `web/src/shell/permission_gate.tsx`: props `permission` / `permissionAny`; children or `<ForbiddenPanel />` |
| Router | Wrap protected routes in `web/src/pages/*` (ops, audit, settings, rtb, billing write) |
| Bootstrap | Fix session bootstrap permissions drift vs live `Snapshot` (P2-SESSION-PERMS related) |
| Forbidden panel | Uses `ErrorBlock` + 403 copy from `admin_error.ts`; optional link home |

**Backend scope:**

- [x] Audit: smoke-matrix primary GETs + ops section list GETs return 403 for MB where `RequirePermission` denies (`TestManagementAPI_RoleMediaBuyerSmokePrimaryGET_holdout`, `web/e2e/permission_route_audit.spec.js`; manifest in `rbac_smoke_route_audit_test.go` + `MEDIA_BUYER_SMOKE_PRIMARY_GET_AUDIT`)
- [x] Route permissions: `bootstrap.user.permissions` + `web/src/lib/route_permissions.ts` (no separate `/session/route-permissions` endpoint)
- [x] Out of scope (documented): full `NAV_GROUPS` + `EXTRA_ROUTE_RULES` parity audit per URL

**DoD:**

- [x] `PermissionGate` on all routes in `NAV_GROUPS` with `permission` / `permissionAny` (global `RoutePermissionGuard`)
- [x] Playwright **L3**: seeded MB role -> `/ops` shows forbidden panel, `GET /api/v1/ops/home` = 403 (`web/e2e/permission_gate.spec.js`; requires `ADMIN_E2E_MB_EMAIL` / `ADMIN_E2E_MB_PASSWORD`)
- [x] Playwright **L1**: smoke-matrix primary GET RBAC for MB (`web/e2e/permission_route_audit.spec.js`, `@L1` API contract)
- [x] No `user?.role === 'MB'` string gates (**RB-C5**)
- [x] `cd web && npm run typecheck` + `bash scripts/ci/admin/web.sh`

**SLA:** N/A (cold path).

**Dependencies:** none.

---

## P1 — operational parity with Keitaro/Binom workflows

**Wave status:** closed (2026-09).

### P1-DOMAINS-BULK-LIFECYCLE

**Status:** closed (2026-09).

**Problem:** `use_domains_page_workspace.ts` supports one hostname at a time (add, park, probe, SSL). No bulk import, burn-list, or auto-attach to campaign pool.

**Backend:**

| Endpoint | Behavior |
| :--- | :--- |
| `POST /api/v1/ops/domains/bulk` | CSV/json hostnames -> queue park + health probe |
| `POST /api/v1/ops/domains/bulk-ssl` | Async job; poll `GET .../jobs/{id}` |
| `PATCH /api/v1/ops/domains/{host}/burn` | Mark burned; remove from TLS allow; optional CF delete |

**Frontend (`domains_directory.tsx`):**

- Bulk paste textarea + file upload
- Job progress panel (reuse reportjob pattern)
- Filter: healthy / degraded / burned

**DoD:**

- [x] 100 domains import completes without blocking HTTP handler (async goroutine worker; poll `GET .../jobs/{id}`)
- [x] Integration test with fake Cloudflare client
- [x] UI coalescing on refresh (**R1** 500 ms)
- [x] OpenAPI + `domains_api.ts`
- [x] E2E **L1**: `web/e2e/domains_bulk.spec.js` POST bulk CSV -> 202 job

**SLA:** bulk job throughput >= 10 domains/min (limited by ACME rate limits, not PG).

---

### P1-TDS-STREAM-UX

**Status:** closed (2026-09).

**Problem:** Flow engine exists (`internal/flow/`, bandit on click) but admin UX is a **JSON textarea** (`flows_directory.tsx`, `flow_detail.tsx`) — not Keitaro-style stream editor.

**User stories:**

- Buyer builds flow: landers + offers + weights visual editor; links campaign `flow_id`.
- Clone flow/campaign; split test presets; preview generated click URL.

**Backend:**

- Mostly done: `flow` CRUD, `selectFlowLanding` on click
- Add: `POST /api/v1/flows/{id}/clone`, validate weights sum via existing `ValidateFlowPathShape`

**Frontend:**

| Component | Detail |
| :--- | :--- |
| `flow_editor_visual.tsx` | Rows: lander picker, offer picker, weight %, device/geo filters (server-stored) |
| Campaign wizard | Link flow in `campaign_wizard_panel.tsx` |
| Offers/landers pickers | Reuse `offers_directory.tsx`, `landers_directory.tsx` comboboxes |

**DoD:**

- [x] No raw JSON required for default path (JSON advanced panel ok)
- [x] Weight sum != 100 -> 400 from server + inline ErrorBlock
- [x] E2E **L2**: create flow -> attach to campaign -> curl `/click` lands on weighted URL
- [x] `go test ./internal/flow/ -run Validate -count=1`

**SLA:** click path unchanged (flow selection after filters; budget in `core.mdc`).

---

### P1-INTEGRATION-ONE-CLICK

**Status:** closed (2026-09).

**Problem:** Integration schemas exist (`deploy/schemas/`, `integration/handlers.go`) but buyer must wire postback URL, CAPI token, and mappings separately.

**User stories:**

- "Connect Admitad / Facebook / Google" wizard: apply inbound + outbound + status preset + postback config in one transaction.
- Campaign Integration tab shows receive URL with macros (`BuildAffiliateReceivePanelURL`).

**Backend:**

- Extend `applySchema` to atomically: postback_config row, conversion-mappings from preset, `integration_schema_id` + `status_integration_schema_id`
- Dry-run endpoint returns sample postback payload

**Frontend:**

| Path | Detail |
| :--- | :--- |
| `campaign_integration_panel` + campaign wizard | Wizard steps: pick network -> credentials -> test postback |
| `campaign_editor` integrations | Copy buttons for click URL + postback URL |

**DoD:**

- [x] Single apply -> all artifacts created; rollback on partial failure (PG tx)
- [x] `tests/integration/postbacks_admin_test.go` extended
- [x] UI: toast only after 2xx (**EC5**)
- [x] E2E **L1**: `web/e2e/integration_one_click.spec.js` dry-run postback template

---

### P1-POSTBACK-HEALTH-DASHBOARD

**Status:** closed (2026-09).

**Problem:** `integrations_postbacks.tsx` has configs / DLQ / campaign status tabs but lacks **SRE-style health**: success %, latency, last error by provider.

**Backend:**

| Endpoint | Detail |
| :--- | :--- |
| `GET /api/v1/integrations/postbacks/health` | Aggregates from `postback_dispatches` (PG): success_rate_24h, p95_latency_ms, last_error per campaign/provider |
| Reuse | `internal/reports/postback_recon.go` queries |

**Frontend:**

- New tab "Health" on postbacks page
- Per-campaign row: RAG badges; link to DLQ retry

**DoD:**

- [x] Metrics match PG dispatch log reconciliation within 1% on integration test fixture
- [x] Alert threshold documented: success < 95% -> runbook link
- [x] No client-side aggregation over full dispatch log (**cold-path**)
- [x] E2E **L1**: `web/e2e/integrations_postbacks_health.spec.js`

**SLA:** health endpoint p95 < 200 ms (CH pre-agg or materialized view).

---

### P1-CAMPAIGN-CLONE-BULK

**Status:** closed (2026-09).

**Problem:** Arbitrage teams duplicate 20 campaigns/day. Clone exists partially via editor; no bulk clone with flow/domain/postback bundle.

**Backend:** `POST /api/v1/campaigns/bulk-clone` (source ids, customer scope, optional rename pattern)

**Frontend:** campaigns directory multi-select + clone dialog (reuse `campaign_wizard_panel` patterns)

**DoD:**

- [x] RBAC: buyer sees only own campaigns
- [x] Clone copies flow_id, postback configs, conversion mappings
- [x] Holdout: budget fields reset; no spend copy (`TestBulkCloneCampaignsHTTP_holdout` flow + postback rows)
- [x] E2E **L1**: `web/e2e/campaign_bulk_clone.spec.js`

---

## P2 — hardening and latency tradeoffs

**Wave status:** closed (2026-09).

### P2-FAST-CLICK-TIER

**Status:** closed (2026-09).

**Problem:** Every `/click` runs full `FilterEngine` + Redis before 302 (`landing_bundle.go`). Correct for fraud-heavy traffic; heavy for pure TDS where buyer wants minimal TTFB.

**Design:**

| Mode | Filters | Redis | Use case |
| :--- | :--- | :--- | :--- |
| `full` (default) | Full chain | <= 1 EVALSHA | Production fraud/budget |
| `light` | license, geo, emergency, registry | 0-1 EVALSHA | Trusted internal streams |
| `redirect_only` | parse + macro only | 0 | **License-gated**; audit log only |

**Config:** campaign flag `click_filter_tier` or env default; **fail closed** if tier misconfigured on licensed fraud features.

**DoD:**

- [x] `light` p99 click redirect < 15 ms on load test (no fraud filters)
- [x] `full` unchanged: `make test-alloc-gate`
- [x] Holdout: `redirect_only` cannot debit budget without explicit flag
- [x] Document in `traffic.mdc` + campaign editor UI select

**SLA:** `light` tier: tracker p95 < 25 ms at 10k RPS (control cohort).

**Frontend:** `campaign_editor_advanced_panel.tsx` — tier select with warning copy.

---

### P2-REGISTRY-STALE-PG-GRACE

**Status:** closed (2026-09).

**Problem:** `REGISTRY_STALE_PG_GRACE=true` (default) allows sync PG read on registry cache miss (`registry_ops.go`). Under misconfigured pub/sub this is the **only** path to "PG on click".

**Tasks:**

- [x] Production profile: document `REGISTRY_STALE_PG_GRACE=false` + alert on `registry_stale`
- [x] Metric: `ad_event_processor_registry_stale_pg_reads_total`
- [x] Circuit: max N PG reads/sec during stale mode -> 503 fail-closed
- [x] Runbook in `docs/DEVELOPMENT.md`

**DoD:**

- [x] Fault test: stale mode + grace off -> 503, zero PG queries on hot path
- [x] Existing `TestLookupCampaign_stalePGGraceDisabled503_holdout` cited in PR

**SLA:** PG reads on hot path = **0** in recommended prod config.

---

### P2-CLICK-PROXY-GUARDRAILS

**Status:** closed (2026-09).

**Problem:** `click_proxy.go` blocks worker on upstream (10-30 s timeouts). Bad upstream = click-to-landing drop.

**Tasks:**

- [x] Hard cap `CLICK_PROXY_TIMEOUT_MS` default 300 ms (config)
- [x] On timeout: fallback 302 to landing URL + metric `ad_event_processor_click_proxy_fallback_total`
- [x] Campaign flag to disable proxy on timeout vs hard fail

**DoD:**

- [x] `go test ./internal/ingest/ -run ClickProxy -count=1`
- [x] Load test: proxy timeout does not exhaust worker pool (`TestClickProxy_holdoutBurstSlowUpstreamBounded`; `click_ingress_latency_drill.sh`)

**SLA:** proxy attempt p95 < 300 ms or fallback.

---

### P2-CLICK-INGRESS-LATENCY-BUDGET

**Status:** closed (2026-09).

**Type:** Technical Task  
**Threat:** T27 — synchronous blocking I/O on click init (full filter chain, blocking upstream proxy, or external verdict HTTP) inflates TTFB/FCP; mobile users abandon before lander paint.

**Problem:** `landing_bundle.go` runs `FilterEngine` + Redis before 302; `click_proxy.go` may block on upstream fetch. Third-party TDS stacks often add a blocking `cURL` to a cloud verdict API on first hop — same failure mode. Cited ops impact: 5–10% session abort on unstable radio when extra RTT exceeds render patience.

**User Story:** As a buyer, I need paid clicks to reach the lander or a deterministic fallback within a bounded latency budget; never a white screen while waiting on an unbounded external verdict.

**Technical Task:**

| Task | Detail |
| :--- | :--- |
| Budget table | Document per-path ceilings: gnet parse + local filters, Redis EVALSHA, click_proxy upstream, safe-page verify — each with env default ms |
| Ban hot-path external verdict | No new sync outbound HTTP to third-party decision APIs on `/click` gnet path; async cache + stale-while-revalidate only |
| Deadline fail-open policy | On `FILTER_TIMEOUT_MS` or proxy timeout: 302 to configured landing (or decoy when policy says fail-closed), metric `ad_click_ingress_latency_budget_exceeded_total{stage}` |
| Dashboard | Grafana: click TTFB p50/p95/p99 by `click_filter_tier`, proxy fallback rate, filter timeout rate |
| Cross-ref | `P2-FAST-CLICK-TIER` (reduce work), `P2-CLICK-PROXY-GUARDRAILS` (proxy cap), `P5-ROUTING-TIMING-CONSTANT-TIME` (pad without adding blocking I/O) |

**Acceptance Criteria:**

- [x] `traffic.mdc` + `docs/DEVELOPMENT.md`: click latency budget table with env knobs
- [x] Holdout: simulated 2 s upstream proxy does not block worker past `CLICK_PROXY_TIMEOUT_MS` (fallback 302)
- [x] Holdout: filter deadline exceeded -> documented route (not hung connection)
- [x] Load test artifact: control cohort click p95 < 50 ms (`core.mdc`) with `full` tier at reference RPS OR explicit waiver with `light` tier default for TDS campaigns
- [x] CI grep gate: no `http.Get` / `curl` / blocking client on `landing_bundle.go` click hot path except click_proxy (allowlisted)

**Dependencies:** P2-FAST-CLICK-TIER, P2-CLICK-PROXY-GUARDRAILS.

**SLA:** click ingress p95 < 50 ms (`core.mdc`); proxy sub-step p95 < 300 ms or fallback.

---

### P2-SESSION-PERMS

**Status:** closed (2026-09).

**Problem:** `GET /api/v1/session/bootstrap` permissions may drift from DB grants vs live policy `Snapshot`.

**Tasks:**

- [x] Bootstrap returns same permission set as `RequirePermission` uses
- [x] On grant/revoke: refetch bootstrap after role change
- [x] Web: nav updates without full re-login

**DoD:**

- [x] `go test ./internal/controlplane/ -run RBAC -count=1`
- [x] UI nav updates after admin grants perm

---

## P3 — honest fraud / review positioning (optional product)

**Wave status:** closed (2026-09).

### P3-SAFE-PAGE-LIMITS-DOC

**Status:** closed (2026-09).

**Problem:** Buyers expect WebGL/headless rejection of all scrapers; stack offers Public Safe Sandbox attestation (`safe_page_attest.go`) and review traffic routing — not universal coverage. Residential gateway masks L4/L7 desync (server Linux stack vs claimed mobile UA).

**Tasks:**

- [x] Operator doc: signal matrix (ingress TCP/TLS/H2, safe-page JS, cross-layer) with **evades** column (CDN, residential IP, attestation off)
- [x] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` — threat catalog T1–T30, Blue Team remediation, operator checklist (P5 parent doc)
- [x] Admin UI: link from fraud presets / safe-page panel to limitations doc
- [x] Cross-ref backlog slugs: P3-MOBILE-BIOMETRICS-CLICK, P3-TLS-JA4-BROWSER-CORPUS, P3-MODERATOR-FINGERPRINT-CORPUS, P3-CROSS-LAYER-DESYNC-POLICY, P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY, P3-APPLE-PRIVATE-RELAY-ASN-POLICY, P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING, P4-*, P5-VISION-MULTIMODAL-MODERATION-AGENTS, P5-CDP-TRUSTED-INPUT-HARDENING, P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK

**DoD:**

- [x] `bash scripts/ci/naming/antifraud_doc.sh` green
- [x] No doc claims "eliminated" Redis ops (`anti-slop.mdc`)
- [x] Doc states residential crawler on datacenter egress is **not** fully detectable at L4 alone

---

### P3-ML-FRAUD-POSITIONING

**Status:** closed (2026-09).

**Problem:** Hot path reads boost snapshot only; LGBM is `cmd/fraud-scorer` batch.

**Tasks:**

- [x] SKU/license copy: ML enforcement = sidecar + CH batch
- [x] Admin: show last model score timestamp per campaign (cold API)

**DoD:**

- [x] No comment or UI implying inline ML on `/track`

---

### P3-RESIDENTIAL-PROXY-EDGE

**Status:** closed (2026-09).

**Problem:** XDP blocks known L4; rotating residential evades. Crawler egress IP looks residential while TCP/TLS stack stays Linux server.

**Tasks:**

- [x] Document: edge XDP = flood + blocklist; `OS_FINGERPRINT_MISMATCH_ENABLED=false` on CDN paths
- [x] Sales copy: `ResidentialProxyFilter` = farm/intel heuristic, not per-session unauthorized agent proof

**DoD:**

- [x] `deploy/vendor/ANTIFRAUD.md` aligned with code
- [x] Moderator corpus work tracked under P3-MODERATOR-FINGERPRINT-CORPUS (not this slug)

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY

**Status:** closed (2026-09).

**Type:** Technical Task (policy + hot-path guardrails)  
**Threat:** T24 — crawler on carrier CGNAT poisons `/24` or single-IP reputation; thousands of legit mobile subscribers inherit deny/safe-page route.

**Problem:** Cited ops FP: 15–20%. `FraudBlacklistFilter` (`SISMEMBER blacklist:fraud`), manual edge deny lists, and BPF LPM promotion can block at IP granularity. `CGNAT_MOBILE_IP_BYPASS` and `syn_subnet_ratelimit_v4` mitigate **rate** not **hard deny** collateral.

**User Story:** As a buyer on mobile carrier traffic, I need fraud enforcement on session/device signals, not blanket punishment of a carrier NAT egress shared with a crawler.

**Technical Task:**

| Task | Detail |
| :--- | :--- |
| Audit deny paths | Inventory every IP-level block: Redis `blacklist:fraud`, XDP sync, nginx deny, `review_traffic_policy` IP tuple |
| CGNAT-aware deny | For mobile ASN tier (`P5-ASN-MOBILE-TIER`): forbid auto-promote `/24` from single-IP fraud event; require `probe_cluster` or device graph corroboration |
| TTL decay | IP deny entries: max TTL env `FRAUD_IP_DENY_TTL_SEC` (default 24h); decay metric on re-hit |
| Bypass hook | Extend `CGNAT_MOBILE_IP_BYPASS` to skip **hard** blacklist route when `netintel.MobileCarrierTier` + no device graph hit |
| Metrics | `ad_cgnat_collateral_route_total{reason}` — safe/decoy route where IP deny skipped due to CGNAT policy |
| CH drill | Report: FP proxy = safe routes on mobile ASN with zero probe/crowd signals |

**Acceptance Criteria:**

- [x] `go test ./internal/filter/ -short -run CGNAT -count=1`
- [x] Holdout: single crawler IP on mobile ASN in blacklist does not hard-route sibling CGNAT session when bypass on
- [x] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` T24 remediation section
- [x] No new `/24` auto-block without human review flag

**Dependencies:** P5-ASN-MOBILE-TIER, P5-XDP-RESIDENTIAL-POLICY-BOUNDARY, P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-APPLE-PRIVATE-RELAY-ASN-POLICY

**Status:** closed (2026-09).

**Type:** Technical Task  
**Threat:** T25 — iCloud Private Relay egress (Cloudflare/Fastly ASNs) misclassified as datacenter; high-value iOS users sent to safe/decoy.

**Problem:** Cited ops FP: 10–15%. `FraudFilter.checkDCASN` and stale DCASN tables treat relay egress like hosting ASN.

**User Story:** As a buyer targeting iOS, I need Private Relay users evaluated on browser/attestation signals, not ASN substring alone.

**Technical Task:**

| Task | Detail |
| :--- | :--- |
| Allowlist feed | Cold-path worker: Apple-published relay egress prefixes / ASN set; versioned snapshot in Redis `netintel:apple_relay:v1` |
| Filter gate | When relay fingerprint matches (ASN + optional `APPLE_PRIVATE_RELAY_UA_HINT`): skip DCASN hard reject; still run UA/JA4/attestation |
| Admin | Fraud preset note: relay exempt from DCASN only, not from behavioral probes |
| Metric | `ad_apple_relay_exempt_total` vs `ad_apple_relay_reject_total` |

**Acceptance Criteria:**

- [x] `go test ./internal/filter/netintel/ -short -run PrivateRelay -count=1`
- [x] Holdout: relay ASN + valid iOS UA passes DCASN; datacenter UA on same ASN still fails cross-layer when enabled
- [x] `PERIMETER_INTEL_DEFENSE.md` T25 row + operator refresh cadence for Apple feed

**Dependencies:** P3-TLS-JA4-BROWSER-CORPUS, P3-CROSS-LAYER-DESYNC-POLICY.

---

### P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING

**Status:** closed (2026-09).

**Type:** Technical Task  
**Threat:** T26 — in-app WebView (Facebook, Instagram, TikTok, etc.) presents truncated UA, non-Safari JA4, missing `Sec-CH-UA`; classifiers flag anomaly -> safe page.

**Problem:** Cited ops FP: 10–15%. Partial mitigations: `social_in_app` preset, `UAMatchesInAppWebView`, `TestSecFetchAnomaly_holdoutWebViewBypass`, `landing_tls_webview_hook_test.go`.

**User Story:** As a buyer running social in-app traffic, I need default-safe classification for known WebView containers without treating every degraded header set as fraud.

**Technical Task:**

| Task | Detail |
| :--- | :--- |
| Corpus | Expand `P3-TLS-JA4-BROWSER-CORPUS` with FB/IG/TikTok WebView JA4 rows; holdout per platform |
| Sec-CH relax | When `UAMatchesInAppWebView`: skip `SecFetchAnomaly` hard fail; log `in_app_webview` reason tag |
| Campaign default | Onboarding wizard: social traffic -> enable `social_in_app` + tooltip for TLS/JA4 limits |
| Attestation | Do not require full safe-page kinematics on first click from in-app; defer to verify hop |
| Metric | `ad_inapp_webview_classified_total{platform}`; CH column `in_app_webview` on click events |

**Acceptance Criteria:**

- [x] `go test ./internal/filter/ -short -run WebView -count=1`
- [x] `go test ./internal/ingest/ -short -run WebView -count=1`
- [x] Holdout: Instagram WebView fixture UA + corpus JA4 -> production route (not decoy) with preset on
- [x] Holdout: desktop Chrome with WebView UA substring alone does not bypass (cross-layer still applies)

**Dependencies:** P3-TLS-JA4-BROWSER-CORPUS, P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-MOBILE-BIOMETRICS-CLICK

**Status:** closed (2026-09).

**Problem:** Gyro/touch biometrics (`summarizeMobileBiometrics`, `mobile_biometrics` CH columns) run on **conversion** when `BEHAVIOR_TELEMETRY_ENABLED` + sandbox attestation. Headless unauthorized agent on `/click` never sends `devicemotion` / touch pressure; flat gyro passes conversion-only checks too late.

**User stories:**

- Buyer enables safe-page attestation; click probe rejects emulator with zero gyro variance before offer redirect.
- Touch events include `force` / `radiusX` / `radiusY` where supported; flat pressure on claimed mobile fails attestation.

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| Telemetry on click | Reuse `BehaviorTelemetryFilter` path for attestation verify (not only `evt.Type == conversion`) behind campaign + env flag |
| Fingerprint wire | Extend `SafePageVerifyFingerprint` + OpenAPI: `touch_force`, `touch_radius_x/y`, gyro sample count |
| Rules | Fail codes: `gyro_flat`, `touch_pressure_missing` (mobile UA only); holdouts mirror `safe_page_attest.go` style |
| CH | Copy `mobile_gyro_*` / `mobile_touch_count` on click rows when telemetry present |

**Frontend scope (`web/`):**

| Task | Path |
| :--- | :--- |
| Attestation probe JS | Ship in safe-page stub / lander template docs (not admin SPA logic) |
| Campaign editor | Safe-page panel: toggle "Require mobile biometrics on click" with limitation tooltip |

**DoD:**

- [x] Holdout: flat gyro on mobile UA -> attestation fail; human curve + gyro noise -> pass
- [x] Holdout: click path skips when attestation off (no false positive on desktop)
- [x] `go test ./internal/track/ -run SafePage -count=1`
- [x] `go test ./internal/ingest/ -run BehaviorTelemetry -count=1`
- [x] Hot path: zero extra Redis; filter work stays local + existing attestation round-trip
- [x] OpenAPI fingerprint schema updated when admin exposes fields

**SLA:** attestation verify handler p95 < 50 ms (cold stub path); no sync PG/CH on verify.

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-TLS-JA4-BROWSER-CORPUS

**Status:** closed (2026-09).

**Problem:** `ja4BrowserCorpusMismatch` and `TLSFingerprintImpersonating` are heuristic. Scanner stack: UA claims iPhone Safari, JA3/JA4 from Chromium/automation — needs version-pinned corpus rows, not blocklist-only.

**Baseline (shipped):** edge capture (`edge-tls-fingerprint.lua`), `DeviceFilter`, `tls_fingerprint_block_enabled`, `review_traffic_policy` TLS safe view.

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| Corpus | Embedded YAML: `ua_family` x `min_ios` / `chrome_major` -> allowed JA4 set; overlay dir same pattern as `TCPSynSigCorpus` |
| Hot signal | Expand `ja4BrowserCorpusMismatch`; metric `ad_tls_ja4_corpus_mismatch_total` |
| Review traffic | Optional: corpus miss routes to safe page when `review_traffic_action=safe_page` (config flag) |
| Cold | Extend `ivt_tcp_edge_correlation` seed cases (Chrome UA + Python JA3 holdout already in `tcp_edge_rule_fault_test.go`) |

**DoD:**

- [x] Holdout: Safari iOS UA + Chromium JA4 -> `tls_ja4_mismatch`
- [x] Holdout: matching corpus row -> no signal
- [x] `go test ./internal/ingest/ -short -run JA4BrowserCorpus -count=1`
- [x] Feed refresh fail-open retains prior snapshot (`testing.mdc`)
- [x] No claim of full ClientHello extension-order parity (document gap -> P4 if needed)

**SLA:** corpus lookup < 1 us p99 (in-memory snapshot); no per-event HTTP.

**Dependencies:** none.

---

### P3-MODERATOR-FINGERPRINT-CORPUS

**Status:** closed (2026-09).

**Problem:** `review_traffic_policy` matches TLS blocklist, CIDR, proxy/VPN, threat intel IP — not a **learned corpus** of scanner JA3/JA4 + TCP sig + sandbox fingerprint tuples from CH.

**User stories:**

- Operator exports unauthorized-agent sessions from `layer-desync-drilldown` / `wire-signal-breakdown`; imports corpus row for future Public Safe Sandbox Zone routing.
- Known AdNet reviewer IP range + TLS hash -> auto safe page without hand-editing blocklist.

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| CH ingest | Materialized or batch job: aggregate `(ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count)` for `review_routed_event=1` |
| Cold API | `POST /api/v1/fraud/moderator-corpus` (RBAC `fraud:write`): upsert tuple -> Redis/CH index |
| Click path | `detectReviewTrafficMatch` consults corpus after existing signals (fail-open if snapshot empty) |
| Admin | Import CSV; preview match count last 7d |

**Frontend scope:**

| Task | Path |
| :--- | :--- |
| Corpus manager | `web/src/domains/fraud/` new section or fraud presets tab: upload, last refresh, match rate |

**DoD:**

- [x] Integration: seed corpus row -> `GET /click` with matching JA3 serves safe view
- [x] Holdout: empty corpus -> behavior unchanged
- [x] `go test ./internal/ingest/ -run ReviewTraffic -count=1`
- [x] RBAC: MB cannot import corpus; Playwright L3 on new page
- [x] No hot-path CH query; snapshot reload <= 60 s

**SLA:** corpus snapshot read on click < 500 ns (atomic pointer); refresh async.

**Dependencies:** P3-RESIDENTIAL-PROXY-EDGE (doc), reports `layer-desync-*` stable.

---

### P3-CROSS-LAYER-DESYNC-POLICY

**Status:** closed (2026-09).

**Problem:** `layer_desync_count` and reports (`layer-desync-summary`, `wire-signal-breakdown`) are **analytics only**. Buyer wants campaign action when contradictions align (e.g. mobile UA + `os_fingerprint_mismatch` + `tls_ja4_mismatch` + `webgl_vendor_mismatch` on same click).

**Backend scope:**

| Task | Detail |
| :--- | :--- |
| Policy | Campaign flag `cross_layer_desync_action`: `off` / `boost` (default) / `safe_page` / `block` |
| Scoring | Reuse existing fraud signal weights in `filters_chain.go`; count L2+ mismatches above threshold on same event |
| CH | Persist `cross_layer_desync_fired` bool on click for funnel reports |
| Review | Do not duplicate full ML; threshold table in campaign fraud config DTO |

**Frontend scope:**

| Task | Path |
| :--- | :--- |
| Campaign fraud panel | Select desync action + threshold (integer 2-5 layers) |
| Copy | Warn: residential IP may still pass individual L2 signals |

**DoD:**

- [x] Holdout: 3 configured mismatch signals -> `safe_page` when action set; 1 signal -> no route change
- [x] Holdout: `off` preserves current behavior
- [x] `go test ./internal/ingest/ -run LayerDesync -count=1`
- [x] `make test-alloc-gate` if hot filter path touched
- [x] OpenAPI campaign fraud fields documented

**SLA:** desync tally <= 200 ns per event (bitmask over existing signals); no extra Redis.

**Dependencies:** P3-TLS-JA4-BROWSER-CORPUS (optional, for richer TLS leg).

---

## P4 — research tier (anti-scanner reconnaissance; not sales SLA)

**Wave status:** closed (2026-09) except XDP TCP option emit (deferred).

Items below close **technical gaps vs dedicated anti-bot stacks**. Do not pitch on sales call until P3 docs ship and load-tier proof exists.

### P4-TCP-SYN-OPTION-CORPUS

**Status:** closed (2026-09); XDP emit deferred.

**Problem:** `TCPSynSigMismatch` hashes `ttl + window + mss + doff` only (`hash_tcp_syn_fields` in `edge_filter.c`). Does not encode full TCP option order (NOP, MSS, SACK, WScale, Timestamps) that distinguishes Linux 6.x from iOS stack behind residential tunnel.

**Scope (current wave):** tracker filter + nginx forward + ops seed path. **XDP bpf emit deferred** to distant backlog (not near-term).

**Tasks:**

- [x] Spec: option-order fingerprint format + corpus file layout (edge + tracker parity) — `pkg/tcpsynopt`, `edge-tcp-sig-v2.lua`
- [x] Tracker: `X-TCP-SIG-V2` parse, `tcp_syn_opt_mismatch` signal, corpus overlay (`TCP_SYN_OPT_CORPUS_ENABLED`, default off)
- [x] Nginx: `edge-tcp-fp-sync` + `edge-ingress` forward `tcp_opt_trace` -> `X-TCP-SIG-V2`
- [x] Ops seed: `internal/edge.Record` `TCPOptTrace` -> Redis `tcp_opt_trace`
- [ ] **Deferred (distant backlog):** XDP extend emit struct / `hash_tcp_syn_opt_order` in `edge_filter.c`; bpf-sync ringbuf
- [x] Holdout: Linux corpus row vs iOS UA -> signal; CDN skip documented (`edge.mdc`, `ANTIFRAUD.md`)

**DoD:**

- [x] `go test ./internal/edge/ -run TCPSyn -count=1` (integration tier) / `go test ./pkg/tcpsynopt/ ./internal/ingest/ -short -run TCPSynOpt -count=1`
- [x] `bash scripts/test/edge/lua_tests.sh unit` (`tcp_sig_v2_test.lua`)
- [x] Feature flag default **off**; `OS_FINGERPRINT_*` skip metrics unchanged when headers absent

**SLA:** XDP emit path unchanged until deferred slice ships (`xdp-bpf.mdc`).

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P4-H2-FRAME-DYNAMICS

**Status:** closed (2026-09).

**Problem:** `L7WireFilter` checks static H2 SETTINGS and pseudo-header order. Does not detect automator multiplex timing (SETTINGS / WINDOW_UPDATE / HEADERS order and priority vs mobile WebKit).

**Tasks:**

- [x] Edge: capture first N frame types + timestamps (ms bucket) -> `X-H2-FRAME-TRACE` (bounded header)
- [x] Tracker: `H2FrameTraceMismatch(ua, trace)` with embedded WebKit/Chrome corpora
- [x] Chaos: `TestChaos_CrossHop_NginxGnet` row for new header

**DoD:**

- [x] Holdout: canned automator trace -> signal; Safari corpus trace -> clean
- [x] Default env **off** (`H2_FRAME_TRACE_ENABLED=false`)
- [x] No dynamic Prometheus labels on hot path

**Dependencies:** none.

---

### P4-CLIENT-RUNTIME-DEEP-PROBES

**Status:** closed (2026-09).

**Problem:** Public Safe Sandbox attestation lacks WebGPU pipeline timing, IEEE 754 canvas noise probes, `Performance.now()` / `Proxy` hook detection on `navigator` — techniques used by unauthorized inspection agents.

**Tasks:**

- [x] Safe-page JS module (opt-in per campaign): shader compile timing bucket, float noise fingerprint, getter timing probe
- [x] Server: evaluate in `EvaluateSafePageAttestation`; fail codes documented
- [x] Privacy review: disclose in operator doc; EU SKU note

**DoD:**

- [x] `FuzzSafePageVerifyParse` extended; no panic on malformed probe payloads
- [x] `go test ./internal/track/ -run SafePage -count=1`
- [x] Explicitly **not** on `/track` hot path — attestation POST only

**Dependencies:** P3-MOBILE-BIOMETRICS-CLICK (shared probe delivery).

---

### P5-WASM-ATTEST-C-MODULE

**Problem:** JS attest/PoW logic is trivially patchable in DevTools; need shared client/server compute module with minimal bundle and bounded server sandbox.

**Scope:** C `wasm32-unknown-unknown`, zero imports, ~2.4 KiB `.wasm`; wazero host for parity/dry-run; browser loader lazy on strict tier only.

**Status:** closed (2026-09) except optional `aad_pow_bench` export.

#### Done

- [x] `wasm/attest/` C module (`attest.c`, `sha256.c`, `abi.h`): PoW parity `pkg/antifraudtelemetry`, IEEE754 float-noise hash, deterministic micro-bench
- [x] SHA-256: FIPS padding in `aad_sha256_final` only (never via `update`); ring-buffer `w[16]` in `transform`; internal `aad_sha256_digest` + export `aad_sha256_one_shot(msg_off, msg_len, out_off)`
- [x] `pkg/wasmattest` wazero sandbox: module size cap, memory page cap, call timeout, zero-import verifier, `SHA256OneShot` host helper
- [x] Holdout tests: PoW parity, float-noise hex, NIST-style vectors (`""`, `"abc"`, 56/64/80 B) vs `crypto/sha256`
- [x] `scripts/build/wasm_attest.sh` + `scripts/ci/static/wasm_attest_gate.sh` (24 KiB cap; current ~2392 B raw)
- [x] `internal/track/wasm_attest_loader.js` + static routes `GET /static/wasm-attest-loader.js`, `GET /static/attest.wasm` (`//go:embed attest.wasm` in `static_assets.go`)
- [x] Strict lander: `antifraud_telemetry.js` `solvePoW('/static/attest.wasm', ...)` on strict tier; safe-page stub loads wasm attest path
- [x] fraudadmin `POST /api/v1/fraud/wasm-attest/dry-run` (`wasm_attest_handlers.go`, wazero sandbox parity)
- [x] `scripts/ci/static/track_hot_bundle_wasm_gate.sh` in `pr_fast.sh` (hot bundle must not reference `attest.wasm`)
- [x] WASM risk pass: no imports, export bounds checks; not on `/track` pixel path (`pkg/wasmattest/doc.go`, `docs/DEVELOPMENT.md`)
- [x] DoD: `bash scripts/ci/static/wasm_attest_gate.sh`; `go test ./pkg/wasmattest/ -short -run WasmAttest -count=1`; bundle <= 24 KiB raw
- [x] DoD: end-to-end strict tier loader fetch + PoW + attestation wasm nonce path wired

#### Not done

- [ ] Optional: `aad_pow_bench` export for PoW loop perf measurement in browser/wazero

---

### P5-STATIC-POLYMORPH-INSTALL

**Status:** closed (2026-09).

**Problem:** All appliances ship identical `attest.wasm` and `track_pixel.js` hashes. One operator blocklist can correlate every tenant.

**Scope (rational MVP):**

| Layer | Behavior |
| :--- | :--- |
| WASM | `WASM_ATTEST_SEED` compile-time junk rodata via `scripts/build/gen_wasm_polymorph_header.sh`; ABI unchanged |
| Pixel | `build_track_pixel.mjs --seed` adds unique banner; semantics unchanged |
| Install | `scripts/install/polymorph_static.sh` writes `${INSTALL_ROOT}/var/static-polymorph/` + manifest |
| Runtime | Tracker loads overrides from `TRACKER_STATIC_POLYMORPH_DIR` or default install path |

**DoD:**

- [x] Distinct SHA-256 per seed; wazero `VerifyModule` on variants (`wasm_polymorph_gate.sh`)
- [x] `go test ./internal/track/ -run Polymorph -count=1`
- [x] Default embed path unchanged for dev/CI (`WASM_ATTEST_SKIP_EMBED=1` on install builds)
- [x] Operator doc: run `polymorph_static.sh` after install; set `TRACKER_STATIC_POLYMORPH_DIR` when non-default

**Not in MVP:** per-request polymorphism, JS AST rewrites, DOM structure randomization.

**Dependencies:** P5-WASM-ATTEST-C-MODULE (done).

---

## P5 — perimeter & commercial intelligence protection (AppSec)

**Theme:** Harden **Public Safe Sandbox Zone** vs **Internal Trusted Production Zone** separation against **unauthorized third-party scanners**, **distributed coordination attacks (Sybil)**, and **commercial intelligence scraping**. Close client attestation gaps; enable measurable perimeter drills.

**Reference:** `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` (threat catalog T1–T16).

**P5 status: closed (2026-09)** except optional/deferred items listed under each card **Not done**.

**Cross-cutting rules:**

| Rule | Detail |
| :--- | :--- |
| Zone naming | Docs/UI: Public Safe Sandbox Zone / Internal Trusted Production Zone (code symbols `safe_page_*` unchanged) |
| Hot path | P5 client JS via `internal/track/` embed; no sync PG on `/click` |
| Verify tier | Each card cites holdout + named drill command |

### Threat catalog (summary)

| ID | Threat | Backlog slug(s) | Residual risk |
| :--- | :--- | :--- | :--- |
| T1 | Dual-origin fetch: sandbox HTML/URL ≠ production zone | P5-DECOY-LANDING-PARITY, P5-CONTENT-DIFF-DRILL, P3-MODERATOR-FINGERPRINT-CORPUS | Medium on corpus miss |
| T2 | Human-in-the-loop Sybil on residential cellular | P5-SYBIL-HUMAN-OPERATOR-RUNBOOK | High (operational controls) |
| T3 | Empty sandbox hydrator (attestation surface open) | P5-SAFE-PAGE-HYDRATOR-CLIENT | Low after ship |
| T4 | Static sandbox DOM ≠ production asset graph | P5-DECOY-LANDING-PARITY, P5-BEHAVIOR-MODEL-HUMANIZATION, P4-CLIENT-RUNTIME-DEEP-PROBES | Medium |
| T5 | Multi-hop redirect fingerprint (DMR) | P5-REDIRECT-PROFILE-COMPLIANCE | Low on strict profile |
| T6 | Third-party telemetry script origin | P5-FIRST-PARTY-PIXEL-ORIGIN | Low when first-party enabled |
| T7 | Server conversion event without browser correlate | P5-CAPI-BROWSER-DEDUP | Medium without operator wiring |
| T8 | iframe sandbox embedding probe | P5-SAFE-PAGE-HYDRATOR-CLIENT | Medium |
| T9 | Internal zone headers in responses | (done) edge strips | Low |
| T10 | Threat intel corpus false positive | P3-MODERATOR-FINGERPRINT-CORPUS QA | Operator tuning |
| T11 | Coordinated probe-operator micro-behavior | P5-CROWD-PROBE-SCORING, P5-BEHAVIOR-MODEL-HUMANIZATION | Medium |
| T12 | Cross-session device reuse after storage wipe | P5-PROBE-CLUSTER-GRAPH, P5-ASN-MOBILE-TIER | Medium |
| T13 | Hybrid AI + crowd wave reconnaissance (distributed testing) | P5-HYBRID-CROWD-WAVE-DETECTION | Medium |
| T14 | Client-edge DOM analysis on buyer browser (on-device ML) | P5-CLIENT-EDGE-DOM-INTEGRITY | High (client out-of-band) |
| T15 | TLS server persona / H2 interrogation (L4 infra fingerprint) | P5-TLS-SERVER-PERSONA-HARDENING, P4-H2-FRAME-DYNAMICS | Medium |
| T16 | Routing timing side-channel (TTFB differential by ingress class) | P5-ROUTING-TIMING-CONSTANT-TIME, P5-CLICK-TIMING-WIRE | Medium |
| T17 | Hot-path telemetry filter heap allocs (GC tail @ 40k RPS) | P5-HOTPATH-TELEMETRY-ZERO-ALLOC | Low (operator alloc-gate paste) |
| T18 | Tier B worker pool saturation (Redis/filter occupancy) | P5-TIERB-OCCUPANCY-BUDGET | Medium (availability; sub-budget doc shipped) |
| T19 | Client RTT probe decoupled from tracker (`/favicon.ico`) | P5-CLIENT-RTT-PROBE-CORRELATION | Low after ship |
| T20 | CH/Redis ingest burst (stream trim / WAL disk) | P5-INGEST-SINK-BURST-RESILIENCE | Medium (fault observable test open) |
| T21 | Residential proxy IPs in XDP blocklist (CGNAT collateral) | P5-XDP-RESIDENTIAL-POLICY-BOUNDARY | Low when policy followed |
| T22 | Client-side safe-page reveal without server verify verdict | P5-HYBRID-SERVER-VERIFY-GATE | Low after ship |
| T23 | Antifraud snapshot scoring blind spots (empty kinematics) | P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS, P5-CDP-TRUSTED-INPUT-HARDENING | Low (optional webgl/MAC/PoW items) |
| T24 | CGNAT `/24` / IP-reputation collateral (legit mobile routed to safe) | P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY, P5-XDP-RESIDENTIAL-POLICY-BOUNDARY | High (15–20% FP cited ops) |
| T25 | Apple Private Relay / iCloud+ egress misclassified as DC | P3-APPLE-PRIVATE-RELAY-ASN-POLICY | Medium (10–15% FP cited ops) |
| T26 | In-app WebView (FB/IG/TikTok) TLS/Sec-CH/JA4 false blocks | P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING | Medium (10–15% FP cited ops) |
| T27 | Sync blocking I/O on click init (TTFB/FCP white screen) | P2-CLICK-INGRESS-LATENCY-BUDGET, P2-FAST-CLICK-TIER, P2-CLICK-PROXY-GUARDRAILS | High (5–10% timeout cited ops) |
| T28 | Vision / multimodal moderation agents (screenshot diff) | P5-VISION-MULTIMODAL-MODERATION-AGENTS, P5-CONTENT-DIFF-DRILL, P5-CLIENT-EDGE-DOM-INTEGRITY | High (client out-of-band) |
| T29 | CDP `Input.dispatch*` with `isTrusted:true` | P5-CDP-TRUSTED-INPUT-HARDENING, P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS | High |
| T30 | Custom Chromium + residential ASN (webdriver scrubbed) | P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK, P3-CROSS-LAYER-DESYNC-POLICY, P5-CROWD-PROBE-SCORING | High |

**Operational audit (2026-09): conversion funnel false positives**

Third-party TDS / moderation stacks and **this tracker** share failure modes when validation is too coarse or synchronous. Map cited ops impact to backlog slugs (targets are operator-measured FP budgets, not guarantees):

| Defect | Mechanism | Cited ops impact | Existing mitigations in tree | Backlog slug |
| :--- | :--- | :--- | :--- | :--- |
| CGNAT IP collateral | `/24` or single-IP deny after crawler on carrier NAT | 15–20% FP | `CGNAT_MOBILE_IP_BYPASS`, `syn_subnet_ratelimit_v4` (rate not hard ban), `FraudBlacklistFilter` per-IP | P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY |
| Apple Private Relay | Relay egress ASN = CDN/DC in stale DCASN table | 10–15% FP | Partial: geo-only; no relay allowlist | P3-APPLE-PRIVATE-RELAY-ASN-POLICY |
| In-app WebView | Truncated UA, JA4/TLS unlike mobile Safari/Chrome | 10–15% FP | `social_in_app` preset, `UAMatchesInAppWebView`, `TestSecFetchAnomaly_holdoutWebViewBypass` | P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING |
| Sync blocking verdict | Blocking HTTP/cURL or full filter+Redis before first byte | 5–10% timeout | `FILTER_TIMEOUT_MS`, `CLICK_PROXY_TIMEOUT_MS` (partial) | P2-CLICK-INGRESS-LATENCY-BUDGET |

**Adversary audit (2026-09): advanced moderation agents**

| Vector | What bypasses naive defenses | Shipped signal | Backlog slug |
| :--- | :--- | :--- | :--- |
| LLM + vision (multi-viewport screenshots) | Public `/lp/{id}/` + sandbox vs prod diff | Decoy parity, DOM lint (regex only) | P5-VISION-MULTIMODAL-MODERATION-AGENTS |
| Residential custom Chromium | Clean `navigator.webdriver`, perfect ASN | `ResidentialProxyFilter`, mobile ASN tier, cross-layer | P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK |
| CDP trusted synthetic input | `isTrusted:true` on dispatched events | Kinematics + bezier; explicit CDP comment in `score.go` | P5-CDP-TRUSTED-INPUT-HARDENING |

---

### P5-SAFE-PAGE-HYDRATOR-CLIENT

**Type:** Technical Task  
**Threat:** T3, T8 — unauthorized agents map sandbox attestation API without client integrity checks.

**User Story:** As a Blue Team engineer, I need the Public Safe Sandbox Zone to execute a signed client attestation probe so server-side `EvaluateSafePageAttestation` receives canvas/audio/WebRTC/timezone/behavior evidence before any Internal Trusted Production Zone iframe is shown.

**Status:** closed (2026-09).

#### Done

- [x] `internal/track/safe_page_hydrator.js` (embed source of truth via `//go:embed` in `safe_page.go`)
- [x] Emit `SafePageVerifyFingerprint` + `events[]` per `safe_page_attest.go`
- [x] Arm listeners (>=18 events); `POST /track/verify` with `campaign_id` from stub query
- [x] On pass: apply production iframe from `html_content`; on fail: remain in sandbox (existing server codes)
- [x] Share listener helpers with P5-BEHAVIOR-MODEL-HUMANIZATION
- [x] Hydrator: canvas A/B (`canvas_retest_enabled`), audio, WebRTC, timezone, WebGL, languages, viewport
- [x] `performance.now()` timestamps on pointer/touch/scroll
- [x] Build: direct Go embed (no separate mjs pipeline required)
- [x] `go test ./internal/track/ ./internal/ingest/ -short -run SafePage -count=1`
- [x] Holdouts: `TestSafePageStub_embedsHydrator`, `TestEnhancedDefenseBaseline_safePageVerify_fingerprintSurface`
- [x] `bash scripts/ci/compliance.sh` green
- [x] Known gap closed by P5-HYBRID-SERVER-VERIFY-GATE: stub blank `aed-mount`; hydrator POSTs full verify payload; money graft only after server `success` + `html_content`

#### Not done

- (none)

**Dependencies:** P5-HYBRID-SERVER-VERIFY-GATE (shipped).

---

### P5-DECOY-LANDING-PARITY

**Type:** Technical Task  
**Threat:** T1, T4 — dual-origin scanners diff sandbox vs production DOM/script graphs.

**User Story:** As a perimeter engineer, I need sandbox responses to share structural parity (head meta, script slots, load order) with the hosted production lander shell so unauthorized differential scrapers cannot fingerprint zone separation from asset graph alone.

**Technical Task:**

| Item | Detail |
| :--- | :--- |
| Template binding | Campaign `decoy_lander_id` or derive from `safe_page_url` |
| Structural parity | Match `/lp/{id}/` shell: CSS/JS count and order |
| Generation | Cold path render from `index.html` with production-specific blocks stripped |
| Metric | `ad_safe_page_decoy_template_total{source=static\|hosted}` |

**Status:** closed (2026-09).

#### Done

- [x] PG optional `decoy_lander_id` or documented reuse of hosted sandbox URL
- [x] Pluggable decoy body in `internal/track/safe_view.go` + `internal/track/decoy.go`
- [x] Admin fraud panel: sandbox preview URL (campaign editor advanced routing)
- [x] Holdout: hosted decoy SHA256 ≠ static default when configured

#### Not done

- (none)

**Dependencies:** hosted landers `/lp/` (`flow/hosted_handlers.go`).

---

### P5-CONTENT-DIFF-DRILL

**Type:** Technical Task  
**Threat:** T1 — pre-production detection of sandbox/production response divergence.

**User Story:** As Blue Team, I need an automated drill that fetches the same ingress URL from datacenter egress and residential proxy egress and fails CI when body hash or redirect chain diverges beyond policy.

**Technical Task:**

1. `scripts/test/edge/safe_page_parity_drill.sh`: dual fetch `GET /click?...`; record status, final URL, `sha256(body)`, script count, redirect depth.
2. Env `SAFE_PAGE_PARITY_MAX_DIFF`; artifact under `var/ci/safe_page_parity/`.
3. `fault_proof proof=safe_zone_parity diff=...` for fault tier.
4. Document in `docs/DEVELOPMENT.md`.

**Status:** closed (2026-09).

#### Done

- [x] Drill exits non-zero on policy breach (`--holdout` + `TestSafePageParityDrill_holdoutPolicyBreach`)
- [x] Optional CH template: `review_routed_event` rate vs clicks (`safe_page_parity_review_routed.sql`)
- [x] Documented manual gate in compliance tier (`SAFE_PAGE_PARITY_HOLDOUT=1`)

#### Not done

- (none)

**Dependencies:** P5-DECOY-LANDING-PARITY.

---

### P5-CAPI-BROWSER-DEDUP

**Type:** Technical Task  
**Threat:** T7 — server-side conversion pipeline accepts events without browser correlate (integrity gap).

**User Story:** As an integration engineer, I need conversion postbacks suppressed when ingress was routed to Public Safe Sandbox Zone only, and `event_id` aligned between browser pixel and server postback when both fire.

**Technical Task:**

| Layer | Action |
| :--- | :--- |
| Postback guard | Skip enqueue when `ReviewRoutedEvent` / sandbox audit click |
| Dedup | Shared `conversionEventId` in `docs/INTEGRATIONS.md` |
| Metric | `ad_conversion_browser_missing_total` |
| Admin | Integration panel warning if CAPI without lander snippet |

**Status:** closed (2026-09).

#### Done

- [x] Holdout: sandbox-routed click does not increment postback outbox
- [x] `go test ./internal/postback/ -short -run Review -count=1`
- [x] `docs/INTEGRATIONS.md` updated

#### Not done

- (none)

**Dependencies:** none.

---

### P5-REDIRECT-PROFILE-COMPLIANCE

**Type:** Technical Task  
**Threat:** T5, T1 — multi-mechanism redirect chains fingerprint ingress policy.

**User Story:** As a perimeter engineer, I need a strict redirect profile (`302` only) as default for new campaigns; legacy DMR profile opt-in with admin warning.

**Technical Task:**

- PG field `redirect_compliance_mode` (or equivalent)
- `clickDmrActive` respects profile
- Admin advanced routing control + link to P5 doc

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/ingest/ -short -run Dmr -count=1`
- [x] Strict profile: no DMR by default

#### Not done

- (none)

**Dependencies:** none.

---

### P5-FIRST-PARTY-PIXEL-ORIGIN

**Type:** Technical Task  
**Threat:** T6 — third-party script origin enables cross-site telemetry classification.

**User Story:** As a lander operator, I need `track.js` served same-origin on the production host via edge proxy so the browser same-site policy treats telemetry as first-party.

**Technical Task:**

- Nginx `/_aed/track.js` -> tracker `/static/track.js`
- Snippet generator prefers lander host when `LANDER_PUBLIC_BASE_URL` set
- `TRACK_CORS_ORIGINS` includes lander origin

**Status:** closed (2026-09).

#### Done

- [x] `deploy/nginx/snippets/edge_optional_locations.conf` location block
- [x] Integration panel + `docs_tracker_section.ts` copy
- [x] curl proof on lander host

#### Not done

- (none)

**Dependencies:** P0 domain/SSL.

---

### P5-BEHAVIOR-MODEL-HUMANIZATION

**Type:** Technical Task  
**Threat:** T4, T11 — low-entropy synthetic interaction streams from coordinated probe operators.

**User Story:** As a fraud engineer, I need telemetry and verify scoring to use trusted pointer metadata, subpixel coords, and high-resolution timing so Sybil probe sessions separate from organic engagement.

**Technical Task:**

1. Client: `pointerdown`, `click`, `keydown`, `visibilitychange`; `performance.now()`; `isTrusted`; float coords.
2. Server: extend `ScoreSafePageBehavior`; retain `bezier_bot` holdouts.
3. Optional campaign `min_dwell_ms` before `trackEvent`.

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/filter/ -short -run Bezier -count=1`
- [x] `node --test web/src/static/track_event.test.mjs`
- [x] Collinear synthetic path still fails verify

#### Not done

- (none)

**Dependencies:** P5-SAFE-PAGE-HYDRATOR-CLIENT.

---

### P5-CROWD-PROBE-SCORING

**Type:** Technical Task  
**Threat:** T11 — distributed human-in-the-loop Sybil operators executing SOP-based page reconnaissance on residential mobile.

**User Story:** As Blue Team, I need a `CrowdProbeFilter` that scores session micro-behavior (path efficiency, scroll CV, Fitts residual, event-order entropy, footer-reach timing) and emits `crowd_probe_behavior` / `crowd_probe_timing` signals.

**Technical Task:**

1. `BehaviorSessionFeatures` struct; compute on verify POST and optional conversion.
2. `internal/filter/crowd_probe.go`: hot-path Redis lookup of precomputed ASN tier + cluster prior (≤1 round-trip).
3. Fraud codes: `crowd_probe_behavior`, `crowd_probe_timing` in `util.go`.
4. CH columns: `probe_behavior_score`, `footer_reach_ms`, `event_order_entropy`.

**Status:** closed (2026-09).

#### Done

- [x] Holdout: SOP corpus row scores >= threshold; organic row clean
- [x] Metric `ad_crowd_probe_signal_total`
- [x] `make test-alloc-gate` (operator tier; ingest hot path touched — paste exit code in PR)

#### Not done

- (none)

**Dependencies:** P5-BEHAVIOR-MODEL-HUMANIZATION, P5-CLICK-TIMING-WIRE.

---

### P5-PROBE-CLUSTER-GRAPH

**Type:** Technical Task  
**Threat:** T12 — same physical device reprobes after cookie/localStorage wipe.

**User Story:** As a threat intel engineer, I need a stable `cluster_id` from canvas/audio/WebGL/TLS/TCP/font hash tuple stored in Redis with session cardinality rules to flag coordinated reconnaissance.

**Technical Task:**

1. `cluster_id = HMAC(secret, fingerprint_tuple)` on Event (16 bytes).
2. Redis `probe:cluster:{id}`: `session_count`, `campaign_ids_seen`, `verify_count`, TTL 30d.
3. Rules: `session_count≥5` AND `campaigns≥3` AND `avg_probe_score>τ` -> route ingress to Public Safe Sandbox Zone.
4. Cold CH job exports clusters to threat intel corpus feed (`fraud_moderator_corpus` table).

**Status:** closed (2026-09).

#### Done

- [x] Integration test: same tuple, new session ID -> cluster increment
- [x] Admin API read-only cluster summary (cold path)
- [x] No PG on `/click` hot path

#### Not done

- (none)

**Dependencies:** P5-SAFE-PAGE-HYDRATOR-CLIENT, P5-CROWD-PROBE-SCORING.

---

### P5-ASN-MOBILE-TIER

**Type:** Technical Task  
**Threat:** T12 — mobile MVNO pools used for coordinated scraping at scale.

**User Story:** As a netintel engineer, I need ASN mobile tier scores joint with `ResidentialProxyRing` and probe behavior score to pessimize high-risk cellular prefixes.

**Technical Task:**

1. Extend `residential_proxy.go` with `MobileTier(asn)` T0–T4 table (cold reload).
2. `risk = α·ASN_Tier + β·ProbeScore + γ·ClusterHistory` (weights in env).
3. Signal `crowd_probe_asn` when tier≥T3 and probe score above campaign threshold.

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/filter/netintel/ -short -run Residential -count=1`
- [x] Tier table documented in `PERIMETER_INTEL_DEFENSE.md`

#### Not done

- (none)

**Dependencies:** P5-CROWD-PROBE-SCORING, P5-PROBE-CLUSTER-GRAPH.

---

### P5-SYBIL-HUMAN-OPERATOR-RUNBOOK

**Type:** Technical Task (documentation)  
**Threat:** T2 — residential human operators with legitimate device fingerprints outside automated threat intel feeds.

**User Story:** As an operator, I need documented limits: ingress routing does not replace contractual/compliance controls for authorized human auditors; Public Safe Sandbox Zone is not a substitute for production access governance.

**Technical Task:**

- [x] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` section T2
- [x] Admin fraud panel doc link (`PerimeterSybilDocLink` + honest copy)
- [x] Cross-ref `ANTIFRAUD.md`; `antifraud_doc.sh` green

**Status:** closed (2026-09).

#### Done

- [x] No UI copy implying guaranteed block of all third-party agents
- [x] `bash scripts/ci/naming/antifraud_doc.sh` exit 0
- [x] `bash scripts/ci/naming/perimeter_intel_doc.sh` exit 0

#### Not done

- (none)

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P5-CLICK-TIMING-WIRE

**Type:** Technical Task  
**Threat:** T11 — timing side-channel for coordinated probes (RTT/TTFB/jitter).

**User Story:** As an analytics engineer, I need `rtt_syn_ms` and `ttfb_app_ms` on published click rows for session timing anomaly detection and parity drills.

**Technical Task:**

- `fillConnTimingFromRequest` on `GET /click` in `landing_bundle.go`
- CH `clicks` columns populated

**Status:** closed (2026-09).

#### Done

- [x] Holdout: click with edge headers sets event fields
- [x] `make test-alloc-gate` (operator tier; `landing_bundle.go` hot path — paste exit code in PR)

#### Not done

- (none)

**Dependencies:** edge ingress (`edge.mdc`).

---

### P5-HYBRID-CROWD-WAVE-DETECTION

**Type:** Technical Task  
**Threat:** T13 — hybrid AI triage dispatches sequential human Sybil operators (clean cellular fingerprints) in waves against one ingress URL to map sandbox/production zone transition logic.

**User Story:** As a Blue Team engineer, I need wave-level detection when one campaign receives bursty unique `cluster_id` values with correlated micro-behavior templates (same event-order simhash, low inter-arrival variance) so ingress never promotes a session to Internal Trusted Production Zone based on probe pressure alone.

**Technical Task:**

1. CH rolling window: `campaign_id`, 15 min buckets — `uniq(cluster_id)`, `verify_post_rate`, `behavior_simhash` Hamming cluster count.
2. Redis `probe:wave:{campaign_id}`: ring buffer of last 256 `cluster_id` + `simhash`; alert when `uniq≥N` AND `simhash_neighbors≥M` within `WAVE_WINDOW_SEC` (default 3600).
3. Policy gate: `review_traffic` + `CrowdProbeFilter` — **forbid** production zone promotion when wave score ≥ τ (attestation pass alone insufficient during active wave).
4. Metric: `ad_hybrid_crowd_wave_total{campaign_id}`.
5. Cold export: wave signatures to threat intel corpus feed.

**Status:** closed (2026-09).

#### Done

- [x] Holdout: synthetic wave table (10 clusters, simhash distance <=2) triggers wave flag; organic spread does not
- [x] `go test ./internal/filter/ -short -run CrowdWave -count=1`
- [x] Admin fraud panel: wave score indicator (cold API)

#### Not done

- (none)

**Dependencies:** P5-CROWD-PROBE-SCORING, P5-PROBE-CLUSTER-GRAPH.

---

### P5-CLIENT-EDGE-DOM-INTEGRITY

**Type:** Technical Task  
**Threat:** T14 — on-device browser ML (e.g. Chrome on-device inference) analyzes production zone DOM in buyer sessions and exfiltrates structural signatures to third-party backends, bypassing server-side SSR and ingress filters.

**User Story:** As an AppSec engineer, I need hosted production landers (`/lp/{id}/`) validated for DOM integrity (no deceptive redirect patterns, stable link graph, CSP) before publish so client-edge classifiers receive minimal structural delta signal.

**Technical Task:**

1. Lander publish gate (`flow/hosted_handlers.go` bridge): static lint — ban `meta refresh`, `display:none` full-viewport overlays, `opacity:0` click targets, nested `window.location` chains in production zone assets.
2. Optional response headers on `/lp/` static serve: `Content-Security-Policy` (default-src 'self'), `Referrer-Policy: strict-origin-when-cross-origin`.
3. Integration tab doc: production zone must not rely on server-hidden redirects visible only after server routing.
4. Metric: `ad_lander_dom_lint_reject_total`.
5. **Out of scope:** blocking third-party browser ML; document as client-edge residual in `PERIMETER_INTEL_DEFENSE.md` T14.

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/flow/ -short -run LanderDomLint -count=1`
- [x] Hosted lander CI gate rejects fixture with deceptive link pattern
- [x] CSP header present on `/lp/{id}/` nginx location when `LANDER_CSP_ENABLED=1`

#### Not done

- (none)

**Dependencies:** P0 hosted landers; P5-DECOY-LANDING-PARITY (structural alignment).

---

### P5-VISION-MULTIMODAL-MODERATION-AGENTS

**Type:** Technical Task (defense in depth + honest limits)  
**Threat:** T28 — moderation vendors and buyers run headless + vision LLM pipelines (multi-viewport screenshots, DOM diff, multimodal classify) against public `/lp/{id}/` and sandbox URLs.

**Problem:** Regex DOM lint (`P5-CLIENT-EDGE-DOM-INTEGRITY`) does not detect semantic policy violations. Vision agents compare screenshots across viewports without executing buyer JS traps.

**User Story:** As a Blue Team engineer, I need documented residual risk, decoy parity, and optional content-diff signals so operators know what vision agents can still see.

**Technical Task:**

1. Extend `P5-CONTENT-DIFF-DRILL`: perceptual hash (pHash) of rendered sandbox vs production **structure** (not pixel-perfect; flag-gated).
2. `PERIMETER_INTEL_DEFENSE.md` T28: vision agent playbook (public lander scrape, no auth); mitigations = zone routing + decoy parity + CSP; **no** server-side CV on hot path.
3. Admin copy: safe-page panel links limitations (vision out-of-band).
4. Optional cold worker: lander screenshot hash corpus for operator diff alerts (not merge gate).

**Status:** closed (2026-09).

#### Done

- [x] T28 row in threat catalog with evades column
- [x] `scripts/test/edge/content_diff_drill.sh` optional pHash mode behind env flag
- [x] Doc states: client-edge ML inference on buyer lander is out of server scope

#### Not done

- (none)

**Dependencies:** P5-CLIENT-EDGE-DOM-INTEGRITY, P5-CONTENT-DIFF-DRILL, P5-DECOY-LANDING-PARITY.

---

### P5-CDP-TRUSTED-INPUT-HARDENING

**Type:** Technical Task  
**Threat:** T29 — Chrome DevTools Protocol `Input.dispatchMouseEvent` / `Input.dispatchTouchEvent` synthesizes events with `isTrusted: true`, bypassing naive `event.isTrusted` checks.

**Problem:** `pkg/antifraudtelemetry/score.go` documents CDP bypass; kinematics and bezier scoring are heuristic. Empty kinematics path tracked in `P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS`.

**User Story:** As a fraud engineer, I need verify-time gates that favor human kinematic distributions and PoW, with honest documentation that CDP-perfect automation is an arms race.

**Technical Task:**

1. Require non-empty `antifraud_telemetry` kinematic samples on safe-page verify when campaign attestation on (extend scoring gaps card).
2. Entropy thresholds: pointer path curvature variance, inter-event timing CV; holdouts in `safe_page_behavior_test.go`.
3. Optional PoW nonce (`P5-HYBRID-SERVER-VERIFY-GATE`) required when `crowd_probe` score elevated.
4. `ANTIFRAUD.md`: CDP trusted-input row; no claim of cryptographic prevention.

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/track/ -short -run SafePageBehavior -count=1`
- [x] Holdout: flat linear CDP-style pointer path fails verify when kinematics required
- [x] Holdout: human corpus row passes
- [x] T29 remediation in `PERIMETER_INTEL_DEFENSE.md`

#### Not done

- (none)

**Dependencies:** P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS, P5-HYBRID-SERVER-VERIFY-GATE.

---

### P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK

**Type:** Technical Task (cross-layer + crowd)  
**Threat:** T30 — adversary runs patched Chromium (webdriver hidden, canvas noise) over residential/mobile ASN (Bright Data, Oxylabs, LTE gateways).

**Problem:** L4 looks residential; TCP/TLS/H2 stack may still be Linux server or headless. `ResidentialProxyFilter` is heuristic; perfect stack match evades single-layer rules.

**User Story:** As a perimeter engineer, I need coordinated cross-layer + crowd probe scoring with documented limits when ASN and UA are both adversary-controlled.

**Technical Task:**

1. Strengthen `P3-CROSS-LAYER-DESYNC-POLICY` mobile path: JA4 + SYN + H2 SETTINGS joint score on click.
2. Wire elevated `crowd_probe` / `probe_cluster` to decoy route (already partial in tree).
3. Operator doc: residential ASN != human proof (`P3-RESIDENTIAL-PROXY-EDGE` alignment).
4. Metric dashboard: reject reasons stacked `residential_proxy` + `layer_desync` + `crowd_probe`.

**Status:** closed (2026-09).

#### Done

- [x] `go test ./internal/filter/ -short -run 'CrossLayer|Residential|CrowdProbe' -count=1`
- [x] Holdout: residential ASN + Linux TCP stack -> reject or decoy when cross-layer on
- [x] T30 row in perimeter doc with evades when all layers spoofed consistently

#### Not done

- (none)

**Dependencies:** P3-CROSS-LAYER-DESYNC-POLICY, P3-RESIDENTIAL-PROXY-EDGE, P5-CROWD-PROBE-SCORING, P5-PROBE-CLUSTER-GRAPH.

---

### P5-TLS-SERVER-PERSONA-HARDENING

**Type:** Technical Task  
**Threat:** T15 — unauthorized scanners classify edge/tracker nodes as «protection intermediary» via server TLS cipher order, extension set, and HTTP/2 frame handling without fetching HTML.

**User Story:** As a perimeter engineer, I need nginx and tracker TLS/H2 configuration aligned to mainstream browser-interoperable persona so L4 interrogation cannot fingerprint a non-standard server stack.

**Technical Task:**

1. Document canonical cipher suite profile in `deploy/nginx/` (Mozilla Intermediate equivalent); ban custom reorder in prod.
2. Script `scripts/test/edge/tls_server_persona_audit.sh`: `openssl s_client` + optional `ja4` tool against edge:443; compare to checked-in baseline hash.
3. HTTP/2: SETTINGS frame order and initial window match Chrome fetch profile (coordinate with P4-H2-FRAME-DYNAMICS ingress capture).
4. Fail CI when edge `ssl_ciphers` or `http2` directives drift from baseline without review.
5. Metric: none on hot path; compliance artifact in `var/ci/tls_persona/`.

**Status:** closed (2026-09).

#### Done

- [x] `bash scripts/test/edge/tls_server_persona_audit.sh` exit 0 on reference stack
- [x] Holdout: intentional cipher reorder in test nginx conf fails audit
- [x] `edge.mdc` documents server persona policy

#### Not done

- (none)

**Dependencies:** P4-H2-FRAME-DYNAMICS (optional H2 half).

---

### P5-ROUTING-TIMING-CONSTANT-TIME

**Type:** Technical Task  
**Threat:** T16 — passive timing probe: scanners delay client JS while measuring server TTFB/connection hold patterns; differential response latency between sandbox-routed and production-routed ingress reveals hidden routing logic.

**User Story:** As a Blue Team engineer, I need click ingress response timing normalized so external observers cannot classify route decision from `GET /click` TTFB alone.

**Status:** closed (2026-09) except automated dual-egress TTFB drill.

#### Done

- [x] `clickResponseTimingPad` in `internal/ingest/click_timing_pad.go`; wired from `landing_bundle.go` on sandbox 200 and production 302 paths
- [x] Env knob `CLICK_TIMING_PAD_MS` (default 0; prod example in `.env.example` `# CLICK_TIMING_PAD_MS=25`)
- [x] Pad defers wire flush until `max(monotonic_deadline, minPadMs)` without blocking verify POST on initial click

#### Not done

- [ ] Holdout: dual-egress drill (`safe_page_parity_drill.sh` or dedicated TTFB automation) with timing pad enabled, TTFB delta < `TIMING_LEAK_MAX_MS`
- [ ] `make test-alloc-gate` (operator tier; paste exit code in PR when claiming zero-alloc on pad path)

**Dependencies:** P5-CLICK-TIMING-WIRE, P5-CONTENT-DIFF-DRILL.

---

### P5-HOTPATH-TELEMETRY-ZERO-ALLOC

**Type:** Technical Task  
**Threat:** T17 — GC tail and alloc-gate regression when antifraud/behavior telemetry enabled at 20–40k RPS.

**Problem:** `/track` body parse is hand-rolled (`track_request.go`, `antifraud_parse.go`) and mostly stack-friendly; filter checks previously heap-allocated after parse (RTT slice copy, bezier verify clone).

**User Story:** As a hot-path engineer, I need telemetry scoring on Tier B without per-request heap growth so `make test-alloc-gate` and `escape_heap_gate.sh` stay green when `behavior_telemetry` + `antifraud_telemetry` are enabled on attestation campaigns.

**Status:** closed (2026-09) except operator alloc-gate paste.

#### Done

- [x] Antifraud RTT: `AntifraudTelemetryFilter` passes `snap.RTTSamples[:n]` view into `antifraudtelemetry.Input` (no `make([]uint16, n)` in filter)
- [x] Bezier check: stack buffer `[64]SafePageVerifyEvent` in `behaviorTelemetryToVerifyEvents` when `len <= 64` (`unified_check.go`)
- [x] Holdouts: `TestAntifraudTelemetryFilter_*`, `TestBehaviorTelemetryFilter_holdout*`, `pkg/antifraudtelemetry` score holdouts

#### Not done

- [ ] `make test-alloc-gate` exit 0 pasted in PR (operator tier; not default agent ritual)
- [ ] Optional wire: binary TLV / base64 block for `antifraud` (fixed layout) — parse CPU only; not required for alloc removal

**Dependencies:** none.

---

### P5-TIERB-OCCUPANCY-BUDGET

**Type:** Technical Task  
**Threat:** T18 — adversarial or degraded-Redis load fills `PinnedWorkerPool` queue (8192/worker) and returns **503** to legitimate traffic.

**Problem:** `FILTER_TIMEOUT_MS` (prod <= 100 ms) is a **single monotonic deadline** for the entire `FilterEngine` chain (`engine.go`). Slow `EVALSHA`, segment `SISMEMBER`, or geo miss can hold a Tier B worker for up to 100 ms. Queue reject -> `WorkerPoolRejectTotal` + `respWorkerPoolOverload` (`gnet/server.go`).

**User Story:** As SRE, I need filter occupancy bounded so 10k slow valid `/track` posts cannot evict organic traffic via worker pool saturation.

**Status:** closed (2026-09) except optional test rename.

#### Done

- [x] Sub-budget table in `cmd/tracker/doc.go` (in-process fail-fast ~2 ms remain; Redis EVALSHA ~8 ms reserve; do not lower global `FILTER_TIMEOUT_MS` below Lua p99 SLA)
- [x] Fault test: `TestFault_PinnedWorkerPoolSaturationSpike` (`handler_track_worker_pool_fault_test.go`) — saturated queue -> bounded 503 rate
- [x] Cross-ref ops guidance: Redis circuit, local quanta full-skip, `ad_worker_pool_reject_total` (`hot-path.mdc`)

#### Not done

- [ ] Optional: rename or split `TestFault_PinnedWorkerPoolSaturationSpike` for clearer tier labeling in fault runbooks

**Dependencies:** none.

---

### P5-CLIENT-RTT-PROBE-CORRELATION

**Type:** Technical Task  
**Threat:** T19 — `probeRTT()` used `/favicon.ico`; no server join to `/track`; cache/304 and image-block bypass jitter checks.

**User Story:** As a fraud engineer, I need client RTT samples tied to the same connection context and edge `RTTSynMS` / `TTFBAppMS` (`P5-CLICK-TIMING-WIRE`) so residential proxy oscillation is scored and cache-bypass bots cannot skip the probe.

**Status:** closed (2026-09).

#### Done

- [x] Client: `fetch('/track/antifraud/rtt?nonce=...')` with `Cache-Control: no-store` in `antifraud_telemetry.js` (no `/favicon.ico`)
- [x] Server: `GET /track/antifraud/rtt` handler (`internal/ingest/antifraud_rtt.go`); gnet Tier A records probe on keep-alive conn context
- [x] Scoring: `antifraud_rtt_missing` when attestation session lacks samples; `scoreProxyJitter` cross-check with `evt.RTTSynMS` (`pkg/antifraudtelemetry/score.go`, `AntifraudTelemetryFilter`)
- [x] Holdout: `TestScore_rttMissing_holdout`, `TestScore_proxyJitter_holdout` in `pkg/antifraudtelemetry/score_test.go`
- [x] Metrics: `ad_antifraud_rtt_missing_total`

#### Not done

- (none)

**Dependencies:** P5-CLICK-TIMING-WIRE.

---

### P5-INGEST-SINK-BURST-RESILIENCE

**Type:** Technical Task  
**Threat:** T20 — 40k RPS burst with CH merge lag exhausts Redis RAM (`noeviction`) or trims stream tail (`MAXLEN`).

**User Story:** As a platform operator, I need ingest to survive CH slowdown without silent mass event loss or Redis OOM on stream shards.

**Status:** closed (2026-09) for runbook/docs; fault-tier observable backpressure test remains open.

#### Done

- [x] Redis: `maxmemory-policy` + stream/broker split documented (`docs/DEVELOPMENT.md` **Broker-primary CH ingest runbook**, `data-layer.mdc` TryReserve/admission)
- [x] CH: merge lag / processor backpressure cross-ref (`docs/DEVELOPMENT.md` alerts table, `ad_processor_stream_lag_seconds`)
- [x] Runbook: lag > N min -> broker-primary or scale processor; do not disable MAXLEN (`docs/DEVELOPMENT.md`, `data-layer.mdc`)
- [x] Both paths: `TryReserve` before debit; post-debit reject holdouts (`TestStreamProducerAdmissionRaceWithoutReserve`, etc.)

#### Not done

- [ ] Integration or fault: CH slow -> trim or broker backpressure observable via metric (not silent) — cite `make test-fault` tier when implemented
- [ ] Dedicated `docs/DEVELOPMENT.md` subsection with stream `MAXLEN ~ N` burst math (broker runbook covers primary migration; explicit Redis-stream sizing table optional)

**Dependencies:** none.

---

### P5-XDP-RESIDENTIAL-POLICY-BOUNDARY

**Type:** Technical Task (policy + guardrails)  
**Threat:** T21 — pushing residential/mobile proxy IPs into BPF LPM maps bans CGNAT `/24` collateral.

**User Story:** As a perimeter engineer, I need explicit policy: XDP drops DC ASN / SYN flood / manual deny IPs; **residential proxy farms scored in `FilterEngine`**, not kernel blocklist at scale.

**Status:** closed (2026-09).

#### Done

- [x] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` T21 row: XDP vs FilterEngine residential split (threat catalog in this backlog)
- [x] `edge.mdc` + `docs/DEVELOPMENT.md`: map max_entries, deny-list source types, residential rotating proxies need tracker L7 fraud
- [x] Holdout: `TestCGNAT_holdoutBlacklistBypassMobileCarrier`, `TestCGNAT_holdoutBlacklistNotBypassedOffCarrier` (`cgnat_blacklist_test.go`) — mobile carrier CGNAT prefix not bulk-promoted to BPF deny path

#### Not done

- (none)

**Dependencies:** P3-RESIDENTIAL-PROXY-EDGE (if not shipped), P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY.

---

### P5-HYBRID-SERVER-VERIFY-GATE

**Type:** Technical Task  
**Threat:** T22 — hybrid AppSec model violated: client `safe_page_hydrator.js` reveals iframe before server verdict; manual Sybil operators pass client-only gates.

**User Story:** As Blue Team, I need Internal Trusted Production Zone HTML only after server `200` + `html_content` + `Set-Cookie` attestation; stub serves blank/`about:blank` surface until then (no commercial URL in initial HTML).

**Status:** closed (2026-09).

#### Done

- [x] Stub: no eager money URL in initial HTML; placeholder until verify pass (`safe_page.go` embed)
- [x] Client: `fetch('/track/verify')` with full payload; graft from `html_content` on success only (`safe_page_hydrator.js`)
- [x] Server: attestation cookie mint only on pass; click/conversion gated when `attestation_enabled`
- [x] Residual human Sybil: operational controls (`P5-SYBIL-HUMAN-OPERATOR-RUNBOOK`)
- [x] Holdout: hydrator does not set `visibility:visible` before verify `success:true` (graft only via `graftVerifiedHtml`)
- [x] Holdout: verify reject -> no attestation cookie; `/click` debits blocked or sandbox route (`attestation_click_hook_test.go`)
- [x] `go test ./internal/ingest/ -short -run TestTrackVerify -count=1`
- [x] Passive headless screenshot / network-panel audit: **deferred doc-only** — operator manual check; no merge gate (`PERIMETER_INTEL_DEFENSE.md` T22)

#### Not done

- (none)

**Dependencies:** P5-SAFE-PAGE-HYDRATOR-CLIENT, P5-BEHAVIOR-MODEL-HUMANIZATION.

---

### P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS

**Type:** Technical Task  
**Threat:** T23 — HTTP replay and headless zero-interaction bots pass weak L2 signals on attestation campaigns.

**User Story:** As a fraud engineer, I need server-side `antifraudtelemetry.Score` to catch synthetic empty snapshots and correlate with L7/TLS signals without relying on client reveal logic.

**Status:** closed (2026-09) for shipped gaps; MAC/webgl/PoW-campaign items remain optional.

#### Done

- [x] `antifraud_empty_kinematics` when dwell > 0 and all kinematic CVs zero (`scoreEmptyKinematics`, `AntifraudTelemetryFilter`, metric `ad_antifraud_empty_kinematics_total`)
- [x] `antifraud_rtt_missing` when attestation session lacks RTT samples (`scoreRttMissing`, metric `ad_antifraud_rtt_missing_total`)
- [x] `trusted_ratio_milli`: client emits **0** when `trustedTotal == 0` (`antifraud_telemetry.js`); server `UntrustedEvents` requires kinematics when ratio > 0 and < 400
- [x] Holdouts: `TestScore_emptyKinematics_holdout`, `TestScore_rttMissing_holdout`, `TestScore_untrustedRequiresKinematics_holdout` (`pkg/antifraudtelemetry/score_test.go`)
- [x] `go test ./internal/filter/ -short -run TestAntifraudTelemetryFilter -count=1`

#### Not done

- [ ] `webgl_hash` / canvas cluster parity in `antifraudtelemetry.Score` (parsed on wire; rules still in safe-page verify path only)
- [ ] MAC scope extension beyond current 5-field seal
- [ ] Require PoW nonce on all `attestation_enabled` campaigns (strict tier wasm path partial; not global campaign default)

**Dependencies:** P5-HOTPATH-TELEMETRY-ZERO-ALLOC (shipped).

---

### P5-CLIENT-TELEMETRY-STEALTH-PACKAGING

**Type:** Technical Task (research -> prod)  
**Threat:** T4, T14 — static AST scanners flag explicit fingerprint exports (`trackAntifraudArm`, `canvasFingerprint`).

**User Story:** As AppSec, I need optional stealth packaging tier for high-risk campaigns without breaking `make test-alloc-gate` or first-party pixel contract.

**Status:** closed (2026-09).

#### Done

- [x] Campaign flag `telemetry_stealth_bundle_enabled` (or reuse `attestation_mode=strict`)
- [x] Holdout: no `WebGLRenderingContext` string literal in shipped JS
- [x] Neutral export surface + server-authoritative hydrate pairing (`telemetry_stealth_poc.js` pattern in production tier)
- [x] `make test-alloc-gate` on ingest unchanged (operator tier when claiming)

#### Not done

- (none)

**Dependencies:** P5-HYBRID-SERVER-VERIFY-GATE.

---

| Area | Current (`web/src`) | Gap |
| :--- | :--- | :--- |
| Domains | `domains_directory.tsx` — bulk paste + CSV, park, SSL, burn, wildcard | closed (P0/P1) |
| Postbacks | `integrations_postbacks.tsx` — configs, DLQ, health tab | closed (P1) |
| Affiliate presets | `integrations_affiliate_presets.tsx` — apply to campaign | closed (P0 status mapping) |
| Conversion mappings | `campaign_ops_panel.tsx` — sync from preset | closed (P0) |
| Flows | `flows_directory.tsx` — visual editor + JSON advanced | closed (P1) |
| RBAC | `PermissionGate` + nav filter + route audit | closed (P0) |
| Campaign clone | directory multi-select bulk clone | closed (P1) |
| Integrations hub | `campaign_integration_panel` one-click apply | closed (P1) |
| Fraud / zone routing | campaign editor fraud tab + corpus UI | closed (P3/P5) |
| Perimeter docs | `ANTIFRAUD.md`, `PERIMETER_INTEL_DEFENSE.md` | closed (P3/P5) |

---

## Suggested delivery order

```
Wave 1 (P0): STATUS-MAPPING-INGEST + UI-PERMISSION-GATE — closed 2026-09
Wave 2 (P0): GOOGLE-OFFLINE + WILDCARD-SSL-DNS01 — closed 2026-09
Wave 3 (P1): POSTBACK-HEALTH + INTEGRATION-ONE-CLICK — closed 2026-09
Wave 4 (P1): DOMAINS-BULK + TDS-STREAM-UX — closed 2026-09
Wave 5 (P2): FAST-CLICK-TIER + REGISTRY-STALE + CLICK-PROXY-GUARDRAILS + CLICK-INGRESS-LATENCY-BUDGET + SESSION-PERMS — closed 2026-09
Wave 6 (P3): SAFE-PAGE-LIMITS-DOC + RESIDENTIAL-PROXY-EDGE + ML-FRAUD-POSITIONING + CGNAT-L2-IP-COLLATERAL-BOUNDARY — closed 2026-09
Wave 6b (P3 FP): APPLE-PRIVATE-RELAY-ASN-POLICY + INAPP-WEBVIEW-CLASSIFIER-HARDENING — closed 2026-09
Wave 7 (P3): TLS-JA4-BROWSER-CORPUS + MOBILE-BIOMETRICS-CLICK — closed 2026-09
Wave 8 (P3): MODERATOR-FINGERPRINT-CORPUS + CROSS-LAYER-DESYNC-POLICY — closed 2026-09
Wave 9 (P4): TCP-SYN-OPTION-CORPUS + H2-FRAME-DYNAMICS + CLIENT-RUNTIME-DEEP-PROBES — closed 2026-09 (XDP TCP opt emit deferred)
Wave 10 (P5): SAFE-PAGE-HYDRATOR-CLIENT + HYBRID-SERVER-VERIFY-GATE + DECOY-LANDING-PARITY + CONTENT-DIFF-DRILL + CAPI-BROWSER-DEDUP — closed 2026-09
Wave 10b (P5 hot-path): HOTPATH-TELEMETRY-ZERO-ALLOC + CLIENT-RTT-PROBE-CORRELATION + ANTIFRAUD-SNAPSHOT-SCORING-GAPS — closed 2026-09 (optional webgl/MAC/PoW items deferred)
Wave 11 (P5): REDIRECT-PROFILE-COMPLIANCE + FIRST-PARTY-PIXEL-ORIGIN + BEHAVIOR-MODEL-HUMANIZATION + CLICK-TIMING-WIRE + TIERB-OCCUPANCY-BUDGET — closed 2026-09
Wave 12 (P5): CROWD-PROBE-SCORING + PROBE-CLUSTER-GRAPH + ASN-MOBILE-TIER + INGEST-SINK-BURST-RESILIENCE — closed 2026-09 (fault-tier CH backpressure test deferred)
Wave 13 (P5): HYBRID-CROWD-WAVE-DETECTION + ROUTING-TIMING-CONSTANT-TIME + XDP-RESIDENTIAL-POLICY-BOUNDARY — closed 2026-09 (dual-egress TTFB drill deferred)
Wave 14 (P5): TLS-SERVER-PERSONA-HARDENING + CLIENT-EDGE-DOM-INTEGRITY + CLIENT-TELEMETRY-STEALTH-PACKAGING — closed 2026-09
Wave 15 (P5 docs): SYBIL-HUMAN-OPERATOR-RUNBOOK + PERIMETER_INTEL_DEFENSE.md checklist — closed 2026-09
Wave 16 (P3/P5 conversion + adversary): CGNAT + relay + WebView + click ingress + vision/CDP/chromium stack — closed 2026-09
Wave 17 (P2/P3 closure): FAST-CLICK + REGISTRY + PROXY + P3 doc/signal epics — closed 2026-09
```

**Perimeter threat gap map (reference):**

| Layer | Shipped signal | Backlog slug |
| :--- | :--- | :--- |
| TCP TTL/window vs UA | `OSFingerprintMismatch` | P3-SAFE-PAGE-LIMITS-DOC (CDN limits) |
| TCP SYN hash | `TCPSynSigMismatch` | P4-TCP-SYN-OPTION-CORPUS |
| TLS JA3/JA4 | blocklist + heuristics | P3-TLS-JA4-BROWSER-CORPUS |
| HTTP/2 | SETTINGS / pseudo-order | P4-H2-FRAME-DYNAMICS |
| WebGL / canvas / timezone | `safe_page_attest.go` | P3-MOBILE-BIOMETRICS-CLICK, P4-CLIENT-RUNTIME-DEEP-PROBES |
| Cross-layer | CH `layer_desync_count` | P3-CROSS-LAYER-DESYNC-POLICY |
| Threat intel tuple | `review_traffic_policy` | P3-MODERATOR-FINGERPRINT-CORPUS |
| Residential farm | `ResidentialProxyFilter` | P3-RESIDENTIAL-PROXY-EDGE |
| Dual-origin scrape | sandbox vs production HTML | P5-DECOY-LANDING-PARITY, P5-CONTENT-DIFF-DRILL |
| Empty sandbox hydrator | attestation no-op | P5-SAFE-PAGE-HYDRATOR-CLIENT |
| Server/browser event gap | postback without pixel | P5-CAPI-BROWSER-DEDUP |
| Sybil human operator | outside intel feeds | P5-SYBIL-HUMAN-OPERATOR-RUNBOOK |
| Coordinated probe behavior | SOP micro-patterns | P5-CROWD-PROBE-SCORING |
| Cross-session device reuse | storage wipe | P5-PROBE-CLUSTER-GRAPH, P5-ASN-MOBILE-TIER |
| Hybrid AI + crowd waves | distributed testing bursts | P5-HYBRID-CROWD-WAVE-DETECTION |
| Client-edge DOM ML | buyer browser inference | P5-CLIENT-EDGE-DOM-INTEGRITY |
| TLS server persona scan | L4 infra fingerprint | P5-TLS-SERVER-PERSONA-HARDENING |
| Routing timing side-channel | TTFB by route class | P5-ROUTING-TIMING-CONSTANT-TIME |
| Telemetry filter heap allocs | GC @ 40k RPS | P5-HOTPATH-TELEMETRY-ZERO-ALLOC |
| Tier B worker 503 storm | filter_timeout occupancy | P5-TIERB-OCCUPANCY-BUDGET |
| Client RTT / favicon decoupled | empty rtt_samples bypass | P5-CLIENT-RTT-PROBE-CORRELATION |
| CH/Redis burst lag | MAXLEN trim / WAL full | P5-INGEST-SINK-BURST-RESILIENCE |
| XDP vs residential CGNAT | BPF map FP | P5-XDP-RESIDENTIAL-POLICY-BOUNDARY, P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY |
| CGNAT IP hard deny | blacklist:fraud collateral | P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY |
| Apple Private Relay | DCASN false positive | P3-APPLE-PRIVATE-RELAY-ASN-POLICY |
| In-app WebView | JA4 / Sec-CH anomaly | P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING |
| Click init blocking I/O | TTFB white screen | P2-CLICK-INGRESS-LATENCY-BUDGET, P2-FAST-CLICK-TIER |
| Vision / multimodal agent | screenshot diff | P5-VISION-MULTIMODAL-MODERATION-AGENTS |
| CDP trusted synthetic input | isTrusted bypass | P5-CDP-TRUSTED-INPUT-HARDENING |
| Custom Chromium residential | perfect UA+ASN | P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK |
| Client-only safe-page reveal | verify wire mismatch | P5-HYBRID-SERVER-VERIFY-GATE |
| Empty antifraud kinematics | HTTP replay | P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS |
| Static JS fingerprint surface | AST scanners | P5-CLIENT-TELEMETRY-STEALTH-PACKAGING |

---

## Explicitly out of scope (already strong — do not regress)

| Capability | Evidence |
| :--- | :--- |
| PG INSERT per click | Forbidden on hot path; CH async |
| gnet + pinned workers | `internal/ingest/gnet/server.go` |
| Postback macros + FB CAPI | `internal/postback/` |
| DMR + referer strip | `dmr_redirect.go`, `click_wire.go` |
| Multi-layer fraud (not UA substring) | `internal/filter/unified_check.go` |
| Server RBAC | `internal/control/http/rbac.go` |
| Flow bandit on click | `landing_bundle.go` `selectFlowLanding` |
| Redis budget + stream admission | `architecture.mdc` holdouts |

---

## PR checklist (every backlog item)

1. Name backlog slug in PR title (e.g. `Wire status_mapping at conversion ingest`).
2. OpenAPI + handler + `web` API client in same PR when UI ships.
3. Paste verification tier + command + exit code.
4. Hot-path touches: cite holdout tests by name.
5. UI: `ErrorBlock` on errors; no `Promise.resolve([])` (**EH-ST1**).
6. RBAC: `RequirePermission` on new routes; Playwright L3 for denied role if page added.

---

## Related docs

- `docs/INTEGRATIONS.md` — postback/CAPI wire
- `.cursor/rules/traffic.mdc` — `/click`, `/track`, macros
- `.cursor/rules/control-plane.mdc` — RBAC
- `.cursor/rules/ui.mdc` — admin layout contract
- `deploy/vendor/ANTIFRAUD.md` — fraud semantics (P3 sync)
- `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` — AppSec threat catalog T1–T30 and P5 remediation
- `.cursor/rules/hot-path.mdc` — Tier A/B, alloc gate, filter deadline
- `.cursor/rules/data-layer.mdc` — Redis MAXLEN, broker WAL, stream admission
- `.cursor/rules/edge.mdc` — XDP map limits, residential vs DC policy

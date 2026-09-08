# Arbitrage / Keitaro-gap closure backlog

Product positioning: **ad-event-processor** is an event/budget/fraud processor with tracking, not a 1:1 Keitaro clone. This backlog closes **operational gaps** that make arbitrage teams reject the stack on a sales call, without regressing hot-path invariants (`architecture.mdc`, `hot-path.mdc`).

**Status baseline (2026-09):** hot path is production-grade (gnet, Redis Lua, async CH). Gaps are mostly **integrations**, **domain ops**, **admin UX**, and **optional TDS latency modes**.

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
| **P3** | Fraud/review positioning + incremental threat-intel signals | Honest limits; tighten gaps vs residential headless crawlers |
| **P4** | Research-tier passive/runtime probes | Not a sales promise; lab + corpus only |
| **P5** | Perimeter & commercial intelligence protection (AppSec) | Anti-scraping, Sybil/coordinated probe defense, zone parity |

---

## P0 — must ship before Keitaro comparison pitch

### P0-STATUS-MAPPING-INGEST

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

- [ ] `MapAffiliateStatus` called from production path (not only `schema_test.go`)
- [ ] Holdout: conversion with `status=sale` + preset maps payout; unknown status -> documented reject or default
- [ ] Apply schema populates or refreshes `campaign_conversion_mappings` without manual PUT
- [ ] UI: preset apply shows success count or field-level 400
- [ ] `go test ./internal/stream/ -run ConversionPayout -count=1`
- [ ] `go test ./internal/integrationschema/ -count=1`
- [ ] Integration test: processor batch with mapped payout + `AssertBudgetInvariant` unaffected

**SLA:** mapping lookup in processor batch: **< 1 ms p99 per event** (in-memory schema cache; no per-event PG).

**Dependencies:** none.

---

### P0-GOOGLE-OFFLINE-CONVERSIONS

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

- [ ] No hardcoded `customers/default` in production path
- [ ] `go test ./internal/postback/ -run Google -count=1` with httptest mock of full job lifecycle
- [ ] Staging gate: `bash scripts/ci/static/capi_staging.sh` extended or sibling `google_offline_staging.sh`
- [ ] DLQ row shows Google API error body (truncated, no secrets)
- [ ] OpenAPI postback config schema updated

**SLA:** dispatch latency same as other providers (async worker); **no** Google HTTP on tracker hot path.

**Dependencies:** none.

---

### P0-WILDCARD-SSL-DNS01

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

- [ ] DNS-01 issuance tested against Cloudflare sandbox or recorded httptest
- [ ] Renew path documented; metric `ad_event_processor_domain_ssl_renew_total`
- [ ] No SSH required for standard deploy (`DOMAIN_SSL_SETUP_ENABLED` path documented)
- [ ] `go test ./internal/platformadmin/domains/ -run Domain -count=1`
- [ ] UI: ErrorBlock on failure; no fake empty table

**SLA:** issuance p95 < 60 s (ACME + DNS propagation); hot path unchanged.

**Dependencies:** `CLOUDFLARE_*` configured (`domain_park.go`).

---

### P0-UI-PERMISSION-GATE

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

- [ ] Audit: every page's primary `GET` already returns 403 for denied role (existing httest)
- [ ] Optional: `GET /api/v1/session/route-permissions` map for UI (or embed in bootstrap)

**DoD:**

- [ ] `PermissionGate` on all routes in `NAV_GROUPS` with `permission` / `permissionAny`
- [ ] Playwright **L3**: seeded MB role -> `/ops` shows forbidden, `GET /api/v1/ops/home` = 403
- [ ] No `user?.role === 'MB'` string gates (**RB-C5**)
- [ ] `cd web && npm run typecheck` + `bash scripts/ci/admin/web.sh`

**SLA:** N/A (cold path).

**Dependencies:** none.

---

## P1 — operational parity with Keitaro/Binom workflows

### P1-DOMAINS-BULK-LIFECYCLE

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

- [ ] 100 domains import completes without blocking HTTP handler (outbox/worker)
- [ ] Integration test with fake Cloudflare client
- [ ] UI coalescing on refresh (**R1** 500 ms)
- [ ] OpenAPI + `domains_api.ts`

**SLA:** bulk job throughput >= 10 domains/min (limited by ACME rate limits, not PG).

---

### P1-TDS-STREAM-UX

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

- [ ] No raw JSON required for default path (JSON advanced panel ok)
- [ ] Weight sum != 100 -> 400 from server + inline ErrorBlock
- [ ] E2E **L2**: create flow -> attach to campaign -> curl `/click` lands on weighted URL
- [ ] `go test ./internal/flow/ -run Validate -count=1`

**SLA:** click path unchanged (flow selection after filters; budget in `core.mdc`).

---

### P1-INTEGRATION-ONE-CLICK

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
| `integrations_platform_campaigns.tsx` | Wizard steps: pick network -> credentials -> test postback |
| `campaign_editor` integrations | Copy buttons for click URL + postback URL |

**DoD:**

- [ ] Single apply -> all artifacts created; rollback on partial failure (PG tx)
- [ ] `tests/integration/postbacks_admin_test.go` extended
- [ ] UI: toast only after 2xx (**EC5**)

---

### P1-POSTBACK-HEALTH-DASHBOARD

**Problem:** `integrations_postbacks.tsx` has configs / DLQ / campaign status tabs but lacks **SRE-style health**: success %, latency, last error by provider.

**Backend:**

| Endpoint | Detail |
| :--- | :--- |
| `GET /api/v1/integrations/postbacks/health` | Aggregates from `postback_dispatches` + Prometheus snapshot: success_rate_24h, p95_latency_ms, last_error per campaign/provider |
| Reuse | `internal/reports/postback_recon.go` queries |

**Frontend:**

- New tab "Health" on postbacks page
- Per-campaign row: RAG badges; link to DLQ retry

**DoD:**

- [ ] Metrics match CH/PG reconciliation within 1% on integration test fixture
- [ ] Alert threshold documented: success < 95% -> runbook link
- [ ] No client-side aggregation over full dispatch log (**cold-path**)

**SLA:** health endpoint p95 < 200 ms (CH pre-agg or materialized view).

---

### P1-CAMPAIGN-CLONE-BULK

**Problem:** Arbitrage teams duplicate 20 campaigns/day. Clone exists partially via editor; no bulk clone with flow/domain/postback bundle.

**Backend:** `POST /api/v1/campaigns/bulk-clone` (source ids, customer scope, optional rename pattern)

**Frontend:** campaigns directory multi-select + clone dialog (reuse `campaign_wizard_panel` patterns)

**DoD:**

- [ ] RBAC: buyer sees only own campaigns
- [ ] Clone copies flow_id, postback configs, conversion mappings
- [ ] Holdout: budget fields reset; no spend copy

---

## P2 — hardening and latency tradeoffs

### P2-FAST-CLICK-TIER

**Problem:** Every `/click` runs full `FilterEngine` + Redis before 302 (`landing_bundle.go`). Correct for fraud-heavy traffic; heavy for pure TDS where buyer wants minimal TTFB.

**Design:**

| Mode | Filters | Redis | Use case |
| :--- | :--- | :--- | :--- |
| `full` (default) | Full chain | <= 1 EVALSHA | Production fraud/budget |
| `light` | license, geo, emergency, registry | 0-1 EVALSHA | Trusted internal streams |
| `redirect_only` | parse + macro only | 0 | **License-gated**; audit log only |

**Config:** campaign flag `click_filter_tier` or env default; **fail closed** if tier misconfigured on licensed fraud features.

**DoD:**

- [ ] `light` p99 click redirect < 15 ms on load test (no fraud filters)
- [ ] `full` unchanged: `make test-alloc-gate`
- [ ] Holdout: `redirect_only` cannot debit budget without explicit flag
- [ ] Document in `traffic.mdc` + campaign editor UI select

**SLA:** `light` tier: tracker p95 < 25 ms at 10k RPS (control cohort).

**Frontend:** `campaign_editor_advanced_panel.tsx` — tier select with warning copy.

---

### P2-REGISTRY-STALE-PG-GRACE

**Problem:** `REGISTRY_STALE_PG_GRACE=true` (default) allows sync PG read on registry cache miss (`registry_ops.go`). Under misconfigured pub/sub this is the **only** path to "PG on click".

**Tasks:**

- [ ] Production profile: document `REGISTRY_STALE_PG_GRACE=false` + alert on `registry_stale`
- [ ] Metric: `ad_event_processor_registry_stale_pg_reads_total`
- [ ] Circuit: max N PG reads/sec during stale mode -> 503 fail-closed
- [ ] Runbook in `docs/DEVELOPMENT.md`

**DoD:**

- [ ] Fault test: stale mode + grace off -> 503, zero PG queries on hot path
- [ ] Existing `TestLookupCampaign_stalePGGraceDisabled503_holdout` cited in PR

**SLA:** PG reads on hot path = **0** in recommended prod config.

---

### P2-CLICK-PROXY-GUARDRAILS

**Problem:** `click_proxy.go` blocks worker on upstream (10-30 s timeouts). Bad upstream = click-to-landing drop.

**Tasks:**

- [ ] Hard cap `CLICK_PROXY_TIMEOUT_MS` default 300 ms (config)
- [ ] On timeout: fallback 302 to landing URL + metric `ad_event_processor_click_proxy_fallback_total`
- [ ] Campaign flag to disable proxy on timeout vs hard fail

**DoD:**

- [ ] `go test ./internal/ingest/ -run ClickProxy -count=1`
- [ ] Load test: proxy timeout does not exhaust worker pool (503 rate bounded)

**SLA:** proxy attempt p95 < 300 ms or fallback.

---

### P2-SESSION-PERMS-SYNC

**Problem:** `GET /api/v1/session/bootstrap` permissions may drift from DB grants vs live policy `Snapshot`.

**Tasks:**

- [ ] Bootstrap returns same permission set as `RequirePermission` uses
- [ ] On grant/revoke: refetch bootstrap after role change
- [ ] Web: nav updates without full re-login

**DoD:**

- [ ] `go test ./internal/controlplane/ -run RBAC -count=1`
- [ ] UI nav updates after admin grants perm

---

## P3 — honest fraud / review positioning (optional product)

### P3-SAFE-PAGE-LIMITS-DOC

**Problem:** Buyers expect WebGL/headless rejection of all scrapers; stack offers Public Safe Sandbox attestation (`safe_page_attest.go`) and review traffic routing — not universal coverage. Residential gateway masks L4/L7 desync (server Linux stack vs claimed mobile UA).

**Tasks:**

- [ ] Operator doc: signal matrix (ingress TCP/TLS/H2, safe-page JS, cross-layer) with **evades** column (CDN, residential IP, attestation off)
- [ ] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` — threat catalog T1–T12, Blue Team remediation, operator checklist (P5 parent doc)
- [ ] Admin UI: link from fraud presets / safe-page panel to limitations doc
- [ ] Cross-ref backlog slugs: P3-MOBILE-BIOMETRICS-CLICK, P3-TLS-JA4-BROWSER-CORPUS, P3-MODERATOR-FINGERPRINT-CORPUS, P3-CROSS-LAYER-DESYNC-POLICY, P4-*

**DoD:**

- [ ] `bash scripts/ci/naming/antifraud_doc.sh` green
- [ ] No doc claims "eliminated" Redis ops (`anti-slop.mdc`)
- [ ] Doc states residential crawler on datacenter egress is **not** fully detectable at L4 alone

---

### P3-ML-FRAUD-POSITIONING

**Problem:** Hot path reads boost snapshot only; LGBM is `cmd/fraud-scorer` batch.

**Tasks:**

- [ ] SKU/license copy: ML enforcement = sidecar + CH batch
- [ ] Admin: show last model score timestamp per campaign (cold API)

**DoD:**

- [ ] No comment or UI implying inline ML on `/track`

---

### P3-RESIDENTIAL-PROXY-EDGE

**Problem:** XDP blocks known L4; rotating residential evades. Crawler egress IP looks residential while TCP/TLS stack stays Linux server.

**Tasks:**

- [ ] Document: edge XDP = flood + blocklist; `OS_FINGERPRINT_MISMATCH_ENABLED=false` on CDN paths
- [ ] Sales copy: `ResidentialProxyFilter` = farm/intel heuristic, not per-session unauthorized agent proof

**DoD:**

- [ ] `deploy/vendor/ANTIFRAUD.md` aligned with code
- [ ] Moderator corpus work tracked under P3-MODERATOR-FINGERPRINT-CORPUS (not this slug)

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-MOBILE-BIOMETRICS-CLICK

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

- [ ] Holdout: flat gyro on mobile UA -> attestation fail; human curve + gyro noise -> pass
- [ ] Holdout: click path skips when attestation off (no false positive on desktop)
- [ ] `go test ./internal/track/ -run SafePage -count=1`
- [ ] `go test ./internal/ingest/ -run BehaviorTelemetry -count=1`
- [ ] Hot path: zero extra Redis; filter work stays local + existing attestation round-trip
- [ ] OpenAPI fingerprint schema updated when admin exposes fields

**SLA:** attestation verify handler p95 < 50 ms (cold stub path); no sync PG/CH on verify.

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-TLS-JA4-BROWSER-CORPUS

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

- [ ] Holdout: Safari iOS UA + Chromium JA4 -> `tls_ja4_mismatch`
- [ ] Holdout: matching corpus row -> no signal
- [ ] `go test ./internal/filter/ -run JA4 -count=1`
- [ ] Feed refresh fail-open retains prior snapshot (`testing.mdc`)
- [ ] No claim of full ClientHello extension-order parity (document gap -> P4 if needed)

**SLA:** corpus lookup < 1 us p99 (in-memory snapshot); no per-event HTTP.

**Dependencies:** none.

---

### P3-MODERATOR-FINGERPRINT-CORPUS

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

- [ ] Integration: seed corpus row -> `GET /click` with matching JA3 serves safe view
- [ ] Holdout: empty corpus -> behavior unchanged
- [ ] `go test ./internal/ingest/ -run ReviewTraffic -count=1`
- [ ] RBAC: MB cannot import corpus; Playwright L3 on new page
- [ ] No hot-path CH query; snapshot reload <= 60 s

**SLA:** corpus snapshot read on click < 500 ns (atomic pointer); refresh async.

**Dependencies:** P3-RESIDENTIAL-PROXY-EDGE (doc), reports `layer-desync-*` stable.

---

### P3-CROSS-LAYER-DESYNC-POLICY

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

- [ ] Holdout: 3 configured mismatch signals -> `safe_page` when action set; 1 signal -> no route change
- [ ] Holdout: `off` preserves current behavior
- [ ] `go test ./internal/ingest/ -run LayerDesync -count=1`
- [ ] `make test-alloc-gate` if hot filter path touched
- [ ] OpenAPI campaign fraud fields documented

**SLA:** desync tally <= 200 ns per event (bitmask over existing signals); no extra Redis.

**Dependencies:** P3-TLS-JA4-BROWSER-CORPUS (optional, for richer TLS leg).

---

## P4 — research tier (anti-scanner reconnaissance; not sales SLA)

Items below close **technical gaps vs dedicated anti-bot stacks**. Do not pitch on sales call until P3 docs ship and load-tier proof exists.

### P4-TCP-SYN-OPTION-CORPUS

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

**Problem:** `L7WireFilter` checks static H2 SETTINGS and pseudo-header order. Does not detect automator multiplex timing (SETTINGS / WINDOW_UPDATE / HEADERS order and priority vs mobile WebKit).

**Tasks:**

- [ ] Edge: capture first N frame types + timestamps (ms bucket) -> `X-H2-FRAME-TRACE` (bounded header)
- [ ] Tracker: `H2FrameTraceMismatch(ua, trace)` with embedded WebKit/Chrome corpora
- [ ] Chaos: `TestChaos_CrossHop_NginxGnet` row for new header

**DoD:**

- [ ] Holdout: canned automator trace -> signal; Safari corpus trace -> clean
- [ ] Default env **off** (`H2_FRAME_TRACE_ENABLED=false`)
- [ ] No dynamic Prometheus labels on hot path

**Dependencies:** none.

---

### P4-CLIENT-RUNTIME-DEEP-PROBES

**Problem:** Public Safe Sandbox attestation lacks WebGPU pipeline timing, IEEE 754 canvas noise probes, `Performance.now()` / `Proxy` hook detection on `navigator` — techniques used by unauthorized inspection agents.

**Tasks:**

- [ ] Safe-page JS module (opt-in per campaign): shader compile timing bucket, float noise fingerprint, getter timing probe
- [ ] Server: evaluate in `EvaluateSafePageAttestation`; fail codes documented
- [ ] Privacy review: disclose in operator doc; EU SKU note

**DoD:**

- [ ] `FuzzSafePageVerifyParse` extended; no panic on malformed probe payloads
- [ ] `go test ./internal/track/ -run SafePage -count=1`
- [ ] Explicitly **not** on `/track` hot path — attestation POST only

**Dependencies:** P3-MOBILE-BIOMETRICS-CLICK (shared probe delivery).

---

## P5 — perimeter & commercial intelligence protection (AppSec)

**Theme:** Harden **Public Safe Sandbox Zone** vs **Internal Trusted Production Zone** separation against **unauthorized third-party scanners**, **distributed coordination attacks (Sybil)**, and **commercial intelligence scraping**. Close client attestation gaps; enable measurable perimeter drills.

**Reference:** `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` (threat catalog T1–T16).

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
| T17 | Hot-path telemetry filter heap allocs (GC tail @ 40k RPS) | P5-HOTPATH-TELEMETRY-ZERO-ALLOC | Medium |
| T18 | Tier B worker pool saturation (Redis/filter occupancy) | P5-TIERB-OCCUPANCY-BUDGET | High (availability) |
| T19 | Client RTT probe decoupled from tracker (`/favicon.ico`) | P5-CLIENT-RTT-PROBE-CORRELATION | High (fraud signal) |
| T20 | CH/Redis ingest burst (stream trim / WAL disk) | P5-INGEST-SINK-BURST-RESILIENCE | High (data loss) |
| T21 | Residential proxy IPs in XDP blocklist (CGNAT collateral) | P5-XDP-RESIDENTIAL-POLICY-BOUNDARY | Critical if misconfigured |
| T22 | Client-side safe-page reveal without server verify verdict | P5-HYBRID-SERVER-VERIFY-GATE | High |
| T23 | Antifraud snapshot scoring blind spots (empty kinematics) | P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS | Medium |

---

### P5-SAFE-PAGE-HYDRATOR-CLIENT

**Type:** Technical Task  
**Threat:** T3, T8 — unauthorized agents map sandbox attestation API without client integrity checks.

**User Story:** As a Blue Team engineer, I need the Public Safe Sandbox Zone to execute a signed client attestation probe so server-side `EvaluateSafePageAttestation` receives canvas/audio/WebRTC/timezone/behavior evidence before any Internal Trusted Production Zone iframe is shown.

**Technical Task:**

1. Implement `internal/track/safe_page_hydrator.js` (embed source of truth).
2. Emit `SafePageVerifyFingerprint` + `events[]` per `safe_page_attest.go`.
3. Arm listeners (≥18 events); `POST /track/verify` with `campaign_id` from stub query.
4. On pass: apply production iframe from `html_content`; on fail: remain in sandbox (existing server codes).
5. Share listener helpers with P5-BEHAVIOR-MODEL-HUMANIZATION.

**Acceptance Criteria:**

- [ ] Hydrator: canvas A/B (`canvas_retest_enabled`), audio, WebRTC, timezone, WebGL, languages, viewport
- [ ] `performance.now()` timestamps on pointer/touch/scroll
- [ ] Build pipeline: `build_safe_page_hydrator.mjs` or extend `build_track_pixel.mjs`
- [ ] `go test ./internal/track/ ./internal/ingest/ -short -run SafePage -count=1`
- [ ] Holdouts: `TestSafePageStub_embedsHydrator`, `TestEnhancedDefenseBaseline_safePageVerify_fingerprintSurface`
- [ ] `bash scripts/ci/compliance.sh` green

**Known gap (2026-09 audit):** hydrator currently calls `revealFrame` client-side before server OK and POSTs `{campaign_id, antifraud}` while `ParseSafePageVerifyRequest` requires `events[]` + `fingerprint`. Attestation cookie and `html_content` money graft are not applied. **Close via P5-HYBRID-SERVER-VERIFY-GATE** (do not mark this card done until server-authoritative unlock ships).

**Dependencies:** none (wire fix tracked in P5-HYBRID-SERVER-VERIFY-GATE).

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

**Acceptance Criteria:**

- [ ] PG optional `decoy_lander_id` or documented reuse of hosted sandbox URL
- [ ] Pluggable decoy body in `internal/track/safe_view.go`
- [ ] Admin fraud panel: sandbox preview URL
- [ ] Holdout: hosted decoy SHA256 ≠ static default when configured

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

**Acceptance Criteria:**

- [ ] Drill exits non-zero on policy breach
- [ ] Optional CH template: `review_routed_event` rate vs clicks
- [ ] Documented manual gate in compliance tier

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

**Acceptance Criteria:**

- [ ] Holdout: sandbox-routed click does not increment postback outbox
- [ ] `go test ./internal/postback/ -short -run Review -count=1`
- [ ] `docs/INTEGRATIONS.md` updated

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

**Acceptance Criteria:**

- [ ] `go test ./internal/ingest/ -short -run Dmr -count=1`
- [ ] Strict profile: no DMR by default

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

**Acceptance Criteria:**

- [ ] `deploy/nginx/snippets/edge_optional_locations.conf` location block
- [ ] Integration panel + `docs_tracker_section.ts` copy
- [ ] curl proof on lander host

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

**Acceptance Criteria:**

- [ ] `go test ./internal/filter/ -short -run Bezier -count=1`
- [ ] `node --test web/src/static/track_event.test.mjs`
- [ ] Collinear synthetic path still fails verify

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

**Acceptance Criteria:**

- [ ] Holdout: SOP corpus row scores ≥ threshold; organic row clean
- [ ] `make test-alloc-gate` if ingest hot path touched
- [ ] Metric `ad_crowd_probe_signal_total`

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

**Acceptance Criteria:**

- [ ] Integration test: same tuple, new session ID -> cluster increment
- [ ] Admin API read-only cluster summary (cold path)
- [ ] No PG on `/click` hot path

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

**Acceptance Criteria:**

- [ ] `go test ./internal/filter/netintel/ -short -run Residential -count=1`
- [ ] Tier table documented in `PERIMETER_INTEL_DEFENSE.md`

**Dependencies:** P5-CROWD-PROBE-SCORING, P5-PROBE-CLUSTER-GRAPH.

---

### P5-SYBIL-HUMAN-OPERATOR-RUNBOOK

**Type:** Technical Task (documentation)  
**Threat:** T2 — residential human operators with legitimate device fingerprints outside automated threat intel feeds.

**User Story:** As an operator, I need documented limits: ingress routing does not replace contractual/compliance controls for authorized human auditors; Public Safe Sandbox Zone is not a substitute for production access governance.

**Technical Task:**

- [ ] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` section T2
- [ ] Admin fraud panel doc link (`StubBanner`)
- [ ] Cross-ref `ANTIFRAUD.md`; `antifraud_doc.sh` green

**Acceptance Criteria:**

- [ ] No UI copy implying guaranteed block of all third-party agents
- [ ] `bash scripts/ci/naming/antifraud_doc.sh` exit 0

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P5-CLICK-TIMING-WIRE

**Type:** Technical Task  
**Threat:** T11 — timing side-channel for coordinated probes (RTT/TTFB/jitter).

**User Story:** As an analytics engineer, I need `rtt_syn_ms` and `ttfb_app_ms` on published click rows for session timing anomaly detection and parity drills.

**Technical Task:**

- `fillConnTimingFromRequest` on `GET /click` in `landing_bundle.go`
- CH `clicks` columns populated

**Acceptance Criteria:**

- [ ] Holdout: click with edge headers sets event fields
- [ ] `make test-alloc-gate` if hot path touched

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

**Acceptance Criteria:**

- [ ] Holdout: synthetic wave table (10 clusters, simhash distance ≤2) triggers wave flag; organic spread does not
- [ ] `go test ./internal/filter/ -short -run CrowdWave -count=1`
- [ ] Admin fraud panel: wave score indicator (cold API)

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

**Acceptance Criteria:**

- [ ] `go test ./internal/flow/ -short -run LanderDomLint -count=1`
- [ ] Hosted lander CI gate rejects fixture with deceptive link pattern
- [ ] CSP header present on `/lp/{id}/` nginx location when `LANDER_CSP_ENABLED=1`

**Dependencies:** P0 hosted landers; P5-DECOY-LANDING-PARITY (structural alignment).

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

**Acceptance Criteria:**

- [ ] `bash scripts/test/edge/tls_server_persona_audit.sh` exit 0 on reference stack
- [ ] Holdout: intentional cipher reorder in test nginx conf fails audit
- [ ] `edge.mdc` documents server persona policy

**Dependencies:** P4-H2-FRAME-DYNAMICS (optional H2 half).

---

### P5-ROUTING-TIMING-CONSTANT-TIME

**Type:** Technical Task  
**Threat:** T16 — passive timing probe: scanners delay client JS while measuring server TTFB/connection hold patterns; differential response latency between sandbox-routed and production-routed ingress reveals hidden routing logic.

**User Story:** As a Blue Team engineer, I need click ingress response timing normalized so external observers cannot classify route decision from `GET /click` TTFB alone.

**Technical Task:**

1. Profile current paths: `review_traffic_policy` early 200 vs full `FilterEngine` 302 — measure p50/p99 delta (load test artifact).
2. `clickResponseTimingPad`: defer wire flush until `max(monotonic_deadline, min_response_bucket)` where `min_response_bucket` = campaign-configured floor (default 25 ms prod / same for sandbox).
3. Sandbox 200 and production 302 responses: pad `Content-Length` body to same size bucket where feasible (static padding comment block in safe view HTML).
4. CH internal only: `route_class` + `ttfb_app_ms` — monitor **internal** stddev; external drill (`safe_page_parity_drill.sh`) must not show route-correlated TTFB > `TIMING_LEAK_MAX_MS` (env, default 15 ms).
5. Do not block TCP connection waiting for verify POST on initial click (verify remains async).

**Acceptance Criteria:**

- [ ] Holdout: with timing pad enabled, dual-egress drill TTFB delta < `TIMING_LEAK_MAX_MS`
- [ ] `make test-alloc-gate` if `landing_bundle.go` hot path touched
- [ ] Document env knobs in `.env.example`

**Dependencies:** P5-CLICK-TIMING-WIRE, P5-CONTENT-DIFF-DRILL.

---

### P5-HOTPATH-TELEMETRY-ZERO-ALLOC

**Type:** Technical Task  
**Threat:** T17 — GC tail and alloc-gate regression when antifraud/behavior telemetry enabled at 20–40k RPS.

**Problem:** `/track` body parse is hand-rolled (`track_request.go`, `antifraud_parse.go`) and mostly stack-friendly, but **filter checks still heap-allocate** after parse:

| Location | Alloc | Trigger |
| :--- | :--- | :--- |
| `AntifraudTelemetryFilter.Check` | `make([]uint16, n)` | `RTTSampleCount > 0` |
| `behaviorTelemetryToVerifyEvents` | `make([]SafePageVerifyEvent, len)` | conversion + bezier check |
| `parseTrackTelemetryEventsArray` | `make([]BehaviorTelemetryEvent, 0, 8)` | scratch `cap < 8` on first parse |

**User Story:** As a hot-path engineer, I need telemetry scoring on Tier B without per-request heap growth so `make test-alloc-gate` and `escape_heap_gate.sh` stay green when `behavior_telemetry` + `antifraud_telemetry` are enabled on attestation campaigns.

**Fix guide (`hot-path.mdc`, `data-layer.mdc`):**

1. **Antifraud RTT:** pass `snap.RTTSamples[:n]` into `antifraudtelemetry.Input` via fixed array + count (no slice copy in filter). Extend `Score()` / `scoreProxyJitter` to accept `[]uint16` view or `*[8]uint16` + `uint8` count.
2. **Bezier check:** run `CheckBezierBot` in-place on `evt.TelemetryEvents` (add adapter that reads `domain.BehaviorTelemetryEvent` without `SafePageVerifyEvent` clone), or stack-buffer `[64]SafePageVerifyEvent` when `len <= trackTelemetryMaxEvents`.
3. **Parse scratch:** ensure `TrackRequest` / `ConnContext` reuse `TelemetryEvents` slice with `cap >= 8` from pool (`Event.Reset` already trims cap > 64).
4. **Optional wire:** binary TLV / base64 block for `antifraud` (fixed layout) to cut parse CPU; not a substitute for filter alloc removal.
5. **Verify:** `make test-alloc-gate`; `bash scripts/ci/static/escape_heap.sh` on touched ingest/filter files; holdout `TestAntifraudTelemetryFilter_*` + `TestBehaviorTelemetryFilter_holdout*`.

**Acceptance Criteria:**

- [ ] Zero new `make`/`append` on filter path for antifraud RTT and bezier when telemetry present
- [ ] `make test-alloc-gate` exit 0 (paste in PR)
- [ ] Holdout: revert alloc fix -> alloc gate or holdout fails

**Dependencies:** none.

---

### P5-TIERB-OCCUPANCY-BUDGET

**Type:** Technical Task  
**Threat:** T18 — adversarial or degraded-Redis load fills `PinnedWorkerPool` queue (8192/worker) and returns **503** to legitimate traffic.

**Problem:** `FILTER_TIMEOUT_MS` (prod <= 100 ms) is a **single monotonic deadline** for the entire `FilterEngine` chain (`engine.go`). Slow `EVALSHA`, segment `SISMEMBER`, or geo miss can hold a Tier B worker for up to 100 ms. Queue reject -> `WorkerPoolRejectTotal` + `respWorkerPoolOverload` (`gnet/server.go`).

**User Story:** As SRE, I need filter occupancy bounded so 10k slow valid `/track` posts cannot evict organic traffic via worker pool saturation.

**Fix guide (`hot-path.mdc`, `architecture.mdc`):**

| Action | Detail |
| :--- | :--- |
| **Do not** | Set global `FILTER_TIMEOUT_MS` to 5–8 ms (violates Lua p99 < 10 ms SLA in `core.mdc`; mass false `filter_timeout`) |
| **Do** | Per-filter sub-deadline: Redis `EVALSHA` budget ~8 ms; in-process filters (telemetry, L7 wire) ~0 ms budget with fast-fail |
| **Do** | Redis circuit / `shard_unavailable` fail-closed before occupying worker (`filter_errors_test.go` infra paths) |
| **Do** | Local quanta full-skip + shadow debit when shard latency spikes (`tradeoffs.mdc`) |
| **Do** | Monitor: `ad_worker_pool_reject_total`, `filter_decisions{filter_timeout}`, `ad_http_request_duration_seconds` p99 |
| **Ops** | Nginx `proxy_request_buffering` on so slow client body does not pin Tier B (attack targets edge buffer, not worker) |

**Acceptance Criteria:**

- [ ] Documented sub-budget table in `cmd/tracker/doc.go` (Redis vs in-process ms)
- [ ] Holdout or fault test: saturated Redis shard -> 503 bounded rate, not unbounded worker hang
- [ ] `TestFault_PinnedWorkerPoolSaturationSpike` or equivalent cited in PR

**Dependencies:** none.

---

### P5-CLIENT-RTT-PROBE-CORRELATION

**Type:** Technical Task  
**Threat:** T19 — `probeRTT()` uses `/favicon.ico` (`antifraud_telemetry.js`); no server join to `/track`; cache/304 and image-block bypass jitter checks.

**Problem:**

- Client: `img.src = '/favicon.ico?rtt=' + start` — may hit browser/Nginx cache (0–1 ms false samples).
- Server: `scoreProxyJitter` requires `len(rtt_samples) >= 3`; empty array is a **no-op** (neither fraud nor cross-check).
- No correlation: tracker does not record favicon probe hits vs subsequent POST `/track`.
- Puppeteer `abort image` -> `rtt_samples: []` -> jitter path disabled.

**User Story:** As a fraud engineer, I need client RTT samples tied to the same connection context and edge `RTTSynMS` / `TTFBAppMS` (`P5-CLICK-TIMING-WIRE`) so residential proxy oscillation is scored and cache-bypass bots cannot skip the probe.

**Fix guide (`hot-path.mdc`, `traffic.mdc`, `edge.mdc`):**

1. **Client:** replace favicon probe with `GET /track/antifraud/rtt` or signed query on existing challenge route (`/track/antifraud/challenge`); `Cache-Control: no-store`; unique nonce per sample.
2. **Server:** gnet handler records probe timestamp + IP + campaign on Tier A; attach to `ConnContext` for `/track` POST on same keep-alive connection when possible.
3. **Scoring:** empty `rtt_samples` when `attestation_enabled` -> `antifraud_rtt_missing` (L2 weak); retain `scoreProxyJitter` cross-check with `evt.RTTSynMS`.
4. **Edge:** optional nginx `location = /track/antifraud/rtt` proxy to tracker (not static favicon).

**Acceptance Criteria:**

- [ ] Holdout: blocked image load -> `antifraud_rtt_missing` on attestation campaign
- [ ] Holdout: synthetic samples [42, 380, 51, 610] + edge RTT mismatch -> jitter signal
- [ ] `go test ./pkg/antifraudtelemetry/ -short -run TestScore -count=1`
- [ ] No `/favicon.ico` in `antifraud_telemetry.js` after ship

**Dependencies:** P5-CLICK-TIMING-WIRE.

---

### P5-INGEST-SINK-BURST-RESILIENCE

**Type:** Technical Task  
**Threat:** T20 — 40k RPS burst with CH merge lag exhausts Redis RAM (`noeviction`) or trims stream tail (`MAXLEN`).

**Problem:** `StreamProducer` uses `XADD` with `MaxLen` + `Approx: true` (default `REDIS_STREAM_MAXLEN` / `STREAM_MAX_LEN` = 10000 per `env.go`). At ~2.4M events/min, processor lag -> **approx trim drops events** (data loss, not necessarily OOM). `CH_INGEST_SOURCE=broker` shifts risk to **mmap WAL disk** (`data-layer.mdc`).

**User Story:** As a platform operator, I need ingest to survive CH slowdown without silent mass event loss or Redis OOM on stream shards.

**Fix guide (`data-layer.mdc`, `architecture.mdc`):**

| `CH_INGEST_SOURCE` | Policy |
| :--- | :--- |
| `redis` | Confirm per-shard `MAXLEN ~ N` sized for max lag SLO; alert on `XINFO` lag and `ad_events_dropped_total` |
| `broker` | Monitor WAL bytes; disk cap + backpressure before hot path accepts unbounded debit |
| Both | `TryReserve` before debit; post-debit reject metric ~0 (`TestStreamProducerAdmissionRaceWithoutReserve`) |

**Ops checklist:**

- [ ] Redis: `maxmemory-policy` documented; streams not the only RAM consumer (dedup, local quanta keys)
- [ ] CH: `parts_to_throw_insert` / merge lag dashboards; processor batch size vs insert latency
- [ ] Runbook: lag > N min -> enable broker-primary or scale processor; do not disable MAXLEN

**Acceptance Criteria:**

- [ ] `docs/DEVELOPMENT.md` or `data-layer.mdc` cross-ref: stream MAXLEN vs burst math
- [ ] Integration or fault: CH slow -> trim or broker backpressure observable via metric (not silent)
- [ ] `make test-fault` tier cited when broker/CH path touched

**Dependencies:** none.

---

### P5-XDP-RESIDENTIAL-POLICY-BOUNDARY

**Type:** Technical Task (policy + guardrails)  
**Threat:** T21 — pushing residential/mobile proxy IPs into BPF LPM maps bans CGNAT `/24` collateral.

**Problem:** XDP blocklist (`edge-bpf-sync` -> pinned maps, max ~786k entries per `internal/edge/doc.go`) suits **known-bad L3/L4** and DC flood. Residential pools (BrightData, Oxylabs, mobile LTE) rotate millions of IPs; bulk sync causes map pressure and false positives on carrier NAT.

**User Story:** As a perimeter engineer, I need explicit policy: XDP drops DC ASN / SYN flood / manual deny IPs; **residential proxy farms scored in `FilterEngine`** (`ResidentialProxyFilter`, `behavior_telemetry`, JA4 corpus), not kernel blocklist at scale.

**Fix guide (`edge.mdc`, `compliance.mdc`, `data-layer.mdc`):**

| Layer | Scope |
| :--- | :--- |
| XDP | DC flood, token bucket, `blacklist:manual` / fraud **single-IP** denies, syn subnet limit |
| Go hot path | `FraudFilter.checkDCASN`, `ResidentialProxyFilter`, `DeviceFilter` JA4, `AntifraudTelemetryFilter` |
| Forbidden | Auto-promote residential intel feed into `blacklist:fraud` -> BPF without human review + CGNAT allowlist |

**Acceptance Criteria:**

- [ ] `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` section: XDP vs FilterEngine residential split
- [ ] `edge.mdc` documents map max_entries and deny-list source types
- [ ] Holdout or doc test: mobile carrier CGNAT prefix not in default BPF sync path

**Dependencies:** P3-RESIDENTIAL-PROXY-EDGE (if not shipped).

---

### P5-HYBRID-SERVER-VERIFY-GATE

**Type:** Technical Task  
**Threat:** T22 — hybrid AppSec model violated: client `safe_page_hydrator.js` reveals iframe before server verdict; manual Sybil operators pass client-only gates.

**Problem:** Production zone promotion must be **server-authoritative** (`EvaluateSafePageAttestation` + attestation cookie). Current hydrator: client-side `dwell_ms` + motion + crypto -> `revealFrame`; verify POST wire mismatch; money `html_content` from `/track/verify` ignored.

**User Story:** As Blue Team, I need Internal Trusted Production Zone HTML only after server `200` + `html_content` + `Set-Cookie` attestation; stub serves blank/`about:blank` surface until then (no commercial URL in initial HTML).

**Fix guide (`hot-path.mdc`, `traffic.mdc`, `control-plane.mdc`):**

1. **Stub:** remove eager `iframe src=safe_page_url`; placeholder only (`safe_page.go` embed).
2. **Client:** `await fetch('/track/verify')` with full `SafePageVerifyRequest` (`events`, `fingerprint`, `antifraud` block); on success `document` graft from `html_content`; on fail stay sandbox.
3. **Server:** `reactTrackVerify` unchanged contract; mint attestation cookie only on pass.
4. **Click path:** `attestationRequired` cookie check on `/click` / conversion when `attestation_enabled`.
5. **Residual:** human Sybil passes verify -> operational controls (`P5-SYBIL-HUMAN-OPERATOR-RUNBOOK`); not client-only dwell thresholds.

**Acceptance Criteria:**

- [ ] Holdout: hydrator does not set `visibility:visible` before verify `success:true`
- [ ] Holdout: verify reject -> no attestation cookie; `/click` debits blocked or sandbox route
- [ ] `go test ./internal/ingest/ -short -run TestTrackVerify -count=1`
- [ ] Passive headless screenshot sees decoy/loader only (no money URL in network panel)

**Dependencies:** P5-SAFE-PAGE-HYDRATOR-CLIENT, P5-BEHAVIOR-MODEL-HUMANIZATION.

---

### P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS

**Type:** Technical Task  
**Threat:** T23 — HTTP replay and headless zero-interaction bots pass weak L2 signals on attestation campaigns.

**Gaps (2026-09 audit):**

| Gap | Current behavior | Fix |
| :--- | :--- | :--- |
| `pointer_cv_milli == 0` | `scoreTemplateBehavior` requires `> 0` | `antifraud_empty_kinematics` when dwell > 0 and all CVs zero |
| `trusted_ratio_milli` | defaults to 1000 when no events | default **0** when `trustedTotal == 0` |
| `webgl_hash` / canvas cluster | parsed, not in `Score()` | reuse `checkWebGLAutomation` rules in filter or score |
| MAC scope | 5 fields only | extend MAC or re-seal at `trackEvent` time |
| PoW optional | skipped without `campaignId` | require when `attestation_enabled` |

**User Story:** As a fraud engineer, I need server-side `antifraudtelemetry.Score` to catch synthetic empty snapshots and correlate with L7/TLS signals (`P3-CROSS-LAYER-DESYNC-POLICY`) without relying on client reveal logic.

**Fix guide (`hot-path.mdc`, `deploy/vendor/ANTIFRAUD.md`):**

1. Implement signals in `pkg/antifraudtelemetry/score.go` + `AntifraudTelemetryFilter`.
2. Wire `webgl_automation` parity from `safe_page_attest.go` into antifraud path.
3. Cross-layer: JA4 + `webgl_hash` desync -> existing `FraudReasonTLSJA4Mismatch` / desync flags when enabled.
4. Verify: `go test ./internal/filter/ -short -run TestAntifraudTelemetryFilter -count=1`; corpus row for empty kinematics holdout.

**Acceptance Criteria:**

- [ ] Holdout: empty kinematics + dwell > 500 ms -> fraud signal
- [ ] Holdout: `trusted_ratio` zero-events -> not 1000
- [ ] `bash scripts/ci/naming/antifraud_doc.sh` green after `ANTIFRAUD.md` slug update

**Dependencies:** P5-HOTPATH-TELEMETRY-ZERO-ALLOC (optional parallel).

---

### P5-CLIENT-TELEMETRY-STEALTH-PACKAGING

**Type:** Technical Task (research -> prod)  
**Threat:** T4, T14 — static AST scanners flag explicit fingerprint exports (`trackAntifraudArm`, `canvasFingerprint`).

**Problem:** PoC exists at `internal/track/telemetry_stealth_poc.js` (stringless resolve, generator FSM, stealth hydrate). Production `antifraud_telemetry.js` still exports obvious symbols.

**User Story:** As AppSec, I need optional stealth packaging tier for high-risk campaigns without breaking `make test-alloc-gate` or first-party pixel contract.

**Fix guide (`hot-path.mdc`, `frontend-modular.mdc` N/A — tracker embed only):**

1. Neutral export surface (`aedSensBootstrap` pattern); lazy-loaded probe chunk post-challenge.
2. Server-authoritative hydrate (`P5-HYBRID-SERVER-VERIFY-GATE`); client remains sensor only.
3. Build gate: `rg` holdout for banned literals in shipped bundle (`telemetry_stealth_poc_test.go` pattern).
4. Do not claim stealth = security; pair with server scoring (`P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS`).

**Acceptance Criteria:**

- [ ] Campaign flag `telemetry_stealth_bundle_enabled` (or reuse `attestation_mode=strict`)
- [ ] Holdout: no `WebGLRenderingContext` string literal in shipped JS
- [ ] `make test-alloc-gate` on ingest unchanged

**Dependencies:** P5-HYBRID-SERVER-VERIFY-GATE.

---


| Area | Current (`web/src`) | Gap |
| :--- | :--- | :--- |
| Domains | `domains_directory.tsx` — single host, park, SSL script | P0 wildcard, P1 bulk |
| Postbacks | `integrations_postbacks.tsx` — configs, DLQ, status | P1 health tab |
| Affiliate presets | `integrations_affiliate_presets.tsx` — **read-only list** | P0 apply + mapping sync |
| Conversion mappings | `campaign_ops_panel.tsx` — manual rows | P0 auto from preset |
| Flows | `flows_directory.tsx` — JSON create | P1 visual editor |
| RBAC | `nav_config.ts` `filterNavItems` only | P0 `PermissionGate` |
| Campaign clone | editor ops | P1 bulk clone |
| Integrations hub | `integrations_nav.tsx` | P1 one-click wizard |
| Fraud / zone routing | campaign editor fraud tab | P3 biometrics, P3 cross-layer, threat intel corpus UI |
| Perimeter docs | `ANTIFRAUD.md` partial | P3-SAFE-PAGE-LIMITS-DOC, P5-SYBIL-HUMAN-OPERATOR-RUNBOOK, `PERIMETER_INTEL_DEFENSE.md` |

---

## Suggested delivery order

```
Wave 1 (P0): STATUS-MAPPING-INGEST + UI-PERMISSION-GATE
Wave 2 (P0): GOOGLE-OFFLINE + WILDCARD-SSL-DNS01
Wave 3 (P1): POSTBACK-HEALTH + INTEGRATION-ONE-CLICK
Wave 4 (P1): DOMAINS-BULK + TDS-STREAM-UX
Wave 5 (P2): FAST-CLICK-TIER + REGISTRY-STALE + CLICK-PROXY-GUARDRAILS
Wave 6 (P3): SAFE-PAGE-LIMITS-DOC + RESIDENTIAL-PROXY-EDGE + ML-FRAUD-POSITIONING
Wave 7 (P3): TLS-JA4-BROWSER-CORPUS + MOBILE-BIOMETRICS-CLICK
Wave 8 (P3): MODERATOR-FINGERPRINT-CORPUS + CROSS-LAYER-DESYNC-POLICY
Wave 9 (P4): TCP-SYN-OPTION-CORPUS + H2-FRAME-DYNAMICS + CLIENT-RUNTIME-DEEP-PROBES (research; flag-gated)
Wave 10 (P5): SAFE-PAGE-HYDRATOR-CLIENT + HYBRID-SERVER-VERIFY-GATE + DECOY-LANDING-PARITY + CONTENT-DIFF-DRILL + CAPI-BROWSER-DEDUP
Wave 10b (P5 hot-path): HOTPATH-TELEMETRY-ZERO-ALLOC + CLIENT-RTT-PROBE-CORRELATION + ANTIFRAUD-SNAPSHOT-SCORING-GAPS
Wave 11 (P5): REDIRECT-PROFILE-COMPLIANCE + FIRST-PARTY-PIXEL-ORIGIN + BEHAVIOR-MODEL-HUMANIZATION + CLICK-TIMING-WIRE + TIERB-OCCUPANCY-BUDGET
Wave 12 (P5): CROWD-PROBE-SCORING + PROBE-CLUSTER-GRAPH + ASN-MOBILE-TIER + INGEST-SINK-BURST-RESILIENCE
Wave 13 (P5): HYBRID-CROWD-WAVE-DETECTION + ROUTING-TIMING-CONSTANT-TIME + XDP-RESIDENTIAL-POLICY-BOUNDARY
Wave 14 (P5): TLS-SERVER-PERSONA-HARDENING + CLIENT-EDGE-DOM-INTEGRITY + CLIENT-TELEMETRY-STEALTH-PACKAGING (flag-gated)
Wave 15 (P5 docs): SYBIL-HUMAN-OPERATOR-RUNBOOK + PERIMETER_INTEL_DEFENSE.md checklist
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
| XDP vs residential CGNAT | BPF map FP | P5-XDP-RESIDENTIAL-POLICY-BOUNDARY |
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
- `deploy/vendor/PERIMETER_INTEL_DEFENSE.md` — AppSec threat catalog T1–T23 and P5 remediation
- `.cursor/rules/hot-path.mdc` — Tier A/B, alloc gate, filter deadline
- `.cursor/rules/data-layer.mdc` — Redis MAXLEN, broker WAL, stream admission
- `.cursor/rules/edge.mdc` — XDP map limits, residential vs DC policy

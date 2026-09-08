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
| **P3** | Fraud/review positioning + incremental moderator signals | Honest limits; tighten gaps vs residential headless crawlers |
| **P4** | Research-tier passive/runtime probes | Not a sales promise; lab + corpus only |

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

**Problem:** Buyers expect WebGL/headless bypass; stack offers safe-page attestation (`safe_page_attest.go`) and review traffic — not magic. Residential gateway masks L4/L7 desync (server Linux stack vs claimed mobile UA).

**Tasks:**

- [ ] Operator doc: signal matrix (ingress TCP/TLS/H2, safe-page JS, cross-layer) with **evades** column (CDN, residential IP, attestation off)
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

- [ ] Document: edge XDP = flood + blocklist, not cloaking; `OS_FINGERPRINT_MISMATCH_ENABLED=false` on CDN paths
- [ ] Sales copy: `ResidentialProxyFilter` = farm/intel heuristic, not per-session moderator proof

**DoD:**

- [ ] `deploy/vendor/ANTIFRAUD.md` aligned with code
- [ ] Moderator corpus work tracked under P3-MODERATOR-FINGERPRINT-CORPUS (not this slug)

**Dependencies:** P3-SAFE-PAGE-LIMITS-DOC.

---

### P3-MOBILE-BIOMETRICS-CLICK

**Problem:** Gyro/touch biometrics (`summarizeMobileBiometrics`, `mobile_biometrics` CH columns) run on **conversion** when `BEHAVIOR_TELEMETRY_ENABLED` + safe-page attestation. Headless moderator on `/click` never sends `devicemotion` / touch pressure; flat gyro passes conversion-only checks too late.

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

**Problem:** `ja4BrowserCorpusMismatch` and `TLSFingerprintImpersonating` are heuristic. Moderator stack: UA claims iPhone Safari, JA3/JA4 from Chromium/automation — needs version-pinned corpus rows, not blocklist-only.

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

**Problem:** `review_traffic_policy` matches TLS blocklist, CIDR, proxy/VPN, moderator intel IP — not a **learned corpus** of moderator JA3/JA4 + TCP sig + safe-page fingerprint tuples from CH.

**User stories:**

- Operator exports moderator sessions from `layer-desync-drilldown` / `wire-signal-breakdown`; imports corpus row for future safe-page routing.
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

## P4 — research tier (anti-moderator crawler; not sales SLA)

Items below close **technical gaps vs dedicated anti-bot stacks**. Do not pitch on sales call until P3 docs ship and load-tier proof exists.

### P4-TCP-SYN-OPTION-CORPUS

**Problem:** `TCPSynSigMismatch` hashes `ttl + window + mss + doff` only (`hash_tcp_syn_fields` in `edge_filter.c`). Does not encode full TCP option order (NOP, MSS, SACK, WScale, Timestamps) that distinguishes Linux 6.x from iOS stack behind residential tunnel.

**Tasks:**

- [ ] Spec: option-order fingerprint format + corpus file layout (edge + tracker parity)
- [ ] XDP: extend emit struct or second hash; nginx `X-TCP-SIG` v2 header
- [ ] Holdout: Linux corpus row vs iOS UA -> signal; CDN skip documented

**DoD:**

- [ ] `go test ./internal/edge/ -run TCPSyn -count=1`
- [ ] `bash scripts/test/edge/lua_tests.sh unit`
- [ ] Feature flag default **off**; `OS_FINGERPRINT_*` skip metrics unchanged when headers absent

**SLA:** XDP emit path unchanged p99 budget (`xdp-bpf.mdc`).

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

**Problem:** Safe-page attestation lacks WebGPU pipeline timing, IEEE 754 canvas noise probes, `Performance.now()` / `Proxy` hook detection on `navigator` — techniques used by moderation inspectors.

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

## Web inventory — current vs required

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
| Fraud / safe-page | campaign editor fraud tab | P3 biometrics toggle, P3 cross-layer policy, P3 moderator corpus UI |
| Fraud docs | none in admin | P3-SAFE-PAGE-LIMITS-DOC link from presets |

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
```

**Moderator-crawler gap map (reference):**

| Layer | Shipped signal | Backlog slug |
| :--- | :--- | :--- |
| TCP TTL/window vs UA | `OSFingerprintMismatch` | P3-SAFE-PAGE-LIMITS-DOC (CDN limits) |
| TCP SYN hash | `TCPSynSigMismatch` | P4-TCP-SYN-OPTION-CORPUS |
| TLS JA3/JA4 | blocklist + heuristics | P3-TLS-JA4-BROWSER-CORPUS |
| HTTP/2 | SETTINGS / pseudo-order | P4-H2-FRAME-DYNAMICS |
| WebGL / canvas / timezone | `safe_page_attest.go` | P3-MOBILE-BIOMETRICS-CLICK, P4-CLIENT-RUNTIME-DEEP-PROBES |
| Cross-layer | CH `layer_desync_count` | P3-CROSS-LAYER-DESYNC-POLICY |
| Moderator intel | `review_traffic_policy` | P3-MODERATOR-FINGERPRINT-CORPUS |
| Residential egress | `ResidentialProxyFilter` | P3-RESIDENTIAL-PROXY-EDGE |

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

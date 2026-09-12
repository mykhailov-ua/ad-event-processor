# Tracker parity backlog — Keitaro / Binom

Product positioning: **ad-event-processor** is an event/budget/fraud processor with tracking, not a 1:1 Keitaro/Binom clone. This file lists **remaining feature gaps** that arbitrage teams cite on sales calls, with falsifiable Definition of Done per item.

**Related (do not duplicate):**

| Document | Scope |
| :--- | :--- |
| `ARBITRAGE_CLOSURE_BACKLOG.md` | P0–P5 operational waves — **closed** (status mapping ingest, wildcard SSL, bulk clone, fast click tier, fraud positioning, etc.) |
| `TRACKER_ENTERPRISE_BACKEND_SPEC.md` | Enterprise CPA/team backend — spec only; overlaps noted below |
| `deploy/vendor/sku.yaml` | License tier gates for commercial packaging |
| `docs/INTEGRATIONS.md` | Shipped integrations truth |

**Rules (mandatory for every delivery):** `hot-path.mdc`, `cold-path.mdc`, `data-layer.mdc`, `anti-slop.mdc`, `control-plane.mdc`, `traffic.mdc`.

---

## Cross-cutting DoD template

Every epic below must satisfy **all** applicable rows before marking **done**.

| Layer | Requirement |
| :--- | :--- |
| **Contract** | OpenAPI path + schema in `api/openapi/`; `make gen` routes; handler DTO `json` tags match sqlc/OpenAPI |
| **RBAC** | `x-permissions` on route; `RequirePermission` in Go; denied-role `httptest` or curl proof — not nav-hide only (`frontend-slop.mdc` **RB-L***) |
| **Hot path** | No sync Postgres/CH/outbox on `/track` or `/click` accept; filter deadline unchanged; alloc gate if ingest touched |
| **Cold path** | `Validate*` before write; `ReadLimitedBody` / `DecodeRequestOrBadRequest`; `writeServiceError` on failures |
| **Data** | Config mutations via outbox when Redis side effect; budget invariant `AssertBudgetInvariant` on spend paths |
| **Admin UI** | `ErrorBlock` on fetch errors; success toast only after 2xx; no fixture IDs in production paths |
| **Tests** | Unit in `pr_fast`; Redis/Lua/CH claims need `make test-integration` or named `*_holdout*` |
| **Docs** | User-facing copy matches code; no "Binom Smart Rotation" unless behavior matches; marketing bullets in `deploy/marketing/site.config.json` updated in same PR when tiering changes |

**Verification tiers (cite in PR):**

| Tier | Command | Use when |
| :--- | :--- | :--- |
| Fast | `bash scripts/ci/pr_fast.sh` | Default merge gate |
| Integration | `make test-integration` | Redis Lua, PG, CH wiring |
| Fault | `make test-fault` | Postback replay, budget rollback |
| Alloc | `make test-alloc-gate` | `/track`, `/click`, filter chain edits |
| Load | `malformed.sh` / operator Prometheus | SLA claims only with measured p99 |

---

## Priority legend

| Priority | Meaning |
| :--- | :--- |
| **P0** | Sales blocker — competitor demo shows it on day one |
| **P1** | Workflow parity — team churns without it within first month |
| **P2** | Differentiator packaging — upsell or retention, not initial pitch |
| **P3** | Optional / honest substitute — document alternative instead of clone |

---

## 1. Conversion processing and postbacks

Competitor refs: Binom **Status Scheme v2**, **Rules**, **Delayed postbacks**; Keitaro **conversion status**, **postback %**, **multiple postbacks**.

### PARITY-STATUS-SCHEME-ENGINE — P0

**Gap:** Inbound S2S accepts `status` / `goal` and maps via `campaign_conversion_mappings` + affiliate YAML presets (`apply-templates`). No rule engine for if/then chains (e.g. `status=hold` -> internal `rejected`, accumulate payout, fire secondary outbound only when `approved`).

**Current:** `internal/stream/conversion_payout.go`, `conversion_reject_rules`, single mapping row per `(campaign, affiliate_status)`.

**Scope:** Cold path (ingest worker / conversion handler); Redis snapshot for hot read if needed — **not** on `/track` accept.

**DoD:**

- [x] OpenAPI: `POST/GET/PATCH /api/v1/campaigns/{id}/status-schemes` with ordered rules (`when`, `then`, `set_internal_status`, `set_payout_mode`, `fire_outbound`)
- [x] PG migration: `campaign_status_scheme_rules`; sqlc queries (`internal/ingest/queries/status_scheme.sql`)
- [x] Worker applies rules on conversion ingest (`StatusSchemeApplier` in `StoreBatch` after payout mapping); holdout `TestStatusScheme_holdMapsRejected_holdout`
- [x] Admin UI: scheme editor on campaign integration panel
- [x] Binom preset import (`PARITY-BINOM-STATUS-SCHEME-IMPORT`); `docs/INTEGRATIONS.md` migration section
- [x] Verify: `go test ./internal/stream/ -short -run TestStatusScheme -count=1` (exit 0); `go test ./internal/postback/ -short -run TestConversionPostbackEnqueuer_skipsStatusSchemeOutbound -count=1` (exit 0)

---

### PARITY-MULTI-OUTBOUND-POSTBACK — P0

**Gap:** `postback_configs.campaign_id` is PK — **one** outbound template per campaign (`internal/ingest/queries/postback.sql`). Binom/Keitaro allow multiple S2S URLs with conditions and per-URL macros.

**Scope:** Cold path + `cmd/postback-sender`.

**DoD:**

- [x] Schema: `campaign_outbound_postbacks` 1:N with `priority`, `enabled`, `url_template`, `trigger_kind` (`conversion`, `status`, `goal`); legacy `postback_configs` fallback when no rows
- [x] Dispatch loop enqueues N outbox events per conversion when triggers match; idempotency key includes `outbound_postback_id`
- [x] Holdout: `TestMultiOutbound_idempotencyDistinctPerPostback_holdout`; worker `resolvePostbackConfig` loads row by ID
- [x] OpenAPI + admin UI on campaign integration panel (list/replace/test)
- [x] Verify: `go test ./internal/postback/ -short -run TestMultiOutbound -count=1` (exit 0)
- [x] Campaign import/export bundle includes `outbound_postbacks[]` (tokens stripped on import like legacy postback)

---

### PARITY-POSTBACK-SAMPLING-PCT — P1

**Gap:** Competitors fire outbound postback on X% of conversions (traffic quality / network contract).

**DoD:**

- [x] Field `sample_percent` (0–100) on outbound postback row; 0 = off, 100 = all
- [x] Deterministic sampling: `hash(click_id, postback_id) % 100 < sample_percent` (stable across retries)
- [x] Metric `ad_postback_sampled_skipped_total`
- [x] Holdout: 100% always fires; 0% never fires
- [x] Verify: `go test ./internal/postback/ -short -run 'TestShouldFireOutboundPostbackSample|TestMultiOutbound_samplePercent' -count=1`

---

### PARITY-PAYOUT-ACCUMULATION — P1

**Gap:** Binom **cnv_status2** / multi-postback payout sum before final status.

**DoD:**

- [x] PG or Redis state: `click_conversion_ledger` keyed by `click_id` with running `payout_micro`, `last_status`
- [x] Status scheme action `accumulate_payout` documented and tested
- [x] Reports: `conversion_payout_micro` reflects accumulated value in CH ingest
- [x] Verify: `go test ./internal/stream/ -short -run TestConversionLedger -count=1`; integration: `tests/integration/conversion_ledger_test.go` (`make test-integration`)

---

### PARITY-DELAYED-POSTBACKS — P2

**Gap:** Schedule outbound S2S after N minutes/hours (Binom delayed postback).

**DoD:**

- [x] Outbox `not_before` on `SEND_POSTBACK` rows; postback worker claims when `not_before <= NOW()`
- [x] Click TTL / retention skip documented (`docs/INTEGRATIONS.md` delayed outbound section)
- [x] Admin UI `delay_seconds` on outbound postback row
- [x] Verify: `go test ./internal/postback/ -short -run 'TestDelayedPostback|TestPostbackNotBefore' -count=1`

---

### PARITY-RULES-FROM-REPORTS — P2

**Gap:** Binom creates optimization rules from report row (pause source, adjust bid).

**DoD:**

- [x] `POST /api/v1/reports/rules` creates `automation_rule` from report filter snapshot (cold path only)
- [x] Worker executes rule via existing campaign pause / budget PATCH (outbox)
- [x] RBAC: `campaigns:write` + audit log
- [x] UI: "Create rule" on source quality report row (`report_rule_create_dialog.tsx`)
- [x] Verify: `go test ./internal/controlplane/ -short -run TestReportRule -count=1`

**Cross-ref:** `TRACKER_ENTERPRISE_BACKEND_SPEC.md` §3.3 outbound DLQ auto-replay, inbound HMAC — partially shipped in `internal/postback/inbound/`; remaining DLQ/state-machine items stay in enterprise spec, not duplicated here.

---

## 2. Traffic distribution and rotation

Competitor refs: Binom **Smart Rotation** (unseen, fix-on, top-to-bottom); Keitaro **streams**, **filters**, **split testing**.

### PARITY-ROTATION-UNSEEN-COOKIE — P0

**Gap:** Flow supports `weighted`, `waterfall`, `waterfall_then_landing` (`internal/flow/types.go`) but no **per-user unseen** lander/offer/path rotation (Binom cookie / fingerprint bucket).

**Scope:** Hot path read: Redis or local snapshot; **no PG on `/click`**.

**DoD:**

- [x] Flow path field `rotation_mode: unseen` (`PathDTO`, `flowPathJSON` in `buildFlowSnapshot`)
- [x] Redis key `{campaign_id}:rot:seen:{visitor_key}` TTL 30d; visitor_key from `click_id`, else `user_id`, else `ip|ua`
- [x] Go selector skips seen landers/offers until pool exhausted, then reset (`rotation_seen.go`)
- [x] Holdout: `TestSelectSnapshot_unseenRotation_secondClickDifferentLander_holdout`
- [x] Admin UI `rotation_mode` picker on flow editor (`flow_editor_visual.tsx`); OpenAPI `FlowPath.rotation_mode`
- [x] Holdout: `TestSelectSnapshot_fixOn_pinsLander_holdout`, `TestSelectSnapshot_unseenDiffersFromFixOn_holdout`
- [x] Holdout: `TestClickRedirect_unseenRotation_secondClickDifferentLander_holdout`, `TestClickRedirect_fixOn_pinsLander_holdout` (gnet `/click`)
- [x] Integration: `TestRotationSeenRedis_unseenThenFixOn_integration` (`make test-integration` Redis tier)
- [ ] Verify: `make test-alloc-gate` (operator tier; selector on click path)

---

### PARITY-ROTATION-FIX-ON — P1

**Gap:** Binom **fix-on** pins chosen LP/offer for session after first hit.

**DoD:**

- [x] `rotation_mode: fix_on` stores first selection in same visitor key as unseen (`selectFixOnLander` / `selectFixOnOffer`)
- [x] Subsequent `/click` on same campaign returns same entity without re-roll (Redis seen set + fix_on selector)
- [x] Holdout: `TestSelectSnapshot_fixOn_pinsLander_holdout`, `TestSelectSnapshot_unseenDiffersFromFixOn_holdout`
- [x] Document visitor key and Redis `{campaign_id}:rot:seen:{visitor_key}` in `docs/INTEGRATIONS.md`
- [x] Integration: `TestRotationSeenRedis_unseenThenFixOn_integration`

---

### PARITY-ROTATION-TOP-TO-BOTTOM — P1

**Gap:** Strict sequential rotation (offer 1 until cap, then offer 2).

**DoD:**

- [x] `rotation_mode: sequential` with explicit ordering index on flow children (`rotation_seen.go`, flow editor option)
- [x] Optional daily cap per entity uses existing conversion cap snapshot or new click cap (see PARITY-OFFER-CLICK-CAPS)
- [x] Verify: `TestSelectSnapshot_sequentialPicksFirstUncappedOffer_holdout`

---

### PARITY-FLOW-UI-MULTI-ENTITY — P0

**Gap:** Visual flow editor models **one LP + one offer per path** (`web/src/domains/creative/flow_path_model.ts`); backend JSON already supports multi-lander/offer.

**DoD:**

- [x] `flow_path_model.ts` allows N landers and N offers per path with weights
- [x] `flow_editor_visual.tsx` renders/adds/removes lander and offer rows per path
- [x] Save round-trip: `visualRowsToFlowPaths` emits multi-ref `FlowPath` JSON
- [x] E2E: `web/e2e/flow_multi_entity.spec.js` PUT flow with two offers + `rotation_mode: unseen`
- [x] Admin routes `/flows` and `/flows/:id` restored (`flows_page.tsx`, `flow_detail_page.tsx`)
- [x] Verify: `bash scripts/ci/admin/web.sh` (operator tier)
- [x] Verify: `npm test -- src/domains/creative/flow_path_model.test.ts` exit 0 (294 pass)

---

### PARITY-TRAFFIC-OPTIMIZER-UI — P1

**Gap:** Traffic optimizer API exists (`internal/trafficoptimizer` / reports) but not exposed as Keitaro-like "optimize this path" workflow.

**DoD:**

- [x] Admin page `/integrations/traffic-optimizer`: presets, rules CRUD, dry-run table, apply -> flow PUT (`integrations_traffic_optimizer.tsx`)
- [x] Server authoritative: weights from `POST .../dry-run` only; client maps arms to flow paths (`traffic_optimizer_apply.ts`)
- [x] RBAC `campaigns:write` gates create/apply/delete; `ErrorBlock` on apply/dry-run errors
- [x] Manual apply via `POST /api/v1/traffic-optimizer/rules/{id}/apply` (same path as worker; records `traffic_optimizer_fires`)
- [x] Verify: `go test ./internal/trafficoptimizer/ -short -run TestHTTPHandlers -count=1`
- [x] Verify: `npm test -- src/domains/integrations/traffic_optimizer_apply.test.ts`
- [x] Verify: `bash scripts/ci/admin/web.sh` (operator tier)

---

### PARITY-OFFER-CLICK-CAPS — P1

**Gap:** Keitaro/Binom offer daily/hourly **click** caps with waterfall fallback. BidShard has conversion caps (~30s snapshot) not click caps.

**Cross-ref:** `TRACKER_ENTERPRISE_BACKEND_SPEC.md` §2.4, §3.2.

**DoD:**

- [x] Redis `INCR` per offer daily/total keys on `/click` (`filter.ReserveOfferClick` in `flow_click.go`)
- [x] Flow selector retries next offer when cap exhausted; metric `ad_offer_click_cap_exhausted_total`
- [x] Holdout: `TestReserveOfferClick_enforcesDailyCap_holdout`
- [x] Holdout: `TestFlowClickCap_secondClickRoutesNextOffer_holdout` (flow select + Redis cap)
- [x] Holdout: `TestClickRedirect_offerClickCap_secondClickStillRedirects_holdout` (gnet `/click` + Redis `INCR`)
- [ ] Verify: `make test-alloc-gate` (operator tier)

---

### PARITY-BUDGET-FAILOVER-REDIRECT — P1

**Gap:** `/click` returns 402 on budget exhaust; competitors optional redirect to backup URL.

**Cross-ref:** `TRACKER_ENTERPRISE_BACKEND_SPEC.md` §3.2 item 5.

**DoD:**

- [x] Campaign field `fallback_click_url` + `budget_failover_mode` (`reject` | `fallback_url` | `flow_next`)
- [x] Hot path: read from registry snapshot (`budget_failover.go`, `tryBudgetFailoverClick`)
- [x] Holdout: fallback returns 302 not 402 when configured (`TestBudgetFailover_redirectsNotDebits_holdout`)
- [x] Verify: `go test ./internal/ingest/ -short -run TestBudgetFailover -count=1`

---

### PARITY-CAMPAIGN-STREAMS-GROUPS — P2

**Gap:** Keitaro **streams** (traffic distribution groups across campaigns). BidShard uses per-campaign flow + TDS templates.

**DoD (minimal v1):**

- [x] `campaign_groups` entity: name, shared default flow reference, member campaign IDs
- [x] Bulk assign campaigns to group via API
- [x] Reports filter by group_id (`click-log` / `clicks`; `ResolveReportCampaignIDs`)
- [x] No requirement to mirror Keitaro UI 1:1 — document mapping in import wizard (`docs/INTEGRATIONS.md`)
- [x] Verify: PG integration test create group + filter list (`TestCampaignGroup_assignListAndReportScope`)

---

## 3. Campaign operations and bulk edit

Competitor refs: Binom/Keitaro **mass edit** (URL, geo, budget, weights).

### PARITY-BULK-CAMPAIGN-PATCH — P0

**Gap:** `POST /api/v1/campaigns/bulk-action` supports pause/resume/archive (≤50). No mass update of `target_url`, `budget_limit`, `geo`, flow weights.

**DoD:**

- [x] OpenAPI `POST /api/v1/campaigns/bulk-patch` body: `campaign_ids[]`, `patch` (allowlisted fields only)
- [x] Server validates each row; partial success response `{ applied, failed[] }` with reasons
- [x] Outbox per campaign via existing `PatchCampaign` (per-campaign TX)
- [x] RBAC: `campaigns:write` + per-id `AuthorizeCampaignIDs`; masked role denied via `PatchCampaign` -> `forbidden`
- [x] Admin UI: bulk select + `Bulk edit` dialog on campaigns directory (`campaign_bulk_patch_dialog.tsx`)
- [x] Holdout: masked role bulk-patch budget -> `forbidden` (`TestPostCampaignBulkPatch_holdoutMapsForbidden`)
- [x] Verify: `go test ./internal/campaign/editor/ -short -run TestPostCampaignBulkPatch -count=1` -> exit 0
- [x] Playwright: `web/e2e/campaigns_bulk_patch.spec.js` bulk timezone patch + list refresh

---

### PARITY-BULK-FLOW-WEIGHT — P1

**Gap:** Mass change rotation weights across campaigns.

**DoD:**

- [x] `bulk-patch` extension `flow_path_weights` absolute weights per `path_index`
- [x] Validation against `flow/validate.go`
- [x] Verify: `go test ./internal/campaign/ -short -run TestApplyFlowPathWeights -count=1`

---

## 4. APIs and external integrations

Competitor refs: Keitaro **KClient**, **Admin API**, **Click API**; Binom **Click API**, **Tracker API**.

### PARITY-CLICK-API-PROGRAMMATIC — P0

**Gap:** `GET /click` issues redirect when hit in browser. No server-side **mint click URL / click_id** without redirect (KClient / `binom_click_api`).

**Scope:** New cold or warm endpoint — not on gnet hot loop.

**DoD:**

- [x] `POST /api/v1/tracker/clicks`: inputs `campaign_id`, optional `click_id` / `params`; returns `click_id`, `click_url`
- [x] Rate limit per API key (`X-API-Key` + `RequireAnyPermissionOrAPIKey` on `POST /api/v1/tracker/clicks`)
- [x] Dedicated audit log row (`MINT_PROGRAMMATIC_CLICK`)
- [x] No Redis debit on mint (debit on `/click` only)
- [x] OpenAPI `trackerMintClick` in `api/openapi/paths/campaigns.yaml`
- [x] `web/src/lib/docs_tracker_section.ts` section "Programmatic click API"
- [x] Link signing append when campaign `link_signing_enabled` (`LinkSigningSecret` on handlers + `MaybeSignProgrammaticClickURL`)
- [x] Holdout: missing `campaign_id` -> 400 (`TestPostMintProgrammaticClick_returnsClickURL` mux path)
- [x] Verify: `go test ./internal/campaign/editor/ -short -run 'TestMintProgrammaticClick|TestPostMintProgrammaticClick' -count=1` -> exit 0

---

### PARITY-TRACKER-JS-SDK — P1

**Gap:** Keitaro KClient PHP/JS; Binom client scripts. BidShard ships `track.js` / `trackEvent` but no drop-in **landing page SDK** (sub-ids, form submit, auto params).

**DoD:**

- [x] `web/src/static/track.js` export documented helpers: `init(campaign)`, `click()`, `conversion(goal, payout)`
- [x] CSP-safe external script tag (`data-campaign-id`); doc in `docs/INTEGRATIONS.md` hosted lander section
- [x] Hosted lander snippet via Integration tab + hosted editor route (`docs/INTEGRATIONS.md` §5)
- [x] Verify: `go test ./internal/track/ -short -run TestTrackPixelContract -count=1`

---

### PARITY-WORDPRESS-PLUGIN — P2

**Gap:** Keitaro WordPress plugin for KClient.

**DoD:**

- [x] Separate repo or `deploy/wordpress/` package: settings page, shortcode, uses Click API + track.js
- [x] Published version pin compatible with Click API contract (`deploy/wordpress/README.md`)
- [x] Not required for core merge gate — track as optional artifact with install doc

---

### PARITY-ADMIN-API-COVERAGE-AUDIT — P1

**Gap:** Competitors document full Admin API. BidShard OpenAPI large but gaps vs UI (traffic optimizer apply, status schemes when built).

**DoD:**

- [x] Script or checklist: `bash scripts/ci/admin/openapi_mutation_coverage.sh`
- [x] `docs/INTEGRATIONS.md` link to `/api/v1/openapi.yaml`
- [x] CI gate: `bash scripts/ci/admin/live_routes.sh` passes for `live: true` routes (operator tier)
- [x] Verify: `bash scripts/ci/admin/web.sh` + `live_routes.sh` exit 0

---

## 5. Landers and creatives

Competitor refs: Binom **Landing page builder**; Keitaro **editor**, **local landing**.

### PARITY-LANDER-WYSIWYG-BUILDER — P2

**Gap:** Hosted ZIP + HTML code editor (`lander_hosted_editor.tsx`, `/lp/{id}/`). No drag-drop WYSIWYG.

**DoD (MVP — not full Webflow clone):**

- [x] Block-based editor: hero, CTA button, image, raw HTML block (`lander_wysiwyg_panel.tsx`)
- [x] Publish pipeline unchanged (versioned ZIP to object store / static host)
- [x] Preview URL before publish (server preview + block srcDoc preview)
- [x] Honest marketing: do not claim "visual builder parity" until blocks ship
- [ ] Verify: E2E publish + GET `/lp/{id}/` 200 (`web/e2e/lander_hosted_publish.spec.js`; operator integration tier)

---

### PARITY-OFFER-WALL-PRODUCT — P3

**Gap:** Marketing Starter bullet **"offer walls"** — no `offer_wall` feature flag or route in tree (likely means multi-offer flow).

**DoD (choose one):**

- [ ] **Option A:** Implement offer-wall template (grid of offers, single page flow type) + SKU feature `offer_wall`
- [x] **Option B:** Remove/reword marketing bullet to "multi-offer flows" with link to flow editor — no fake feature name
- [x] Verify: no `offer_wall` token in marketing (`site.config.json`); flow editor is canonical multi-offer surface

---

### PARITY-LP-LOCAL-SERVE — P1

**Gap:** Keitaro local landing without CDN. BidShard hosted lander exists — verify parity for self-hosted path.

**DoD:**

- [x] Document: nginx `/lp/` routing, cache headers, custom domain on lander host (`docs/INTEGRATIONS.md` §5)
- [x] Admin: hosted-files API deploy instructions + hosted editor snippet path
- [x] Missing pieces filed as bugs with DoD in this section (none open)
- [x] Verify: `docs/INTEGRATIONS.md` hosted lander section matches `internal/flow/list_landers.go`

---

## 6. Cost, economics, and reporting

Competitor refs: Binom **update costs** rules; Keitaro **cost sync** (FB/Google/TikTok).

### PARITY-COST-SYNC-TIER-LIMITS — P1 (commercial, not competitor gap)

**Gap:** 20+ cost sync networks shipped — **ahead** of Keitaro on breadth. Risk: Starter/Pro unlimited sync blocks Scale upsell.

**DoD:**

- [x] `sku.yaml`: `max_cost_sync_networks` per SKU (pilot=0 manual only, starter=3, pro=15, scale+=unlimited)
- [x] `licensingadmin.EnforceCostSyncNetworkCap` on cost-sync credential upsert and manual run (`billingadmin/cost_sync_handlers.go`)
- [x] Holdout: over-limit / disabled tier -> 429 `LIMIT_EXCEEDED` (`ErrDeploymentCostSyncNetworkLimit`)
- [x] Marketing `site.config.json` starter/pro cost sync bullets aligned
- [x] Admin UI `LIMIT_EXCEEDED` upgrade CTA in `panel_error.tsx` (links to Settings)
- [x] Verify: `go test ./internal/licensing/entitlements/ -short -run TestCostSyncTier -count=1`

---

### PARITY-MARGIN-GUARD-TIER-MOVE — P1 (commercial)

**Gap:** `margin_guard: true` on Pilot in `sku.yaml` — competitors do not give margin automation on entry tier; hurts Scale differentiation.

**DoD:**

- [x] `margin_guard` enabled Pro+ only (`SanitizeFeaturesForSKU` clears starter/pilot; `sku.yaml` margin_guard: false)
- [x] `internal/marginguard` HTTP gate via existing `licenseGate("margin_guard")` (unchanged wiring)
- [x] Marketing `site.config.json` Margin Guard tier Pro+; removed from Starter feature list
- [x] Admin UI `/integrations/margin-guard` restored (`integrations_margin_guard.tsx`; license gate via API 403 FEATURE_REQUIRED)
- [x] Verify: `go test ./internal/controlplane/ -short -run TestMarginGuardLicense -count=1`

---

### PARITY-MANUAL-COST-UI — P1

**Gap:** Manual cost per campaign/source in UI (competitors daily spreadsheet workflow).

**DoD:**

- [x] `PUT /api/v1/campaigns/{id}/manual-cost` manual cost entry with date range
- [x] PG `campaign_costs` (`network=manual`) feeds true-ROI via existing cost snapshot path
- [x] UI on campaign stats panel (`campaign_manual_cost_form.tsx`)
- [x] Verify: `go test ./internal/campaign/ -short -run TestPutManualCampaignCost -count=1`

---

### PARITY-REPORT-BUILDER-PARITY — P2

**Gap:** Binom custom columns / slices; Keitaro custom metrics.

**DoD:**

- [ ] Export hub supports saved report defs (partial — extend existing `export_hub`)
- [ ] Column picker matches CH schema fields documented in OpenAPI
- [ ] No client-side ROI math — server query params only
- [ ] Verify: `web/e2e/export_hub.spec.js` extended

---

## 7. Import, migration, and presets

### PARITY-BINOM-STATUS-SCHEME-IMPORT — P1

**Depends on:** PARITY-STATUS-SCHEME-ENGINE.

**DoD:**

- [x] `internal/migrationsource` maps Binom export status scheme JSON to native rules
- [x] Import preview shows unmappable rules count (`unmapped_status_scheme_rules`, warnings)
- [x] Holdout: golden file `testdata/binom_status_scheme_campaign.json`
- [x] Verify: `go test ./internal/migrationsource/ -short -run TestBinomStatusScheme -count=1`

---

### PARITY-KEITARO-STREAM-IMPORT — P2

**DoD:**

- [ ] Keitaro stream -> `campaign_groups` or flow mapping documented
- [ ] Import wizard UI step with dry-run counts
- [ ] Verify: fixture import test

---

## 8. Team, access, and multi-user

Competitor refs: Keitaro **admin users**, **permissions**; Binom **user access**.

### PARITY-TEAM-SCOPE-ENFORCEMENT — P1

**Cross-ref:** `TRACKER_ENTERPRISE_BACKEND_SPEC.md` §2.1 — `ScopeTeam` unused.

**DoD:**

- [x] Campaign list filters by ownership via `teamscope.ResolveListScope` (`campaign/list_query.go`)
- [x] Masked role owner query override blocked (`teamscope.AllowOwnerQueryOverride`)
- [x] Economics redaction on true-ROI for masked roles (`redactTrueROIRows`; compare_previous prev rows redacted)
- [x] Verify: `go test ./internal/reports/ -short -run TestRedactTrueROI -count=1`
- [x] Verify: `go test ./internal/controlplane/ -short -run TestTeamScope -count=1`

---

## 9. Security and cloaking (honest positioning)

### PARITY-BUILTIN-CLOAKER — P3

**Gap:** Binom markets built-in **cloaker**. BidShard uses antifraud + silent reject + XDP (different model).

**DoD (documentation, not necessarily code):**

- [ ] `deploy/vendor/ANTIFRAUD.md` section: map Binom cloaker expectations to silent reject, geo, ML boost, safe page
- [ ] Sales cheat sheet: what we do **not** promise (moderator fingerprint 100%)
- [ ] Optional: campaign-level "safe page URL" already exists — document as cloaker substitute
- [ ] Verify: `bash scripts/ci/naming/antifraud_doc.sh`

---

## 10. Shipped strengths (no backlog — do not regress)

Use on competitive matrix; not implementation tasks.

| Area | BidShard | Keitaro/Binom typical |
| :--- | :--- | :--- |
| Budget invariant + Redis Lua debit | Yes | Partial / plugin |
| Silent reject + fraud stream | Yes | Limited |
| CAPI outbound bundle | Yes | Varies |
| Cost sync network breadth | 20+ | Often 3–5 native |
| ClickHouse analytics | Yes | MySQL |
| Telegram Mini App | Yes | Rare |
| Keitaro/Binom import | `migrationsource` | N/A |
| Broker WAL ingest (Scale+) | Yes | N/A |
| XDP / edge rate limit | Optional SKU | N/A |

---

## 11. Suggested delivery waves

| Wave | Epics | Goal |
| :--- | :--- | :--- |
| **W1** | PARITY-STATUS-SCHEME-ENGINE, PARITY-MULTI-OUTBOUND-POSTBACK, PARITY-BULK-CAMPAIGN-PATCH, PARITY-CLICK-API-PROGRAMMATIC | Survive Binom postback + bulk edit demo |
| **W2** | PARITY-ROTATION-UNSEEN-COOKIE, PARITY-FLOW-UI-MULTI-ENTITY, PARITY-OFFER-CLICK-CAPS, PARITY-BUDGET-FAILOVER-REDIRECT | Survive smart rotation + caps demo |
| **W3** | PARITY-COST-SYNC-TIER-LIMITS, PARITY-MARGIN-GUARD-TIER-MOVE, PARITY-TEAM-SCOPE-ENFORCEMENT, PARITY-TRAFFIC-OPTIMIZER-UI | Monetization + team accounts |
| **W4** | PARITY-LANDER-WYSIWYG-BUILDER, PARITY-WORDPRESS-PLUGIN, PARITY-RULES-FROM-REPORTS | Retention / long tail |
| **W5** | PARITY-ROTATION-TOP-TO-BOTTOM, PARITY-POSTBACK-SAMPLING-PCT, PARITY-CLICK-API tail | Rotation + outbound sampling + API hardening |
| **W6** | PARITY-DELAYED-POSTBACKS, PARITY-OFFER-WALL (option B), admin/live_routes verify closure | Delayed S2S + marketing honesty + gate closure |

---

## 12. Explicit non-goals

| Item | Reason |
| :--- | :--- |
| 1:1 Binom UI clone | `ui.mdc` contract; server-authoritative admin |
| MySQL-backed reporting | CH is canonical |
| Per-click ML on `/click` | `hot-path.mdc`; batch `cmd/fraud-scorer` only |
| Redis `KEYS` / admin FLUSH | `cold-path.mdc` |
| PHP KClient drop-in | Use HTTP Click API + JS SDK instead |
| Guarantee "unlimited" events on Pilot | `sku.yaml` monthly_events caps |

---

## Changelog

| Date | Change |
| :--- | :--- |
| 2026-09-12 | Initial parity backlog extracted from codebase audit vs Keitaro/Binom; cross-ref closed `ARBITRAGE_CLOSURE_BACKLOG.md` |
| 2026-09-12 | W2: gnet `/click` holdouts for unseen/fix_on rotation and offer click caps; rotation docs in `INTEGRATIONS.md`; integration test `flow_rotation_integration_test.go` |
| 2026-09-12 | W2: restore admin `/flows` + `/flows/:id` routes; W3 start: cost sync network caps, margin guard Pro+ tier, team scope holdouts |
| 2026-09-12 | W3: traffic optimizer admin UI (`/integrations/traffic-optimizer`); tier tests green (`TestCostSyncTier`, `TestTeamScope`, `TestMarginGuardLicense`) |
| 2026-09-12 | W3 tails: margin guard UI, cost-sync LIMIT_EXCEEDED CTA, true-ROI masked compare fix, optimizer apply endpoint |
| 2026-09-12 | P1 closure: payout ledger, bulk flow weights, manual cost API/UI, Binom status scheme import, tracker SDK, LP local serve docs, OpenAPI mutation coverage script |
| 2026-09-12 | W6: delayed outbound postbacks (`delay_seconds`, `outbox_events.not_before`); offer-wall marketing reword; `web.sh` + `live_routes.sh` green |

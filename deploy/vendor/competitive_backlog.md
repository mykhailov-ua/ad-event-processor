# Competitive parity backlog

Gap list vs cloud trackers (BeMob-class) and self-hosted trackers (Keitaro, Binom). Derived from operator comparisons and current tree (`README.md`, `sku.yaml`, `enhanced_defense_baseline_audit_test.go` product-scope gates).

**Not in scope for this file:** antifraud signal and ML work - see `ANTIFRAUD.md` and `antifraud_backlog.md` (when present). **Compliance:** defensive perimeter only (`compliance.mdc`); no outbound attack tooling.

Each item uses a semantic slug for cross-reference in PRs and docs. **Do not mark a slug closed** until every checked gate below applies to that PR (skip N/A lines only when the touch surface truly did not change).

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `parity_blocker` | Blocks positioning against Keitaro/Binom for typical affiliate workflows |
| `growth` | Improves win rate vs BeMob without changing deploy model |
| `enterprise_optional` | Large build; license-gated or separate SKU surface |
| `pricing_gtm` | Packaging, trial, or onboarding - not a hot-path code change |

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`, `hot-path.mdc`)
- [ ] Verification commands pasted in PR with package path (no unrun claims - `quality.mdc`)
- [ ] Holdout or fault test added when behavior is non-obvious (`testing.mdc`)
- [ ] Doc claims match code; no microbench cited as prod SLA (`anti-slop.mdc` lie modes)
- [ ] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [ ] No new thin `*_gate.sh` that only re-invokes existing gates (`anti-slop.mdc`)
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `README.md`, `docs/`, `deploy/vendor/` (`naming.mdc`)

Rule: `core.mdc` commit policy (when landing code)

- [ ] Imperative commit title names concrete surface; no `refactor:` / milestone tokens (`core.mdc`)
- [ ] Docs-only changes ship in the same commit as code (`core.mdc`)

---

## Summary

| Slug | Priority | Competitor gap | Rough surface |
| :--- | :--- | :--- | :--- |
| `local_lander_zip_hosting` | parity_blocker | Keitaro/Binom ZIP upload | control + edge static |
| `lander_panel_code_editor` | parity_blocker | In-panel HTML edit | admin UI + object store |
| `visual_flow_builder_ui` | growth | Keitaro flow UI | `web/` flows pages |
| `moderator_intel_feed` | growth | Shipped: signed `moderator_intel_v1` + `moderator_ip` signal | `pkg/moderatorintel` + tracker loader |
| `review_traffic_policy` | growth | Shipped: per-campaign `review_traffic_action` matrix | tracker + campaign config |
| `one_click_compose_bootstrap` | growth | Shipped: `scripts/install/appliance_bootstrap.sh` | install + compose |
| `guided_first_campaign_wizard` | growth | Shipped: `/campaigns/wizard` multi-step onboarding | admin UI |
| `tiktok_cost_sync` | growth | Shipped: spend pull + OAuth refresh | `internal/costsync` |
| `microsoft_ads_cost_sync` | growth | Shipped: Reporting API v13 campaign spend + OAuth refresh | `internal/costsync` |
| `snapchat_cost_sync` | growth | Shipped: ad account stats + OAuth refresh | `internal/costsync` |
| `linkedin_cost_sync` | growth | Shipped: adAnalytics campaign spend + OAuth refresh | `internal/costsync` |
| `pinterest_cost_sync` | growth | Shipped: campaigns analytics + OAuth refresh | `internal/costsync` |
| `trafficstars_cost_sync` | growth | Shipped: campaigns statistics v2 + offline key refresh | `internal/costsync` |
| `richads_cost_sync` | growth | Shipped: advertiser reports API + API key | `internal/costsync` |
| `galaksion_cost_sync` | growth | Shipped: advertiser statistics + token or login | `internal/costsync` |
| `ad_platform_campaign_api` | enterprise_optional | Shipped: pause/resume/budget + status sync | `internal/platformsync` |
| `managed_saas_tenant_plane` | enterprise_optional | Shipped phase 1: `managed_saas` SKU + isolated compose cell | `docs/MANAGED_SAAS.md` |
| `extended_trial_selfserve` | pricing_gtm | Shipped: 14d pilot SKU + license upgrade CTA | `sku.yaml` + license UI |
| `workspace_billing_split` | pricing_gtm | BeMob paid workspaces | tenants + invoices |

---

## `local_lander_zip_hosting`

**Priority:** parity_blocker

**Gap:** Keitaro and Binom host landing files inside the tracker. ~~We only store lander metadata (`landers.url`) and redirect/proxy to external URLs (`gateNoLocalLanders = redirect_and_proxy_only`).~~ **Shipped** (`gateLocalLanders = local_zip_hosting`).

**Current state:** Admin ZIP upload to `LANDER_STORE_ROOT`, versioned `lander_assets`, nginx `/lp/` static (or control fallback), flow snapshot resolves hosted URL when `hosted_asset_id` set.

**Target:**

- Admin upload ZIP (size cap, MIME allowlist, path traversal guards).
- Extract to versioned object store (local `var/landers/` dev; S3-compatible prod).
- Serve on campaign/flow hostname via nginx or tracker static handler with cache headers.
- Flow `lander_id` resolves to hosted path when `url` empty and `hosted_asset_id` set.

**Dependencies:** `settings_domains_page` TLS allowlist; per-tenant storage quota in license limits.

### Done gates

Rule: `architecture.mdc` / `hot-path.mdc`

- [x] Static assets served from edge/nginx or cold object store - **no** per-request PG/CH on `/click` hot path
- [ ] If tracker serves bytes: `make test-alloc-gate` and `bash scripts/ci/escape_heap_gate.sh` on touched hot files
- [x] Registry snapshot for hosted path resolution - no per-click sqlc fetch (`architecture.mdc`)

Rule: `cold-path.mdc`

- [ ] Upload handler uses `pkg/coldpath.ReadLimitedBody` / `DefaultMaxBody` (64 KiB default; raise cap explicitly for ZIP with documented max) — ZIP uses `ParseMultipartForm` + `io.LimitReader` at `LANDER_MAX_ZIP_BYTES`
- [ ] `bash scripts/ci/cold_path_static_gate.sh` and `cold_path_json_gate.sh`
- [x] Config mutation + `outbox_events` in same PG txn when lander publish affects tracker (`control-plane.mdc`)

Rule: `compliance.mdc` (edge static route)

- [x] No direct BPF map writes from admin handlers; deny lists still via outbox -> Redis -> sync
- [ ] `bash scripts/ci/compliance.sh` if `deploy/nginx/**` or `internal/edge/**` touched

Rule: `testing.mdc`

- [ ] `*_integration_test.go` with `integration:` skip reason, real store helper, behavioral asserts
- [ ] `bash scripts/ci/integration_test_slop_gate.sh` on new integration tests
- [ ] E2E: upload ZIP -> `GET /click` -> hosted asset 200 with expected body hash

Rule: `ui.mdc` (upload UI)

- [x] DTO fields match Go handler JSON tags; `apiConfirmed` on upload/publish
- [x] `cd web && npm run typecheck` and `bash scripts/ci/admin_web.sh`

Rule: product scope

- [x] Flip or remove `gateNoLocalLanders` in `enhanced_defense_baseline_audit_test.go` (E2E click->hosted hash still open)

---

## `lander_panel_code_editor`

**Priority:** parity_blocker

**Gap:** Competitors edit lander HTML/JS in the admin panel. ~~We have no LP builder (`gateNoVisualEditors = no_grapejs_lp_builder`).~~ **Shipped** text editor (`gateHostedLanderEditor = hosted_lander_code_editor`). GrapeJS drag-and-drop builder remains out of scope.

**Current state:** Hosted lander file tree, textarea editor, draft versions, signed preview URL, publish to live.

**Target:**

- File tree for hosted lander versions (index.html, assets/).
- Monaco or CodeMirror editor in `web/` with save -> new immutable version.
- Preview URL with signed token; publish promotes version to live.

**Dependencies:** `local_lander_zip_hosting`.

### Done gates

Rule: `ui.mdc` / `anti-slop.mdc` admin slop

- [x] No `(skeleton)` / demo KPIs / silent empty tables on error
- [x] `renderErrorBlock` on fetch failures; `StubBanner` only for real 501
- [x] `apiConfirmed` on save/publish; no "Saved" toast before 2xx
- [x] JSDoc on new `web/**` functions and module constants (`quality.mdc`)
- [x] `cd web && npm run typecheck` and `bash scripts/ci/admin_web.sh`

Rule: `cold-path.mdc`

- [x] Version save uses `ReadLimitedBody` / documented max per file
- [x] Preview token verify uses constant-time compare where secrets involved
- [x] `go test ./internal/controlplane/... -short` on touched handlers

Rule: `architecture.mdc`

- [x] Editor and preview are cold path only; live traffic reads published snapshot

Rule: product scope

- [x] Remove or gate-flip `gateNoVisualEditors` only when editor ships

---

## `visual_flow_builder_ui`

**Priority:** growth

**Gap:** Keitaro visual flow editor. ~~We have declarative JSON paths in API (`flows.paths`) and list UI stub scope (`gateNoFlowBuilderUI = declarative_backend_lists_only`).~~ **Shipped** path builder UI (`gateVisualFlowBuilder = visual_flow_builder_ui`). Geo/device path filters remain future work.

**Current state:** `/campaigns/flows/:id/builder` edits weighted paths, landers, and offers; `PUT /api/v1/flows/{id}` validates graph server-side and publishes `flow:reload`.

**Target:**

- Drag-and-drop path editor: landers, offers, filters (geo, device), weights.
- Validate graph server-side (no orphan lander_id, weight sum rules).
- Keep hot path `CampaignFlowTable.Select` unchanged - only config shape enrichment.

### Done gates

Rule: `hot-path.mdc` / `architecture.mdc`

- [x] `go test ./internal/ingestion/ -run=Flow -count=1` golden fixtures unchanged for same JSON paths
- [x] No new Redis RTT or PG call in `flow_click.go` / `CampaignFlowTable.Select`
- [ ] If hot structs change: `make test-alloc-gate` on `internal/ingestion/`

Rule: `cold-path.mdc` / `control-plane.mdc`

- [x] Flow graph validation in controlplane service - no N+1 lander lookups in loop
- [x] Flow update publishes `flow:reload` (same channel as lander publish)

Rule: `ui.mdc`

- [x] Form fields match `FlowDTO` / path JSON schema from Go
- [ ] `bash scripts/ci/report_live_routes_gate.sh` if new `/campaigns/*` route is `live: true`
- [x] `admin_web.sh` + typecheck

Rule: product scope

- [x] Flip `gateNoFlowBuilderUI` when visual editor replaces declarative-only UX

---

## `moderator_intel_feed`

**Priority:** growth

**Gap:** Articles claim "true cloaking" needs daily-updated moderator and residential-proxy IP sets for Meta, Google, TikTok review traffic. ~~We have fraud signals (DC, TLS, residential intel SKU) but no dedicated moderator corpus.~~ **Shipped** signed pull feed (`gateModeratorIntelFeed = moderator_intel_feed`).

**Current state:** `pkg/moderatorintel` parses `moderator_intel_v1` with HMAC signature and TTL. Tracker loads into in-memory LPM (`ModeratorIPTable`); `/click` serves safe view when campaign `moderator_intel_enabled` and IP matches. SKU `moderator_intel_feed` on Scale+; default off per campaign.

**Target:**

- Optional signed feed channel (similar to GeoIP updater): `moderator_intel_v1` with source attribution and TTL.
- Map to L1-high or dedicated `moderator_ip` signal; default off per campaign.
- Document fail-open on stale feed; never wipe last good snapshot (`testing.mdc` feed pattern).

**Compliance note:** Defensive classification only - show `safe_page_url` to matched review traffic, not offensive probing.

**Dependencies:** `review_traffic_policy` for operator-visible policy matrix.

### Done gates

Rule: `compliance.mdc`

- [x] Feed fetch is vendor -> appliance pull only; workers do not probe visitor IPs
- [x] No SYN/UDP or "hack-back" tooling in repo

Rule: `architecture.mdc` / `hot-path.mdc`

- [x] Hot path reads in-memory snapshot only (same pattern as CIDR / residential table)
- [x] No `internal/fraud` ML inference on tracker
- [x] Corrupt refresh retains last good snapshot (`TestModeratorIntel_FeedRefreshFailClosed_RetainsSnapshot`)

Rule: `licensing.mdc`

- [x] Feature gated in `sku.yaml` (`moderator_intel_feed`: false starter/pro/pilot, true scale+)

Rule: `anti-slop.mdc` documentation

- [x] `moderator_ip` row in `deploy/vendor/ANTIFRAUD.md`
- [x] No claim of "guaranteed no ban" or daily moderator DB in README without code proof

Rule: `testing.mdc`

- [x] Holdout: feed line matches -> LPM hit; corrupt feed does not empty table
- [x] Campaign flag gate: `TestModeratorIntelHook_campaignFlagGate`
- [x] `go test ./pkg/moderatorintel/... ./internal/ingestion/ -run ModeratorIntel -short`

Rule: `control-plane.mdc` / `ui.mdc`

- [x] Campaign PATCH `moderator_intel_enabled` in DTO + admin checkbox
- [x] `bash scripts/ci/admin_web.sh` after web changes

Rule: product scope

- [x] Flip `gateModeratorIntelFeed` in `enhanced_defense_baseline_audit_test.go`

---

## `review_traffic_policy`

**Priority:** growth

**Gap:** Safe page triggers only on fraud/placement reject or `silent_reject_enabled`. Competitors route review traffic to a white lander before offer logic without looking like a hard block. ~~No per-campaign action matrix.~~ **Shipped** (`gateReviewTrafficPolicy = review_traffic_policy`).

**Current state:** Unified pre-filter policy on `/click` for TLS fingerprint, CIDR, proxy/VPN block, and moderator intel signals. Campaign field `review_traffic_action`: `safe_page` (default), `block`, `passthrough`. CH column `review_routed_event` on `clicks`.

**Target:**

- Campaign policy: `review_traffic_action` = `safe_page` | `block` | `passthrough`.
- Bind to signals: moderator feed, manual CIDR list, proxy/VPN block, TLS fingerprint (IPv rotation remains separate).
- Click: safe view, 403 block, or passthrough to offer; track `review_routed_event` in CH.

**Dependencies:** `moderator_intel_feed` for moderator signal binding.

### Done gates

Rule: `hot-path.mdc`

- [x] Unified policy in `review_traffic_policy.go`; no new PG/Redis on `/click`
- [x] Holdout tests: block, passthrough, default safe page
- [ ] `make test-alloc-gate` on touched ingest files

Rule: `anti-slop.mdc` antifraud semantics

- [x] Defensive routing only; no offensive probing copy in docs
- [x] Safe naming: `review_traffic_*` (no cloaking tokens in code or UI)

Rule: `data-layer.mdc`

- [x] CH migration `00023_review_routed_event_column.sql`
- [x] `clickhouse_store` writes `review_routed_event` on clicks

Rule: `control-plane.mdc`

- [x] Campaign PATCH `review_traffic_action` in DTO + admin select
- [x] `bash scripts/ci/admin_web.sh` after web changes

Rule: product scope

- [x] Flip `gateReviewTrafficPolicy` in `enhanced_defense_baseline_audit_test.go`

---

## `one_click_compose_bootstrap`

**Priority:** growth

**Gap:** BeMob registers and runs traffic immediately. ~~Fresh clone is sources-only (`make gen`, GeoIP, BPF, env, compose).~~ **Shipped** `scripts/install/appliance_bootstrap.sh`.

**Current state:** One command runs deps check, `make gen`, pilot license seed (when vendor key present), optional MaxMind GeoIP pull, compose up, `seed_admin.sh`, `smoke_local.sh`, and prints click URL + admin login + template import curl.

**Target:**

- `scripts/install/appliance_bootstrap.sh`: check deps, run codegen, seed demo license, pull GeoIP, `compose up` health wait.
- Print click URL + admin login + integration template import command.
- Document minimum VPS RAM per profile (`ingest-only` vs `full`).

### Done gates

Rule: `anti-slop.mdc` CI honesty

- [x] Script exits non-zero on failed step (no `exit 0` on failure)
- [x] Not a matryoshka gate - does not re-run `pr_fast.sh` internals twice

Rule: `core.mdc` / `naming.mdc`

- [x] No `===` banners or Unicode dashes in script echo/log (`naming.mdc`)
- [x] Artifacts under `var/` or `bin/` - not repo root clutter

Rule: `development.mdc`

- [x] `docs/DEVELOPMENT.md` and README bootstrap section updated in same commit
- [ ] Documented smoke on clean Ubuntu VM or CI job artifact log attached to PR

Rule: `licensing.mdc`

- [x] Demo license uses `pilot` SKU or documented `license-issue` one-liner - no secrets in repo

---

## `guided_first_campaign_wizard`

**Priority:** growth

**Gap:** Beginners need campaign URL, postback URL, and lander wiring explained. Competitors surface this in UI; we documented in `integration_kit.ts` strings only.

**Current state:** **Shipped** (`gateGuidedFirstCampaignWizard = guided_first_campaign_wizard`). Route `/campaigns/wizard` walks template create, traffic macros, lander URL, test click, and postback dry-run.

**Target:**

- Wizard: traffic source template -> click URL with macros -> lander (hosted or external) -> test click -> test postback.
- Link to Cost Sync and CAPI only as optional steps.

### Done gates

Rule: `ui.mdc`

- [x] Wizard steps call real `/api/v1` routes (`live: true` only with backend)
- [x] `apiConfirmed` on campaign create/update; errors via `ErrorBlock` / inline step errors
- [x] No hardcoded demo KPIs; template import uses existing integration API
- [x] JSDoc on wizard helpers; `typecheck` + `admin_web.sh`

Rule: `anti-slop.mdc`

- [x] No user-visible "fixed" for features still in `competitive_backlog.md` open slugs

Rule: `cold-path.mdc`

- [x] Test dispatch endpoints use existing postback test routes - body limits enforced

---

## `tiktok_cost_sync`

**Priority:** growth

**Gap:** TikTok outbound CAPI existed; daily spend pull was missing for true ROI.

**Current state:** `internal/costsync` supports facebook, google, **tiktok**, taboola, outbrain, tonic_rsoc, system1_rsoc. Adapter: `provider_tiktok.go` (`report/integrated/get`, campaign-level spend, `Access-Token` header). OAuth refresh: `refreshTikTokOAuth` via `TIKTOK_APP_ID` / `TIKTOK_APP_SECRET`. Admin: `tiktok` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchTikTokCosts` with httptest fixture

Rule: `testing.mdc`

- [x] `provider_tiktok_integration_test.go` with `integration:` prefix and recorded fixtures
- [x] `integration_test_slop_gate.sh`

Rule: `ui.mdc`

- [x] Cost sync UI fields match Go DTO; credentials not echoed after save

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated: TikTok in Cost Sync row when shipped

---

## `microsoft_ads_cost_sync`

**Priority:** growth

**Gap:** Microsoft Ads click schema existed; daily spend pull was missing for true ROI vs Keitaro/Binom.

**Current state:** `internal/costsync` adapter `provider_microsoft_ads.go` uses Reporting API v13 async Campaign Performance CSV (`Submit` / `Poll` / download). Credential: `account_id` = advertising account id; `extra_config.customer_id`, `extra_config.developer_token`. OAuth refresh: `refreshMicrosoftOAuth` via `MICROSOFT_ADS_CLIENT_ID` / `MICROSOFT_ADS_CLIENT_SECRET`. Admin: `microsoft_ads` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchMicrosoftAdsCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `microsoft_ads` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `snapchat_cost_sync`

**Priority:** growth

**Gap:** Snapchat click schema existed; daily spend pull was missing for true ROI vs Keitaro/Binom.

**Current state:** `internal/costsync` adapter `provider_snapchat.go` uses Marketing API `GET /v1/adaccounts/{id}/stats` with `breakdown=campaign`, `granularity=DAY`, `fields=spend` (micro-currency). OAuth refresh: `refreshSnapchatOAuth` via `SNAPCHAT_CLIENT_ID` / `SNAPCHAT_CLIENT_SECRET`. Admin: `snapchat` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchSnapchatCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `snapchat` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `linkedin_cost_sync`

**Priority:** growth

**Gap:** LinkedIn click schema existed; daily spend pull was missing for true ROI vs Keitaro/Binom.

**Current state:** `internal/costsync` adapter `provider_linkedin.go` uses Marketing API `GET /rest/adAnalytics` (`pivot=CAMPAIGN`, `timeGranularity=DAILY`, `costInUsd`). OAuth refresh: `refreshLinkedInOAuth` via `LINKEDIN_CLIENT_ID` / `LINKEDIN_CLIENT_SECRET`. Admin: `linkedin` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchLinkedInCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `linkedin` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `pinterest_cost_sync`

**Priority:** growth

**Gap:** Pinterest click schema existed; daily spend pull was missing for true ROI vs Keitaro/Binom.

**Current state:** `internal/costsync` adapter `provider_pinterest.go` lists campaigns then calls `GET /v5/ad_accounts/{id}/campaigns/analytics` (`SPEND_IN_DOLLAR`, `granularity=DAY`). OAuth refresh: `refreshPinterestOAuth` via `PINTEREST_CLIENT_ID` / `PINTEREST_CLIENT_SECRET`. Admin: `pinterest` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchPinterestCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `pinterest` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `trafficstars_cost_sync`

**Priority:** growth

**Gap:** TrafficStars click schema existed; daily spend pull was missing for pop/SSP parity vs Voluum Automizer.

**Current state:** `internal/costsync` adapter `provider_trafficstars.go` calls `POST /v2/campaigns/statistics` with `group_by=campaign`. Offline API key from TrafficStars profile stored in `refresh_token`; worker exchanges via `refreshTrafficStarsOAuth` (`grant_type=refresh_token`). Admin: `trafficstars` in `COST_SYNC_NETWORKS`.

**Target (residual):** none (shipped).

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] OAuth token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchTrafficStarsCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `trafficstars` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `richads_cost_sync`

**Priority:** growth

**Gap:** RichAds click schema existed; advertiser spend API was undocumented publicly but required for tracker parity.

**Current state:** `internal/costsync` adapter `provider_richads.go` calls `GET https://api.richads.com/api/reports/` with `segment=campaign_id` (override via `extra_config.segment`). API key from RichAds account Settings. Admin: `richads` in `COST_SYNC_NETWORKS`.

**Target (residual):** confirm segment field names per account type if API returns empty rows.

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] API key via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchRichAdsCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `richads` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `galaksion_cost_sync`

**Priority:** growth

**Gap:** Galaksion click schema existed; API access is account-specific but Voluum lists full Automizer integration.

**Current state:** `internal/costsync` adapter `provider_galaksion.go` calls `GET /v1/advertiser/statistics` on `ssp2-api.galaksion.com` with `groupBy=["campaign"]`. Auth: stored API token or `POST /v1/auth` with `extra_config` email/password (`account_id` + `password`). Admin: `galaksion` in `COST_SYNC_NETWORKS`.

**Target (residual):** optional `extra_config.base_url` for operator overrides when Galaksion rotates API host.

### Done gates

Rule: `cold-path.mdc`

- [x] Adapter in `internal/costsync` - no tracker import
- [x] Token storage via existing credential pattern; no secrets in API responses
- [x] Unit test `TestFetchGalaksionCosts_Httptest` with httptest fixture

Rule: `ui.mdc`

- [x] Cost sync UI lists `galaksion` network

Rule: `anti-slop.mdc` / README

- [x] README Integrations table updated

---

## `ad_platform_campaign_api`

**Priority:** enterprise_optional

**Gap:** None of the compared trackers fully replace ad platform UIs, but some buyers expect pause/budget sync from tracker.

**Current state:** **Shipped** (`gateAdPlatformCampaignAPI = ad_platform_campaign_api`). Cold-path worker `internal/platformsync` syncs Meta/Google campaign status; admin API supports link CRUD, dry-run pause/resume/budget with idempotency keys. License flag `ad_platform_campaign_api` on enterprise + vendor license SKU.

**Target (minimal):**

- Read-only campaign status sync for Google Ads + Meta (optional).
- Write path limited to pause/resume and daily budget cap with idempotency keys.

**Out of scope v1:** Creative upload, audience editing, policy review.

### Done gates

Rule: `architecture.mdc` / `boundaries.mdc`

- [x] Worker runs in control/processor - **zero** hot-path coupling to `/track` or `/click`
- [x] No per-request external HTTP from tracker

Rule: `cold-path.mdc` / `control-plane.mdc`

- [x] Idempotency keys on mutating calls; audit log row per mutation
- [x] Dry-run mode returns diff without vendor write (`X-Dry-Run: 1`)
- [x] `platformsync/mutation_fault_test.go` idempotency key contract + httptest vendor mocks

Rule: `anti-slop.mdc`

- [x] README still states no full campaign CRUD if only pause/budget ships

Rule: `licensing.mdc`

- [x] Feature flag `ad_platform_campaign_api` in JWT / `sku.yaml` (enterprise + license)

---

## `managed_saas_tenant_plane`

**Priority:** enterprise_optional

**Gap:** BeMob cloud - data on vendor infra, no VPS. We are self-hosted only for on-prem SKUs.

**Current state:** **Shipped phase 1** (`gateManagedSaasTenantPlane = managed_saas_tenant_plane`). JWT `deployment_mode` (`on_prem` | `managed_saas`); SKU `managed_saas`; vendor bootstrap `scripts/install/managed_saas_cell_bootstrap.sh`; compose overlay `docker-compose.managed-saas-cell.yaml` (one isolated project per buyer). `docs/MANAGED_SAAS.md` covers export/residency runbook. Expanded tenant IDOR probe catalog in `tenant_idor_catalog.go`. **Not** a shared multi-tenant control plane.

**Target (future):**

- Multi-tenant control plane with isolated PG schema or DB per tenant.
- Vendor-operated compose per customer or k8s namespace.
- Data residency and export API documented.

**Not a small item.** Treat as separate product line; do not blur with on-prem SKU in docs.

### Done gates

Rule: `architecture.mdc`

- [x] Tenant isolation proof: cross-tenant IDOR tests on `/api/v1` (expanded catalog; holdout in `api_fault_test.go`)
- [x] Hot path still no per-request PG - registry scoped per tenant snapshot (unchanged; one cell = one deployment)

Rule: `data-layer.mdc`

- [x] Migrations documented; shard/outbox semantics per tenant (`docs/MANAGED_SAAS.md` phase-1 cell isolation)

Rule: `licensing.mdc` / `naming.mdc`

- [x] Separate SKU + `deployment_mode` in JWT - on-prem docs unchanged
- [x] No `BidShard` / legacy tokens in new surfaces

Rule: `anti-slop.mdc`

- [x] `SALES.md` distinguishes SaaS vs on-prem pricing - no competitor anchor table

---

## `extended_trial_selfserve`

**Priority:** pricing_gtm

**Gap:** Keitaro 14-day trial; BeMob free/low tier. Pilot was 10 days, 5k RPS, vendor-issued JWT.

**Current state:** **Shipped** (`gateExtendedTrialSelfserve = extended_trial_selfserve`). Pilot SKU `valid_days: 14` (same 5k RPS cap). `GET /api/v1/license/status` exposes `days_to_expiry`, `plan_code`, `max_rps`, `upgrade_plan_code`, `trial_self_serve_url` (`VENDOR_TRIAL_SELF_SERVE_URL`). `/settings/license` shows pilot request + Starter upgrade CTA. Telegram flow: `cmd/vendor-trial-bot` + trial registry.

**Target:**

- Self-serve pilot request flow (Telegram bot already exists: `cmd/vendor-trial-bot`).
- Optional 14-day pilot SKU with same RPS cap.
- Clear upgrade path to starter in admin license screen.

### Done gates

Rule: `licensing.mdc`

- [x] `sku.yaml` + `license-issue` support new duration without breaking HWID bind rules
- [x] Trial registry: no second pilot on same `telegram` / `hwid` (existing `trialregistry` tests)
- [ ] `make license-verify` only if crypto paths touched (not required this change)

Rule: `ui.mdc`

- [x] License screen shows `days_to_expiry` and upgrade CTA from `GET /api/v1/license/status`
- [x] `npm run typecheck` + `license_upgrade` unit tests

Rule: `anti-slop.mdc`

- [x] RPS cap cites JWT `max_rps` (`formatLicenseRpsCap`) - no unlimited hype

---

## `workspace_billing_split`

**Priority:** pricing_gtm

**Gap:** BeMob charges extra for team workspaces. We have `max_tenants` per SKU but no per-workspace invoice split.

**Current state:** **Shipped** (`gateWorkspaceBillingSplit = workspace_billing_split`). Optional `customers.cost_center` per tenant; `PATCH /api/v1/customers/{id}/cost-center`; `GET /api/v1/billing/usage/export` CSV from `billing.usage_daily` (operational meter, not financial truth). Ledger export chunk uses `max_export_chunk_bytes` from license. `CreateCustomer` enforces `max_tenants`. Team overview exposes `cost_center`.

**Dependencies:** None for tracker; reporting + ledger views only.

### Done gates

Rule: `cold-path.mdc`

- [x] Reports read CH/PG aggregates - async workers only
- [x] Export chunk respects `max_export_chunk_bytes` from license (`sku.yaml`)

Rule: `architecture.mdc`

- [x] Financial truth remains Postgres `balance_ledger` - meter is operational/analytics

Rule: `testing.mdc`

- [x] `AssertBudgetInvariant` unchanged on ledger paths
- [x] Tenant-scoped report test with holdout for cross-tenant leak

Rule: `ui.mdc`

- [x] Money formatted via `formatUsdDecimal` / `formatAmountMicro` (`ui.mdc`)

---

## Explicit non-goals (competitor marketing vs our stack)

| Claim in competitor articles | Our stance |
| :--- | :--- |
| "Guaranteed no ban on FB gambling" | Not verifiable; do not document |
| Unlimited RPS on $40 license | We enforce `max_rps` in JWT |
| ML on hot path | Batch sidecars only (`ANTIFRAUD.md`) |
| XDP stops residential fraud | L3/L4 only; docs must stay honest |
| Pay-per-click cloud packages | On-prem unlimited events; price by RPS/hosts |

---

## Doc wiring (after items ship)

When closing a slug, update in the **same commit** as code:

- [ ] `README.md` Features section (if operator-visible)
- [ ] `deploy/vendor/SALES.md` only if SKU limits change
- [ ] `bash scripts/ci/antifraud_doc_gate.sh` when touching vendor antifraud docs
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` on `deploy/vendor/`
- [ ] Cross-reference antifraud slugs in `antifraud_backlog.md` by name, not ticket IDs

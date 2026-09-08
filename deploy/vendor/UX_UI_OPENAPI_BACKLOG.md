# Backlog: close admin UI to OpenAPI v0.6.0

Companion: `UX_SCENARIOS_OPENAPI.md` (347 atomic UX-IDs).

Scope: **operator-facing** `/api/v1` on control plane (`:8188`). Out of scope: tracker ingest, webhooks, static LP URLs opened in a browser tab.

## Definition of done

| Tier | Meaning | Verification |
|------|---------|--------------|
| L3 | One OpenAPI operation = one discoverable admin intent; mutations follow UX-A..N (`frontend-slop.mdc`); `ErrorBlock` on failure; toast only after 2xx | Handler round-trip test (M1+) or Playwright L1+ |
| L4 | L3 on live API (`admin_dev=0`); no silent `Promise.resolve(undefined)` on errors; license/501 surfaces explicit `StubBanner` or `ErrorBlock`, not empty tables | `bash scripts/ci/admin/web.sh` + scoped E2E |

**Not done:** path string in `openapi.d.ts` only; nav hide without server 403 proof; `chart_mock=1` as default demo.

## Inventory (honest snapshot)

| Metric | Value |
|--------|------:|
| OpenAPI HTTP operations | 347 |
| Out of admin UI scope | 9 |
| In frontend scope | 338 |
| SPA routes (`app_routes.tsx`) | 93 |
| `export async function` in `web/src/api/*_api.ts` | ~289 |
| Report catalog keys (`ReportCatalogEntries`) | 20 |
| Report handlers without catalog row | 22 |
| Domains at L3+ (estimate) | ~18 |
| Domains needing L2->L4 work | ~15 |

---

## Out of scope (no SPA work)

| operationId | Tag | Reason |
|-------------|-----|--------|
| `billingCryptoWebhook` | billing | Payment provider webhook |
| `telegramWebhook` | telegram | Bot webhook receiver |
| `opsRumIngest` | ops | RUM beacon POST from browser shell |
| `landersServePreview` | landers | Draft LP URL; open from editor |
| `landersServePublished` | landers | Live LP URL; copy/open from directory |
| `getBrandsById` | brands | Duplicate stub; use `brandsGet` |
| `getCampaignsListFacets` | campaigns | Duplicate stub; use `campaignsListFacets` |
| `getCampaignsMetricsTotals` | campaigns | Duplicate stub; use `campaignsListMetricsTotals` |
| `getCampaignsTargetCountries` | campaigns | Duplicate stub; use `campaignsListTargetCountries` |

---

## Phase 1 -- Reports and analytics navigation (P0)

Largest gap: OpenAPI documents **52** report operations; catalog exposes **20**; **22** handlers have no catalog row; Telegram report paths use nested URLs incompatible with `reports/:key`.

### 1.1 Expand report catalog (backend + admin hub)

Add `ReportCatalogEntries` rows for handlers that already exist (permissions, `DefaultRange`, `ExportFormats` per sibling entries):

| Report key | OpenAPI path / operationId |
|------------|---------------------------|
| `campaign-geo-device` | `reportCampaignGeoDevice` |
| `conversion-type-payout` | `reportConversionTypePayout` |
| `customer-portfolio` | `reportCustomerPortfolio` |
| `data-quality` | `reportDataQuality` |
| `daypart-heatmap` | `reportDaypartHeatmap` |
| `discrepancy-buy-sell` | `reportDiscrepancyBuySell` |
| `edge-parity` | `reportEdgeParity` |
| `geo-roi` | `reportGeoRoi` |
| `keywords` | `reportKeywords` |
| `layer-desync-summary` | `reportLayerDesyncSummary` |
| `ml/feature-spikes` | `reportMlFeatureSpikes` |
| `ml/score-distribution` | `reportMlScoreDistribution` |
| `ml/shadow-delta` | `reportMlShadowDelta` |
| `postback-reconciliation` | `reportPostbackReconciliation` |
| `rtb-geo-device` | `reportRtbGeoDevice` |
| `rtb-no-bid-reasons` | `reportRtbNoBidReasons` |
| `source-quality` | `reportSourceQuality` |
| `spend-velocity` | `reportSpendVelocity` |
| `traffic-sources` | `reportTrafficSources` |
| `true-roi` | `reportTrueRoi` |

`click-log`: already at `/reports/click-log`; add catalog row or cross-link from hub (avoid duplicate UX).

`campaign-stats`: confirm handler vs OpenAPI parity before catalog entry.

**Acceptance:** `/reports` lists all keys; `bash scripts/ci/admin/live_routes.sh` passes with updated catalog count; RBAC rows match `RequiredPermissions`.

### 1.2 Telegram reports routing

OpenAPI: `GET /api/v1/reports/telegram/*` (6) + `POST .../telegram/export`.

| Task | Detail |
|------|--------|
| Nested SPA routes | Add `reports/telegram/*` routes (or `reports/telegram/:segment`) instead of single-segment `:key` |
| Hub section | Telegram category in reports page from catalog or static nav |
| Runner | Map keys to paths in `report_paths.ts` overrides (same pattern as `rtb-overview`) |
| Export | Wire `exportTelegramReport` button on each telegram report where applicable |

**operationIds:** `reportTelegram`, `reportTelegramSummary`, `reportTelegramBots`, `reportTelegramFunnel`, `reportTelegramFraud`, `reportTelegramPremium`, `reportTelegramExport`.

### 1.3 Report jobs and aliases

| Task | operationId |
|------|-------------|
| Verify cancel/download error paths | `reportCancelJob`, `reportDownloadJob` |
| Alias `reportClicks` -> click-log runner (same handler) | Document in catalog as alias only |
| Export-only bulk evidence UX | `fraud-evidence-pack-bulk` in `EXPORT_ONLY_REPORT_KEYS` |

### 1.4 CI

Extend `live_routes.sh` or add `report_catalog_openapi_parity.sh`: every catalog key has OpenAPI GET (or documented export-only); flag handlers without catalog after phase 1.1.

---

## Phase 2 -- OpenAPI contract holes (backend first, then UI) (P0)

UI cannot reach L3 until API exposes the operation.

| Epic slug | Missing OpenAPI / backend | UI work after API |
|-----------|---------------------------|-------------------|
| `offers_crud_complete` | `GET/PATCH/DELETE /api/v1/offers/{id}` | Offers directory: edit row, delete confirm (UX-E), detail |
| `flows_delete` | `DELETE /api/v1/flows/{id}` | Flow detail: delete with 409 if referenced |
| `brands_mutate` | `PATCH/DELETE /api/v1/brands/{id}` | Brands directory row actions |
| `customers_patch_expand` | Rich `PATCH /api/v1/customers/{id}` beyond cost center | Customer detail tabs match DTO |
| `campaign_openapi_stub_cleanup` | Remove or implement stub `get*` operationIds | Drop dead references in `openapi.d.ts` codegen |

**Offers today:** `offersList`, `offersCreate` only -- L2 cap until API grows.

---

## Phase 3 -- Degraded and license-gated surfaces (P1)

Pages exist; contract closure requires **live** behavior, not `StubBanner` as default.

| Epic slug | Domain | Work |
|-----------|--------|------|
| `rtb_license_live_proof` | rtb (10 ops) | Enterprise SKU E2E: deals CRUD, shadow diff, floors apply, validate bid; 403 -> license banner, other errors -> `ErrorBlock` |
| `dashboards_live_no_mock` | dashboards (7 ops) | Role dashboards + campaign dashboard; ban default `chart_mock`; document query flag in ops docs only |
| `campaign_editor_501_audit` | campaigns | Editor shell, save, overview sheet: 501 -> `StubBanner`, not empty form |
| `selfserve_publisher_live` | selfserve, publisher | Portal pages: replace silent `Promise.resolve(undefined)` with `ErrorBlock`; payment intent + statements E2E |

---

## Phase 4 -- Mutation script audit (UX-A..N, EH-*) (P1)

Cross-cutting pass per domain workspace (`use_*_workspace.ts`, editor panels). Priority domains with write paths:

| Priority | Domain | Key operationIds / flows |
|----------|--------|--------------------------|
| P0 | campaigns | `campaignsPatch`, `campaignsPublish`, `campaignsBulkMutate`, `campaignsImport`, wizard session |
| P0 | landers | `landersUpdate`, `landersDelete`, hosted file PUT -> publish |
| P0 | billing | `billingVoidInvoice`, `billingRetryInvoiceDelivery`, export jobs |
| P1 | integrations | `postbacksUpdateConfig`, `costSyncUpsertCredential`, `integrationApplySchema` |
| P1 | team | `teamInviteMember`, budget approve/deny |
| P1 | ops | `opsRetryDlq`, `opsAddBlacklist`, `opsShard0Catchup` |
| P2 | automation family | dry-run before save: `automationDryRunRule`, `trafficOptimizerDryRunRule` |

**Checklist per mutation:** confirm dialog (delete); 409 message; no navigate without response id (UX-C); no toast before await (UX-D); coalesced refresh (UX-G).

---

## Phase 5 -- Domain completion matrix

Status: **current -> target**. Work slug links to phases above.

| Tag | Ops | Route | API module | Now | Target | Work slug |
|-----|----:|-------|------------|-----|--------|-----------|
| campaigns | 51 | yes | campaigns_api | L3 | L4 | `campaigns_e2e_mutations` |
| landers | 12 | yes | landers_api | L3 | L4 | `landers_e2e_hosted_editor` |
| billing | 23 | yes | billing_api | L3 | L4 | `billing_void_retry_e2e` |
| reports | 52 | partial | reports_api | L2 | L4 | `reports_catalog_nav_telegram` |
| ops | 29 | yes | ops_api | L3 | L4 | `ops_dlq_retry_e2e` |
| integrations + cost-sync + postbacks | 23 | yes | integrations_api | L3 | L4 | `integration_schemas_apply_e2e` |
| platform-campaigns | 8 | yes | integrations_api | L2 | L4 | `platform_links_mutations` |
| rtb | 10 | yes | rtb_api | L2 | L4 | `rtb_license_live_proof` |
| dashboards | 7 | yes | dashboards_api | L2 | L4 | `dashboards_live_no_mock` |
| fraud-admin | 7 | yes | fraud_api | L2 | L4 | `fraud_scope_and_presets_ops` |
| telegram | 13 | partial | telegram_api | L2 | L4 | `telegram_reports_and_mint_scope` |
| flows + offers | 6 | yes | flows_api, offers_api | L2 | L3 | `flows_delete_api_and_ui`, `offers_crud_complete` |
| brands | 8 | yes | brands_api | L2 | L3 | `brands_crud_api_and_ui` |
| supply | 12 | yes | supply_api | L3 | L4 | `supply_preview_inline_panel` |
| automation | 6 | yes | automation_api | L3 | L4 | `automation_dry_run_e2e` |
| traffic-optimizer | 6 | yes | traffic_optimizer_api | L3 | L4 | `traffic_optimizer_dry_run_e2e` |
| smart-alerts | 6 | yes | smart_alerts_api | L3 | L4 | `smart_alerts_ack_e2e` |
| margin-guard | 4 | yes | margin_guard_api | L3 | L4 | `margin_guard_policy_e2e` |
| team | 7 | yes | team_api | L3 | L4 | `team_invite_approve_e2e` |
| selfserve | 8 | yes | selfserve_api | L2 | L4 | `selfserve_portal_payment_e2e` |
| publisher | 2 | yes | publisher_api | L2 | L4 | `publisher_dashboard_live` |
| customers | 2 | yes | customers_api | L2 | L3 | `customer_detail_tabs_complete` |
| domains | 6 | yes | domains_api | L3 | L4 | `domains_e2e` |
| settings | 4 | yes | settings_api | L3 | L4 | `settings_bootstrap_apply_e2e` |
| audit | 2 | yes | audit_api | L3 | L4 | `audit_export_csv_e2e` |
| report-schedules | 5 | yes | report_schedules_api | L3 | L4 | `report_schedules_crud_e2e` |
| views | 5 | yes | saved_views_api | L3 | L4 | `saved_views_e2e` |
| disputes | 1 | yes | platform_api | L2 | L3 | `disputes_list_filters` |
| support | 2 | yes | platform_api | L3 | L4 | `support_feedback_submit_e2e` |
| recon | 1 | yes | ops_api | L2 | L3 | `recon_runs_table` |
| auth, meta, public | 9 | yes | auth_api | L4 | L4 | done |
| eula, license | 4 | yes | platform_api | L3 | L4 | `eula_gate_first_login`, `license_setup_polish` |
| consent | 1 | partial | platform_api | L1 | L3 | `consent_capture_flow` |
| command-palette | 5 | global | command_palette_api | L3 | L4 | `palette_open_metric_verify` |

### 5.1 Small gaps (single ops)

| operationId | Work |
|-------------|------|
| `telegramMintClick` | Document as Mini App server-only; optional operator debug panel under Telegram hub |
| `consentRecord` | First-login or settings consent step; wire `postConsent` from `platform_api.ts` |
| `supplyPreviewAdsTxt`, `supplyPreviewSellersJSON` | Inline preview panel (fetch + `ErrorBlock`) instead of raw `href` only |
| `commandPaletteOpen` | Verify fired on Ctrl+K open; metric-only, no user-visible screen |

---

## Phase 6 -- Verification and regression gates (P0 ongoing)

| Gate | Purpose |
|------|---------|
| `bash scripts/ci/admin/web.sh` | typecheck, live routes, UI slop |
| `bash scripts/ci/admin/live_routes.sh` | catalog keys vs SPA after phase 1 |
| Playwright L1+ per epic slug | Mutation round-trip, not heading-only |
| Tier column in `UX_SCENARIOS_OPENAPI.md` | Auto-generated L0-L4 from `web/src` + catalog (future script) |
| `admin_dev=0` smoke | Falsify mock-tier claims (`frontend-slop.mdc` T0 ban) |

Suggested Playwright order: `campaigns` -> `landers` -> `billing` -> `reports/jobs` -> `team` -> `integrations/postbacks`.

---

## Suggested execution order

```
Phase 1.1 catalog expansion (unblocks 20+ report UX-IDs in hub)
  -> Phase 1.2 telegram report routes
  -> Phase 2 API gaps (offers, flows, brands) in parallel with backend
  -> Phase 4 mutation audit on campaigns/landers/billing
  -> Phase 3 RTB/dashboards live proof
  -> Phase 6 E2E expansion per closed epic
```

## Effort rough order (engineering weeks, one squad)

| Phase | Estimate |
|-------|----------|
| 1 Reports + telegram nav | 2-3 |
| 2 API contract gaps | 1-2 (split with backend) |
| 3 License/mock surfaces | 1 |
| 4 Mutation audit (all domains) | 2-3 |
| 5 Remaining L2->L4 polish | 2 |
| 6 CI + E2E | 1+ ongoing |

**Total to L4 on frontend scope (~338 ops):** ~8-12 weeks depending on backend phase 2 and E2E depth.

---

## Tracking

Use epic slugs in PR titles (imperative, concrete surface per `core.mdc`). Example: `Wire report catalog rows for geo-roi and spend-velocity`.

Do not track checkbox state in this file; issue tracker lives outside repo.

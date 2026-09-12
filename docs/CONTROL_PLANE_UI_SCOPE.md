# Control Plane UI scope

Operator decision (2026-03): admin SPA is **Control Plane only** — configure tenants, campaigns, team, platform settings, ops controls, and integration wiring. **Not** in scope for now: reports/BI, dashboards, RTB console, creative hubs, fraud/automation portals, buyer-facing polish, and directory UX chrome.

This document is the source of truth for **KEEP vs FREEZE** and a **backend parity audit** (`/api/v1` on `:8188` vs `web/`). Routes stay registered when frozen; freeze means **no active product work** and **sidebar deprioritization**, not deletion.

Related: `docs/DEVELOPMENT.md` (admin dev), `web/WEB.md` (T1 live tier), `.cursor/rules/frontend-patterns.mdc` (паттерны фронтенда), `.cursor/rules/ui.mdc`, `.cursor/rules/frontend-slop.mdc`, `.cursor/rules/frontend-primitives.mdc`.

---

## KEEP (active Control Plane)

| Surface | UI routes | Backend packages | Why CP |
| :--- | :--- | :--- | :--- |
| Auth / session | `/login`, `/activate`, `/invite/accept` | `internal/control/http`, `platformadmin` session | Bootstrap |
| Customers | `/customers`, `/customers/:id` | `platformadmin`, `billingadmin` (tabs) | Tenant config |
| Campaigns list | `/campaigns` | `internal/campaign` list/facets/metrics/bulk/clone | Operate campaigns |
| Campaign editor | `/campaigns/:id/edit` | `campaign` editor/wizard/publish/fraud/integration | Core CP workflow |
| Team | `/team` | `platformadmin` team | RBAC, invites, budget approvals |
| Settings | `/settings`, `/settings/license` | `platformadmin`, `licensingadmin` | Platform + license |
| Customer billing tabs | customer detail: Balance, Ledger, Statement, Wallet, Tax | `billingadmin` per-customer | Money gates clone/spend |
| Billing ops (minimal) | `/billing/invoices/:id`, void/retry from customer context | `billingadmin` invoice detail | Operator correction |
| Ops | `/ops/*` (home, dlq, blacklist, outbox, shards, domains, ml-model, recon, consent, rum, metrics) | `opsadmin`, `doctor` | Stack control |
| Audit | `/audit` | `opsadmin` audit | Read-only trail |
| Integrations | `/integrations/*` (cost-sync, postbacks, schemas, platform-campaigns, affiliate-presets) | `billingadmin`, `campaign`, `platformadmin` | External wiring |
| Export Hub | `/exports` (legacy: `/reports/jobs`, `/billing/exports`) | `internal/reports`, `reportjob`, `billingadmin` exports | Async BI export; no in-browser report tables in CP |

### KEEP adjuncts (not full frozen hubs)

These use APIs from frozen **sections** but are required for the campaign CP path:

| Need | UI location | APIs | Note |
| :--- | :--- | :--- | :--- |
| Flow pick / routing | campaign editor (`use_campaign_editor_load`, advanced routing) | `GET /api/v1/flows/:id`, list flows | No `/creative` hub required |
| Campaign fraud policy | campaign editor fraud panel | `GET/PATCH /api/v1/campaigns/:id/fraud`, `POST .../fraud/preview` | Not `/fraud` hub |
| Publish / validate | campaign editor actions | `POST .../validate`, `publish`, `publish-check`, `smoke` | |
| Clone | list + editor | `POST .../clone`, `bulk-clone`, `clone-preview` | Needs customer balance (seed or ledger) |
| Integration panel | campaign editor | `GET .../integration-panel`, `integration-health`, schema apply | |
| Postbacks / cost-sync | integrations section + editor deps | `/api/v1/postbacks/*`, `/api/v1/cost-sync/*` | |

---

## KEEP route shell and error contract

Canonical shells: `web/src/shell/directory_page_shell.tsx` (`DirectoryPageShell`), `web/src/shell/customer_tab_shell.tsx` (`CustomerTabShell`), `web/src/shell/editor_page_shell.tsx` (`EditorPageShell`), `web/src/domains/ops/ops_page_shell.tsx` (`OpsPageWithLoad`), `web/src/domains/integrations/integrations_nav.tsx` (`IntegrationsPageWithLoad`), `web/src/shell/page_chrome.tsx` (`PageChrome`), `web/src/shell/auth_page_layout.tsx` (`AuthPageLayout`, standalone auth). **Error phases:** blocking = full-page `ErrorBlock` / `panelError` before content; refresh band = stale snapshot + error above content; panelError/StubBanner = `panelError()` or `StubBanner` (403/501/degraded); mutation actionError = save/export/void errors in alerts slot or inline `ErrorBlock`.

### Auth and bootstrap

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/login` | `web/src/pages/login_page.tsx` | standalone (`AuthPageLayout`) | mutation actionError | `ErrorBlock` on failed login; `PageSkeleton` while meta bootstraps |
| `/activate` | `web/src/pages/activate_page.tsx` | standalone | blocking, mutation actionError | Owner onboarding; success state inline |
| `/invite/accept` | `web/src/pages/invite_accept_page.tsx` | standalone | mutation actionError | Client validation + API `ErrorBlock` |
| `/setup` | `web/src/pages/setup_page.tsx` | standalone redirect | none | Legacy URL; redirects to `/activate` or `/login` |
| `/forbidden` | `web/src/pages/forbidden_page.tsx` | standalone (`AdminErrorPage`) | blocking ErrorBlock | RBAC deny surface |
| License setup (gate) | `web/src/pages/license_setup_page.tsx` | standalone | blocking, mutation actionError | Rendered when `licenseNeedsSetup` in `ProtectedLayout`; not a registered path |

### Customers

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/customers` | `web/src/pages/customers_page.tsx` | `DirectoryPageShell` | blocking, refresh band, mutation alerts | `customers_directory.tsx` |
| `/customers/:id` (header) | `web/src/pages/customer_detail_page.tsx` | section stack (no `PageChrome`) | blocking panelError, refresh band | `customer_detail.tsx`; tab bar below header |
| tab: profile | `customer_detail_profile_tab.tsx` | inline `Card` | mutation panelError | No `CustomerTabShell`; save errors via `panelError` |
| tab: balance | `customer_detail_balance_tab.tsx` | `CustomerTabShell` | blocking, refresh band | |
| tab: ledger | `customer_detail_ledger_tab.tsx` | inline section | blocking ErrorBlock, mutation export ErrorBlock | Pagination + CSV export |
| tab: statement | `customer_detail_statement_tab.tsx` | `CustomerTabShell` | blocking, refresh band | Month picker + load action |
| tab: wallet | `customer_detail_wallet_tab.tsx` | `CustomerTabShell` | blocking, refresh band | Hidden when `payment_enabled` false |
| tab: tax | `customer_detail_tax_tab.tsx` | inline section | blocking ErrorBlock, mutation save ErrorBlock | |
| tab: forecast | `customer_detail_forecast_tab.tsx` | `CustomerTabShell` | blocking, refresh band | CH-degraded via handler |
| tab: payments | `customer_detail_payments_tab.tsx` | `CustomerTabShell` | blocking, refresh band | Gated on `payment_enabled` |

### Campaigns, team, settings

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/campaigns` | `web/src/pages/campaigns_page.tsx` | `DirectoryPageShell` | blocking, refresh band, mutation alerts | `campaigns_directory.tsx` |
| `/campaigns/:id/edit` | `web/src/pages/campaign_editor_page.tsx` | `EditorPageShell` | blocking panelError, mutation panelError | `EditorStatusBanners` for save/publish/validate |
| `/team` | `web/src/pages/team_page.tsx` | `DirectoryPageShell` | blocking, refresh band, mutation actionError | `team_overview.tsx` |
| `/settings` | `web/src/pages/settings_page.tsx` | `PageChrome` | none (nav wrapper) | `SectionNav`; `<Outlet />` for index + access |
| `/settings` (index) | `web/src/pages/settings_license_page.tsx` | embedded in `PageChrome` | blocking ErrorBlock, mutation actionError | `SettingsMain`: platform PATCH + license apply |
| `/settings/license` | (redirect) | — | — | `Navigate` to `/settings` (`app_routes.tsx`) |
| `/settings/access` | `web/src/domains/access/access_roles.tsx` | embedded in `PageChrome` | blocking ErrorBlock, mutation actionError | `AccessRolesView`; YAML validate/apply |

### Ops (`OPS_NAV_ITEMS`, 13 subroutes)

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/ops` | `web/src/pages/ops_page.tsx` | `OpsPageWithLoad` | blocking panelError, refresh band, mutation alerts | `ops_home.tsx` |
| `/ops/health` | `web/src/pages/ops_health_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | `ops_health.tsx` |
| `/ops/sync-errors` | `web/src/pages/ops_sync_errors_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | DLQ alias redirects here |
| `/ops/blacklist` | `web/src/pages/ops_blacklist_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/incidents` | `web/src/pages/ops_incidents_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/outbox` | `web/src/pages/ops_outbox_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/shards` | `web/src/pages/ops_shards_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/ml-model` | `web/src/pages/ops_ml_model_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/domains` | `web/src/pages/ops_domains_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | Rotation subset (not creative hub) |
| `/ops/recon` | `web/src/pages/ops_recon_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/consent` | `web/src/pages/ops_consent_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | |
| `/ops/rum` | `web/src/pages/ops_rum_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | Read-only; POST is telemetry ingress |
| `/ops/metrics` | `web/src/pages/ops_metrics_page.tsx` | `OpsPageWithLoad` | blocking, refresh, alerts | Live metrics stream |

Ops domain wrappers use `opsPanelError` -> `panelError` (`ops_nav.tsx`).

### Integrations

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/integrations` | `web/src/pages/integrations_hub_page.tsx` | `PageChrome` | none | Static `HubLinkGrid`; no list fetch |
| `/integrations/cost-sync` | `web/src/pages/integrations_cost_sync_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | `integrations_cost_sync.tsx` |
| `/integrations/api-keys` | `web/src/pages/integrations_api_keys_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | Service accounts |
| `/integrations/postbacks` | `web/src/pages/integrations_postbacks_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | |
| `/integrations/debugger` | `web/src/pages/integrations_debugger_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | |
| `/integrations/schemas` | `web/src/pages/integrations_schemas_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | |
| `/integrations/platform-campaigns` | `web/src/pages/integrations_platform_campaigns_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | Scope gate uses bare `PageChrome` when customer unset |
| `/integrations/affiliate-presets` | `web/src/pages/integrations_affiliate_presets_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | |
| `/integrations/google-sheets` | `web/src/pages/integrations_google_sheets_page.tsx` | `IntegrationsPageWithLoad` | blocking, refresh, alerts | OAuth connect flow |

Integrations fetch errors use `integrationsPanelError` -> `AdminError` (`integrations_nav.tsx`).

### Exports, audit, billing

| Route | Page file | Shell | Error phases | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `/exports` | `web/src/pages/exports_page.tsx` | `PageChrome` | blocking + refresh catalog ErrorBlock, mutation ErrorBlock, StubBanner | `export_hub.tsx`; catalog fetch via `useResource` + `catalogHasSnapshot`; job failure `export-job-error` + `export-job-refresh` retry |
| `/exports/schedules` | `web/src/pages/export_schedules_page.tsx` | `PageChrome` | blocking ErrorBlock, mutation ErrorBlock | `export_schedules_page_view.tsx` |
| `/audit` | `web/src/pages/audit_page.tsx` | `DirectoryPageShell` | blocking, refresh band, mutation export | Sync CSV via `DirectoryMutationError` |
| `/billing/invoices/:id` | `web/src/pages/invoice_detail_page.tsx` | `PageChrome` | blocking panelError, refresh band, mutation actionError | `billingPanelError`; void/retry actions |

Cross-ref error phase helpers: `web/src/shell/panel_error.tsx`, `web/src/shell/directory_load_state.ts`, `.cursor/rules/frontend-patterns.mdc` (**Pattern 7**).

## E3 implementation status

E3 = KEEP surfaces ship **blocking / refresh / mutation** error phases (table above) plus **L3** Playwright rows (`stubApiRoute` or `page.route` with HTTP 500 stimulus + `ErrorBlock` / `role=alert`; no `.or(unavailable)`). Proof levels: `web/e2e/README.md`.

| ID | Surface | Status | Shell phases | L3 E2E |
| :--- | :--- | :--- | :--- | :--- |
| E3.1 | Auth | **done** | `login_page.tsx`, `activate_page.tsx`, `invite_accept_page.tsx`, `license_setup_page.tsx`, `eula_gate.tsx` | `login.spec.js` bad-password 401 stub |
| E3.2 | Customers | **done** | `DirectoryPageShell` + `CustomerTabShell`; wallet gated on `payment_enabled`; tax field map | `customer_detail_billing.spec.js` balance GET 500 |
| E3.3 | Campaigns list | **done** | `campaigns_directory.tsx` blocking + refresh + mutation alerts; create/wizard mutex | `campaigns_bulk_pause.spec.js` bulk POST 500 |
| E3.4 | Campaign editor | **done** | `EditorPageShell` + `EditorStatusBanners` 501 StubBanner; integration health degraded | `campaign_publish.spec.js` publish-check 501 stub |
| E3.5 | Team | **done** | split overview/roster refresh tokens; `DirectoryFetchError` refresh band | `team_invite.spec.js` POST 400; `team_member_patch.spec.js` PATCH 500 |
| E3.6 | Settings | **done** | `settings_main.tsx` meta ErrorBlock; `access_roles_view.tsx` YAML/409 errors | `settings.spec.js` meta GET 500 |
| E3.7 | Ops | **done** | `OpsPageWithLoad` on all ops subroutes; metrics stream error banner; domains SSL 501 | `ops_console.spec.js` home GET 500 |
| E3.8 | Audit | **done** | `audit_directory.tsx` directory shell + export adjunct | `audit.spec.js` list GET 500 |
| E3.9 | Integrations | **done** | `IntegrationsPageWithLoad`; debugger/affiliate errors in `alerts` slot | `integrations_hub.spec.js` affiliate presets GET 500 |
| E3.10 | Billing | **done** | `invoice_detail.tsx` `billingPanelError` + `BILLING_UNAVAILABLE` StubBanner | `billing_invoice_detail.spec.js` invoice GET 500 |

**Done bar:** row marked **done** when shell phases are wired (table above) **and** at least one L3 Playwright row uses `stubApiRoute` / `stubApiGetError` / `page.route` with 4xx/5xx + `ErrorBlock` (no `.or(unavailable)`). Shared helpers: `web/e2e/helpers.js` (`stubApiRoute`, `stubApiGetError`, `expectErrorBlockVisible`). Gate: `rg -l 'stubApiRoute|status:\\s*500' web/e2e/*.spec.js` count >= 5 (`ui_slop.sh` **E3 KEEP error contract**). Cross-cutting RBAC L3: `permission_gate.spec.js`, `session_perms_nav.spec.js`.

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E3 KEEP error contract**).

## E4 implementation status

E4 = **RBAC deep-link UX**: client route guard (`RoutePermissionGuard` + `route_permissions.ts`) is **UX only**; server `RequirePermission` is **authoritative**. Nav hide (`filterControlPlaneNavGroups`) is not authorization — deep links must hit `RoutePermissionGuard` or API `403` + `ErrorBlock`.

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E4-1 | CP nav `/team` permission | **done** | `control_plane_scope.ts`: `permissionAny: ['team:read', 'campaigns:read']` aligned with `nav_config.ts` |
| E4-2 | `route_permissions.ts` deep-link rules | **done** | `EXTRA_ROUTE_RULES`: `/dashboards`, `/alerts`, `/exports/schedules`; `formatRoutePermissionRequirement` |
| E4-3 | Deep-link deny UX (RB-C6) | **done** | `RoutePermissionGuard` in `app_shell.tsx` -> `ForbiddenPanel` (403 + permission slug); no empty table on denied ops/audit/settings |
| E4-4 | Bootstrap permissions operator warning | **done** | See warning below; `useSessionRefetchOnVisibility` coalesces role grant lag |
| E4-5 | Forbidden page shows permission slug | **done** | `ForbiddenPanel` + `/forbidden?require=`; `AdminErrorPage` `detail` prop |
| E4-6 | `permission_route_audit.spec.js` L3 | **done** | MB deep links `/ops`, `/audit`, `/settings`: UI 403 + permission slug + API 403; @L1 smoke matrix retained |
| E4-7 | `session_perms_nav.spec.js` API 403 | **done** | Before role grant: Audit nav hidden **and** `GET /api/v1/audit` returns 403 (RB-L2) |

**Operator warning (E4-4):** `GET /api/v1/session/bootstrap` `permissions` reflect the signed-in user's **current** grants. Changing role in DB (or via team PATCH) without re-login may lag until visibility refetch (`useSessionRefetchOnVisibility` -> `refreshSession` + `refetchSession`). Nav filter (`filterControlPlaneNavGroups`) is **not** authorization — deep links must hit `RoutePermissionGuard` or API `403` + `ErrorBlock`/`panelError`.

### Deep-link contract (media buyer example)

Seeded role `MB` (`internal/control/http/rbac.go`: `campaigns:read`, `campaigns:write`, `customers:read` only — no `shards:read`, `audit:read`, `settings:read`).

| Deep link | Route guard (`route_permissions.ts`) | SPA UX (no nav) | Primary API | Server |
| :--- | :--- | :--- | :--- | :--- |
| `/ops`, `/ops/*` | `shards:read` | `ForbiddenPanel` (403) in main; no ops table chrome | `GET /api/v1/ops/home` (and ops section GETs) | `403` `FORBIDDEN` |
| `/audit` | `audit:read` | `ForbiddenPanel` (403) | `GET /api/v1/audit` | `403` `FORBIDDEN` |
| `/settings`, `/settings/access` | `settings:read` / `access:read` | `ForbiddenPanel` (403) on index; access tab gated by `PermissionGate` | `GET /api/v1/settings/platform` | `403` `FORBIDDEN` |

Allowed MB deep links (same guard model): `/customers`, `/campaigns`, `/campaigns/:id/edit` (`campaigns:read` or `campaigns:read:masked`), `/team` (`team:read` or `campaigns:read`), `/exports`, `/integrations`, `/alerts`, `/dashboards/*` — page fetch may succeed; mutations still enforced per handler.

**Done bar:** E4 row **done** when route guard + bootstrap permissions are wired (table above) **and** at least one Playwright row in each gate file passes on T1+ with `ADMIN_E2E_MB_EMAIL` / `ADMIN_E2E_MB_PASSWORD`. Server holdout: `TestManagementAPI_RoleMediaBuyerSmokePrimaryGET_holdout` + `rbac_smoke_route_audit_test.go` manifest aligned with `MEDIA_BUYER_SMOKE_PRIMARY_GET_AUDIT`.

Gate references: `web/e2e/permission_gate.spec.js`, `web/e2e/permission_route_audit.spec.js`, `web/e2e/session_perms_nav.spec.js`.

---

## E5 implementation status

E5 = **FREEZE surfaces: honest UX** — no in-browser BI tables on deprioritized routes; deep links show export stub, redirect, or FREEZE banner instead of fake empty data.

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E5-1 | `/alerts` FREEZE + shell | **done** | `smart_alerts_page_view.tsx`: FREEZE `StubBanner`, `DirectoryPageShell` (rules fetch), `CustomerScopeGate`; EH-ST1 (`undefined` not `Promise.resolve([])`) |
| E5-2 | Dashboards FREEZE banner | **done** | `dashboard_page_view.tsx`: `dashboard-freeze-banner` when not `?chart_mock=1`; `DashboardStaleBanner` unchanged |
| E5-3 | Phase D report stubs | **done** | E2: `ReportExportStubRoute` on `reports/*` (`report_export_stub_page.tsx`) |
| E5-4 | Unknown `/reports/*` stub | **done** | No blind `/exports` redirect; `catalogUnknown` banner via `export_only_report_stub_message.ts` |
| E5-5 | Frozen hub E2E | **done** | `web/e2e/freeze_redirect.spec.js` (`@freeze`): `/fraud/*`, `/rtb/*` -> `/exports`; `/creative` -> `/campaigns`; click-log stays on stub |
| E5-6 | Click log export stub E2E | **done** | `web/e2e/click_log.spec.js` (`@L1`): export mode banner, no `GET /api/v1/reports/click-log` on load |

**Done bar:** FREEZE deep link shows redirect, `ExportOnlyReportStub`, or FREEZE `StubBanner` — never silent empty table pretending live BI. Removed legacy fraud/rtb/creative table specs documented in `web/e2e/README.md` (**@freeze removed**).

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E5 FREEZE contract**); Playwright: `freeze_redirect.spec.js`, `click_log.spec.js`.

---

## E6 implementation status

E6 = **request fan-out, coalescing, state ownership** (RF-*, R1, RP-4/5). Canonical helpers: `web/src/lib/coalesced_user_action.ts`, `web/src/hooks/use_coalesced_refresh_token.ts`.

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E6-1 | RF-1 facets off list refresh | **done** | `use_campaigns_page_list.ts`: facets deps `[customerId]` only; list/metrics on `refreshToken` / `listAuxFetchKey` |
| E6-2 | RF-8 team split refresh tokens | **done** | `use_team_page_workspace.ts`: `overviewRefreshToken` vs `rosterRefreshToken`; coalesced bumps |
| E6-3 | RF-6 invoice ledger id change | **done** | `use_invoice_detail_page_workspace.ts`: id effect resets cursor only; `resetLedger()` on void |
| E6-4 | RF-4 customers combobox cache | **done** | `fetchCustomersComboboxCached`; paginated directory uses `listCustomers` directly only |
| E6-5 | R1 coalescing audit | **done** | Dashboard, export schedules, smart alerts, Google Sheets, ops shards use `useCoalescedBumpRefresh` |
| E6-6 | RP-4 command palette lanes | **done** | `use_command_palette.ts`: `catalogError`/`catalogLoading` vs `searchError`/`searchLoading` |
| E6-7 | RP-5 palette row component | **done** | `command_palette_row.tsx`; no duplicate `CommandItem` blocks |
| E6-8 | Column prefs ownership | **done** | Default-only prefs in `use_campaigns_directory_workspace.ts`; holdout in `campaign_list_columns.test.ts` |
| E6-9 | UX-I create/wizard mutex copy | **done** | `campaigns_list_toolbar.tsx` `overlaysBusy` hint |

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E6 coalescing contract**).

---

## E7 implementation status

E7 = **shell / layout standardization** (VL-08, VL-10, VL-11, `DirectoryPageShell`, `CustomerTabShell`, `EditorPageShell`).

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E7-1 | `DirectoryPageShell` adoption | **done** | `integrations_hub.tsx`, `export_hub.tsx` (catalog fetch), `settings_main.tsx`, `export_schedules_page_view.tsx` (schedules list); team/alerts/customers/campaigns/audit already wired |
| E7-2 | `CustomerTabShell` audit | **done** | `customer_detail_ledger_tab.tsx`, `customer_detail_tax_tab.tsx`; balance/statement/wallet/forecast/payments already wired |
| E7-3 | `EditorPageShell` audit | **done** | `campaign_editor.tsx` uses `EditorPageShell` with campaign load `fetchState` |
| E7-4 | VL-11 filter row height | **done** | KEEP directories use default `Input`/`SelectTrigger`/`Button` (no manual `h-7`/`h-8`/`h-9`); `ui_slop.sh` control-height gate |
| E7-5 | VL-10 scroll chain (`min-h-0`) | **done** | `sheet.tsx` `SheetBody` scroll chain restored; campaign import sheet uses `SheetHeader` + `SheetBody` |
| E7-6 | VL-08 `justify-between` purge | **done** | `customer_detail_forecast_tab.tsx` CardHeader uses explicit grid columns |
| E7-7 | `admin_spacing` token adoption | **done** | No raw `gap-5+` under `web/src/domains`; existing `ui_slop.sh` gap gate |

`DirectoryPageShell` blocking/refresh footers: `blockingErrorFooter` / `refreshErrorFooter` for export catalog retry (`export_hub.tsx`).

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E7 shell layout contract**).

---

## E8 implementation status

E8 = **E2E proof upgrade** — L3 error contracts on KEEP routes; frozen spec drift removed; CI tier honesty.

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E8-A1 | customers list L3 | **done** | `customers_list.spec.js` GET 500 + `Could not load customers` |
| E8-A2 | campaigns list L3 | **done** | `campaigns_filters.spec.js` GET 500 |
| E8-A3 | campaign editor save L3 | **done** | `campaign_editor.spec.js` PATCH 400 field message |
| E8-A4 | export hub failed job L3 | **done** | `export_hub.spec.js` failed job poll + `export_hub.spec.js` L1 catalog |
| E8-A5 | ops home MB 403 L3 | **done** | `permission_route_audit.spec.js` (pre-E8) |
| E8-A6 | postbacks save L3 | **done** | `integrations_postbacks_save.spec.js` PUT 400 |
| E8-A7 | settings platform 409 L3 | **done** | `settings.spec.js` PATCH 409 |
| E8-A8 | team invite 400 L3 | **done** | `team_invite.spec.js` tagged `@L3` |
| E8-B1 | fraud `*.spec.js` removal | **done** | README **@freeze removed** table; no fraud specs in tree |
| E8-B2 | rtb specs | **done** | Redirect covered by `freeze_redirect.spec.js` |
| E8-B3 | creative flows | **done** | `freeze_redirect.spec.js` `/creative` -> `/campaigns` |
| E8-B4 | automation rules | **done** | Redirect pattern in `freeze_redirect.spec.js` |
| E8-B5 | portals smoke | **done** | Documented removed; redirect to `/exports` |
| E8-B6 | dashboards specs | **done** | `dashboards_adops.spec.js` L1 only; FREEZE banner in UI |
| E8-B7 | L1s `.or(empty)` ban | **done** | `helpers.js` FE2 comment; `ui_slop.sh` bans `table.or(empty)` |
| E8-C1 | pr_fast smoke honesty | **done** | `web.sh` + `web_e2e_smoke.sh` label mount smoke, not wiring proof |
| E8-C2 | Nightly KEEP matrix | **done** | `web_e2e_keep_proof.sh` (16 specs); runs before full matrix in nightly |

Existing L3 specs tagged `@L3`: `audit`, `billing_invoice_detail`, `customer_detail_billing`, `login`, `ops_console`, `permission_gate`, `permission_route_audit`, `session_perms_nav`, `settings`, `team_member_patch`.

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E8 E2E proof contract**); Playwright: `bash scripts/ci/admin/web_e2e_keep_proof.sh` (live `:8188`).

---

## E9 implementation status

E9 = **backend-ahead KEEP adjuncts** — wire live cold APIs that had client stubs or docs-only references; no full fraud/disputes hubs and no core-nav additions.

| ID | Task | Status | Notes |
| :--- | :--- | :--- | :--- |
| E9-1 | `GET /api/v1/fraud/probe-clusters/{id}` | **done** | `campaign_fraud_signals_panel.tsx` in campaign editor ops strip; `audit:read` gate |
| E9-2 | `GET /api/v1/fraud/crowd-waves/{campaign_id}` | **done** | Same panel; on-demand load for current campaign |
| E9-3 | `PATCH /api/v1/ops/fraud/presets/{name}` | **done** | `ops_fraud_preset_panel.tsx` on ops home; list via `GET /api/v1/fraud/presets`; patch requires `shards:write` |
| E9-4 | `fraud_api.ts` extension | **done** | `getProbeClusterSummary`, `getCrowdWaveSummary`; `CrowdWaveSummary` manual type until OpenAPI adds schema |
| E9-5 | `GET /api/v1/disputes` | **done** | `/disputes` paginated directory; `customers:read`; not in core nav |
| E9-6 | `POST /api/v1/support/feedback` | **done** | `/support/feedback` form + meta; auth-only route; not in core nav |

Gate: `bash scripts/ci/admin/ui_slop.sh` (**E9 backend-ahead contract**).

---

## FREEZE (no active work)

| Surface | UI routes | Backend | Freeze reason |
| :--- | :--- | :--- | :--- |
| Reports hub + runners | `/reports`, `/reports/*`, `/report-schedules`, `/views` | `internal/reports`, `reportjob` (~40+ CH reports) | BI, not CP |
| Dashboards | `/dashboards/:role`, `/dashboards/campaign/:id` | `internal/dashboardadmin` | Charts/KPIs |
| RTB | `/rtb/*` | `internal/rtbadmin` | Auction ops, license-gated |
| Creative hub | `/creative`, `/flows`, `/landers`, `/offers`, `/brands`, `/supply`, `/domains` | `internal/flow`, `brand`, `supply`, `platformadmin/domains` | Content studio |
| Fraud hub | `/fraud/*` | `internal/fraudadmin` | Policy tools beyond per-campaign panel |
| Automation | `/automation/*`, traffic-optimizer, smart-alerts, margin-guard | respective packages | Optimization product |
| Portals | `/portals`, `/selfserve`, `/publisher/*`, `/telegram/*`, `/forecast/campaign` | `campaign/selfserve`, `dashboardadmin`, `telegram` | Buyer/publisher portals |
| Billing directory | `/billing` (invoice list hub), `/billing/exports` | `billingadmin` summary/list/exports | Operator BI; keep per-customer + invoice detail |
| Directory polish | column resize, export UX, overview sheet depth, campaign dashboard links from list | n/a | UX chrome |
| Disputes / support (low) | `/disputes`, `/support/feedback` | `billingadmin`, `platformadmin` | Wired (E9); secondary deep-link only; no core nav |

**Remove from default nav (done):** Dashboard, Billing (hub), Reports, RTB, Fraud, Creative, Automation, Portals trimmed from Control Plane sidebar. **Core nav:** Customers, Campaigns, Team, Settings. **Operations:** Ops, Audit, Integrations, **Exports** (`/exports`). Implemented in `web/src/lib/control_plane_scope.ts` + `web/src/lib/tracker_nav.ts`.

---

## Export Hub

**Purpose:** BI and report surfaces do not render heavy browser tables in Control Plane mode. The operator configures filters and runs async export jobs (or sync CSV when the result set is small). Data stays on the server / ClickHouse; the SPA submits jobs and polls for download URLs.

**Canonical route:** `/exports` in **Operations** nav -- unified catalog, job form, and job status/history. Legacy deep-link aliases remain registered: `/reports/jobs`, `/billing/exports` redirect to `/exports` with query preserved (`ReportJobsRoute`, `BillingExportsRoute` in `web/src/app_routes.tsx` / `web/src/shell/export_hub_legacy_redirect.tsx`). Typed catalog keys under `/reports/*` render `ReportExportStubRoute` (`web/src/pages/report_export_stub_page.tsx`); unknown keys fall through to `/exports?report_key=...`.

### KEEP tables vs export-only

| Pattern | Routes / surfaces | CP behavior |
| :--- | :--- | :--- |
| **KEEP tables** | Customers, campaigns list, ops, audit list, integrations | Operational CRUD; **Export** is an adjunct toolbar action (CSV where handler supports sync export) |
| **Export-only** | Customer-scoped reports (`CustomerReportPage` keys), telegram / ml / evidence / campaign-stats runners, fraud reasons table view | No `DirectoryTable` body in CP mode; `ExportOnlyReportStub` + link to Export Hub with pre-filled `report_key` |

Export-only catalog keys today include typed sets in `web/src/lib/report_paths.ts` (`TYPED_CUSTOMER_REPORT_KEYS`, `TYPED_TELEGRAM_REPORT_KEYS`, `TYPED_ML_REPORT_KEYS`, `TYPED_CAMPAIGN_STATS_REPORT_KEYS`, `TYPED_EVIDENCE_PACK_REPORT_KEYS`, `TYPED_EXPORT_ONLY_REPORT_KEYS`, `TYPED_RTB_REPORT_KEYS`, `TYPED_OPS_REPORT_KEYS`). Reference implementation: `web/src/pages/report_export_stub_page.tsx` + `web/src/shell/export_only_report_stub.tsx` (`buildReportJobsHref` -> `buildExportHubHref`).

### Job flow

```
customer_id + from + to + report_key + format
  -> POST report job (or billing export when ledger-scoped)
  -> poll GET job status
  -> download via job download URL (or billing export download)
```

Reuse existing APIs:

| Job kind | API | UI today |
| :--- | :--- | :--- |
| Report catalog jobs | `POST/GET /api/v1/reports/jobs`, download on completed job | `/exports` Export Hub (`export_hub.tsx`, `use_export_hub_page_workspace.ts`) |
| Billing ledger exports | `POST/GET /api/v1/billing/exports`, download | `/exports` Export Hub (same form; kind `billing`) |
| Audit CSV (sync) | `GET /api/v1/audit/export` | `/exports` Export Hub (kind `audit`; direct download) |

Export Hub form maps one UX to both backends by `report_key` / export kind (catalog entry metadata).

### Catalog sources

| Source | Entries |
| :--- | :--- |
| `GET /api/v1/reports/catalog` | Dynamic report keys, titles, async-capable flags |
| Static CP entries (Phase C) | Audit CSV (`GET /api/v1/audit/export`), campaign list CSV export, billing ledger export |

Pre-fill deep links via `buildExportHubHref` in `web/src/lib/export_hub_paths.ts` (`report_key`, `customer_id`, `from`, `to`, `format`, `job_id`, `kind`). `buildReportJobsHref` in `web/src/lib/report_paths.ts` delegates to `buildExportHubHref` with `kind=report`.

### Phase rollout

| Phase | Deliverable | Status |
| :--- | :--- | :--- |
| **A** | This spec + cross-ref in `frontend-primitives.mdc` | **done** (this section) |
| **B** | Primitives catalog (`ExportOnlyReportStub`, date range / searchable select shell roles) | **done** |
| **C** | MVP `/exports` route, Operations nav, legacy alias redirects | **done** (`exports_page.tsx`, `control_plane_scope.ts`, `ReportJobsRoute`, `BillingExportsRoute`) |
| **D** | Stub report pages when `CONTROL_PLANE_NAV_ENABLED`; frozen report routes show export stub instead of table | **done** (`report_export_stub_page.tsx`, `ReportExportStubRoute` on `reports/*`) |

Cross-ref: `web/src/lib/export_hub_paths.ts` (`buildExportHubHref`), `web/src/lib/report_paths.ts` (`buildReportJobsHref`, `reportRequiresCustomerScope`), `web/src/pages/report_export_stub_page.tsx`, `.cursor/rules/frontend-primitives.mdc` (**Export Hub UI**).

---

## Backend vs frontend parity

Legend: **match** = UI calls live handler; **partial** = conditional (CH, license, nil dep, 501); **backend-only** = API exists, no admin page; **ui-only** = page exists, backend edge case.

### Auth, session, platform

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `POST /api/v1/auth/{login,logout,refresh}` | live | login flow | match |
| `GET /api/v1/auth/me`, `GET /api/v1/session/bootstrap` | live | app shell | match |
| `GET /api/v1/meta` | live | `MetaProvider` | match |
| `GET/PATCH /api/v1/settings/platform` | live | `/settings` | match |
| `GET /api/v1/license/status`, `POST .../apply` | live | `/settings/license` | match |
| `POST /api/v1/public/{activate,invite/accept}` | live | onboarding pages | match |
| `POST /api/v1/ops/roles/reload` | live | ops home action | match |

### Customers

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/customers`, `GET/PATCH .../{id}` | live | directory + profile tab | match |
| `GET .../balance`, `ledger`, `balance/export` | live | customer tabs | match |
| `GET .../billing/statement`, `forecast` | live / partial (CH) | customer tabs | partial if CH down |
| `GET .../wallet`, `payments` | live | wallet tab (gated on `payment_enabled`) | match |
| `GET/PUT .../tax-profile` | live | tax tab | match |
| `GET /api/v1/disputes` | live | `/disputes` directory (E9) | match (secondary; no nav) |

### Campaigns

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/campaigns`, `list-facets`, metrics batch | live | `/campaigns` | match |
| `POST bulk`, `bulk-clone`, `clone` | live | list actions | match (balance prerequisite) |
| `GET/PATCH /api/v1/campaigns/{id}` | live | editor | match |
| Editor bundle: `editor`, `validate`, `publish*`, `diff`, `macro-preview` | live | editor | match |
| `GET/PATCH .../fraud`, `POST .../fraud/preview` | live | fraud panel | match |
| `GET .../integration-panel`, `integration-health` | live | integration panel | match |
| Wizard `GET/POST /api/v1/campaigns/wizard/session` | live | wizard | match |
| Migrate/import ` /api/v1/campaigns/migrate/*`, `/import/*` | live | editor import flows | match |
| `GET /api/v1/campaigns/{id}/stats` | live (CH) | no dedicated page | backend-only (use reports freeze) |
| `GET /api/v1/dashboards/campaign/{id}` | live / 501 if reader nil | `/dashboards/campaign/:id` | match but **FREEZE** |
| `POST /api/v1/selfserve/campaigns` (create from list) | live | campaigns list create | match (selfserve API used from admin) |

### Team

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/team/overview`, `members`, `budget-approvals` | live | `/team` | match |
| `POST members`, `PATCH .../{id}`, approve/deny | live | team actions | match |

### Billing

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Per-customer ledger/statement/wallet | live | customer tabs | **KEEP** |
| `GET /api/v1/billing/invoices/{id}`, void, deliveries | live | `/billing/invoices/:id` | **KEEP** |
| `GET /api/v1/billing/summary`, `invoices` list, `invariant`, exports | live | `/billing`, `/billing/exports` | match but **FREEZE** hub |
| `POST /api/v1/billing/crypto/webhook` | live | no UI (ingress) | backend-only |
| `GET /api/v1/cost-sync/*` | live | integrations cost-sync | **KEEP** |

### Ops

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/ops/home`, dlq, outbox, shards, blacklist | live | ops subpages | match |
| `GET /api/v1/audit`, `audit/export` | live | `/audit` | match |
| `GET /api/v1/ops/domains/*` | live | `/ops/domains` | match |
| `GET /api/v1/recon/runs` | live | `/ops/recon` | match |
| `GET/POST /api/v1/ops/rum` | live | `/ops/rum` (GET only) | partial (no ingest UI; POST is telemetry ingress) |
| `GET /api/v1/ops/dashboard/{metrics,stream}` | live | `/ops/metrics` | match |
| `PATCH /api/v1/ops/fraud/presets/{name}` | live | ops home preset panel (E9) | match |

### Integrations

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `/api/v1/postbacks/*` | live | integrations postbacks | match |
| `/api/v1/integration/schemas`, templates, affiliate-presets | live | integrations pages | match |
| `/api/v1/platform-campaigns/*` | live | platform-campaigns page | match |
| `/api/v1/domains/*` (admin) | live / SSL 501 if script missing | `/domains` (creative) | match but **FREEZE** hub; ops has rotation subset |

### Fraud (hub vs campaign)

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Campaign `.../fraud` | live | editor panel | **KEEP** |
| `GET /api/v1/fraud/labels`, overrides, presets, corpus, decisions | live | `/fraud/*` | match but **FREEZE** |
| `GET /api/v1/fraud/probe-clusters/{id}` | live | campaign editor ops strip (E9) | match |
| `GET /api/v1/fraud/crowd-waves/{campaign_id}` | live | campaign editor ops strip (E9) | match |
| `POST /api/v1/fraud/wasm-attest/dry-run` | partial (501 disabled) | no page | backend-only |

### Reports, dashboards, RTB (all FREEZE)

| Domain | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Reports catalog + ~40 report keys | live / CH partial | `/reports/*` | match (freeze) |
| Report jobs, schedules, saved views | live | jobs + portals pages | match (freeze) |
| Dashboards buyer/adops/cfo/... | live | `/dashboards/:role` | match (freeze) |
| RTB deals, shadow, floors, validate | live / license | `/rtb/*` | match (freeze) |

### Creative, automation, portals (FREEZE)

| Domain | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Flows, landers, offers, brands, supply | live | `/creative/*` | match (freeze); editor uses flows API only |
| Automation, traffic-optimizer, smart-alerts, margin-guard | live | `/automation/*` | match (freeze) |
| Selfserve, publisher, telegram | live | `/portals/*` | match (freeze) |

---

## Gap summary

### Backend ahead of UI (KEEP backlog candidates)

| API | Suggested CP surface |
| :--- | :--- |
| _(none — E9 wired)_ | Probe/crowd-wave, ops preset patch, disputes list, support feedback |

### UI ahead of or wider than CP scope

| UI | Issue |
| :--- | :--- |
| Campaign list → `/dashboards/campaign/:id` | Links into FREEZE dashboards |
| Campaign list metrics columns | Depends on CH batch metrics (degraded without CH) |
| `/campaigns` create via selfserve API | Works but blurs admin vs portal boundary |
| Buyer dashboard in Core nav | Should move to FREEZE / remove from nav |

### Conditional / degraded (not stub catalog)

| Condition | Symptom | Correct UI behavior |
| :--- | :--- | :--- |
| ClickHouse down | reports + some forecasts/metrics 503 or `stale` | `ErrorBlock`; no fake rows |
| `CampaignDashboard` reader nil | `GET /dashboards/campaign/{id}` 501 | StubBanner on frozen page |
| Domain SSL script missing | domains SSL 501 | StubBanner (integrations/creative) |
| RTB not in license | 403 on `/api/v1/rtb/*` | StubBanner on frozen RTB |
| Stripe disabled | `payment_enabled: false` | Hide wallet/payments tab |
| Insufficient customer balance | clone 400 | Show server error; seed `db seed-clone-balance` in dev |

`stubRouteCatalog` in `internal/controlplane/admin_static.go` is **empty** — no global 501 stubs. Failures are real handler responses.

---

## Inventory scale

| Layer | Count | Source |
| :--- | :--- | :--- |
| Catalogued `/api/v1` routes | ~362 | `internal/controlplane/routecatalog/catalog.go` |
| Admin SPA top-level nav items | 14 | `web/src/lib/nav_config.ts` |
| Report runner routes | ~40 | `web/src/app_routes.tsx` |
| Ops subroutes | 13 | `web/src/domains/ops/ops_nav.tsx` |

OpenAPI parity: `api/openapi/` + `internal/openapi/documented_routes.go`. Regenerate TS: `make openapi-types`.

---

## Verification (scope audit)

```bash
# Control healthy (T1)
curl -sf http://127.0.0.1:8188/health

# Session bootstrap (auth)
curl -sf -c /tmp/aed.cookie -b /tmp/aed.cookie \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@test.local","password":"Password123!"}' \
  http://127.0.0.1:8188/api/v1/auth/login
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/session/bootstrap | head -c 200

# KEEP samples
curl -sf -b /tmp/aed.cookie 'http://127.0.0.1:8188/api/v1/customers?limit=1'
curl -sf -b /tmp/aed.cookie 'http://127.0.0.1:8188/api/v1/campaigns?limit=1'
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/team/overview
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/settings/platform
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/ops/home

# FREEZE samples (still live on backend)
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/reports/catalog
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/dashboards/buyer
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/rtb/deals
```

Admin static gates (when touching `web/`): `npm run typecheck`, `bash scripts/ci/admin/web.sh`.

---

## Implementation status

| Item | Status |
| :--- | :--- |
| This scope doc | **done** |
| Export Hub spec (Phase A) | **done** (this doc) |
| Nav trim (remove FREEZE from sidebar) | **done** (`control_plane_scope.ts`, `nav_config.ts`, `tracker_nav.ts`) |
| `ControlPlaneFrozenGate` on outlet | **done** |
| Unstyled HTML frames (`control_plane_*_frame.tsx`) | **done** |
| Remove campaign list → dashboard links | **done** |
| Link from `DEVELOPMENT.md` | done |
| Export Hub route `/exports` (Phase C) | **done** |
| Export-only report stubs (Phase D) | **done** |

# Admin UI E2E proof levels

Playwright specs under this directory are classified below. Do not cite smoke or nightly runs as handler wiring proof without naming the level. Proof level is **not** repeated in spec file comments; this README is canonical.

| Level | Name | Minimum shape | Proves |
| :--- | :--- | :--- | :--- |
| **L0** | Route shell | `loginAsAdmin` + route heading | Auth cookie and route mount |
| **L1s** | Soft read | `table.or(empty).or(stub\|forbidden)` | Deprecated; do not add new specs at this level |
| **L1** | Read contract | `waitForResponse` GET `/api/v1/...` 200 + JSON field + DOM bound to payload | Live list/read path |
| **L2** | Mutation contract | `waitForResponse` POST/PATCH 2xx + follow-up GET or visible row change | Create/update path |
| **L3** | Error contract | Stimulus 4xx/5xx + `ErrorBlock` copy; no `.or(unavailable)` | Fail-closed UI |

## Tier

| Tier | When | Falsify |
| :--- | :--- | :--- |
| **T1** | Live `/api/v1` on `:8188` (`ADMIN_E2E_BASE_URL` default) | `curl -sf :8188/health`; `skipUnlessIntegrationReady` |
| **T2** | Embedded `web/dist` from control binary | Same-origin live API |

There is no in-browser mock API tier. Chart preview (`?chart_mock=1`) is dashboard-only and does not prove list/report handlers.

## Shared helpers

`web/e2e/helpers.js`:

| Helper | Use |
| :--- | :--- |
| `gotoLive`, `gotoLiveAwaitGet`, `gotoLiveAwaitResponse` | T1 navigation + GET wait |
| `isApiGet`, `isApiPost`, `isApiPatch` | Response predicates |
| `expectApiListBoundToDom` | L1 list/empty bind (FE2-safe `.or()` on row surfaces only) |
| `ADMIN_SMOKE_ROUTE_READS`, `OPS_SECTION_READS` | Smoke matrix route tables |
| `applyCustomerScopeIfPrompted`, `ensureCustomerScopeLoaded` | Customer-scoped pages |
| `openFirstCampaignEditor`, `fetchFirstCampaignId` | Campaign editor flows |
| `integrationRunToken`, `randomHex32`, `integration*Reason/Domain/Suffix` | Unique integration fixtures |

## L1 read specs (Phase 4)

| File | Primary GET |
| :--- | :--- |
| `smoke_matrix.spec.js` | Per-route via `ADMIN_SMOKE_ROUTE_READS` + `/fraud` hub |
| `customers_list.spec.js` | `/api/v1/customers` |
| `billing_filters.spec.js` | `/api/v1/billing/invoices` (+ filtered refetch) |
| `billing_invoice_detail.spec.js` | Invoice + ledger + deliveries on detail open |
| `audit.spec.js` | `/api/v1/audit` |
| `campaign_editor.spec.js` | `/api/v1/campaigns/:id` on editor open |
| `campaigns_filters.spec.js` | `/api/v1/campaigns` list bind; status GET; pacing overlay; Report link |
| `campaign_single_clone.spec.js` | POST `/api/v1/campaigns/{id}/clone` (L2 API) |
| `click_log.spec.js` | Export-only stub on `/reports/click-log`; no GET `/api/v1/reports/click-log` on load; Export Hub link with `report_key=click-log` |
| `export_hub.spec.js` | `/api/v1/reports/catalog` on `/exports` |
| `freeze_redirect.spec.js` | `@freeze` frozen deep links: fraud/rtb -> `/exports`, creative -> `/campaigns`, click-log stays on export stub |
| `customer_detail_billing.spec.js` | `/api/v1/customers`, `/api/v1/customers/:id` |
| `dashboards_adops.spec.js` | `/api/v1/dashboards/adops` campaigns[] DOM bind |
| `integrations_hub.spec.js` | Per integrations section GET |
| `integrations_postbacks_health.spec.js` | `/api/v1/postbacks/health` on Health tab |
| `ops_blacklist.spec.js` | `/api/v1/ops/blacklist` |
| `ops_console.spec.js` | `/api/v1/ops/home` + `OPS_SECTION_READS` |
| `ops_dlq.spec.js` | `/api/v1/ops/dlq/inbox` |
| `settings.spec.js` | `/api/v1/settings/platform` |
| `sidebar.spec.js` | `/api/v1/session` after login |
| `team.spec.js` | `/api/v1/team/overview` |
| `team_member_patch.spec.js` | `/api/v1/team/members` |
| `command_palette.spec.js` | `/api/v1/command-palette/routes` |

## @freeze removed specs (Control Plane scope)

Frozen routes no longer mount in-browser table runners. Do not recreate full fraud/rtb/creative table specs; use `freeze_redirect.spec.js` for redirect honesty and `click_log.spec.js` for export-only report stubs.

| Former spec | Replacement |
| :--- | :--- |
| `automation_rules.spec.js` | **@freeze removed** — route redirects to `/exports` (`freeze_redirect.spec.js` pattern) |
| `creative_flows.spec.js` | **@freeze removed** — `/creative` -> `/campaigns` in `freeze_redirect.spec.js` |
| `dashboards.spec.js` | **@freeze removed** — buyer dashboard frozen |
| `edge_parity.spec.js` | **@freeze removed** — ops report frozen |
| `flow_stream.spec.js` | **@freeze removed** — flows frozen |
| `fraud_decision.spec.js` | **@freeze removed** |
| `fraud_integrations.spec.js` | **@freeze removed** |
| `fraud_labels.spec.js` | **@freeze removed** |
| `fraud_presets.spec.js` | **@freeze removed** — `/fraud/presets` -> `/exports` in `freeze_redirect.spec.js` |
| `fraud_presets_patch.spec.js` | **@freeze removed** |
| `portals_smoke.spec.js` | **@freeze removed** — portals redirect to `/exports` |
| `reports.spec.js` | **@freeze removed** — `/reports` index redirects to `/exports` |
| `rtb.spec.js` | **@freeze removed** |
| `rtb_deals.spec.js` | **@freeze removed** — `/rtb/deals` -> `/exports` in `freeze_redirect.spec.js` |

Run frozen redirect coverage:

```bash
cd web/e2e && npx playwright test --grep @freeze
```

## L2 write specs (`@write` tag)

Mutation specs carry `{ tag: '@write' }` on each test:

| File | Mutation |
| :--- | :--- |
| `campaigns_bulk_pause.spec.js` | POST `/api/v1/campaigns/bulk` |
| `campaign_bulk_clone.spec.js` | POST `/api/v1/campaigns/bulk-clone` |
| `campaign_single_clone.spec.js` | POST `/api/v1/campaigns/{id}/clone` |
| `campaign_publish.spec.js` | POST validate + GET publish-check |
| `fraud_labels_write.spec.js` | POST `/api/v1/fraud/labels` |
| `fraud_overrides_write.spec.js` | POST `/api/v1/fraud/overrides` |
| `integrations_actions.spec.js` | cost-sync run, platform sync, DLQ retry |
| `settings_platform_patch.spec.js` | PATCH `/api/v1/settings/platform` |
| `settings_apply.spec.js` | POST `/api/v1/settings/platform/apply` |
| `team_invite.spec.js` | POST `/api/v1/team/members` |

## L3 error specs (`@L3` tag)

| File | Stimulus | Assert |
| :--- | :--- | :--- |
| `customers_list.spec.js` | GET `/api/v1/customers` 500 | `Could not load customers`; no empty table |
| `campaigns_filters.spec.js` | GET `/api/v1/campaigns` 500 | `Could not load campaigns`; no empty table |
| `campaign_editor.spec.js` | PATCH `/api/v1/campaigns/:id` 400 | `Could not save campaign` + field message |
| `export_hub.spec.js` | GET `/api/v1/reports/jobs/:id` failed | `Export failed` on poll |
| `settings.spec.js` | GET `/api/v1/meta` 500; PATCH platform 409 | metadata + save `ErrorBlock` |
| `audit.spec.js` | GET `/api/v1/audit` 500 | `Could not load audit log`; no export toolbar |
| `ops_console.spec.js` | GET `/api/v1/ops/home` 500 | `Could not load ops snapshot` |
| `permission_route_audit.spec.js` | MB deep-link `/ops`, `/audit`, `/settings` | 403 UI + API |
| `integrations_postbacks_save.spec.js` | PUT `/api/v1/postbacks/config/:id` 400 | save error alert |
| `customer_detail_billing.spec.js` | GET balance 500 | `Could not load balance` |
| `billing_invoice_detail.spec.js` | GET invoice 500 | `Could not load invoice` |
| `team_invite.spec.js` | POST `/api/v1/team/members` 400 | mutation alert + message |
| `team_member_patch.spec.js` | PATCH `/api/v1/team/members/:id` 500 | mutation alert |
| `login.spec.js` | POST `/api/v1/auth/login` 401 | login `ErrorBlock` |
| `session_perms_nav.spec.js` | API 403 before grant | nav hidden |
| `permission_gate.spec.js` | forbidden route | `ForbiddenPanel` |

## Retained L0 only (not wiring proof)

| File | Pattern |
| :--- | :--- |
| `onboarding.spec.js` | Install/login/setup headings |
| `login.spec.js` | Auth shell |
| `bootstrap.spec.js`, `settings_bootstrap.spec.js` | Boot paths |
| `fraud_hub.spec.js` | Hub card links (no list GET on `/fraud`) |
| `ui_buttons_audit.spec.js` | DOM chrome audit |
| `ui_clickability_audit.spec.js` | Cross-route enabled-button clickability (`trial` click + hit-test) |
| `keyboard_navigation.spec.js` | A11y shell |
| `campaign_flows.spec.js` | Dialog open only |
| `campaign_import_validate.spec.js` | Control visibility |
| `report_jobs.spec.js` | Form shell (job GET only when `job_id` set) |

## Commands

```bash
# T1 stack required for integration specs
bash scripts/dev/aed-admin up

# Curated mount smoke (not L1/L3 wiring proof)
ADMIN_WEB_E2E_SMOKE=1 bash scripts/ci/admin/web.sh

# KEEP L1+L3 proof per core route (nightly tier 1)
bash scripts/ci/admin/web_e2e_keep_proof.sh

# Full matrix (nightly tier 2)
ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh

# L3 error subset
cd web/e2e && npx playwright test --grep @L3

# L2 mutation subset (@write tag)
cd web && npx playwright test --grep @write

# L1 read subset (examples)
cd web/e2e && npx playwright test \
  smoke_matrix.spec.js customers_list.spec.js billing_filters.spec.js \
  audit.spec.js ops_console.spec.js click_log.spec.js

# @freeze redirect subset
cd web/e2e && npx playwright test freeze_redirect.spec.js
```

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
| `automation_rules.spec.js` | `/api/v1/automation/rules` |
| `campaign_editor.spec.js` | `/api/v1/campaigns/:id` on editor open |
| `campaigns_filters.spec.js` | `/api/v1/campaigns` list bind |
| `click_log.spec.js` | `/api/v1/reports/click-log` on Apply |
| `creative_flows.spec.js` | `/api/v1/flows`, `/api/v1/landers` |
| `flow_stream.spec.js` | `/api/v1/flows/validate` 400; visual weight ErrorBlock; L2 flow -> campaign -> `/click` redirect |
| `customer_detail_billing.spec.js` | `/api/v1/customers`, `/api/v1/customers/:id` |
| `dashboards.spec.js` | `/api/v1/dashboards/buyer` on Apply |
| `fraud_labels.spec.js` | `/api/v1/fraud/labels` |
| `fraud_presets.spec.js` | `/api/v1/fraud/presets` |
| `integrations_hub.spec.js` | Per integrations section GET |
| `integrations_postbacks_health.spec.js` | `/api/v1/postbacks/health` on Health tab |
| `ops_blacklist.spec.js` | `/api/v1/ops/blacklist` |
| `ops_console.spec.js` | `/api/v1/ops/home` + `OPS_SECTION_READS` |
| `ops_dlq.spec.js` | `/api/v1/ops/dlq/inbox` |
| `portals_smoke.spec.js` | Self-serve / publisher reads |
| `reports.spec.js` | `/api/v1/reports/catalog` |
| `rtb_deals.spec.js` | `/api/v1/rtb/deals` |
| `settings.spec.js` | `/api/v1/settings/platform` |
| `sidebar.spec.js` | `/api/v1/session` after login |
| `team.spec.js` | `/api/v1/team/overview` |
| `team_member_patch.spec.js` | `/api/v1/team/members` |
| `fraud_presets_patch.spec.js` | `/api/v1/fraud/presets` |
| `command_palette.spec.js` | `/api/v1/command-palette/routes` |

## L2 write specs (`@write` tag)

Mutation specs carry `{ tag: '@write' }` on each test:

| File | Mutation |
| :--- | :--- |
| `campaigns_bulk_pause.spec.js` | POST `/api/v1/campaigns/bulk` |
| `campaign_publish.spec.js` | POST validate + GET publish-check |
| `fraud_labels_write.spec.js` | POST `/api/v1/fraud/labels` |
| `fraud_overrides_write.spec.js` | POST `/api/v1/fraud/overrides` |
| `integrations_actions.spec.js` | cost-sync run, platform sync, DLQ retry |
| `settings_patch.spec.js` | PATCH `/api/v1/settings/platform` |
| `settings_apply.spec.js` | POST `/api/v1/settings/platform/apply` |
| `team_invite.spec.js` | POST `/api/v1/team/members` |

## L3 error specs

| File | Notes |
| :--- | :--- |
| `ops_forbidden_mb.spec.js` | MB role GET `/api/v1/ops/home` 403 |

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

# Curated smoke bundle
ADMIN_WEB_E2E_SMOKE=1 bash scripts/ci/admin/web.sh

# Full matrix (nightly)
ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh

# L2 mutation subset (@write tag)
cd web && npx playwright test --grep @write

# L1 read subset (examples)
cd web/e2e && npx playwright test \
  smoke_matrix.spec.js customers_list.spec.js billing_filters.spec.js \
  audit.spec.js ops_console.spec.js fraud_labels.spec.js
```

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
| **T0** | `admin_dev=1` mock intercept | `admin_dev=0` on `:8188` |
| **T1** | Live `/api/v1` on `:8188`, mock banner absent | `assertLiveApiMode(page)` after `?admin_dev=0` |

## Upgraded specs (L1+)

Shared helpers: `web/e2e/helpers.js` (`gotoLive`, `isApiGet`, `isApiPost`, `isApiPatch`, `expectApiListBoundToDom`, `applyCustomerScopeIfPrompted`, `fetchSessionCustomerId`, `ensureCustomerScopeLoaded`, `ensureIntegrationCustomerScope`, `openFirstCampaignEditor`, `fetchFirstCampaignId`, `randomHex32`, `integrationRunToken`, `integrationTeamInviteEmail`, `integrationSettingsTrackingDomain`, `integrationFraudLabelReason`, `integrationCampaignValidateSuffix`).

| File | Level | Notes |
| :--- | :--- | :--- |
| `campaigns_filters.spec.js` | **L1** + L0 filter tests | `waitForResponse` on GET `/api/v1/campaigns` |
| `campaigns_bulk_pause.spec.js` | **L2** | POST `/api/v1/campaigns/bulk` + refreshed list status |
| `ops_forbidden_mb.spec.js` | **L3** | Mock MB role; GET `/api/v1/ops/home` 403 + blocking error |
| `customers_list.spec.js` | **L1** | GET `/api/v1/customers` + row/empty bind |
| `automation_rules.spec.js` | **L1** | GET `/api/v1/automation/rules` |
| `creative_flows.spec.js` | **L1** | GET `/api/v1/flows`, `/api/v1/landers` |
| `integrations_hub.spec.js` | **L0** nav + **L1** section reads | Per-section GET on integrations routes |
| `portals_smoke.spec.js` | **L1** | Self-serve invoices, report schedules; publisher **L1/L3** |
| `rtb_deals.spec.js` | **L1** | GET `/api/v1/rtb/deals` when licensed |
| `team_member_patch.spec.js` | **L1** | GET `/api/v1/team/members` |
| `fraud_presets_patch.spec.js` | **L1** | GET `/api/v1/fraud/presets` |
| `command_palette.spec.js` | **L1** | GET `/api/v1/command-palette/routes` + option row |
| `fraud_labels_write.spec.js` | **L2** | POST `/api/v1/fraud/labels` + list refresh + row |
| `fraud_overrides_write.spec.js` | **L2** | POST `/api/v1/fraud/overrides` + success copy |
| `settings_patch.spec.js` | **L2** | PATCH `/api/v1/settings/platform` + status |
| `settings_apply.spec.js` | **L2** | POST `/api/v1/settings/platform/apply` + `written_path` |
| `team_invite.spec.js` | **L2** | POST `/api/v1/team/members` 201 + roster row |
| `campaign_publish.spec.js` | **L2** | POST `/validate` + GET `/publish-check` on editor gate |
| `integrations_actions.spec.js` | **L2** | cost-sync run 202, platform sync 204, DLQ retry 200 |

## Retained L0 only (not wiring proof)

| File | Level | Pattern |
| :--- | :--- | :--- |
| `smoke_matrix.spec.js` | L0 | Heading-only route matrix |
| `ops_console.spec.js` | L0 | Ops section nav; no 404 text |
| `ui_buttons_audit.spec.js` | L0 | DOM chrome audit only |

## Commands

```bash
# Curated smoke bundle (mix of L0/L1; includes L1 campaigns read)
ADMIN_WEB_E2E_SMOKE=1 bash scripts/ci/admin/web.sh

# Full matrix (nightly)
ADMIN_WEB_E2E_NIGHTLY=1 bash scripts/ci/admin/web_e2e_nightly.sh

# Single upgraded spec
cd web/e2e && npx playwright test customers_list.spec.js

# L1+ directory read specs
cd web/e2e && npx playwright test \
  customers_list.spec.js automation_rules.spec.js creative_flows.spec.js \
  integrations_hub.spec.js portals_smoke.spec.js rtb_deals.spec.js \
  team_member_patch.spec.js fraud_presets_patch.spec.js command_palette.spec.js

# L2 write/mutation specs
cd web/e2e && npx playwright test \
  fraud_labels_write.spec.js fraud_overrides_write.spec.js \
  settings_patch.spec.js settings_apply.spec.js team_invite.spec.js \
  campaign_publish.spec.js integrations_actions.spec.js
```

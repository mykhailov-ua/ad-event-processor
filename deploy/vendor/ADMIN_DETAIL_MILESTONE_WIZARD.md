# ADMIN_DETAIL_MILESTONE_WIZARD

Detail/editor: /campaigns/wizard.

**Status:** DRAFT  
**Slug:** `admin_detail_wizard`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS`  
**Blocks:** —  
**Pattern:** wizard  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| TS-only form fields | Invented PATCH keys | OpenAPI Patch*Request only |
| Toast before 2xx | Optimistic save | apiConfirmed after response |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client KPI math | Sum sub-resources in browser | Stats endpoints only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per wizard |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Saved" toast before PATCH 2xx
- Form fields not on Go PATCH struct

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/campaigns/wizard` — Server-driven CampaignWizardSession; step payload per OpenAPI; commit creates campaign
- Legacy: `first_campaign_wizard_page.tsx`

### Out of scope

- Client-derived KPIs
- TS-only PATCH fields

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| POST action | CampaignWizardSessionRequest | action (create\|update\|commit) |
| POST session_id | CampaignWizardSessionRequest | session_id |
| POST customer_id | CampaignWizardSessionRequest | customer_id |
| POST template_key | CampaignWizardSessionRequest | template_key enum |
| POST step | CampaignWizardSessionRequest | step enum |
| POST payload | CampaignWizardSessionRequest | payload per step schema |
| POST idempotency_key | CampaignWizardSessionRequest | idempotency_key |
| POST publish | CampaignWizardSessionRequest | publish boolean on commit |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | First campaign wizard | static |
| steps | WizardStepBar | campaign → integration → flow → budget | URL step param |
| tab_campaign | WizardBasicsStep | name, traffic_template_id, click_query_params | CampaignWizardTrafficSourceStep |
| tab_integration | WizardIntegrationStep | integration_schema, affiliate_network, tracking_domain | CampaignWizardIntegrationTemplateStep |
| tab_flow | WizardFlowStep | flow_name, lander, offer URLs | CampaignWizardFlowSkeletonStep |
| tab_budget | WizardBudgetStep | budget_limit_micro, publish | CampaignWizardBudgetStep + commit |
| tab_templates | WizardTemplatesPick | template list for customer | GET selfserve/templates |
| platform_ctx | WizardPlatformHints | click URL from settings/platform | GET settings/platform |
| error | ErrorBlock | Session POST failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| campaign | Basics | POST wizard/session action=create\|update step=traffic_source |
| integration | Integration | POST wizard/session step=integration_template |
| flow | Flow | POST wizard/session step=flow_skeleton |
| budget | Budget | POST wizard/session step=budget action=commit |
| templates | Templates | GET /api/v1/selfserve/templates?customer_id= |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/wizard` |
| Nav group | Commercial → Campaigns → Wizard |
| Permission | campaigns:write |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/campaigns/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| step | — | campaign | Wizard step in URL |
| customer_id | customer_id |  | required UUID for templates + session |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |
| Tabs | Separate GET per tab | No client merge of tab payloads |
| GET detail | Full DTO from handler | Render as returned |
| PATCH | Fields ⊆ OpenAPI Patch*Request | apiConfirmed after 2xx |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/first_campaign_wizard_page.tsx | Thin compose; tabs route optional |
| web/src/ui/campaigns/campaigns_detail.tsx | Detail shell |
| web/src/ui/campaigns/*.module.css | Section CSS modules |
| web/src/helpers/campaigns_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../models/traffic_source_templates.js, ../helpers/first_campaign_wizard_model.js.

**Legacy page:** `first_campaign_wizard_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Success toast before 2xx | apiConfirmed on commit POST |
| TS-only session fields | CampaignWizardSessionRequest keys only |
| Client wizard state only | Every step via POST session |
| Skip customer_id | Require UUID before create |
| models/traffic_source_templates | No models/ import in new wizard |
| Wrapper stack | No WizardChrome |
| Flex page root | CSS Grid step layout |
| Platform URL fiction | click template from GET settings/platform |
| Commit without server | Show API 400 errors |
| Piecemeal wizard page | Atomic ui/campaigns/wizard/ |
| ghost_* / silent_reject | Not on wizard — fraud is post-create |
| Double navigation | step param single source of truth |
| apiConfirmed per step | Toast only on commit 2xx unless API returns early |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm CampaignWizardSessionRequest + step payloads | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/campaigns_wizard_api.ts | getSession, postSession, listTemplates | Compiles |
| 4 | web/src/ui/campaigns/wizard/* | Step panels per 4.1 | surface gate |
| 5 | web/src/pages/first_campaign_wizard_page.tsx | Compose; step URL | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /campaigns/wizard | Resolves |
| 7 | Directory toolbar | Wizard link on campaigns directory | nav |
| 8 | Legacy cleanup | Remove models/traffic_source_templates import | rg models/traffic web/src/pages/first_campaign_wizard_page.tsx |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Session POST | Manual: advance each step | POST wizard/session per step |
| apiConfirmed commit | Manual: finish wizard | toast after commit 2xx |
| customer_id guard | Manual: missing customer_id | validation error UI |
| Templates fetch | Manual: templates tab | GET selfserve/templates |
| No models import | rg 'models/traffic' web/src/ui/campaigns/wizard/ | no matches |
| ErrorBlock | Manual: block session POST | ErrorBlock visible |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `first_campaign_wizard_page.tsx` replaced or delegates to `ui/campaigns/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/campaigns/`, helpers, and page compose in one slice; no half migration.


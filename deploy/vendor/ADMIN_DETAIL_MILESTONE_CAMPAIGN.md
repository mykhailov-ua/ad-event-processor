# ADMIN_DETAIL_MILESTONE_CAMPAIGN

Detail/editor: /campaigns/:id.

**Status:** DRAFT  
**Slug:** `admin_detail_campaign`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS`  
**Blocks:** `admin_detail_telegram` (embedded tab); dedicated route optional  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| ghost_* UI | Legacy fraud section copy | silent_reject_enabled label from API; ANTIFRAUD.md semantics |
| silent_reject lie | Toggle implies per-IP ghosting | Campaign flag only; no ML auto-enable |
| TS-only form fields | Extra PATCH keys in campaign_admin_api | PatchCampaignRequest keys only |
| Toast before 2xx | Optimistic save/publish | apiConfirmed after PATCH/POST 2xx |
| Client stats | useMemo on hourly[] | GET .../stats and dashboards/campaign only |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Buyer model import | estimateDeliveryPct in overview | Remove models/buyer.js from route |
| Demo KPIs | Hardcoded pacing % | stats endpoint fields only |
| Flex header | page-header flex row | PageChrome grid section |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_detail_pattern |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy campaign_detail_page.tsx | Monolith 1400+ lines | ui/campaigns/ tab modules |
| Skip masked tab guard | Show fraud to masked | allowedTabIds holdout |
| Skip publish confirm | One-click publish | Confirm modal |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- ghost_* labels in fraud tab
- PATCH silent_reject_enabled from ML action UI
- estimateDeliveryPct or buyer model in new UI
- "Saved" toast before PATCH 2xx

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/campaigns/:id` under `web/src/ui/campaigns/`
- Zone TabBar + URL `?tab=`; masked role tab allowlist
- Config/fraud/postback forms ⊆ OpenAPI PATCH/PUT bodies
- Stats/events from handler only; FreshnessBadge when envelope provides
- Publish actions with confirm modal + apiConfirmed

### Out of scope

- Client pacing/KPI math (estimateDeliveryPct, buyer model)
- Auto-enable silent_reject from ML UI
- ghost_* column headers or labels
- Per-IP ghosting toggle marketing copy

**Not on page (explicit):** `Client health merge from buyer dashboard`, `Flex page-header layout`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| platform-campaigns | May lack dedicated sync API on tab | Link to ADMIN_INTEGRATIONS_MILESTONE_PLATFORM_CAMPAIGNS or StubBanner |
| Masked tabs | campaigns:read:masked hides monetization tabs | allowedTabIds parity with legacy |

### Stop triggers (revert slice; do not compensate)

- Auto-enable silent_reject from ML UI — forbidden
- Client pacing math for KPIs
- Wrapper stack CampaignDetailChrome

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| PATCH name | PatchCampaignRequest | name |
| PATCH status | PatchCampaignRequest | status |
| PATCH budget_limit | PatchCampaignRequest | budget_limit |
| PATCH budget_limit_micro | PatchCampaignRequest | budget_limit_micro |
| PATCH pacing_mode | PatchCampaignRequest | pacing_mode |
| PATCH daily_budget_micro | PatchCampaignRequest | daily_budget_micro |
| PATCH timezone | PatchCampaignRequest | timezone |
| PATCH freq_limit | PatchCampaignRequest | freq_limit |
| PATCH freq_window | PatchCampaignRequest | freq_window |
| PATCH target_countries | PatchCampaignRequest | target_countries |
| PATCH target_url | PatchCampaignRequest | target_url |
| PATCH safe_page_url | PatchCampaignRequest | safe_page_url |
| PATCH safe_page_enabled | PatchCampaignRequest | safe_page_enabled |
| PATCH dmr_enabled | PatchCampaignRequest | dmr_enabled |
| PATCH cidr_block_enabled | PatchCampaignRequest | cidr_block_enabled |
| PATCH proxy_vpn_block_enabled | PatchCampaignRequest | proxy_vpn_block_enabled |
| PATCH moderator_intel_enabled | PatchCampaignRequest | moderator_intel_enabled |
| PATCH review_traffic_action | PatchCampaignRequest | review_traffic_action |
| PATCH tls_fingerprint_block_enabled | PatchCampaignRequest | tls_fingerprint_block_enabled |
| PATCH conn_type_policy | PatchCampaignRequest | conn_type_policy |
| PATCH link_signing_enabled | PatchCampaignRequest | link_signing_enabled |
| PATCH link_signing_ttl_sec | PatchCampaignRequest | link_signing_ttl_sec |
| PATCH attestation_enabled | PatchCampaignRequest | attestation_enabled |
| PATCH attestation_mode | PatchCampaignRequest | attestation_mode |
| PATCH attestation_ttl_sec | PatchCampaignRequest | attestation_ttl_sec |
| PATCH referrer_filter | PatchCampaignRequest | referrer_filter |
| PATCH click_delivery | PatchCampaignRequest | click_delivery |
| PATCH proxy_upstream_url | PatchCampaignRequest | proxy_upstream_url |
| PATCH proxy_rewrite_assets | PatchCampaignRequest | proxy_rewrite_assets |
| PATCH start_at | PatchCampaignRequest | start_at |
| PATCH end_at | PatchCampaignRequest | end_at |
| PATCH daypart_hours | PatchCampaignRequest | daypart_hours |
| PATCH flow_id | PatchCampaignRequest | flow_id |
| PATCH brand_id | PatchCampaignRequest | brand_id |
| PATCH ingress_cost_config | PatchCampaignRequest | ingress_cost_config |
| PATCH traffic_template_id | PatchCampaignRequest | traffic_template_id |
| PATCH click_query_params | PatchCampaignRequest | click_query_params |
| Fraud PATCH preset | PatchCampaignFraudRequest | preset |
| Fraud PATCH fraud_threshold_pass | PatchCampaignFraudRequest | fraud_threshold_pass |
| Fraud PATCH fraud_threshold_suspect | PatchCampaignFraudRequest | fraud_threshold_suspect |
| Fraud PATCH fraud_threshold_ivt | PatchCampaignFraudRequest | fraud_threshold_ivt |
| Fraud PATCH fraud_threshold_block | PatchCampaignFraudRequest | fraud_threshold_block |
| Fraud PATCH silent_reject_enabled | PatchCampaignFraudRequest | silent_reject_enabled |
| Fraud PATCH behavior_flags | PatchCampaignFraudRequest | behavior_flags |
| Fraud PATCH canvas_retest_enabled | PatchCampaignFraudRequest | canvas_retest_enabled |
| Fraud PATCH cgnat_ip_policy_enabled | PatchCampaignFraudRequest | cgnat_ip_policy_enabled |
| Fraud PATCH accept_lang_geo_enabled | PatchCampaignFraudRequest | accept_lang_geo_enabled |
| Fraud PATCH json_serialization_enabled | PatchCampaignFraudRequest | json_serialization_enabled |
| Fraud PATCH conversion_reject_rules | PatchCampaignFraudRequest | conversion_reject_rules |
| PUT provider | UpdatePostbackConfigRequest | provider |
| PUT url_template | UpdatePostbackConfigRequest | url_template |
| PUT api_token | UpdatePostbackConfigRequest | api_token |
| PUT target_event | UpdatePostbackConfigRequest | target_event |
| PUT test_event_code | UpdatePostbackConfigRequest | test_event_code |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Breadcrumb Campaigns → customer → name | GET campaign + customer context |
| chrome | PageChrome | campaign.name; status_label/status_tone chip | GET campaign |
| toolbar | CampaignToolbar | Save, Publish, Pause/Resume, Delete, Export | PATCH/POST per x-permissions |
| zones | ZoneTabBar | Performance \| Setup \| Monetization | URL ?tab= + zone map |
| freshness | FreshnessBadge | stale, freshness_label on stats envelope | GET .../stats |
| tab_overview | CampaignOverview | KPI strip; pacing health | GET .../stats; dashboards/campaign |
| tab_stats | CampaignStatsPanel | Hourly chart; time range in URL | GET .../stats |
| tab_events | CampaignEventsGrid | Paginated events | GET .../events |
| tab_config | CampaignConfigForm | PatchCampaignRequest fields | GET/PATCH campaign |
| tab_tracking | CampaignTrackingPanel | click URL template, macros, health rows | integration-health; macro-preview |
| tab_filters | CampaignFiltersPanel | placement blocks; geo/device toggles on PATCH | PATCH + placement-blocks |
| tab_fraud | CampaignFraudPanel | PatchCampaignFraudRequest; no ghost_* labels | GET/PATCH .../fraud |
| tab_postbacks | CampaignPostbackPanel | UpdatePostbackConfigRequest | postbacks/config/{campaign_id} |
| tab_margin | CampaignMarginPanel | margin_breach, window KPIs | GET .../margin |
| publish | PublishActions | publish-check, smoke, validate | POST publish-* endpoints |
| error | ErrorBlock | Tab fetch/PATCH failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| overview | Overview | GET /api/v1/campaigns/{id}; GET .../stats; GET .../dashboards/campaign/{id} |
| stats | Statistics | GET /api/v1/campaigns/{id}/stats |
| events | Event log | GET /api/v1/campaigns/{id}/events?limit&offset |
| config | Configuration | GET/PATCH /api/v1/campaigns/{id} |
| tracking | Integration | GET campaign; POST .../macro-preview; integration-health |
| filters | Filters | PATCH filter fields; POST .../placement-blocks |
| creative | Creative | GET campaign; brands/creatives sub-routes |
| telegram | Telegram | Link /campaigns/:id/telegram or embedded section |
| postbacks | CAPI & Postbacks | GET/PUT /api/v1/postbacks/config/{campaign_id}; POST .../test |
| fraud | Fraud | GET/PATCH /api/v1/campaigns/{id}/fraud |
| margin | Margin guard | GET /api/v1/campaigns/{id}/margin |
| platform | Platform sync | GET integration panel; platform-campaigns when API exists |


**Not on page (explicit):** Client health merge from buyer dashboard, Flex page-header layout.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/:id` |
| Nav group | Commercial → Campaigns → Detail |
| Permission | campaigns:read or campaigns:read:masked |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

ContextBar → PageChrome (+Freshness) → Toolbar → ZoneTabBar → [tab panel] → ErrorBlock

| Invariant | Value |
| :--- | :--- |
| Page grid | CSS Grid; no flex page root |
| Zone tabs | Masked role: overview, stats, config only (legacy allowedTabIds) |
| Config form | PATCH sends only dirty keys present in PatchCampaignRequest |

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
| tab | — | overview | Zone + tab; masked role hides Setup/Monetization tabs |
| events_offset | offset | 0 | events tab pagination |

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
| web/src/pages/campaign_detail_page.tsx | Thin compose; tabs route optional |
| web/src/ui/campaigns/campaigns_detail.tsx | Detail shell |
| web/src/ui/campaigns/*.module.css | Section CSS modules |
| web/src/helpers/campaigns_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../components/campaign_*, ../models/buyer.js, client health merge.

**Legacy page:** `campaign_detail_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| ML ghosting UI | Decisions per ANTIFRAUD.md; not per-IP toggle lie |
| ghost_* headers | Use silent_reject_* canonical naming |
| silent_reject auto-enable | ML action must not PATCH flag silently |
| Client stats | stats endpoint only; no hourly reduce |
| Toast before publish 2xx | apiConfirmed on publish-check/smoke |
| Toast before PATCH 2xx | apiConfirmed on config/fraud save |
| TS-only PATCH fields | Cross-check PatchCampaignRequest |
| Buyer health merge | No parallel buyer dashboard fetch for KPIs |
| Client daypart parse | Send daypart_hours array per OpenAPI |
| Wrapper stack | No CampaignDetailChrome |
| Flex page root | CSS Grid sections |
| Masked tab leak | Hide setup/monetization tabs when masked |
| Double status chip | One StatusBadge from status_label |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Inventory PatchCampaignRequest + PatchCampaignFraudRequest | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/campaigns_api.ts | get/patchCampaign, fraud, postback helpers | Compiles |
| 4 | web/src/ui/campaigns/campaign_detail.tsx | Zone TabBar + shell | surface gate |
| 5 | web/src/ui/campaigns/campaign_*_panel.tsx | One module per tab | Form fields = contract_rows |
| 6 | web/src/pages/campaign_detail_page.tsx | Compose; ?tab=; masked allowlist | <= ~120 lines |
| 7 | web/src/app_routes.tsx | Route /campaigns/:id | Resolves |
| 8 | Legacy cleanup | Drop models/buyer.js + components/campaign_* imports | rg estimateDeliveryPct web/src/pages/campaign_detail_page.tsx |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'ghost_' web/src/ui/campaigns/
rg 'silent_reject' api/openapi/components/schemas/campaign.yaml
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No ghost labels | rg 'ghost_' web/src/ui/campaigns/ | no matches |
| PATCH parity | Manual: save config field | network PATCH body keys in OpenAPI |
| apiConfirmed save | Manual: save fraud tab | toast after 2xx only |
| Masked tabs | Manual: masked session | fraud/postbacks tabs hidden |
| Publish confirm | Manual: publish | confirm modal shown |
| Stats source | rg 'estimateDeliveryPct' web/src/ui/campaigns/ | no matches |
| Error tab | Manual: block GET .../stats | ErrorBlock on stats tab |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `campaign_detail_page.tsx` replaced or delegates to `ui/campaigns/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/campaigns/`, helpers, and page compose in one slice; no half migration.


# ADMIN_DASHBOARD_MILESTONE_BUYER

Dashboard: Buyer.

**Status:** DRAFT  
**Slug:** `admin_dashboard_buyer`  
**Depends on:** admin_page_chrome  
**Blocks:** —  
**Pattern:** dashboard  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Demo KPIs | Hardcoded numbers in TSX | Every value from API DTO field |
| Client sum/avg | useMemo over campaigns[] or series[] | Handler aggregates only |
| RoleDashboard slop | additionalProperties:true treated as any | Import handler DTO via openapi-types |
| Date drift | from/to not in URL | useSearchParams sync on Apply |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Skeleton forever | Placeholder KPI cards | ErrorBlock or real data |
| Freshness invented | Chip without API field | Omit until freshness DTO present |
| Copy role_dashboard_page | Monolith reuse | ui/dashboard/<role> sections |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per dashboard |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy role_dashboard_page | Monolith reuse | ui/<domain>/ per-role section |
| Client portfolio math | utilization_pct in browser | Use API utilization_pct |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Dashboard wired" without handler path for primary API
- Demo KPI literals in production TSX
- ghost_* UI labels on fraud dashboard

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Route `/dashboards/buyer`
- Portfolio KPIs, campaigns health, recommendations, alerts, fraud strip
- API: `GET /api/v1/dashboards/buyer` (`dashboardBuyer`)
- KPI keys: kpis.spend_micro, kpis.revenue_micro, kpis.profit_micro, kpis.conversions, kpis.cpa_micro, kpis.roi_pct, active, paused, archived, impressions_7d, clicks_7d, overspend_count
- FreshnessBadge from freshness DTO only
- RBAC: `campaigns:read | campaigns:read:masked` per OpenAPI x-permissions

### Out of scope

- Client aggregation of KPIs or series rollups
- Demo/hardcoded numbers
- Hand-written DTO diverging from handler json tags

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET | GET /api/v1/dashboards/buyer | BuyerPortfolioDTO |
| KPIs | kpis | MetricsBlockDTO |
| Campaigns | campaigns[] | BuyerCampaignPortfolioRowDTO |
| Recs | recommendations[] | RecommendationCardDTO + actions |
| Alerts | alerts[] | AlertCardDTO |
| Fraud | fraud | CustomerFraudOverviewDTO optional |
| Series | series[] | DashboardSeriesPointDTO label, impressions, blocks, spend_micros |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Buyer dashboard | static |
| freshness | FreshnessBadge | kpis.freshness.stale, ch_lag_seconds, as_of | kpis.freshness |
| kpis | KpiGrid | repeat(auto-fill, minmax(12rem, 1fr)) | portfolio KPI keys |
| campaigns | CampaignHealthGrid | campaigns[] status, utilization_pct, margin_breach | campaigns[] |
| recs | RecommendationCards | recommendations[] title, actions | recommendations[] |
| alerts | AlertCards | alerts[] level, route | alerts[] |
| fraud | FraudOverviewStrip | fraud.* when present | fraud optional |
| links | DrillDownLinks | /reports/*, /campaigns/:id | permissions |
| error | ErrorBlock | Fetch failure | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/dashboards/buyer` |
| Permission | campaigns:read \| campaigns:read:masked |
| `live` | true |
| Handler | internal/dashboardadmin/handlers.go |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome + FreshnessBadge
  -> KpiGrid: spend | revenue | profit | conversions | cpa | roi | active | paused | impressions_7d | clicks_7d
  -> CampaignHealthGrid (subgrid)
  -> RecommendationCards | AlertCards
  -> FraudOverviewStrip (optional)
  -> DrillDownLinks

| Invariant | Value |
| :--- | :--- |
| Page grid | CSS Grid on page root; no flex page layout |
| KPI grid | repeat(auto-fill, minmax(12rem, 1fr)) in rem |
| Money | formatMicro on *_micro fields; prefer *_display when API adds it |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/dashboard/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--dashboard-cols`: repeat(auto-fill, minmax(12rem, 1fr)) |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| customer_id | customer_id |  | Required on role dashboards except operator |
| from | from |  | RFC3339; ReportFromQuery |
| to | to |  | RFC3339; ReportToQuery |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/dashboards/buyer?customer_id={uuid}&from={iso}&to={iso}
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/role_dashboard_page.tsx | Thin compose; URL params -> fetch |
| web/src/ui/dashboard/buyer_dashboard.tsx | KPI + sections |
| web/src/ui/dashboard/*.module.css | Section grids |
| web/src/helpers/dashboard_api.ts | Typed dashboard fetch |
| web/src/types/generated/openapi.d.ts | Generated types |

**Legacy page:** `role_dashboard_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Demo KPIs | Every card maps to handler json field; no literals |
| Client KPI math | No useMemo sum/avg over campaigns[] or series[] |
| Flex page root | CSS Grid sections per ui.mdc |
| Silent catch → empty | ErrorBlock on blocking fetch failure |
| Freshness invented | Use kpis.freshness or envelope freshness only |
| ghost_* labels | silent_reject_* canonical on fraud surfaces |
| Stale range ignored | Refetch when from/to URL params change |
| Wrapper stack | No *DashboardChrome around PageChrome |
| Nested chip chrome | FreshnessBadge rendered directly in PageChrome slot |
| live without handler | Route live only when GET op registered |
| OpenAPI RoleDashboard lie | RoleDashboard is additionalProperties; bind typed handler DTO |
| Series client rollup | Render series[] points as returned; no daily merge in browser |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | GET /api/v1/dashboards/buyer | Confirm operation_id + handler DTO fields | openapi_gate.sh + handler test |
| 2 | internal/dashboardadmin/handlers.go | Cross-check json tags vs milestone KPI keys | grep json tags |
| 3 | make openapi-types | Regenerate openapi.d.ts | typecheck passes |
| 4 | web/src/helpers/dashboard_api.ts | Typed fetch with from/to/customer_id | Compiles |
| 5 | web/src/ui/dashboard/* | KpiGrid + sections per grid_ascii | check_ui_surface_gate.sh |
| 6 | web/src/pages/role_dashboard_page.tsx | Compose + URL param sync | Lazy import resolves |
| 7 | web/src/app_routes.tsx | Route + nav entry | report_live_routes honest |

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
| No demo KPIs | rg 'demo\|placeholder\|12345' web/src/ui/dashboard/ | no hardcoded KPI literals |
| KPI key parity | rg 'kpis.spend_micro' web/src/ui/dashboard/ | maps to API field name |
| Error state | Manual: block API | ErrorBlock visible; no empty KPI grid |
| Freshness chip | Manual: stale response | FreshnessBadge uses API stale/as_of |
| URL range | Manual: change from/to | Refetch; period in response updates |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `role_dashboard_page.tsx` replaced or delegates to `ui/dashboard/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/dashboard/`, helpers, and page compose in one slice; no half migration.


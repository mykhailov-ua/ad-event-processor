# ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS

Reference directory #2 after customers. Bulk actions + multi-filter list.

**Status:** DRAFT  
**Slug:** `admin_directory_campaigns`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** `admin_detail_campaign`  
**Pattern:** admin_directory_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Client sortRows | Legacy campaigns_page.tsx | rg table_sort on route |
| Buyer health merge | parallelAll buyer dashboard | List API labels only |
| Bulk without schema | Invented action names | OpenAPI POST .../bulk body |
| Masked read leak | Show full customer on masked role | RBAC + masked fields from API |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Import in toolbar without API | Dead button | Hide until OpenAPI POST import |
| Pause per-row only | N+1 pause calls | Bulk endpoint when multi-select |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_directory_pattern |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Keep buildUrl without q/sort | Minimal URL sync | All OpenAPI query params in URL |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Server search" if q removed from OpenAPI
- "Bulk wired" without POST .../bulk in same PR
- sortRows on items[] for any column

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/campaigns` under `web/src/ui/campaigns/`
- URL-driven filters: customer_id, status, q, sort, order, pacing_mode (per OpenAPI)
- Grid: name (link), status_label + status_tone chip, budget_display or budget_limit, pacing summary, updated_at
- Row link → `/campaigns/{id}`; touchCustomerContext when customer scoped
- Bulk selection + POST `/api/v1/campaigns/bulk` when permitted
- Buyer bound customer: default customer_id filter from session

### Out of scope

- Campaign detail (`ADMIN_DETAIL_MILESTONE_CAMPAIGN.md`)
- Wizard / migrate routes
- Client-side buyer dashboard merge (legacy fetches buyer dashboard for health — move to server labels or separate widget milestone)
- RecentCustomers widget (defer)

**Not on page (explicit):** `RecentCustomers`, `client sortRows on fetched page`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| status_label / status_tone | Handler sets json status_label; not in OpenAPI Campaign schema | Use status + displayLabel in UI until OpenAPI extended; no invented tone enum |
| Buyer health badges | Legacy parallelAll fetchBuyerDashboard for CampaignHealthBadge sort | Remove buyer dashboard fetch; list Campaign fields only |
| budget_display | Campaign schema has budget_limit string only | formatUsdDecimal(budget_limit) in UI — no budget_display field |
| freshness_label | Not on CampaignListResponse | Omit FreshnessBadge until envelope extended |
| Bulk delete | campaignsBulkMutate enum is pause\|resume only | No delete in bulk bar; per-row delete out of scope |

### Stop triggers (revert slice; do not compensate)

- Wrapper stack (`CampaignsDirectoryChrome`) — revert
- Bulk action without OpenAPI body schema — stop and fix contract first

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List op | GET /api/v1/campaigns — campaignsList | customer_id, status, q, sort (name\|updated_at\|spend), order, pacing_mode, limit, offset |
| Bulk op | POST /api/v1/campaigns/bulk — campaignsBulkMutate | body: action pause\|resume, campaign_ids[] |
| Row | Campaign schema — campaign.yaml | id, name, status, budget_limit, current_spend, pacing_mode, updated_at |
| List envelope | CampaignListResponse | items, total, limit, offset, filters_applied, sort |
| RBAC | x-permissions | campaigns:read, campaigns:read:masked |
| Handler | internal/campaign/handlers.go | campaignsList |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "Campaigns"; subtitle total | list.total |
| toolbar | CampaignsToolbar | Create wizard link, migrate link, import — permission gated | static routes + permissions |
| filters | CampaignsFilterPanel | customer_id, status, q, pacing_mode, sort, order; Apply → URL | URL params |
| bulk | BulkActionBar | Pause/resume/delete selection → bulk API | POST .../bulk |
| content | CampaignsGrid | role=grid | items[] |
| col_name | grid cell | Name + link to detail | item.name, item.id |
| col_status | StatusBadge | status_label + status_tone | API only |
| col_budget | grid cell | budget_display or formatted budget_limit | API string/micros |
| col_pacing | grid cell | Pacing summary from API | handler field |
| col_updated | grid cell | updated_at locale | item.updated_at |
| footer | PaginationBar | limit/offset/total | envelope |
| error | ErrorBlock | List failure | fetch error |


**Not on page (explicit):** RecentCustomers, client sortRows on fetched page.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns` |
| Nav group | Commercial → Campaigns |
| Icon | `megaphone` |
| Permission | campaigns:read or campaigns:read:masked |
| `live` | true |
| Handler | internal/campaign/handlers.go — campaignsList |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ Campaigns                         {total} total         │
├ Toolbar ────────────────────────────────────────────────┤
│ [Wizard] [Migrate] [Import]                             │
├ FilterPanel ─────────────────────────────────────────────┤
│ Customer [▼] Status [▼] Search [____] Sort [▼] [Apply] │
├ Bulk (when selection) ──────────────────────────────────┤
│ [Pause] [Resume] …                                      │
├ Content (role=grid) ──────────────────────────────────────┤
│ --campaigns-cols: minmax(14rem,2fr) 9rem 9rem 7rem 9rem │
│ Name | Status | Budget | Pacing | Updated               │
├ Footer ───────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--campaigns-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| Sortable headers | name, updated_at, spend (OpenAPI enum) only |
| Checkbox column | Leading column for bulk; does not affect --campaigns-cols data columns |


**Sortable headers:** name, updated_at, spend
**Non-sortable headers:** status, budget, pacing

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/campaigns/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--campaigns-cols`: minmax(14rem,2fr) 9rem 9rem 7rem 9rem |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| limit | limit | 50 |  |
| offset | offset | 0 |  |
| customer_id | customer_id |  | from URL; buyer session default |
| status | status |  | empty = all |
| q | q |  | server search |
| sort | sort | updated_at | name \| updated_at \| spend |
| order | order | desc | asc \| desc |
| pacing_mode | pacing_mode |  | optional filter |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/campaigns?limit=50&offset=0&customer_id={uuid}&status=active&q=test&sort=name&order=asc
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/campaigns_page.tsx | Compose: useSearchParams + fetch + sections |
| web/src/ui/campaigns/campaigns_directory.tsx | Section assembly |
| web/src/ui/campaigns/campaigns_toolbar.tsx | Toolbar region |
| web/src/ui/campaigns/campaigns_filter.tsx | FilterPanel draft + Apply |
| web/src/ui/campaigns/campaigns_grid.tsx | role=grid + rows |
| web/src/ui/campaigns/campaigns_directory.module.css | Page section grid |
| web/src/ui/campaigns/campaigns_grid.module.css | Column template |
| web/src/helpers/campaigns_api.ts | list(params, signal) |
| web/src/types/generated/openapi.d.ts | Generated types |
| web/src/ui/campaigns/campaigns_bulk_bar.tsx | Bulk selection + campaignsBulkMutate |
| web/src/ui/campaigns/campaigns_bulk_bar.module.css | Bulk bar grid |
| web/src/helpers/campaign_actions.ts | pause/resume single row if needed |
| web/src/helpers/campaign_admin_api.ts | import/clone — toolbar only when permitted |


**Remove from this route (legacy):** ../components/*, ../lib/table_sort.js, ../types/campaign.js, buyer dashboard merge for sort.

**Legacy page:** `campaigns_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Client filter on items[] | Forbidden; URL refetch only |
| Buyer dashboard side fetch | Health badges from parallel GET dashboards/buyer — remove |
| sortRows on items[] | Legacy table_sort — URL sort/order refetch |
| Bulk delete invented | OpenAPI bulk is pause\|resume only — no delete action |
| Bulk without confirm | Confirm modal before campaignsBulkMutate POST |
| Double status chrome | One StatusBadge per cell; no nested frames |
| Search UI without q | q exists in OpenAPI — wire URL param or omit field |
| status_label without OpenAPI | Use status string + displayLabel until schema extended |
| Wrapper stack | No CampaignsDirectoryChrome — flat ui/campaigns/ sections |
| Portal filter listbox | Inline drop on sort/status selects |
| N+1 pause per row | Use POST /api/v1/campaigns/bulk for multi-select |
| Import without idempotency | POST import requires Idempotency-Key header |
| Buyer scope leak | Default customer_id in URL for bound buyer session |
| RecentCustomers on page | Defer widget — not directory pattern |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm list op query params match URL | Manual or test: param change affects response |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck passes |
| 3 | web/src/helpers/campaigns_api.ts | Typed list fetch | Compiles; no hand-written DTO |
| 4 | web/src/ui/campaigns/* | Sections per 4.1–4.4 | check_ui_surface_gate.sh pass |
| 5 | web/src/pages/* | URL sync compose | No table_sort / client filter on items[] |
| 6 | web/src/app_routes.tsx | Route loads | Lazy import resolves |
| 7 | POST /api/v1/campaigns/bulk | BulkActionBar → campaignsBulkMutate body (pause\|resume) | 2xx + list refetch |
| 8 | Legacy cleanup | Remove table_sort, buyer dashboard merge, components/ imports | rg table_sort web/src/pages/campaigns_page.tsx empty |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'sortRows' web/src/pages/campaigns_page.tsx
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No client sort | rg 'table_sort\|sortRows' web/src/pages/campaigns_page.tsx web/src/ui/campaigns/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| Bulk pause | Manual: select 2 rows → Pause | campaignsBulkMutate 2xx; list refetch |
| Buyer scope | Manual: buyer session opens /campaigns | customer_id defaulted in URL |
| Search | Manual: q=foo in URL | server filters rows; no client filter |
| Sort | Manual: sort=spend&order=desc | row order changes |
| Masked read | Manual: campaigns:read:masked session | no PII leak in grid cells |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `campaigns_page.tsx` replaced or delegates to `ui/campaigns/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/campaigns/`, helpers, and page compose in one slice; no half migration.


# ADMIN_DIRECTORY_MILESTONE_RTB_DEALS

RTB deals directory with inline create/edit modal; full array fetch (no pagination).

**Status:** DRAFT  
**Slug:** `admin_directory_rtb_deals`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** —  
**Pattern:** admin_directory_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| UUID path for deal | Wrong id type on PATCH | Use int64 id from row |
| 501 treated as empty | Legacy stub path hides errors | ErrorBlock unless true 501 stub route |
| Modal fields invented | geo_mask in form without API | RtbDealCreateSpec only |
| Delete without confirm | One-click delete | Confirm modal |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| PaginationBar | No list envelope | Omit footer |
| Flex page header | page-header__row flex | Grid sections |
| settings:write fallback | Legacy uses settings:write for mutate | Prefer rtb:write per OpenAPI |
| Copy monolith page | 300-line rtb_deals_page | ui/rtb/ sections |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch rtb_deals_page in place | Keep Modal in page | ui/rtb/deal_form_modal |
| Skip delete flow | Edit only | DELETE wired with confirm |
| Skip int64 id typing | String id in URL | OpenAPI int64 path |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Paginated deals list" without OpenAPI limit/offset
- Form fields not on RtbDealCreateSpec
- Delete without confirm modal

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/rtb/deals` under `web/src/ui/rtb/`
- Grid: id, deal_id, floor_micro, customer_id, pacing, seats, updated_at
- Create/Edit modal with RtbDealCreateSpec fields from OpenAPI
- Delete row with confirm → DELETE .../rtb/deals/{id}
- Breadcrumb link back to /rtb/integration

### Out of scope

- RTB integration profile (`ADMIN_DETAIL_MILESTONE_RTB.md`)
- Server pagination (rtbListDeals returns full array)
- geo_mask / cat_mask editing unless exposed in PATCH schema

**Not on page (explicit):** `PaginationBar`, `client sortRows`, `501 stub forever on live handler`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| No pagination | rtbListDeals returns RtbDeal[] without limit/offset | No PaginationBar; full fetch; window if N>500 |
| geo_mask / cat_mask | On RtbDeal schema but not in legacy modal | Omit columns until PATCH body documents editable masks |
| Path id type | Deal path id is int64 not UUID | Use deal.id (int64) for PATCH/DELETE paths |

### Stop triggers (revert slice; do not compensate)

- Modal fields not on RtbDealCreateSpec — revert
- Wrapper RtbDealsChrome — revert

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List | GET /api/v1/rtb/deals — rtbListDeals | array of RtbDeal |
| Create | POST /api/v1/rtb/deals — rtbCreateDeal | RtbDealCreateSpec body |
| Update | PATCH /api/v1/rtb/deals/{id} — rtbPatchDeal | RtbDealCreateSpec body |
| Delete | DELETE /api/v1/rtb/deals/{id} — rtbDeleteDeal | confirm modal |
| Row | RtbDeal | id, deal_id, floor_micro, customer_id, pacing, seats, updated_at |
| RBAC | x-permissions | rtb:read / rtb:write |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "RTB deals" | deals.length |
| context | ContextBar | Breadcrumb RTB → Deals | static routes |
| toolbar | DealsToolbar | Create deal button | POST when rtb:write |
| content | DealsGrid | role=grid | deals[] |
| col_deal_id | grid cell | deal_id monospace | item.deal_id |
| col_floor | grid cell | formatAmountMicro(floor_micro) | item.floor_micro |
| col_customer | grid cell | customer_id | item.customer_id |
| col_pacing | grid cell | pacing label | item.pacing |
| col_actions | grid cell | Edit / Delete | rtb:write |
| modal | DealFormModal | Create/edit form | RtbDealCreateSpec |
| error | ErrorBlock | List/save failure | fetch error |
| loading | grid skeleton | placeholder rows | loading |
| empty | EmptyState | No deals + create CTA | deals.length === 0 |


**Not on page (explicit):** PaginationBar, client sortRows, 501 stub forever on live handler.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/rtb/deals` |
| Nav group | RTB → Deals |
| Icon | `handshake` |
| Permission | rtb:read (list); rtb:write (mutate) |
| `live` | true |
| Handler | internal/rtbadmin/handlers.go — rtbListDeals, rtbCreateDeal, rtbPatchDeal, rtbDeleteDeal |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ RTB deals                                               │
├ Toolbar ────────────────────────────────────────────────┤
│ [Create deal]                                           │
├ Content (role=grid) ────────────────────────────────────┤
│ --rtb-deals-cols: 5rem minmax(8rem,1fr) 7rem … 8rem      │
│ ID | Deal ID | Floor | Customer | Pacing | Seats | Updated│
├ (no pagination — full array) ───────────────────────────┤
└─────────────────────────────────────────────────────────┘
Modal: deal_id, customer_id, floor_micro, pacing, seats
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--rtb-deals-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| No pagination | Full array from rtbListDeals |
| int64 path id | PATCH/DELETE use numeric id |


**Non-sortable headers:** deal_id, floor, customer, pacing, seats, updated

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/rtb/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--rtb-cols`: 5rem minmax(8rem,1fr) 7rem minmax(10rem,1fr) 6rem 5rem 8rem |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/rtb/deals
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/rtb_deals_page.tsx | Compose: fetch + modal state |
| web/src/ui/rtb/deals_directory.tsx | Section assembly |
| web/src/ui/rtb/deals_grid.tsx | role=grid + rows |
| web/src/ui/rtb/deal_form_modal.tsx | Create/edit form |
| web/src/ui/rtb/deals_directory.module.css | Page section grid |
| web/src/ui/rtb/deals_grid.module.css | Column template |
| web/src/helpers/rtb_api.ts | fetchRtbDeals, create, patch, delete |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../components/modal.js, ../components/breadcrumbs.js, <table> data-table.

**Legacy page:** `rtb_deals_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Wrong path id type | int64 not UUID on PATCH/DELETE |
| Pagination fiction | No envelope — no PaginationBar |
| 501 stub as success | Show ErrorBlock on real errors |
| Modal fields invented | OpenAPI body only |
| Delete no confirm | Confirm modal required |
| Flex page layout | Grid sections |
| <table> grid | role=grid subgrid |
| Toast before save 2xx | apiConfirmed |
| settings:write only gate | Document rtb:write primary |
| geo_mask column | Not in legacy modal — omit |
| Wrapper stack | No RtbDealsChrome |
| Silent catch → empty | ErrorBlock on load fail |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/rtb.yaml | Confirm deal CRUD ops + RtbDealCreateSpec | openapi_gate |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/rtb_api.ts | list/create/patch/delete typed | int64 id paths |
| 4 | web/src/ui/rtb/* | Grid + modal sections | surface gate |
| 5 | web/src/pages/rtb_deals_page.tsx | Thin compose | no <table> |
| 6 | web/src/app_routes.tsx | Route /rtb/deals | loads |
| 7 | Delete confirm | Confirm modal before DELETE | manual test |
| 8 | Legacy cleanup | Remove components/modal import | rg '../components/modal' web/src/pages/rtb_deals_page.tsx empty |

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
| No client sort | rg 'table_sort\|sortRows' web/src/pages/rtb_deals_page.tsx web/src/ui/rtb/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| Create | Manual: create deal | POST 201; grid refetch |
| Edit | Manual: patch deal | PATCH 2xx |
| Delete | Manual: delete with confirm | DELETE 2xx; row removed |
| No pagination | rg PaginationBar web/src/ui/rtb/ | no matches |
| Error | Manual: block API | ErrorBlock visible |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `rtb_deals_page.tsx` replaced or delegates to `ui/rtb/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/rtb/`, helpers, and page compose in one slice; no half migration.


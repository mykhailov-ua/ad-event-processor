# ADMIN_DIRECTORY_MILESTONE_BRANDS

Brands directory (route gap) — customer-scoped brand grid + create; creatives sub-panel per API.

**Route gap:** register `/brands` in `app_routes.tsx` before `live: true`.

**Status:** DRAFT  
**Slug:** `admin_directory_brands`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** —  
**Pattern:** admin_directory_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| live without route | Catalog live before app_routes | Register route first |
| Missing customer_id | brandsList 400 | URL required param |
| Fleet-wide brand list | API requires customer scope | No admin-all brands until API exists |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| live:true early | Nav link 404 | gap_route until route ships |
| PaginationBar | No list envelope | Omit |
| Creatives without brand row | Orphan panel | Expand selected brand only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_directory_pattern |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Skip nav_config | Route only | Nav + permission in same PR |
| Skip creatives panel | Brands only | Milestone includes creatives sub-route |
| Hand-written Brand type | types/brand.js | openapi.d.ts |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Brands directory live" without /brands in app_routes.tsx
- List fetch without customer_id query param
- Pagination without OpenAPI limit/offset

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Register `/brands` route + nav entry
- Customer scope: required customer_id query (OpenAPI CustomerIdQueryRequired)
- Brand grid: name, id, updated_at, freq_limit/freq_window
- Create brand modal → POST /api/v1/brands
- Row expand or side panel: creatives list GET .../brands/{id}/creatives
- Create creative when campaigns:write

### Out of scope

- Campaign brand picker embedded in campaign editor
- Server pagination (brandsList returns array)
- Global brand list without customer_id

**Not on page (explicit):** `PaginationBar`, `brand list without customer_id`, `client sortRows`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| Route gap | No /brands in legacy app_routes or pages | Register route + nav before live:true |
| customer_id required | brandsList requires customer_id query param | FilterPanel or session default — cannot list all brands fleet-wide |
| No ListEnvelope | brandsList returns Brand[] array | No PaginationBar; full fetch per customer |
| Creatives no pagination | brandCreativesList returns array | Sub-panel lists all creatives for brand |

### Stop triggers (revert slice; do not compensate)

- live:true before route registered — revert
- Brand grid without customer_id — API will 400

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List brands | GET /api/v1/brands — brandsList | customer_id required query |
| Create brand | POST /api/v1/brands — brandsCreate | CreateBrandRequest |
| List creatives | GET /api/v1/brands/{id}/creatives — brandCreativesList | BrandCreative[] |
| Create creative | POST /api/v1/brands/{id}/creatives — brandCreativesCreate | create body |
| Row | Brand | id, customer_id, name, freq_limit, freq_window, updated_at |
| RBAC | x-permissions | campaigns:read / campaigns:write |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "Brands" | brands.length |
| filters | BrandsFilterPanel | customer_id required; Apply → URL | customer_id query |
| toolbar | BrandsToolbar | Create brand | POST /brands |
| content | BrandsGrid | role=grid brand rows | brands[] |
| col_name | grid cell | brand name | item.name |
| col_id | grid cell | CopyableUuid id | item.id |
| col_updated | grid cell | updated_at locale | item.updated_at |
| col_freq | grid cell | freq_limit / freq_window | item.freq_* |
| creatives | BrandCreativesPanel | expand row creatives grid | GET .../creatives |
| modal | BrandCreateModal | name + customer_id | CreateBrandRequest |
| error | ErrorBlock | List/create failure | errors |
| loading | grid skeleton | placeholder rows | loading |
| empty | EmptyState | No brands for customer | array empty |


**Not on page (explicit):** PaginationBar, brand list without customer_id, client sortRows.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/brands` |
| Nav group | Commercial → Brands (new) |
| Icon | `tag` |
| Permission | campaigns:read (list); campaigns:write (create) |
| `live` | true |
| Handler | internal/brand/handlers.go — brandsList, brandsCreate, brandCreativesList |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ Brands                                                  │
├ FilterPanel ────────────────────────────────────────────┤
│ Customer ID [___________] [Apply]  (required)           │
├ Toolbar ────────────────────────────────────────────────┤
│ [Create brand]                                          │
├ Content (role=grid) ────────────────────────────────────┤
│ --brands-cols: minmax(12rem,2fr) 10rem 8rem 7rem        │
│ Name | ID | Updated | Freq cap                          │
├ Creatives panel (expanded row) ─────────────────────────┤
│ Creative name | landing_url | weight | status           │
└─────────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--brands-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| customer_id required | No fetch without customer_id in URL |
| No pagination | Array response — no PaginationBar |


**Non-sortable headers:** name, id, updated_at, freq

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/brands/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--brands-cols`: minmax(12rem,2fr) 10rem 8rem 7rem |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| customer_id | customer_id |  | required — CustomerIdQueryRequired |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/brands?customer_id={uuid}
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/brands_page.tsx | Compose: useSearchParams + fetch + sections |
| web/src/ui/brands/brands_directory.tsx | Section assembly |
| web/src/ui/brands/brands_toolbar.tsx | Toolbar region |
| web/src/ui/brands/brands_filter.tsx | FilterPanel draft + Apply |
| web/src/ui/brands/brands_grid.tsx | role=grid + rows |
| web/src/ui/brands/brands_directory.module.css | Page section grid |
| web/src/ui/brands/brands_grid.module.css | Column template |
| web/src/helpers/brands_api.ts | list(params, signal) |
| web/src/types/generated/openapi.d.ts | Generated types |
| web/src/ui/brands/brand_creatives_panel.tsx | Creatives sub-grid |
| web/src/ui/brands/brand_create_modal.tsx | Create brand form |
| web/src/helpers/nav_config.ts | Add /brands nav entry |


**Remove from this route (legacy):** N/A — new route.

**Legacy page:** `gap — no legacy page`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| live without route | Register app_routes + nav first |
| Missing customer_id | API 400 — block fetch until set |
| Fleet-wide list fiction | customer_id required in OpenAPI |
| PaginationBar | Array response — omit |
| Wrapper BrandsDirectoryChrome | Flat ui/brands/ |
| Campaign picker scope | Embedded picker is different surface |
| Creative fields invented | BrandCreative schema only |
| Toast before create 2xx | apiConfirmed |
| Flex page layout | Grid sections |
| Silent catch → empty | ErrorBlock |
| Skip creatives API | brandCreativesList wired on expand |
| Wrong permission slug | campaigns:read not brands:read |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | web/src/app_routes.tsx | Add /brands route | Route resolves |
| 2 | web/src/helpers/nav_config.ts | Nav entry campaigns:read | visible with permission |
| 3 | api/openapi/paths/campaigns.yaml | Confirm brands + creatives ops | openapi_gate |
| 4 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 5 | web/src/helpers/brands_api.ts | list/create + creatives helpers | customer_id param |
| 6 | web/src/ui/brands/* | Grid + modal + creatives panel | surface gate |
| 7 | web/src/pages/brands_page.tsx | URL customer_id compose | no fetch without id |
| 8 | Set live flag honest | nav + route + handler | report_live_routes_gate |

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
| No client sort | rg 'table_sort\|sortRows' web/src/pages/brands_page.tsx web/src/ui/brands/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| Route | Manual: open /brands | page loads not 404 |
| customer_id | Manual: missing param | no fetch / validation message |
| Create brand | Manual: POST create | 201 + grid refetch |
| Creatives | Manual: expand row | creatives list loads |
| Nav | Manual: nav shows Brands | permission gated |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `gap — no legacy page` replaced or delegates to `ui/brands/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/brands/`, helpers, and page compose in one slice; no half migration.


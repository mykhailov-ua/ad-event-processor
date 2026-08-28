# ADMIN_DIRECTORY_MILESTONE_CUSTOMERS

Reference directory page for the admin SPA rebuild. Copy pattern for `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS` and other W2 lists.

**Status:** DRAFT  
**Slug:** `admin_directory_customers`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** `admin_detail_customer`  
**Pattern:** `admin_directory_pattern` (`ADMIN_MILESTONES_REQUIREMENTS.md`)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Search filter wired | Legacy `customers_page.tsx` filters client-side; OpenAPI has no `q` | `rg 'name: q' api/openapi/paths/platform.yaml` — empty until backend ships |
| Server sort works | Handler parses `sort`/`order` but `ListCustomers` ignores them today | Integration test or manual: change `order` and assert row order changes |
| `balance_display` field | Pages backlog typo; schema has `balance` string only | `Customer` in `api/openapi/components/schemas/platform.yaml` |
| Create customer button | No `POST /api/v1/customers` in OpenAPI | `rg 'post:' api/openapi/paths/platform.yaml` under customers |
| Freshness chip | `CustomerListResponse` has no `freshness` / `freshness_label` | Omit chip until handler adds envelope field |
| PageChrome exists | Foundation milestone may not be shipped yet | `test -f web/src/ui/system/page_chrome.tsx` |
| Types from OpenAPI | Agents invent `CustomerDTO` in `types/customer.js` | Use `web/src/types/generated/openapi.d.ts` after `make openapi-types` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Copy legacy page | Reuse `sortRows`, `FilterToolbar`, `components/` | New `ui/customers/` + URL-driven fetch |
| Client search | `useMemo` filter on `items[]` | Defer search UI until `q` query param exists on handler |
| Client sort on balance | Sortable headers for non-API columns | Sort headers only for `name`, `created_at` (OpenAPI enum) |
| Flex page layout | `flex items-center` on page root | CSS Grid sections per `ui.mdc` |
| Phantom create action | Toolbar "New customer" with no POST | No create control until OpenAPI + handler |
| Silent empty on error | `catch` → empty table | `ErrorBlock` on failed list fetch |
| Extend `components/` | New files under `web/src/components/` | `ui/customers/` + `ui/system/` only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch `customers_page.tsx` in place | Smallest diff | Replace page compose; delete legacy imports from this route |
| Keep `table_sort.js` | Reuse lib | Remove client sort; URL `sort`/`order` only |
| Skip tenant redirect | Ignore edge case | Tenant role still redirects to `/customers/:id` |
| Skip backend sort gap | UI-only PR | Section 5 step 1: wire SQL sort or document blocker in PR |
| `-short` only verification | Fast green | `admin_web.sh` + `check_ui_surface_gate.sh` in section 7 |

### 1.4 Forbidden claims until verified

- "Server-side search" without `q` on `GET /api/v1/customers`
- "Sort by balance" without OpenAPI enum + store implementation
- "Done" without section 7 commands pasted with exit codes
- "Reference directory shipped" if legacy `sortRows` still on `/customers`

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

---

## 2. Scope

### In scope

- Rebuild `/customers` as reference **directory pattern** page
- Server pagination via `limit`, `offset`
- Server sort via URL `sort` (`name` \| `created_at`) and `order` (`asc` \| `desc`) — requires handler/store fix if sort not applied today
- Grid columns: name (link), balance, currency, active_campaigns, created_at
- Tenant users: redirect to own `/customers/:id` (preserve legacy behavior)
- Row click → `/customers/:id` with `touchCustomerContext`
- Replace legacy `web/src/pages/customers_page.tsx` compose to use `ui/customers/`
- Nav entry already exists (`nav_config.ts`); route already registered (`app_routes.tsx`)

### Out of scope

- Customer detail (`ADMIN_DETAIL_MILESTONE_CUSTOMER.md`)
- Create customer (no POST list endpoint)
- Search by name/ID (`q` param — backend milestone first; see API gaps)
- `RecentCustomers` sidebar widget (defer; not in directory pattern)
- Export / bulk actions
- `balance_display` server field (format `balance` in UI only)

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| `q` search | Not in OpenAPI or handler | No search field; do not client-filter |
| SQL sort | Handler parses sort; store may ignore | Step 1 must wire `ListCustomers` sort or PR notes gap |
| `freshness_label` | Not on list envelope | No freshness chip on PageChrome |
| `POST /customers` | Missing | No create button |

### Stop triggers (revert slice; do not compensate)

- Operator rejects grid layout → revert `ui/customers/` slice; update section 4.3 only
- Fix adds wrapper stack (`CustomersDirectoryChrome`) instead of flat sections

---

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List operation | `GET /api/v1/customers` — `customersList` in `api/openapi/paths/platform.yaml` | `limit`, `offset`, `sort`, `order` |
| Response schema | `CustomerListResponse` in `api/openapi/components/schemas/platform.yaml` | `items`, `total`, `limit`, `offset`, `sort` |
| Row schema | `Customer` | `id`, `name`, `balance`, `currency`, `active_campaigns`, `created_at` |
| RBAC | `x-permissions: [customers:read]` | Hide nav/route if permission missing (shell) |
| Tenant scope | `isTenantUser` + `customer_id` on session | Redirect to detail |
| Handler | `internal/platformadmin/customers_handlers.go` | `listCustomers` |
| Global UI rules | G1–G10 in `ADMIN_MILESTONES_REQUIREMENTS.md` | All apply |

---

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| `chrome` | `PageChrome` | Title "Customers"; optional total count subtitle | `total` from list response |
| `toolbar` | `CustomersToolbar` | Secondary link "Open billing" → `/billing` | static route |
| `filters` | `CustomersFilterPanel` | Sort field + order; Apply → URL | URL params `sort`, `order` |
| `content` | `CustomersGrid` | `role="grid"` row list | `items[]` |
| `col_name` | grid cell | Customer name; navigates to detail | `item.name`, `item.id` |
| `col_balance` | grid cell | Balance display | `item.balance` via `formatUsdDecimal` |
| `col_currency` | grid cell | Currency code | `item.currency` (default display `USD` if empty) |
| `col_campaigns` | grid cell | Active campaign count | `item.active_campaigns` |
| `col_created` | grid cell | Created date (locale format) | `item.created_at` |
| `footer` | `PaginationBar` | Prev/next page | `limit`, `offset`, `total` |
| `error` | `ErrorBlock` | Blocking list failure | fetch error |
| `loading` | grid skeleton | 5 placeholder rows | `loading` state |
| `empty` | empty state | No rows copy + link to billing | `items.length === 0` && !loading |

**Not on page (explicit):** search input, create button, freshness chip, `RecentCustomers`.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/customers` |
| Nav group | Commercial → Customers (`web/src/helpers/nav_config.ts`) |
| Icon | `users` |
| Permission | `customers:read` |
| `live` | `true` — handler registered |
| Detail link | Row / name → `/customers/{id}` |
| Tenant | `isTenantUser` → `replace` navigate to `/customers/{customer_id}` |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────┐
│ title: Customers          meta: "{total} total"      │
├ Toolbar ────────────────────────────────────────────┤
│ [Open billing]                                       │
├ FilterPanel ───────────────────────────────────────┤
│ Sort: [name ▼]  Order: [asc ▼]  [Apply]             │
├ Content (role=grid) ────────────────────────────────┤
│ --customers-cols: minmax(12rem,2fr) 7rem 4rem 6rem 8rem │
│ Name | Balance | Currency | Campaigns | Created       │
│ ... rows ...                                         │
├ Footer ─────────────────────────────────────────────┤
│ PaginationBar                                        │
└─────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each data row uses same `--customers-cols` template |
| Sortable headers | `name`, `created_at` only — click sets URL `sort`/`order` and refetches |
| Non-sortable headers | `balance`, `currency`, `active_campaigns` — no sort affordance |
| Max page width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer sticky within page grid |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/customers/*.module.css` only |
| Page file | `customers_page.tsx` imports domain components; no CSS modules on page |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Grid class | `.directory` on page root; `.grid` on content |
| Row hover | `--color-row-hover` on clickable rows |
| Sort icon | `ui/system/` icon component; active column `aria-sort` |
| Listbox | Sort/order selects: inline drop, wrapper `width: 100%` |
| Chip | N/A (no freshness) |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit` (default 50), `offset`, `total` | URL → refetch on page change |
| Sort | `sort` enum `name`, `created_at`; `order` `asc`, `desc` | URL → refetch on Apply / header click |
| Search | **Not implemented** — no `q` | **No search UI** until OpenAPI + handler |
| List body | `CustomerListResponse` | Render `items` as returned |
| Balance format | `balance` string from API | `formatUsdDecimal` display only |
| Filters echo | `sort` object in response | Optional: show active sort from response |
| Tenant guard | session `customer_id` | Redirect; skip list fetch |

**URL param mapping:**

| URL param | API query | Default |
| :--- | :--- | :--- |
| `limit` | `limit` | `50` |
| `offset` | `offset` | `0` |
| `sort` | `sort` | `created_at` |
| `order` | `order` | `desc` |

Fetch example:

```
GET /api/v1/customers?limit=50&offset=0&sort=name&order=asc
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| `web/src/pages/customers_page.tsx` | Compose: `useSearchParams`, tenant redirect, wire fetch |
| `web/src/ui/customers/customers_directory.tsx` | Section assembly |
| `web/src/ui/customers/customers_toolbar.tsx` | Toolbar region |
| `web/src/ui/customers/customers_filter.tsx` | FilterPanel draft + Apply |
| `web/src/ui/customers/customers_grid.tsx` | Grid + sort headers + rows |
| `web/src/ui/customers/customers_directory.module.css` | Page section grid |
| `web/src/ui/customers/customers_grid.module.css` | Column template + rows |
| `web/src/helpers/customers_api.ts` | `listCustomers(params, signal)` → typed response |
| `web/src/types/generated/openapi.d.ts` | Generated types (`make openapi-types`) |

**Remove from this route (legacy):** imports from `../components/*`, `../lib/table_sort.js`, `../types/customer.js` (migrate to generated types).

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Copy legacy `sortRows` | Delete usage; holdout: no `table_sort` import in `customers_page.tsx` |
| Search without `q` API | No search field until backend milestone |
| Sort headers on balance/currency | Only `name`, `created_at` sortable |
| Wrapper stack | No `CustomersDirectoryChrome`; one folder `ui/customers/` |
| Double chip chrome | No freshness chip (not in API) |
| Portal filter listbox | Inline drop on sort selects |
| Silent `catch` → empty table | `ErrorBlock` on list error |
| `live: true` without API | Handler exists at `GET /api/v1/customers` |
| Demo KPIs | Show `total` from API only |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |
| Tenant list leak | Redirect tenant before fetch |
| Perf: full list render | Server page size 50; windowing N/A at this page size |

---

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | `internal/platformadmin/customers.go`, SQL | Wire `sort`/`order` into `ListCustomers` if not already applied | Changing URL `sort` changes row order (manual or test) |
| 2 | `api/openapi/paths/platform.yaml` | Add `q` param only if step 1 backend search ships in same PR | OpenAPI matches handler |
| 3 | `make openapi-types` | Regenerate `openapi.d.ts` | `npm run typecheck` passes |
| 4 | `web/src/helpers/customers_api.ts` | `listCustomers({ limit, offset, sort, order }, signal)` | Unit-free compile; uses generated types |
| 5 | `web/src/ui/customers/*` | Build sections per 4.1–4.4 | `check_ui_surface_gate.sh` pass |
| 6 | `web/src/pages/customers_page.tsx` | URL sync + compose; tenant redirect | No `table_sort` / client filter |
| 7 | `web/src/app_routes.tsx` | Confirm lazy import path unchanged | Route loads |
| 8 | Legacy cleanup | Remove unused imports from old customers page patterns | `rg table_sort web/src/pages/customers_page.tsx` empty |

**Optional follow-up PR (not this milestone):** `q` search backend + filter field; `POST /customers` + create toolbar.

---

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin path | N/A hot-path SLA | Not `/track`; no p99 gate |
| Render | Initial paint | No layout shift on load | Skeleton in grid body only |
| Scroll | Frame budget | < 16 ms/frame at 50 rows | React profiler (optional) |
| Re-render | Sort Apply | No full-app commit | Filter state local until Apply |

Shell/list at 50 rows: server pagination sufficient; windowing not required (`react.mdc` G9 threshold 500+).

---

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
go test ./internal/controlplane/ -short -run TestQueryBudget_ListCustomers -count=1
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Types | `cd web && npm run typecheck` | exit 0 |
| Surface gate | `bash scripts/ci/check_ui_surface_gate.sh` | exit 0; grid + modules under `ui/customers/` |
| Admin orchestrator | `bash scripts/ci/admin_web.sh` | exit 0 |
| No client sort | `rg 'table_sort|sortRows' web/src/pages/customers_page.tsx web/src/ui/customers/` | no matches |
| Tenant redirect | Manual: tenant session opens `/customers` | lands on `/customers/{id}` |
| Pagination | Manual: >50 customers | offset changes; total stable |
| Sort | Manual: toggle `sort=name&order=asc` | row order changes (after step 1) |
| Error | Manual: stop control plane | `ErrorBlock` visible, not empty table |

PR body must paste commands **actually run** with exit codes.

---

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] Legacy `customers_page.tsx` no longer uses client `sortRows` / search filter
- [ ] All new CSS in `web/src/ui/customers/*.module.css`
- [ ] List fetch uses URL params only
- [ ] Tenant redirect preserved
- [ ] Verification output pasted in PR
- [ ] Commit title names surface, e.g. `Rebuild customers directory page under ui/customers`

---

## 9. Rollback

Revert `web/src/ui/customers/`, `web/src/helpers/customers_api.ts`, and `customers_page.tsx` in one slice. Restore previous `customers_page.tsx` from git if needed. Do not leave half-migrated imports.

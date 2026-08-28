# ADMIN_PUBLISHER_MILESTONE

Publisher portal at /publisher with performance, statements, supply validation tabs.

**Status:** DRAFT  
**Slug:** `admin_publisher`  
**Depends on:** admin_page_chrome  
**Blocks:** —  
**Pattern:** dashboard  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per dashboard |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Done" without section 7 commands and exit codes
- "Wired" without handler path for primary API
- Client-side business rules on cold path (G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`))

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/publisher` — seller-scoped KPIs; statements; supply validation tab
- Tab state: dashboard | statements | supply (legacy publisher_page.tsx)
- KPIs from publisher/dashboard only — impressions, fill_rate, ecpm, revenue

### Out of scope

- Buyer self-serve routes
- Client ROI math on statement rows

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Dashboard | GET /api/v1/publisher/dashboard | PublisherDashboard kpis |
| Statements | GET /api/v1/publisher/statements | PublisherStatement list |
| Supply | GET supply validation (integrations) | SupplyValidation — tab optional |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Publisher dashboard | static |
| context | SellerContext | seller_id, publisher_account_id | dashboard DTO |
| tabs | TabBar | dashboard \| statements \| supply | local tab state |
| kpis | PublisherKpiGrid | impressions, fill_rate, ecpm, revenue | GET .../dashboard |
| statements | StatementsGrid | statement periods | GET .../statements items[] |
| supply | SupplyValidationPanel | sellers.json / ads.txt validation | supply validation API |
| error | ErrorBlock | Blocking dashboard failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| dashboard | Performance | GET /api/v1/publisher/dashboard |
| statements | Statements | GET /api/v1/publisher/statements |
| supply | Supply validation | supply validation API |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/publisher` |
| Permission | publisher role / seller scope |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome → SellerContext → TabBar → [KpiGrid | StatementsGrid | SupplyPanel] → ErrorBlock

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/publisher/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Tabs | Separate GET per tab | No client merge of tab payloads |
| GET detail | Full DTO from handler | Render as returned |
| PATCH | Fields ⊆ OpenAPI Patch*Request | apiConfirmed after 2xx |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/publisher_page.tsx | Compose |
| web/src/ui/publisher/* | Publisher sections + CSS modules |
| web/src/helpers/publisher_api.ts | fetchPublisherDashboard, fetchPublisherStatements |

**Legacy page:** `publisher_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Demo KPIs | Every metric from publisher/dashboard DTO |
| Client fill_rate math | Use API fill_rate field |
| Silent statements fail | Show empty only when API returns []; error on fail |
| Flex tab row | Grid tab bar; no flex page root |
| Supply tab 501 | StubBanner when validation API 501 |
| Seller id hardcoded | Display dashboard.seller_id from API |
| ghost_* in fraud cross-links | silent_reject_* canonical in linked reports |
| Operator publisher conflation | Route guard publisher role only |
| Client statement sort | Server order only |
| Copy legacy btn--* | Button from ui/system/ |
| Partial load hide error | dashboard error blocks page even if statements OK |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | GET /api/v1/publisher/dashboard | Confirm OpenAPI + handler | openapi_gate.sh |
| 2 | GET /api/v1/publisher/statements | Confirm list envelope | handler test |
| 3 | make openapi-types | Regenerate types | typecheck |
| 4 | web/src/helpers/publisher_api.ts | Typed fetch helpers | Compiles |
| 5 | web/src/ui/publisher/* | KPI grid + statements + supply tabs | surface gate |
| 6 | web/src/pages/publisher_page.tsx | Compose tabs | Route resolves |
| 7 | web/src/app_routes.tsx | Register /publisher | Publisher role guard |

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
| KPI source | Manual: compare UI to API JSON | values match dashboard.kpis |
| Statements error | Manual: block statements API | error or tab-level ErrorBlock |
| Supply tab | Manual: open supply tab | validation panel or StubBanner |
| No demo literals | rg 'demo\|placeholder' web/src/ui/publisher/ | no matches |
| Role guard | Manual: non-publisher session | redirect or 403 |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `publisher_page.tsx` replaced or delegates to `ui/publisher/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/publisher/`, helpers, and page compose in one slice; no half migration.


# ADMIN_DETAIL_MILESTONE_CUSTOMER

Detail/editor: /customers/:id.

**Status:** DRAFT  
**Slug:** `admin_detail_customer`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| balance_display typo | Pages backlog field name | Use Customer.balance string + formatUsdDecimal |
| TS-only tax fields | Extra keys on PUT body | TaxProfile schema only (4 fields) |
| Toast before 2xx | pushToast on click | apiConfirmed after PUT tax-profile 2xx |
| Client campaign sort | sortRows on embedded list | Link to /campaigns?customer_id=; no client sort |
| Phantom profile PATCH | Agents assume PatchCustomerRequest | rg PatchCustomer api/openapi — empty |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Copy components/ | Reuse billing widgets from legacy tree | ui/customers/ + ui/system/ only |
| Silent tax error | catch → empty form | ErrorBlock on tax GET failure |
| Demo forecast | Hardcoded chart points | Handler series or 503 UI |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_detail_pattern |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch customer_detail_page in place | Smallest diff | New ui/customers detail modules |
| Skip tenant guard | Operator-only testing | Preserve legacy 403 tenant boundary |
| Skip apiConfirmed | Direct fetch in save | confirmed_api helper |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Profile saved" without PATCH handler
- balance_display in TS types
- PUT tax-profile without apiConfirmed
- sortRows / table_sort on campaigns tab

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/customers/:id` under `web/src/ui/customers/`
- TabBar with URL `?tab=`; lazy fetch per tab
- Tax profile save via PUT .../tax-profile + apiConfirmed
- Tenant guard: 403 when tenant opens foreign customer id
- Replace legacy `customer_detail_page.tsx` compose

### Out of scope

- PATCH customer profile (no PatchCustomerRequest today)
- Wallet PATCH (GET-only in OpenAPI)
- Client ledger aggregation or balance math
- balance_display invented server field

**Not on page (explicit):** `Freshness chip (not on Customer GET)`, `Client sortRows on campaigns mini-grid`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| PatchCustomerRequest | GET /customers/{id} only; no PATCH in platform.yaml | Overview tab read-only; no Save on profile until OpenAPI + handler |
| Wallet PATCH | GET .../wallet only | Wallet tab display-only |
| Forecast CH down | billing/forecast may 503 | StubBanner or ErrorBlock; no demo chart |

### Stop triggers (revert slice; do not compensate)

- Wrapper stack (`CustomerDetailChrome`) — revert
- Invented PATCH /api/v1/customers/{id} without OpenAPI
- Billing tab POST not in OpenAPI

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET Customer | GET /api/v1/customers/{id} | id, name, balance, currency, cost_center, active_campaigns, total_spend, created_at, updated_at |
| PUT country_code | PUT .../tax-profile (TaxProfile) | country_code |
| PUT tax_region | PUT .../tax-profile (TaxProfile) | tax_region |
| PUT tax_scheme | PUT .../tax-profile (TaxProfile) | tax_scheme |
| PUT tax_rate_bps | PUT .../tax-profile (TaxProfile) | tax_rate_bps |
| GET Balance | GET .../customers/{id}/balance | balance_micro, currency, recent lines |
| GET Ledger | GET .../customers/{id}/ledger | items[], total, limit, offset |
| GET Statement | GET .../customers/{id}/billing/statement | periods, amounts from handler |
| GET Forecast | GET .../customers/{id}/billing/forecast | series[], ch_unavailable |
| GET Wallet | GET .../customers/{id}/wallet | balance_micro, payment_provider_configured, burn_days_estimate |
| GET Payments | GET .../customers/{id}/payments | items[], total |
| PATCH gap | No PatchCustomerRequest in OpenAPI | Profile name/cost_center not editable until handler + schema |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Breadcrumb Customers → name | GET customer.name |
| chrome | PageChrome | Title = customer.name; subtitle id truncated | GET customer |
| toolbar | CustomerToolbar | Export balance CSV; Open billing link | GET .../balance/export; static /billing |
| tabs | TabBar | overview \| balance \| ledger \| statement \| forecast \| wallet \| tax \| payments \| campaigns \| api_keys | URL ?tab= |
| kpi_strip | CustomerKpiStrip | balance, currency, active_campaigns, total_spend | GET customer fields |
| tab_overview | CustomerOverviewPanel | id, created_at, updated_at read-only dl | GET customer |
| tab_balance | BalancePanel | balance_micro slice + recent lines | GET .../balance |
| tab_ledger | LedgerGrid | Paginated ledger lines | GET .../ledger |
| tab_statement | StatementPanel | Statement periods; download | GET .../billing/statement |
| tab_forecast | ForecastWidget | CH forecast series; 503 → StubBanner | GET .../billing/forecast |
| tab_wallet | WalletPanel | balance_micro, provider readiness (read-only) | GET .../wallet |
| tab_tax | TaxProfileForm | PUT TaxProfile body fields | GET/PUT .../tax-profile |
| tab_payments | PaymentsGrid | Payment history rows | GET .../payments |
| tab_campaigns | CustomerCampaignsGrid | Top campaigns link-out | GET /api/v1/campaigns?customer_id |
| error | ErrorBlock | Blocking GET/PUT failure per tab | fetch errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| overview | Overview | GET /api/v1/customers/{id} |
| balance | Balance | GET /api/v1/customers/{id}/balance |
| ledger | Ledger | GET /api/v1/customers/{id}/ledger?limit&offset |
| statement | Statement | GET /api/v1/customers/{id}/billing/statement |
| forecast | Forecast | GET /api/v1/customers/{id}/billing/forecast |
| wallet | Wallet | GET /api/v1/customers/{id}/wallet |
| tax | Tax profile | GET/PUT /api/v1/customers/{id}/tax-profile |
| payments | Payments | GET /api/v1/customers/{id}/payments |
| campaigns | Campaigns | GET /api/v1/campaigns?customer_id={id} |
| api_keys | API keys | GET/POST /api/v1/selfserve/api-keys (tenant scope) |


**Not on page (explicit):** Freshness chip (not on Customer GET), Client sortRows on campaigns mini-grid.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/customers/:id` |
| Nav group | Commercial → Customers → Detail |
| Permission | customers:read |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

ContextBar → PageChrome → Toolbar → TabBar → [tab panel] → ErrorBlock

| Invariant | Value |
| :--- | :--- |
| Page grid | CSS Grid sections; no flex page root |
| Tab panel | Single scroll region per tab; min-height: 0 |
| KPI strip | repeat(auto-fill, minmax(12rem, 1fr)) on overview only |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/customers/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| tab | — | overview | TabBar sync; refetch on change |

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
| web/src/pages/customer_detail_page.tsx | Thin compose; tabs route optional |
| web/src/ui/customers/customers_detail.tsx | Detail shell |
| web/src/ui/customers/*.module.css | Section CSS modules |
| web/src/helpers/customers_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../components/*, ../types/customer.js, ../lib/table_sort.js.

**Legacy page:** `customer_detail_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Success toast before 2xx | apiConfirmed after PUT tax-profile |
| TS-only PATCH fields | No PatchCustomerRequest — overview read-only |
| balance_display | Format Customer.balance only |
| Client ledger math | Display GET .../ledger rows as returned |
| Client forecast math | No useMemo reduce on series |
| Wrapper stack | No CustomerDetailChrome; flat sections |
| Flex page root | CSS Grid per ui.mdc |
| Silent catch → empty tab | ErrorBlock per tab fetch |
| Tenant list leak | 403 before fetch when tenant id mismatch |
| Wallet edit UI | GET-only wallet — no fake Save |
| Phantom POST payments | No create UI unless POST documented on tab |
| Copy legacy components/ | Migrate to ui/customers/ |
| Masked field leak | Hide operator-only tabs when permission missing |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/platform.yaml + billing.yaml | Confirm GET customer + PUT TaxProfile fields | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/customers_api.ts | getCustomer, putTaxProfile, tab fetch helpers | No hand-written CustomerDTO |
| 4 | web/src/ui/customers/customer_detail.tsx | TabBar + section assembly per 4.1 | check_ui_surface_gate.sh |
| 5 | web/src/ui/customers/customer_*_panel.tsx | One module per tab region | PATCH/PUT fields match contract_rows |
| 6 | web/src/pages/customer_detail_page.tsx | Compose; ?tab=; tenant guard | <= ~120 lines |
| 7 | web/src/app_routes.tsx | Route /customers/:id | Lazy import resolves |
| 8 | Legacy cleanup | Remove components/* imports from route | rg table_sort\|sortRows customer_detail |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'balance_display' web/src/ui/customers/
rg 'PatchCustomerRequest' api/openapi/
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No balance_display | rg balance_display web/src/ui/customers/ | no matches |
| apiConfirmed tax | Manual: save tax profile | toast only after 2xx |
| Tenant 403 | Manual: tenant opens foreign id | ErrorBlock 403 |
| Tab refetch | Manual: ?tab=ledger | ledger API called once |
| No client sort | rg 'table_sort\|sortRows' web/src/pages/customer_detail_page.tsx web/src/ui/customers/ | no matches |
| OpenAPI parity | Cross-check TaxProfile PUT body vs form | 4 fields only |
| Error tab | Manual: block ledger GET | ErrorBlock on ledger tab |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `customer_detail_page.tsx` replaced or delegates to `ui/customers/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/customers/`, helpers, and page compose in one slice; no half migration.


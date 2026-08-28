# ADMIN_DIRECTORY_MILESTONE_BILLING

Operator invoice directory at /billing — KPI strip + invoice grid; legacy wallet/ledger tabs out of scope.

**Status:** DRAFT  
**Slug:** `admin_directory_billing`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** `admin_detail_invoice`  
**Pattern:** admin_directory_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Client KPI math | Sum invoices in browser for KPI strip | billingSummary endpoint only |
| Demo overdue count | Hardcoded KPI cards | API summary fields or hide strip |
| Ship all tabs | Scope creep from legacy billing_page | Invoices directory only |
| sortRows on invoices | Legacy table_sort | No client sort |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| due_at column | Field not in OpenAPI | Omit until schema ships |
| KPI without shards:read | Show strip with zeros | Hide strip |
| Wallet tab in same PR | Monolith carryover | Separate milestone |
| amount_display field | Invented DTO | formatAmountMicro |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy billing_page.tsx | 600-line monolith | ui/billing invoices only |
| Skip summary permission check | Always fetch summary | Gate on shards:read |
| Keep TabBar | wallet/ledger in page | Drop tabs from this milestone |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Billing directory done" with wallet/ledger tabs still in same page compose
- "KPI strip wired" without shards:read or billingSummary 2xx
- due_at or amount_display columns without OpenAPI fields
- sortRows on items[]

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild invoice directory portion of `/billing` under `web/src/ui/billing/`
- Admin fleet invoice grid: id, customer_id, billing_month, total_micro, status, currency
- Optional KPI strip from GET /api/v1/billing/summary when session has shards:read
- URL filters: customer_id, status, month, min_total, limit, offset per OpenAPI
- Row → `/billing/invoices/{id}`
- Buyer bound customer: default customer_id filter from session

### Out of scope

- Invoice detail (`ADMIN_DETAIL_MILESTONE_INVOICE.md`)
- Legacy wallet/ledger/exports/disputes tabs (separate milestones or detail)
- Client KPI math on invoice rows
- Crypto payment / self-serve billing panels
- RecentCustomers widget

**Not on page (explicit):** `wallet tab`, `ledger tab`, `exports tab`, `disputes tab`, `client sortRows`, `due_at column`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| due_at / amount_display | Invoice/InvoiceSummary have total_micro not due_at or amount_display | Format total_micro in UI; omit due column until OpenAPI field exists |
| status_label | InvoiceSummary.status string only — no status_label/tone | displayLabel(status) — no invented tone chip |
| billingSummary permission | billingSummary requires shards:read not customers:read | Hide KPI strip when permission missing — do not fake KPIs |
| Legacy multi-tab page | billing_page.tsx has wallet/ledger/exports/disputes | This milestone ships invoices directory + optional KPI only |

### Stop triggers (revert slice; do not compensate)

- Ship all billing tabs in one PR — split per milestone
- Demo overdue KPI literals — revert

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List op | GET /api/v1/billing/invoices — billingListInvoices | customer_id, month, status, min_total, limit, offset |
| List envelope | InvoiceListResponse | items, total, limit, offset |
| Row | Invoice / InvoiceSummary | id, customer_id, billing_month, total_micro, status, currency |
| Summary | GET /api/v1/billing/summary — billingSummary | invoiced_mtd_micro, invoice_count_mtd, undelivered_invoice_notifications |
| RBAC list | x-permissions | customers:read |
| RBAC summary | x-permissions | shards:read |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "Billing" | static |
| kpis | BillingSummaryStrip | MTD invoiced, invoice count, undelivered notifications | GET .../summary |
| filters | BillingFilterPanel | customer_id, status, month, min_total; Apply → URL | URL params |
| content | InvoicesGrid | role=grid invoice columns | items[] |
| col_id | grid cell | Invoice id link to detail | item.id |
| col_customer | grid cell | customer_id short + link context | item.customer_id |
| col_month | grid cell | billing_month | item.billing_month |
| col_total | grid cell | formatAmountMicro(total_micro) | item.total_micro |
| col_status | grid cell | status string | item.status |
| footer | PaginationBar | limit/offset/total | envelope |
| error | ErrorBlock | List/summary failure | fetch error |
| loading | grid skeleton | placeholder rows | loading |
| empty | EmptyState | No invoices copy | items.length === 0 |


**Not on page (explicit):** wallet tab, ledger tab, exports tab, disputes tab, client sortRows, due_at column.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/billing` |
| Nav group | Commercial → Billing |
| Icon | `receipt` |
| Permission | customers:read (invoices); shards:read (summary KPI) |
| `live` | true |
| Handler | internal/billingadmin/handlers.go — billingListInvoices, billingSummary |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ Billing                                                 │
├ KPI strip (optional shards:read) ───────────────────────┤
│ Invoiced MTD | Invoice count | Undelivered notifications│
├ FilterPanel ────────────────────────────────────────────┤
│ Customer [____] Status [▼] Month [____] Min total [__]  │
├ Content (role=grid) ────────────────────────────────────┤
│ --billing-cols: 10rem minmax(10rem,1.5fr) 7rem 8rem 7rem 5rem │
│ ID | Customer | Month | Total | Status | Currency       │
├ Footer ─────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--billing-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| KPI strip optional | Hidden without shards:read — no demo numbers |
| No client sort | No sort params on billingListInvoices |


**Non-sortable headers:** id, customer, month, total, status, currency

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/billing/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--billing-cols`: 10rem minmax(10rem,1.5fr) 7rem 8rem 7rem 5rem |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| limit | limit | 50 |  |
| offset | offset | 0 |  |
| customer_id | customer_id |  | buyer session default |
| status | status |  | invoice status filter |
| month | month |  | MonthQuery YYYY-MM |
| min_total | min_total |  | int64 micros floor |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/billing/invoices?limit=50&offset=0&customer_id={uuid}&status=open&month=2026-08
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/billing_page.tsx | Compose: useSearchParams + fetch + sections |
| web/src/ui/billing/billing_directory.tsx | Section assembly |
| web/src/ui/billing/billing_toolbar.tsx | Toolbar region |
| web/src/ui/billing/billing_filter.tsx | FilterPanel draft + Apply |
| web/src/ui/billing/billing_grid.tsx | role=grid + rows |
| web/src/ui/billing/billing_directory.module.css | Page section grid |
| web/src/ui/billing/billing_grid.module.css | Column template |
| web/src/helpers/billing_api.ts | list(params, signal) |
| web/src/types/generated/openapi.d.ts | Generated types |
| web/src/ui/billing/billing_summary_strip.tsx | KPI strip from billingSummary |
| web/src/ui/billing/billing_summary_strip.module.css | KPI grid |


**Remove from this route (legacy):** ../components/billing_* panels from invoice route, ../lib/table_sort.js, TabBar wallet/ledger/exports from directory milestone slice.

**Legacy page:** `billing_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Client KPI math | Use billingSummary only |
| Demo overdue KPI | No hardcoded numbers |
| due_at column | Not in InvoiceSummary schema |
| status_label chip | status string only — no tone API |
| All tabs at once | wallet/ledger out of scope |
| Buyer scope leak | Default customer_id for bound buyer |
| sortRows legacy | Remove table_sort |
| RecentCustomers widget | Not on directory page |
| Summary 403 shown as zeros | Hide KPI strip on error |
| Flex page layout | Grid sections |
| Wrapper BillingDirectoryChrome | Flat ui/billing/ |
| Silent catch → empty | ErrorBlock on list fail |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/billing.yaml | Confirm billingListInvoices + billingSummary | openapi_gate |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/billing_api.ts | listInvoices + getSummary helpers | header/body parse only |
| 4 | web/src/ui/billing/* | KPI strip + invoice grid sections | surface gate |
| 5 | web/src/pages/billing_page.tsx | Invoices compose + URL params | no wallet TabBar in slice |
| 6 | web/src/app_routes.tsx | Route /billing | loads |
| 7 | Permission gates | KPI strip shards:read; list customers:read | manual role matrix |
| 8 | Legacy cleanup | Remove table_sort from billing page | rg table_sort web/src/pages/billing_page.tsx empty |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'table_sort' web/src/pages/billing_page.tsx
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No client sort | rg 'table_sort\|sortRows' web/src/pages/billing_page.tsx web/src/ui/billing/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| KPI permission | Manual: session without shards:read | KPI strip hidden |
| Invoice row link | Manual: click row | navigates to /billing/invoices/{id} |
| Filters | Manual: customer_id in URL | refetch filtered list |
| No client sort | rg table_sort web/src/ui/billing/ | no matches |
| Error | Manual: block API | ErrorBlock visible |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `billing_page.tsx` replaced or delegates to `ui/billing/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/billing/`, helpers, and page compose in one slice; no half migration.


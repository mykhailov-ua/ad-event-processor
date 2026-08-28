# ADMIN_DETAIL_MILESTONE_INVOICE

Detail/editor: /billing/invoices/:id.

**Status:** DRAFT  
**Slug:** `admin_detail_invoice`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_BILLING`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| TS-only void fields | Invented POST body keys | OpenAPI void endpoint body only |
| Toast before 2xx | Void toast on click | apiConfirmed after POST void 2xx |
| Client line math | Re-sum ledger lines | Display GET invoice.lines + ledger-lines rows |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client KPI math | Sum sub-resources in browser | Stats endpoints only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_detail_pattern |
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

- Rebuild `/billing/invoices/:id` under `web/src/ui/billing/`
- Display Invoice.lines from GET (no client recomputation of tax)
- Ledger-lines cursor pagination; append on Load more
- Void + delivery retry with confirm modal + apiConfirmed

### Out of scope

- PATCH invoice (no PatchInvoiceRequest in OpenAPI)
- Client tax/subtotal math
- TS-only invoice fields

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| PATCH invoice | No mutating PATCH on invoice resource | Void via POST .../void only |

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET Invoice | GET /api/v1/billing/invoices/{id} | id, customer_id, billing_month, subtotal_micro, tax_micro, total_micro, currency, tax_scheme, tax_rate_bps, lines[], pdf_url, status |
| GET Ledger lines | GET .../invoices/{id}/ledger-lines | items[], total, next_cursor |
| GET Deliveries | GET .../invoices/{id}/deliveries | items[] delivery attempts |
| GET PDF | GET .../invoices/{id}/pdf | application/pdf binary |
| POST Void | POST .../invoices/{id}/void | no PATCH body — confirm modal + apiConfirmed |
| POST Retry | POST .../invoices/{id}/deliveries/retry | confirm + apiConfirmed |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Billing → customer → invoice id | GET invoice.customer_id |
| chrome | PageChrome | Invoice {billing_month}; status chip | GET invoice |
| toolbar | InvoiceToolbar | Download PDF; Void; Retry delivery | GET pdf; POST void; POST deliveries/retry |
| tabs | TabBar | header \| lines \| deliveries \| pdf | URL ?tab= |
| tab_header | InvoiceHeaderPanel | subtotal_micro, tax_micro, total_micro, lines[] | GET invoice |
| tab_lines | InvoiceLedgerGrid | Cursor-paged ledger lines | GET .../ledger-lines |
| tab_deliveries | InvoiceDeliveriesGrid | delivery status rows + retry | GET .../deliveries |
| tab_pdf | InvoicePdfLink | PDF download anchor | GET .../pdf blob |
| footer | LedgerCursorFooter | Load more when next_cursor | next_cursor from API |
| error | ErrorBlock | Blocking GET/POST failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| header | Invoice | GET /api/v1/billing/invoices/{id} |
| lines | Ledger lines | GET /api/v1/billing/invoices/{id}/ledger-lines?cursor&limit |
| deliveries | Deliveries | GET /api/v1/billing/invoices/{id}/deliveries |
| pdf | PDF | GET /api/v1/billing/invoices/{id}/pdf |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/billing/invoices/:id` |
| Nav group | Commercial → Billing → Invoice |
| Permission | customers:read |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

ContextBar → PageChrome → Toolbar → TabBar → [panel] → Cursor footer

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/billing/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| tab | — | header | TabBar optional; lazy load deliveries |

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
| web/src/pages/invoice_detail_page.tsx | Thin compose; tabs route optional |
| web/src/ui/billing/billing_detail.tsx | Detail shell |
| web/src/ui/billing/*.module.css | Section CSS modules |
| web/src/helpers/billing_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../types/index.js Invoice* imports.

**Legacy page:** `invoice_detail_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Success toast before 2xx | apiConfirmed on void/retry |
| TS-only PATCH fields | No invoice PATCH — POST mutations only |
| Client tax math | Use subtotal_micro/tax_micro from GET |
| Silent delivery error | toast only — show ErrorBlock on tab | ErrorBlock on deliveries fetch fail |
| Void without confirm | Confirm modal before POST void |
| Retry without confirm | Confirm modal before POST deliveries/retry |
| Wrapper stack | No InvoiceDetailChrome |
| Flex page root | CSS Grid sections |
| Copy legacy types/index.js | Generated openapi.d.ts types |
| Phantom PATCH | No edit-in-place invoice form |
| PDF inline iframe | Download via GET pdf blob only unless milestone waives |
| Double status chip | One StatusBadge from invoice status |
| Cursor leak | Pass next_cursor from API; no client invent |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/billing.yaml | Confirm invoice GET + void + deliveries ops | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/billing_api.ts | getInvoice, ledgerLines, deliveries, void, retry | Compiles |
| 4 | web/src/ui/billing/invoice_detail.tsx | TabBar + header panel | surface gate |
| 5 | web/src/ui/billing/invoice_*_panel.tsx | Lines grid + deliveries grid | No client sum |
| 6 | web/src/pages/invoice_detail_page.tsx | Compose; permissions for void | <= ~120 lines |
| 7 | web/src/app_routes.tsx | Route /billing/invoices/:id | Resolves |
| 8 | Legacy cleanup | Remove types/index.js InvoiceDTO duplicates | typecheck uses generated types |

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
| apiConfirmed void | Manual: void invoice (test env) | toast after 2xx |
| apiConfirmed retry | Manual: retry delivery | toast after 2xx |
| No client sum | rg 'reduce\(' web/src/ui/billing/invoice_ | no ledger aggregation |
| PDF download | Manual: download PDF | blob from GET .../pdf |
| Cursor paging | Manual: load more ledger lines | next_cursor appended |
| ErrorBlock | Manual: block GET invoice | ErrorBlock visible |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `invoice_detail_page.tsx` replaced or delegates to `ui/billing/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/billing/`, helpers, and page compose in one slice; no half migration.


# ADMIN_DIRECTORY_MILESTONE_AUDIT

Audit log directory with CSV export; array body + X-Total-Count header pagination.

**Status:** DRAFT  
**Slug:** `admin_directory_audit`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** —  
**Pattern:** admin_directory_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Invented total field | Assumes ListEnvelope JSON | Read X-Total-Count header |
| Export without permission | Show button without audit:read | Gate toolbar |
| Filter fiction | actor/action UI without API | api_gaps — no filters |
| Silent export fail | toast only | Show truncation banner + error |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| JSON total field | body.total in helper | Header parse only |
| Date filter UI | Not in OpenAPI | Defer filter milestone |
| Copy <table> layout | Legacy audit_page | role=grid subgrid |
| Export success without download | Missing blob handling | api_blob.ts pattern |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Keep page state pagination | No URL offset | Sync limit/offset in URL |
| Skip export truncation UI | Download only | X-Export-Truncated banner |
| Patch audit_page in place | Smallest diff | ui/audit/ sections |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Server-side actor filter" without OpenAPI param on auditList
- "total from response body" — use X-Total-Count
- Export button without audit:read permission

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/audit` under `web/src/ui/audit/`
- Paginated grid via limit/offset + X-Total-Count header
- Columns: created_at, action, target_type+target_id, admin_id
- redact_pii toggle → URL/query param (default true per legacy)
- Export CSV toolbar → GET .../audit/export?format=csv
- Show X-Export-Truncated + X-Next-Cursor banner when export truncated

### Out of scope

- actor/action/date text filters (not in OpenAPI list op)
- ListEnvelope with total field in JSON body
- PII unmask for non-audit:read roles

**Not on page (explicit):** `actor filter`, `action filter`, `date range filter`, `invented total JSON field`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| List envelope | auditList returns array + X-Total-Count header, not ListEnvelope | Parse total from header in audit_api.ts; never invent body.total |
| actor/action/date filters | Not on GET /api/v1/audit OpenAPI params | No filter panel until backend + OpenAPI milestone |
| Export customer_id filter | auditExport has optional customer_id; list does not | Export-only param; do not add to list UI unless list op gains param |
| changes/metadata columns | AuditLog has changes object — list UI shows summary only | No expand row client-side PII parsing beyond redact_pii flag |

### Stop triggers (revert slice; do not compensate)

- Invented filter UI without OpenAPI query params — stop
- Wrapper AuditDirectoryChrome — revert

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List op | GET /api/v1/audit — auditList | limit, offset, redact_pii |
| List body | AuditLog[] | id, admin_id, action, target_type, target_id, created_at |
| Total | X-Total-Count response header | integer — not JSON total |
| Export | GET /api/v1/audit/export — auditExport | format=csv, redact_pii, cursor, customer_id |
| Export headers | X-Export-Truncated, X-Next-Cursor | show truncation banner |
| RBAC | x-permissions | audit:read |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "Audit log"; subtitle entry count | X-Total-Count |
| toolbar | AuditToolbar | Export CSV button | GET .../export |
| filters | AuditFilterPanel | redact_pii checkbox only | URL redact_pii |
| content | AuditGrid | role=grid | rows[] |
| col_time | grid cell | created_at locale | row.created_at |
| col_action | grid cell | action string | row.action |
| col_target | grid cell | target_type + target_id short | row.target_* |
| col_admin | grid cell | admin_id short monospace | row.admin_id |
| export_banner | AlertBanner | truncated export notice | X-Export-Truncated |
| footer | PaginationBar | limit/offset pages | header total |
| error | ErrorBlock | List/export failure | fetch error |
| loading | grid skeleton | 5 placeholder rows | loading |
| empty | EmptyState | No audit entries copy | rows.length === 0 |


**Not on page (explicit):** actor filter, action filter, date range filter, invented total JSON field.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/audit` |
| Nav group | Settings → Audit |
| Icon | `list` |
| Permission | audit:read |
| `live` | true |
| Handler | internal/platformadmin/audit_logs.go — auditList |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ Audit log                         {total} entries       │
├ Toolbar ────────────────────────────────────────────────┤
│ [Export CSV]                                            │
├ FilterPanel ────────────────────────────────────────────┤
│ [x] Redact PII in changes/metadata                      │
├ Content (role=grid) ────────────────────────────────────┤
│ --audit-cols: 10rem 8rem minmax(10rem,1.2fr) 8rem       │
│ Time | Action | Target | Admin                          │
├ Footer ─────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--audit-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| Header total | total from X-Total-Count only |
| No actor/action columns sort | No sort params in OpenAPI |


**Non-sortable headers:** created_at, action, target, admin_id

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/audit/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--audit-cols`: 10rem 8rem minmax(10rem,1.2fr) 8rem |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| limit | limit | 50 |  |
| offset | offset | 0 |  |
| redact_pii | redact_pii | true | boolean; legacy default true |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/audit?limit=50&offset=0&redact_pii=true
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/audit_page.tsx | Compose: useSearchParams + fetch + sections |
| web/src/ui/audit/audit_directory.tsx | Section assembly |
| web/src/ui/audit/audit_toolbar.tsx | Toolbar region |
| web/src/ui/audit/audit_filter.tsx | FilterPanel draft + Apply |
| web/src/ui/audit/audit_grid.tsx | role=grid + rows |
| web/src/ui/audit/audit_directory.module.css | Page section grid |
| web/src/ui/audit/audit_grid.module.css | Column template |
| web/src/helpers/audit_api.ts | list(params, signal) |
| web/src/types/generated/openapi.d.ts | Generated types |
| web/src/helpers/api_blob.ts | Export blob + truncation headers |
| web/src/ui/audit/audit_export_banner.tsx | Truncation notice |


**Remove from this route (legacy):** ../components/filter_toolbar.js, <table> data-table, page state page index without URL offset.

**Legacy page:** `audit_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Invented total field | ListEnvelope lie — header only |
| Actor/action filter UI | Not in OpenAPI — blocked |
| Silent catch → empty | ErrorBlock on list fail |
| Export without format=csv | auditExport requires format enum csv |
| Ignore export truncation | Show X-Next-Cursor banner |
| redact_pii not in URL | Toggle must refetch with query param |
| <table> instead of grid | ui.mdc violation |
| Wrapper stack | No AuditDirectoryChrome |
| Page-only pagination | URL offset/limit sync |
| PII expand client-side | Respect redact_pii; no manual unmask |
| Flex page layout | CSS Grid sections |
| Duplicate ErrorBlock | Early return vs inline — one pattern |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/platform.yaml | Confirm auditList + auditExport params | openapi_gate |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/audit_api.ts | listAudit parses X-Total-Count | unit compile |
| 4 | web/src/ui/audit/* | Grid + toolbar + redact toggle | surface gate |
| 5 | web/src/pages/audit_page.tsx | URL sync offset + compose | no <table> |
| 6 | Export flow | api_blob + truncation banner | manual CSV download |
| 7 | web/src/app_routes.tsx | Route /audit | loads |
| 8 | Legacy cleanup | Remove filter_toolbar/table | rg data-table web/src/pages/audit_page.tsx empty |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'X-Total-Count' web/src/helpers/audit_api.ts
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No client sort | rg 'table_sort\|sortRows' web/src/pages/audit_page.tsx web/src/ui/audit/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| Header total | Manual: compare X-Total-Count vs row count | total matches header |
| redact_pii | Manual: toggle off | refetch with redact_pii=false |
| Export | Manual: Export CSV | file downloads; truncation banner if header set |
| Pagination | Manual: offset in URL | refetch new page |
| Permission | Manual: session without audit:read | route hidden or forbidden |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `audit_page.tsx` replaced or delegates to `ui/audit/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/audit/`, helpers, and page compose in one slice; no half migration.


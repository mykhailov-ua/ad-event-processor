# ADMIN_REPORTS_HUB_MILESTONE

Reports catalog hub at /reports.

**Status:** DRAFT  
**Slug:** `admin_reports_hub`  
**Depends on:** admin_page_chrome  
**Blocks:** all ADMIN_REPORT_MILESTONE_*  
**Pattern:** admin_report_pattern  
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
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_report_pattern |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- ghost-impression-funnel in REPORT_CATALOG keys
- live:true on retired report keys
- live:true without registered SPA route

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/reports` — cards from GET /api/v1/reports/catalog rows[]
- `live: true` in `web/src/models/report.ts` REPORT_CATALOG only when dedicated SPA route + GET handler exist
- Retired keys (`campaign-unit-economics`, `source-margin`) render RETIRED_REPORT_ALTS link — never live:true
- Non-live cards probe stub via probeStubReport; no false live navigation
- Hub export: POST /api/v1/reports/jobs with ReportJobSpec.report_key + poll/download flow
- Saved views panel: GET /api/v1/views?customer_id= when session scoped

### Out of scope

- Individual report grids (separate ADMIN_REPORT_MILESTONE_* specs)
- Client-side report catalog filtering beyond API permissions
- ghost-impression-funnel as catalog card key (canonical: silent-reject-impression-funnel)

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Catalog | GET /api/v1/reports/catalog | ReportCatalogRow: key, title, description, category, required_permissions, export_formats |
| Live flag | web/src/models/report.ts REPORT_CATALOG | live:true only when route + handler shipped |
| Retired alts | RETIRED_REPORT_ALTS | campaign-unit-economics → /reports/placements; source-margin → /reports/source-quality |
| Export | POST /api/v1/reports/jobs | ReportJobSpec: customer_id, report_key, from, to, format |
| Job poll | GET /api/v1/reports/jobs/{id} | ReportJobStatus.status until complete\|failed |
| Download | GET /api/v1/reports/jobs/{id}/download | CSV/JSON when job complete |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Reports | static |
| toolbar | ReportsHubToolbar | Date preset + export selected report_key | POST /api/v1/reports/jobs |
| saved | SavedViewsPanel | Pinned saved views for customer | GET /api/v1/views |
| cards | ReportCatalogCards | Card per catalog row.key; icon from reportIcon() | GET /api/v1/reports/catalog |
| retired | RetiredReportAlt | Link to canonical alt per RETIRED_REPORT_ALTS | report.ts static map |
| error | ErrorBlock | Catalog or saved-views fetch failure | fetch error |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/reports` |
| `live` | true |
| Handler | internal/reports/catalog.go — getReportCatalog |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/reports/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Report rows | Handler row keys | No client aggregation |
| Export | POST `/api/v1/reports/jobs` + poll | When API supports |


Fetch example:

```
GET /api/v1/reports/catalog
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/ui/reports/reports_hub.tsx | Hub cards + export toolbar |
| web/src/pages/report_hub_page.tsx | Compose (migrate from legacy) |
| web/src/models/report.ts | REPORT_CATALOG, RETIRED_REPORT_ALTS, reportHref() |
| web/src/helpers/report_api.ts | submitReportExport, pollReportJob, downloadReportExport |

**Legacy page:** `report_hub_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| live without route | REPORT_CATALOG live:true requires app_routes.tsx path |
| Retired card navigates to dead route | retired:true cards use RETIRED_REPORT_ALTS only |
| ghost_* catalog keys | No ghost-impression-funnel card; use silent-reject-impression-funnel |
| Stub probe as live | probeStubReport before promoting card to live:true |
| Export toast before 202 | Poll job after POST /reports/jobs |
| Client catalog filter | Use API FilterReportCatalog permissions; no client RBAC |
| Demo KPI on hub | No hardcoded counts on cards |
| Silent catch → empty | ErrorBlock when catalog fetch fails |
| Catalog/handler key drift | catalog.go ReportCatalogEntries key matches report_keys.go export list |
| Buyer card leakage | buyer:true cards respect session customer scope |
| Flex hub layout | CSS Grid card region per ui.mdc |
| Double export chrome | One toolbar export flow; not per-card hidden POST |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | GET /api/v1/reports/catalog | Confirm ReportCatalogResponse handler | openapi_gate.sh |
| 2 | web/src/models/report.ts | REPORT_CATALOG live flags + RETIRED_REPORT_ALTS | Matches shipped routes |
| 3 | web/src/ui/reports/reports_hub.tsx | Cards from API or static catalog | Retired alts render |
| 4 | web/src/helpers/report_api.ts | Export job POST/poll/download helpers | Manual export flow works |
| 5 | web/src/app_routes.tsx | Register /reports hub route | Lazy import resolves |

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
| Catalog fetch | GET /api/v1/reports/catalog | rows[] keys match handlers |
| Retired alt | Manual: open campaign-unit-economics card | links to /reports/placements |
| Export POST | POST /api/v1/reports/jobs placements | 202 + job id |
| Export poll | GET /api/v1/reports/jobs/{id} | status reaches complete |
| Export download | GET /api/v1/reports/jobs/{id}/download | file downloads |
| live gate | bash scripts/ci/report_live_routes_gate.sh | exit 0 when web/ exists |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `report_hub_page.tsx` replaced or delegates to `ui/reports/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/reports/`, helpers, and page compose in one slice; no half migration.


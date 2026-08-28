# ADMIN_REPORT_MILESTONE_EDGE_PARITY_REPORT

Report: edge-parity.

**Route gap:** register `/reports/edge-parity` in `app_routes.tsx` before `live: true`.

**Status:** DRAFT  
**Slug:** `admin_report_edge_parity`  
**Depends on:** admin_page_chrome, ADMIN_REPORTS_HUB_MILESTONE  
**Blocks:** —  
**Pattern:** admin_report_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| ghost_* columns | Legacy naming in UI | silent_reject_* from handler |
| Client aggregation | useMemo reduce on rows | Display handler rows only |
| Export job lie | Toast before POST 202 | Poll job status until terminal |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Stub forever | report_stub_page never replaced | Dedicated ui/reports section |
| live without route | Catalog live before route | Register app_routes first |
| Generic column fiction | Invented keys not in OpenAPI/handler | Column list in section 4.5 |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_report_pattern |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Report wired" without GET handler for report key
- ghost_* column headers in new UI
- ghost-impression-funnel catalog title or column labels

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/reports/edge-parity` — Edge vs tracker parser parity drift
- API: `GET /api/v1/reports/edge-parity`
- Column contract: Handler row keys (ReportMapEnvelope): edge_ingress, tracker_events, divergence_pct, alert, blacklist_stale, edge_blocked_total, shard_mismatch_hint
- FilterPanel params ⊆ OpenAPI query list
- Export via POST /api/v1/reports/jobs + poll when export_supported
- Display silent_reject_* canonical names (no ghost_* UI labels)

### Out of scope

- Client row aggregation or compare-period math
- Rename handler columns in browser
- ghost-impression-funnel as user-visible title (legacy API alias only)

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET | GET /api/v1/reports/edge-parity | reportEdgeParity |
| Columns | Handler row keys (ReportMapEnvelope): edge_ingress, tracker_events, divergence_pct, alert, blacklist_stale, edge_blocked_total, shard_mismatch_hint | handler |
| Envelope | rows + freshness (+ next_cursor when paginated) | ReportMapEnvelope or typed response |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Report title from catalog or static (edge-parity) | GET /api/v1/reports/catalog or reportTitle() |
| freshness | FreshnessBadge | freshness.stale, freshness.ch_lag_seconds, freshness_label when present | envelope.freshness |
| filters | ReportFilterPanel | customer_id, from, to, compare, campaign_id, limit, offset, cursor | URL searchParams → query string |
| toolbar | ReportToolbar | Export → POST /api/v1/reports/jobs; optional saved views | jobs API + /api/v1/views when permitted |
| content | report_edge_parity | KPI tiles from envelope rows or summary fields | rows[] or typed response |
| footer | PaginationBar | limit/offset/cursor when API paginates | envelope next_cursor/total |
| error | ErrorBlock | Report fetch failure | fetch error |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/reports/edge-parity` |
| `live` | false |
| Handler | internal/reports/ — `reportEdgeParity` |

### 4.3 Layout and placement (grid contract)


**Non-sortable headers:** edge_ingress, tracker_events, divergence_pct, alert, blacklist_stale, edge_blocked_total, shard_mismatch_hint

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/reports/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| customer_id | customer_id |  | CustomerIdQueryRequired |
| from | from | now-7d | ReportFromQuery RFC3339 |
| to | to | now | ReportToQuery RFC3339 |
| compare | compare |  | ReportCompareQuery: previous\|1\|true |
| campaign_id | campaign_id |  | CampaignIdQuery optional |
| limit | limit | 50 | LimitQuery |
| offset | offset | 0 | OffsetQuery |
| cursor | cursor |  | CursorQuery |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |
| Report rows | Handler row keys | No client aggregation |
| Export | POST `/api/v1/reports/jobs` + poll | When API supports |


Fetch example:

```
/api/v1/reports/edge-parity?customer_id={uuid}&from={iso}&to={iso}
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/ui/reports/report_edge_parity.tsx | ReportKpiStrip section |
| web/src/ui/reports/report_edge_parity.module.css | Grid/heatmap CSS module |
| web/src/helpers/reports_api.ts | fetchReport('edge-parity', params) |
| web/src/pages/report_route_pages.tsx | Compose until ui/reports migration |

**Legacy page:** `ops_edge_parity_page.tsx (ops cross-link)`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| live without route | Register app_routes.tsx before REPORT_CATALOG live:true |
| Stub forever | Replace legacy report_stub_page / query shell for this key |
| Client aggregation | Display handler rows only; no useMemo reduce/sum |
| ghost_* column headers | Canonical silent_reject_* per naming.mdc / ui.mdc |
| Flex page layout | CSS Grid sections per ui.mdc admin_report_pattern |
| Silent catch → empty | ErrorBlock on blocking fetch failure |
| Export before 202 | Poll GET /api/v1/reports/jobs/{id} after POST jobs |
| Client compare math | compare=previous query param; delta from handler compare.* |
| Client pagination total | limit/offset/cursor/total from API envelope only |
| Filter debounce spam | Single refetch on Apply; not per keystroke |
| Catalog/API drift | Card key matches GET path; openapi_gate.sh green |
| Hand-written DTO | Column keys from openapi.d.ts; no invented row fields |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/ops_reports.yaml | Confirm `reportEdgeParity` params + response schema | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/reports_api.ts | fetchReport('edge-parity', params) | Compiles |
| 4 | web/src/app_routes.tsx | Register gap route `/reports/edge-parity` + nav/catalog entry | Route resolves |
| 5 | web/src/ui/reports/report_edge_parity.tsx | ReportKpiStrip section | check_ui_surface_gate.sh |
| 6 | web/src/models/report.ts | Set live:true only after route + handler wired | report_live_routes_gate.sh |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| Large row sets | Admin cold path | Server limit/offset/cursor windowing | react.mdc G9 |
| Date apply | Filter change | Single refetch on Apply | No debounce on calendar Apply |
| Export job | Cold path | POST jobs 202 → poll → download | Not synchronous GET export |

## 7. Verification (paste in PR)

```bash
Register `/reports/edge-parity` in app_routes before catalog live:true
curl -s '/api/v1/reports/edge-parity?customer_id={uuid}&from={iso}&to={iso}' | jq '.freshness'
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No ghost labels | rg 'ghost_' web/src/ui/reports/report_edge_parity | no matches |
| Freshness chip | Manual: load with CH lag | FreshnessBadge from envelope.freshness |
| URL refetch | Manual: change from/to Apply | single refetch; params in URL |
| Error state | Manual: block GET | ErrorBlock visible; no silent empty table |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `ops_edge_parity_page.tsx (ops cross-link)` replaced or delegates to `ui/reports/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/reports/`, helpers, and page compose in one slice; no half migration.


# ADMIN_OPS_MILESTONE_EDGE_PARITY

Ops: /ops/edge-parity.

**Status:** DRAFT  
**Slug:** `admin_ops_edge_parity`  
**Depends on:** admin_page_chrome  
**Blocks:** —  
**Pattern:** report  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Destructive no confirm | One-click retry/catchup/blacklist | apiConfirmed + ConfirmModal |
| Client metrics | Sum DLQ rows or chart points in browser | opsDashboardMetrics only |
| SSE-only refresh | No poll when stream drops | 30s poll + optional EventSource |
| 503 shards hard fail | Discard IncidentSnapshot in 503 body | Render degraded shard grid |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Fake retry success | Toast without 2xx | apiConfirmed |
| Demo incident KPIs | Hardcoded red tiles | opsDashboardSummary fields only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per report |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy ops_home_page monolith | Reuse legacy components/ | ui/ops/ sections |
| Skip confirm on blacklist dry-run | Dry-run bypasses confirm OK; POST must confirm | Separate code paths |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Ops wired" without OpenAPI opId for primary GET
- One-click POST catchup/retry/blacklist without confirm
- Client sum of metric points for KPI tiles

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/ops/edge-parity` — Parser/nginx vs gnet parity metrics (ops lens)
- API: GET /api/v1/reports/edge-parity
- Confirm modal on destructive/catchup/retry/blacklist mutations

### Out of scope

- Hot-path tracker changes
- Client-derived metrics or Prometheus math
- Mandatory SSE when operator disables stream

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Report | GET /api/v1/reports/edge-parity | parity metrics; differential_count=0 target |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Edge parity | static |
| filters | ReportFilterPanel | from/to range | URL params |
| content | EdgeParityGrid | differential_count rows | reports/edge-parity |
| freshness | FreshnessBadge | stale, ch_lag_seconds | envelope |
| links | DrillDownLinks | /reports/edge-parity | static |
| error | ErrorBlock | Report fetch failure | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/ops/edge-parity` |
| Permission | shards:read (read); shards:write (mutations) |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome → Toolbar → [Filters] → Content → [Footer] → ErrorBlock

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--ops-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/ops/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Primary fetch | GET /api/v1/reports/edge-parity | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/ops_edge_parity_page.tsx | Compose only |
| web/src/ui/ops/* | Ops sections + CSS modules |
| web/src/helpers/ops_api.ts | Ops API + apiConfirmed helpers |
| web/src/types/generated/openapi.d.ts | Generated types |

**Legacy page:** `ops_edge_parity_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| differential_count client math | Display handler rows only |
| ghost_* labels | Use silent_reject_* in linked fraud reports only |
| Destructive no confirm | apiConfirmed + ConfirmModal on catchup/retry/blacklist POST/DELETE |
| Silent catch → empty | ErrorBlock on blocking fetch; partial errors in AlertBanner |
| Client metric math | Prometheus series from opsDashboardMetrics only; no row sums |
| Flex page root | CSS Grid sections per ui.mdc |
| Toast before 2xx | apiConfirmed on POST catchup/retry/blacklist |
| Permission leak | Gate catchup/retry on shards:write; blacklist on shards:write |
| SSE required lie | SSE optional; poll fallback when stream unavailable |
| Invented ops fields | DTO fields from OpenAPI ops_reports.yaml only |
| live without route | Register app_routes + nav before catalog live |
| Copy legacy components/ | Rebuild ui/ops/; no FilterToolbar import from components/ |
| 503 as hard error | Shards 503 may carry IncidentSnapshot body — render degraded grid |
| Double chip chrome | StatusBadge one layer; symmetric padding |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/ops_reports.yaml | Confirm ops opIds + x-permissions | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/ops_api.ts | Typed fetch + apiConfirmed wrappers | Compiles |
| 4 | web/src/ui/ops/* | Sections per 4.1 grid contract | check_ui_surface_gate.sh |
| 5 | web/src/pages/ops_edge_parity_page.tsx | Thin compose; permission gates | No *.module.css on page |
| 6 | web/src/app_routes.tsx | Route + lazy import | Resolves |
| 7 | web/src/ui/shell/nav | Ops nav group links | RBAC hides unauthorized |
| 8 | Confirm wiring | catchup/retry/blacklist use confirm_ui | Manual: cancel keeps state |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |
| SSE reconnect | Ops home optional | Backoff + poll fallback | No tight reconnect loop |
| Parallel fan-out | Ops home sub-requests | Partial errors banner | No silent slot drop |

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Confirm catchup | Manual: Run catch-up → cancel | No POST; state unchanged |
| Confirm retry | Manual: DLQ retry → cancel | No POST |
| Confirm blacklist | Manual: Block IP → cancel | No POST |
| ErrorBlock | Manual: block primary GET | ErrorBlock visible |
| No client aggregation | rg 'reduce\(\|useMemo.*sum' web/src/ui/ops/ | no matches |
| Permission gate | Manual: session without shards:write | catchup/retry buttons hidden |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `ops_edge_parity_page.tsx` replaced or delegates to `ui/ops/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/ops/`, helpers, and page compose in one slice; no half migration.


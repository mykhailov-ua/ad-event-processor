# ADMIN_DASHBOARD_MILESTONE_OVERVIEW

Dashboard: Overview.

**Status:** DRAFT  
**Slug:** `admin_dashboard_overview`  
**Depends on:** admin_page_chrome  
**Blocks:** —  
**Pattern:** dashboard  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Demo KPIs | Hardcoded numbers in TSX | Every value from API DTO field |
| Client sum/avg | useMemo over campaigns[] or series[] | Handler aggregates only |
| RoleDashboard slop | additionalProperties:true treated as any | Import handler DTO via openapi-types |
| Date drift | from/to not in URL | useSearchParams sync on Apply |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Skeleton forever | Placeholder KPI cards | ErrorBlock or real data |
| Freshness invented | Chip without API field | Omit until freshness DTO present |
| Copy role_dashboard_page | Monolith reuse | ui/dashboard/<role> sections |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per dashboard |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy role_dashboard_page | Monolith reuse | ui/<domain>/ per-role section |
| Client portfolio math | utilization_pct in browser | Use API utilization_pct |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Dashboard wired" without handler path for primary API
- Demo KPI literals in production TSX
- ghost_* UI labels on fraud dashboard

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Route `/`
- Role-based nav cards from session; product meta — not a KPI dashboard API
- API: `GET /api/v1/meta; GET /api/v1/session` (`metaGet; sessionGet`)
- KPI keys: nav_groups, permissions, product_version
- FreshnessBadge from freshness DTO only
- RBAC: `session (unauthenticated -> login)` per OpenAPI x-permissions
- Cards deep-link to dashboards/reports per session permissions

### Out of scope

- Client aggregation of KPIs or series rollups
- Demo/hardcoded numbers
- Hand-written DTO diverging from handler json tags

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| No overview KPI endpoint | Legacy home is nav hub only | Link to role dashboards; no demo spend/revenue cards |

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Meta | GET /api/v1/meta | MetaResponse |
| Session | GET /api/v1/session | nav + permissions |
| No KPI API | — | Do not invent overview KPIs |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Overview / home | static |
| cards | NavCardGrid | repeat(auto-fill, minmax(14rem, 1fr)) | session nav filtered by permissions |
| meta | MetaStrip | version, environment hints | GET /api/v1/meta |
| error | ErrorBlock | session/meta fetch failure | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/` |
| Permission | session (unauthenticated -> login) |
| `live` | true |
| Handler | internal/platformadmin/handlers.go (meta); session handler |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome
  -> NavCardGrid (auto-fill minmax 14rem)
     [Dashboards] [Campaigns] [Reports] [Integrations] ...
  -> MetaStrip (optional footer)

| Invariant | Value |
| :--- | :--- |
| Page grid | CSS Grid on page root; no flex page layout |
| KPI grid | repeat(auto-fill, minmax(12rem, 1fr)) in rem |
| Money | formatMicro on *_micro fields; prefer *_display when API adds it |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/dashboard/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--dashboard-cols`: repeat(auto-fill, minmax(12rem, 1fr)) |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Primary fetch | GET /api/v1/meta; GET /api/v1/session | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |


Fetch example:

```
GET /api/v1/meta; GET /api/v1/session?customer_id={uuid}&from={iso}&to={iso}
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/overview_page.tsx | Thin compose; URL params -> fetch |
| web/src/ui/dashboard/overview_dashboard.tsx | KPI + sections |
| web/src/ui/dashboard/*.module.css | Section grids |
| web/src/helpers/dashboard_api.ts | Typed dashboard fetch |
| web/src/types/generated/openapi.d.ts | Generated types |

**Legacy page:** `overview_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Demo KPIs | Every card maps to handler json field; no literals |
| Client KPI math | No useMemo sum/avg over campaigns[] or series[] |
| Flex page root | CSS Grid sections per ui.mdc |
| Silent catch → empty | ErrorBlock on blocking fetch failure |
| Freshness invented | Use kpis.freshness or envelope freshness only |
| ghost_* labels | silent_reject_* canonical on fraud surfaces |
| Stale range ignored | Refetch when from/to URL params change |
| Wrapper stack | No *DashboardChrome around PageChrome |
| Nested chip chrome | FreshnessBadge rendered directly in PageChrome slot |
| live without handler | Route live only when GET op registered |
| OpenAPI RoleDashboard lie | RoleDashboard is additionalProperties; bind typed handler DTO |
| Series client rollup | Render series[] points as returned; no daily merge in browser |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | GET /api/v1/meta; GET /api/v1/session | Confirm operation_id + handler DTO fields | openapi_gate.sh + handler test |
| 2 | internal/platformadmin/handlers.go (meta); session handler | Cross-check json tags vs milestone KPI keys | grep json tags |
| 3 | make openapi-types | Regenerate openapi.d.ts | typecheck passes |
| 4 | web/src/helpers/dashboard_api.ts | Typed fetch with from/to/customer_id | Compiles |
| 5 | web/src/ui/dashboard/* | KpiGrid + sections per grid_ascii | check_ui_surface_gate.sh |
| 6 | web/src/pages/overview_page.tsx | Compose + URL param sync | Lazy import resolves |
| 7 | web/src/app_routes.tsx | Route + nav entry | report_live_routes honest |

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
| No demo KPIs | rg 'demo\|placeholder\|12345' web/src/ui/dashboard/ | no hardcoded KPI literals |
| KPI key parity | rg 'nav_groups' web/src/ui/dashboard/ | maps to API field name |
| Error state | Manual: block API | ErrorBlock visible; no empty KPI grid |
| Freshness chip | Manual: stale response | FreshnessBadge uses API stale/as_of |
| URL range | Manual: change from/to | Refetch; period in response updates |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `overview_page.tsx` replaced or delegates to `ui/dashboard/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/dashboard/`, helpers, and page compose in one slice; no half migration.


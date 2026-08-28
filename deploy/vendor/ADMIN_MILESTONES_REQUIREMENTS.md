# Admin UI milestones — requirements catalog

**Status:** DRAFT (requirements catalog; **98** per-milestone specs; deep sections 1.1–4.7; regen: `python3 scripts/dev/gen_admin_milestone_specs.py`)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`  
**Page routes and APIs:** `admin_web_pages_backlog.md`  
**Template skeleton:** `MILESTONE_TEMPLATE.md`  
**Index:** `admin_ui_redesign_backlog.md`

Each milestone has a spec file under `deploy/vendor/`. Implementation is **not startable** until spec status is `REVIEW`, page-specific 4.3 is complete (extend `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS.md` depth where needed), and section 7 verification is pasted on ship.

---

## Milestone dependency graph

```
admin_contract_gate
  └── admin_tokens
        └── admin_shell
              ├── admin_page_chrome
              │     ├── admin_directory_pattern → ADMIN_DIRECTORY_MILESTONE_<PAGE>
              │     ├── admin_detail_pattern    → ADMIN_DETAIL_MILESTONE_<PAGE>
              │     └── admin_report_pattern    → ADMIN_REPORT_MILESTONE_<REPORT>
              ├── admin_integrations_hub      → ADMIN_INTEGRATIONS_MILESTONE_<AREA>
              ├── ADMIN_DASHBOARD_MILESTONE_<ROLE>
              ├── ADMIN_OPS_MILESTONE_<PAGE>
              ├── ADMIN_SETTINGS_MILESTONE_<PAGE>
              └── ADMIN_SELFERVE_MILESTONE_<PAGE>
admin_campaigns_migrate (after admin_directory_pattern + API)
ADMIN_FRAUD_MILESTONE_ADMIN (after admin_detail_pattern)
```

---

## Global requirements (every UI milestone)

| ID | Requirement |
| :--- | :--- |
| G1 | Cold path: server owns filter, sort, aggregate, labels (`ui.mdc`) |
| G2 | Layout: CSS Grid sections only; no flex on page/section (`ui.mdc`) |
| G3 | CSS: `*.module.css` under `web/src/ui/<domain>/` only; pages ≤ ~120 lines (`frontend-modular.mdc`) |
| G4 | Types: `web/src/types/generated/openapi.d.ts`; no invented PATCH fields |
| G5 | Errors: `ErrorBlock` on blocking fetch fail; no silent empty table |
| G6 | `live: true` / nav link only when `/api/v1` handler exists |
| G7 | Pitfalls table 4.7 filled (`ui-backlog.mdc`) |
| G8 | One PR = one page + one `ui/<domain>/` + one helper unless spec says split |
| G9 | Lists 500+ rows: server pagination + `react.mdc` windowing/prefetch |
| G10 | No new `web/src/components/` monolith; use `ui/system/` or domain folder |

---

# Foundation milestones

## `admin_contract_gate`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_CONTRACT_MILESTONE_GATE.md` |
| **Depends on** | — |
| **Blocks** | `admin_tokens`, all page milestones |
| **Pattern** | gate (no UI page) |

### Purpose

OpenAPI is the single wire contract for admin UI types and route catalog. Handlers and YAML stay in sync before any TS field work.

### In scope

- `api/openapi/openapi.yaml` and path fragments match live `cmd/control` handlers
- `make openapi-types` → `web/src/types/generated/openapi.d.ts`
- Catalog test: every registered `/api/v1` admin route documented or allowlisted
- `report_live_routes_gate.sh` inputs: route registry vs OpenAPI report keys

### Out of scope

- React components, embed, Docker
- New handler behavior (separate backend milestone)

### Requirements

| ID | Requirement | Done when |
| :--- | :--- | :--- |
| C1 | `bash scripts/ci/openapi_gate.sh` exits 0 | CI green |
| C2 | No handler-only DTO fields missing from OpenAPI on touched routes | Spectral + catalog test |
| C3 | `ListEnvelope` shape documented for directory handlers (`items`, `total`, `limit`, `offset`, `freshness`, `filters_applied`, `sort`) | Schema in OpenAPI |
| C4 | Report routes in catalog match `GET /api/v1/reports/*` paths | `admin_web_pages_backlog.md` cross-check |
| C5 | `x-permissions` on mutating operations where handler enforces RBAC | OpenAPI review |

### Verification

```bash
bash scripts/ci/openapi_gate.sh
make openapi-types
cd web && npm run typecheck
```

---

## `admin_tokens`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_TOKENS_MILESTONE.md` |
| **Depends on** | `admin_contract_gate` |
| **Blocks** | `admin_shell` |
| **Pattern** | shell (design system only) |

### Purpose

Design tokens in `rem` only. No page layout; establishes variables consumed by `ui/system/` and domain modules.

### In scope

- `web/src/styles/tokens.css` — spacing, typography, color semantic tokens
- Document token names in milestone section 4.4
- Remove hard-coded hex from **new** CSS modules (legacy globals may remain until migrated)

### Out of scope

- Components, routes, API

### Requirements

| ID | Requirement | Done when |
| :--- | :--- | :--- |
| T1 | Spacing scale: `--space-1` … `--space-*` in `rem` only | No px spacing in tokens file |
| T2 | Typography: `--text-*-leading`, font size tokens | Referenced by system components |
| T3 | Semantic colors: surface, border, text, danger, success — no product hex in domain CSS | `check_ui_literals` if wired |
| T4 | Sidebar width `--sidebar-width` documented | `shell.css` or module uses var |
| T5 | Chip/badge tokens: symmetric padding variable | `ui.mdc` chips section |

### Verification

```bash
bash scripts/ci/admin_web.sh
rg '#[0-9a-fA-F]{3,8}' web/src/ui --glob '*.module.css' || true
```

---

## `admin_shell`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_SHELL_MILESTONE.md` |
| **Depends on** | `admin_tokens` |
| **Blocks** | all authenticated pages |
| **Pattern** | shell |

### Purpose

Login, bootstrap, app shell (sidebar, nav, session guard), embed `web/dist` in control binary.

### Routes (see `admin_web_pages_backlog.md` W0)

| Route | APIs |
| :--- | :--- |
| `/login` | `POST /api/v1/session` |
| `/bootstrap` | `GET/POST /api/v1/settings/platform/bootstrap` |
| `/install/done` | static |
| guarded `/*` | `GET /api/v1/session`, `GET /api/v1/meta`, EULA, license |

### 4.1 Page inventory

| Region | Requirement |
| :--- | :--- |
| Login | email, password, submit, API error text, redirect on 2xx |
| Bootstrap | platform bootstrap fields from OpenAPI only |
| Shell sidebar | nav groups; collapse; search optional; user footer |
| Session guard | loading → authenticated / login redirect / `ForbiddenPage` |
| EULA gate | modal or gate page; `POST /api/v1/eula/accept` before app |
| `#root` | empty in `index.html` / `login.html` |

### 4.3 Layout

- Shell: grid `sidebar | main` — sidebar fixed width token; main is outlet for routes
- No flex on page-level sections inside main (delegated to PageChrome milestone)

### 4.6 File map

| Path | Role |
| :--- | :--- |
| `web/src/ui/shell/` | sidebar, layout, login form |
| `web/src/pages/login_page.tsx` | thin compose |
| `web/src/app_boot.tsx`, `app_routes.tsx` | guard + router |
| `web/embed.go` | serve `web/dist` |
| `internal/controlplane/admin_static_stub/` | removed when dist ships |

### Requirements

| ID | Requirement | Done when |
| :--- | :--- | :--- |
| S1 | `go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1` | pass |
| S2 | Login POST session; cookie set; redirect `/` | manual |
| S3 | Unauthenticated → `/login`; RBAC fail → forbidden page | manual |
| S4 | Nav items respect role (session or static map documented) | manual |
| S5 | `cd web && npm run typecheck` | pass |
| S6 | No text inside `<div id="root">` in boot HTML | static route test |

### Verification

```bash
cd web && npm run typecheck
bash scripts/ci/admin_web.sh
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
```

---

## `admin_page_chrome`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_PAGE_CHROME_MILESTONE.md` |
| **Depends on** | `admin_shell` |
| **Blocks** | directory, detail, report, dashboard pages |
| **Pattern** | shell (shared chrome) |

### Purpose

Reusable page frame: ContextBar, PageChrome, Toolbar slot, Footer slot, system primitives.

### 4.1 Components (`web/src/ui/system/` + `web/src/ui/shell/page_chrome.tsx`)

| Component | Requirement |
| :--- | :--- |
| `PageChrome` | title slot; optional badge slot (direct child — no wrapper chrome) |
| `FreshnessBadge` | `freshness_label`, `stale` from API; one layer chip |
| `ErrorBlock` | API error message + retry |
| `PageSkeleton` | grid-shaped skeleton |
| `EmptyState` | one sentence + action when filters inactive |
| `Button` | replaces `btn--*` BEM on pages |
| `PaginationBar` | prev/next + range; emits offset only |

### Requirements

| ID | Requirement | Done when |
| :--- | :--- | :--- |
| P1 | Section stack documented: ContextBar → PageChrome → Toolbar → Content → Footer | milestone 4.3 |
| P2 | Chip: symmetric `padding: var(--space-1)`; no nested frame | visual + `ui.mdc` |
| P3 | Field `Select` / listbox: inline drop, wrapper `width: 100%` | dropdown contract |
| P4 | `bash scripts/ci/check_ui_surface_gate.sh` | pass on sample page |

### Verification

```bash
bash scripts/ci/check_ui_surface_gate.sh
cd web && npm run typecheck
```

---

# Pattern milestones (templates for pages)

## `admin_directory_pattern`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_DIRECTORY_MILESTONE_<PAGE>.md` (one per page) |
| **Depends on** | `admin_page_chrome` |
| **Example pages** | customers, campaigns, audit, billing invoices, rtb deals, flows |

### Purpose

Server-paginated list: URL params drive fetch; grid renders `items` without client full-list sort/filter.

### 4.1 Required regions

| Region | Requirement |
| :--- | :--- |
| PageChrome | title; `freshness_label` chip if list envelope includes freshness |
| Toolbar | primary actions (create, export) — permission-gated |
| FilterPanel | draft → Apply copies to URL; fields map 1:1 to query params |
| Content | `role="grid"` + subgrid; columns in `--<page>-cols` |
| Footer | PaginationBar: `limit`, `offset`, `total` from API |

### 4.5 API contract

- `GET` list endpoint with `limit`, `offset`, optional `q`, `sort`, `order`, domain filters
- Response: `ListEnvelope` or documented list DTO with `items`, `total`
- Display: `*_display`, `status_label`, `status_tone` from handler

### 4.7 Pitfalls (mandatory)

| Pitfall | Prevention |
| :--- | :--- |
| Client `sortRows` / `filter` on `items` | forbidden; URL refetch |
| Portal filter listbox | inline drop only |
| Double freshness chrome | badge in PageChrome slot only |

### Implementation order (section 5)

1. Confirm OpenAPI list operation + handler
2. `make openapi-types`
3. `helpers/<domain>_api.ts` — `list(params, signal)`
4. `ui/<domain>/<domain>_directory.tsx` + grid + CSS modules
5. `pages/<domain>_page.tsx` — `useSearchParams` + compose
6. Register route + nav

### Verification

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
```

### Page instances

| Milestone file | Route | Primary API |
| :--- | :--- | :--- |
| `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS` | `/customers` | `GET /api/v1/customers` | **Spec:** `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS.md` |
| `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS` | `/campaigns` | `GET /api/v1/campaigns` |
| `ADMIN_DIRECTORY_MILESTONE_FLOWS` | `/campaigns/flows` | `GET /api/v1/flows` |
| `ADMIN_DIRECTORY_MILESTONE_AUDIT` | `/audit` | `GET /api/v1/audit` |
| `ADMIN_DIRECTORY_MILESTONE_BILLING` | `/billing` | `GET /api/v1/billing/invoices` |
| `ADMIN_DIRECTORY_MILESTONE_RTB_DEALS` | `/rtb/deals` | `GET /api/v1/rtb/deals` |

Each instance copies this pattern + column spec in 4.3 from `admin_web_pages_backlog.md`.

---

## `admin_detail_pattern`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_DETAIL_MILESTONE_<PAGE>.md` |
| **Depends on** | `admin_page_chrome`; directory milestone for same entity recommended |
| **Example pages** | customer, campaign, invoice, settings platform |

### Purpose

GET + PATCH (or PUT) detail: form fields ⊆ Go struct; toast only after 2xx (`apiConfirmed`).

### 4.1 Required regions

| Region | Requirement |
| :--- | :--- |
| PageChrome | entity title; status chip from `status_label` |
| Toolbar | Save, Publish, Delete — per OpenAPI permissions |
| Content | form sections grouped by domain; read-only stats panels separate |
| Footer | optional sticky save bar |

### 4.5 API contract

- `GET /api/v1/.../{id}` — full DTO
- `PATCH` body fields exactly match OpenAPI `Patch*Request`
- Validation errors: 400 field map rendered per field
- Sub-resources: separate tabs call sub-GETs (ledger, stats, fraud)

### 4.7 Pitfalls

| Pitfall | Prevention |
| :--- | :--- |
| TS-only form fields | every field on Go PATCH + OpenAPI first |
| Success toast before 2xx | `apiConfirmed` pattern |
| Client-derived KPIs | use handler stats endpoints |

### Page instances

| Milestone file | Route | APIs |
| :--- | :--- | :--- |
| `ADMIN_DETAIL_MILESTONE_CUSTOMER` | `/customers/:id` | customers + billing sub-routes |
| `ADMIN_DETAIL_MILESTONE_CAMPAIGN` | `/campaigns/:id` | campaign CRUD, stats, fraud, publish |
| `ADMIN_DETAIL_MILESTONE_INVOICE` | `/billing/invoices/:id` | invoice, pdf, deliveries |
| `ADMIN_DETAIL_MILESTONE_RTB` | `/rtb/integration` | integration-profile, shadow-diff |
| `ADMIN_DETAIL_MILESTONE_FLOW_BUILDER` | `/campaigns/flows/:id/builder` | flows PATCH |
| `ADMIN_DETAIL_MILESTONE_LANDER` | `/campaigns/landers/:id/editor` | hosted-editor |
| `ADMIN_DETAIL_MILESTONE_TELEGRAM` | `/campaigns/:id/telegram` | telegram bots/postbacks |
| `ADMIN_DETAIL_MILESTONE_WIZARD` | `/campaigns/wizard` | wizard session |
| `ADMIN_SETTINGS_MILESTONE_PLATFORM` | `/settings` | platform settings |
| `ADMIN_SETTINGS_MILESTONE_LICENSE` | `/settings/license` | license status/apply |
| `ADMIN_SETTINGS_MILESTONE_DOMAINS` | `/settings/domains` | domains CRUD |
| `ADMIN_SETTINGS_MILESTONE_TEAM` | `/team` | team members, approvals |
| `ADMIN_SETTINGS_MILESTONE_SUPPORT` | `/support/feedback` | feedback POST |

---

## `admin_report_pattern`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_REPORT_MILESTONE_<REPORT>.md` |
| **Depends on** | `admin_page_chrome` |
| **Hub** | `/reports` uses `GET /api/v1/reports/catalog` |

### Purpose

CH-backed report: date range and filters in URL; grid or heatmap; freshness; optional compare period.

### 4.1 Required regions

| Region | Requirement |
| :--- | :--- |
| PageChrome | report title from catalog; freshness chip |
| FilterPanel | `from`, `to` (RFC3339); `campaign_id`, `customer_id` when in OpenAPI; compare toggle |
| Content | grid or heatmap per report shape; `role="grid"` for tables |
| Toolbar | Export → `POST /api/v1/reports/jobs` + poll + download |

### 4.5 API contract

- `GET /api/v1/reports/<report-key>` with documented query params
- Rows use server column keys — no rename to legacy `ghost_*` in UI (`silent_reject_*` canonical)
- `stale`, `freshness_label`, `ch_lag_seconds` displayed from envelope

### 4.6 Performance (`react.mdc`)

- Large row sets: server limit or pagination if API supports; else row windowing
- Date change → single refetch; debounce not on calendar apply

### 4.7 Pitfalls

| Pitfall | Prevention |
| :--- | :--- |
| `live: true` on catalog card without route | register route or mark not live |
| Stub page forever | `report_stub_page` only until milestone ships |
| Client aggregation of rows | display handler rows only |

### Report milestone files (one per report route)

Create `ADMIN_REPORT_MILESTONE_<SLUG>` where `<SLUG>` is UPPER_SNAKE of report key:

| Report key | Route | API |
| :--- | :--- | :--- |
| placements | `/reports/placements` | `GET .../placements` |
| keywords | `/reports/keywords` | `GET .../keywords` |
| pacing-drift | `/reports/pacing-drift` | `GET .../pacing-drift` |
| filter-rejects | `/reports/filter-rejects` | `GET .../filter-rejects` |
| fraud-breakdown | `/reports/fraud-breakdown` | `GET .../fraud-breakdown` |
| silent-reject-impression-funnel | `/reports/silent-reject-impression-funnel` | `GET .../silent-reject-impression-funnel` |
| spend-velocity | `/reports/spend-velocity` | `GET .../spend-velocity` |
| daypart-heatmap | `/reports/daypart-heatmap` | `GET .../daypart-heatmap` |
| campaign-geo-device | `/reports/campaign-geo-device` | `GET .../campaign-geo-device` |
| geo-roi | `/reports/geo-roi` | `GET .../geo-roi` |
| source-quality | `/reports/source-quality` | `GET .../source-quality` |
| ivt-by-source | `/reports/ivt-by-source` | `GET .../ivt-by-source` |
| click-log | `/reports/clicks` | `GET .../click-log` |
| conversion-type-payout | `/reports/conversion-type-payout` | `GET .../conversion-type-payout` |
| postback-reconciliation | `/reports/postback-reconciliation` | `GET .../postback-reconciliation` |
| rtb-overview | `/reports/rtb/overview` | `GET .../rtb/overview` |
| rtb-no-bid-reasons | `/reports/rtb/no-bid-reasons` | `GET .../rtb/no-bid-reasons` |
| rtb-geo-device | `/reports/rtb/geo-device` | `GET .../rtb/geo-device` |
| traffic-sources | `/reports/traffic-sources` | `GET .../traffic-sources` |
| discrepancy-buy-sell | `/reports/discrepancy-buy-sell` | `GET .../discrepancy-buy-sell` |
| true-roi | `/reports/true-roi` | `GET .../true-roi` |
| cost-sync-coverage | `/reports/cost-sync-coverage` | `GET .../cost-sync-coverage` |
| campaign-overview | `/reports/campaign-overview` | `GET .../campaign-overview` |
| customer-portfolio | `/reports/customer-portfolio` | `GET .../customer-portfolio` |
| data-quality | `/reports/data-quality` | `GET .../data-quality` |
| edge-parity | `/reports/edge-parity` or ops | `GET .../edge-parity` |
| telegram summary/funnel/bots/premium/fraud | `/reports/telegram/*` | `GET .../telegram/*` |

**Gap reports (milestone required before `live: true`):**  
`layer-desync-summary`, `layer-desync-drilldown`, `fraud-evidence-pack`, `signal-effectiveness`, `rtt-split-tunnel`, `campaign-toggle-cohort`, `wire-signal-breakdown`, `customer-fraud-*`, `ml/*` — add route in `app_routes.tsx` per `admin_web_pages_backlog.md`.

---

## `admin_integrations_hub`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_INTEGRATIONS_MILESTONE_HUB.md` (landing) + `ADMIN_INTEGRATIONS_MILESTONE_<AREA>.md` |
| **Depends on** | `admin_shell` |
| **Pattern** | hub |

### Hub page `/integrations` (if routed) or nav group only

- Cards linking to integration subpages
- `StubBanner` on 501 endpoints

### Sub-milestones

| Milestone | Route | APIs | Requirements summary |
| :--- | :--- | :--- | :--- |
| `ADMIN_INTEGRATIONS_MILESTONE_COST_SYNC` | `/integrations/cost-sync` | cost-sync networks, credentials, run, history | credential forms per network schema; run + history grid |
| `ADMIN_INTEGRATIONS_MILESTONE_POSTBACKS` | `/integrations/postbacks` | postbacks config, dlq, test | per-campaign config; DLQ retry |
| `ADMIN_INTEGRATIONS_MILESTONE_SCHEMAS` | `/integrations/schemas` | integration schemas CRUD, apply | directory + apply modal |
| `ADMIN_INTEGRATIONS_MILESTONE_TEMPLATES` | `/integration/templates/import` | templates catalog, import | import bundle flow |
| `ADMIN_INTEGRATIONS_MILESTONE_SUPPLY` | `/integrations/supply` | sellers, ads-txt, validation | dual grid sellers + ads.txt |
| `ADMIN_INTEGRATIONS_MILESTONE_MARGIN_GUARD` | `/integrations/margin-guard` | policies, activity, overrides | policy editor + activity |
| `ADMIN_INTEGRATIONS_MILESTONE_SMART_ALERTS` | `/integrations/smart-alerts` | rules, history, ack | rule CRUD + event ack |
| `ADMIN_INTEGRATIONS_MILESTONE_AUTOMATION` | `/integrations/automation` | presets, rules, dry-run | rule editor + dry-run panel |
| `ADMIN_INTEGRATIONS_MILESTONE_PLATFORM_CAMPAIGNS` | gap: new route | platform-campaigns links, sync | link external campaigns; pause/resume/budget |

Each sub-milestone: hub pattern or directory + detail mix per `admin_web_pages_backlog.md` W4.

---

## `admin_campaigns_migrate`

| Field | Value |
| :--- | :--- |
| **Spec file** | `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE.md` |
| **Depends on** | `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS`, API ready |
| **Route** | `/campaigns/migrate` |

### Requirements

| ID | Requirement |
| :--- | :--- |
| M1 | Source picker: `GET /api/v1/campaigns/migrate/sources` |
| M2 | Preview table: `POST .../migrate/preview` or pull preview |
| M3 | Import confirm: `POST .../migrate/import` with idempotency |
| M4 | Error rows from API displayed; no client validation of business rules |
| M5 | Step wizard in URL or explicit step state; back/next |

### APIs

`migrate/sources`, `migrate/preview`, `migrate/import`, `migrate/pull/preview`, `migrate/pull/import`

---

# Dashboard milestones

| Milestone | Route | API | Requirements |
| :--- | :--- | :--- | :--- |
| `ADMIN_DASHBOARD_MILESTONE_OVERVIEW` | `/` | `GET /api/v1/meta` | role-based links; no demo KPIs |
| `ADMIN_DASHBOARD_MILESTONE_BUYER` | `/dashboards/buyer`, `/campaigns/portfolio` | `GET /api/v1/dashboards/buyer` | KPI cards from DTO; drill-down links |
| `ADMIN_DASHBOARD_MILESTONE_ADOPS` | `/dashboards/adops` | `GET .../adops` | pacing/delivery KPIs |
| `ADMIN_DASHBOARD_MILESTONE_CFO` | `/dashboards/cfo` | `GET .../cfo` | spend/ledger highlights |
| `ADMIN_DASHBOARD_MILESTONE_FRAUD` | `/dashboards/fraud` | `GET .../fraud` | fraud KPIs |
| `ADMIN_DASHBOARD_MILESTONE_OPERATOR` | `/dashboards/operator` | `GET .../operator` | stack health tiles |
| `ADMIN_DASHBOARD_MILESTONE_CAMPAIGN` | `/dashboards/campaign/:id` | `GET .../campaign/{id}` | single campaign KPI strip |

**Pattern:** dashboard = PageChrome + KPI grid (CSS grid) + optional table; all numbers from API; freshness on stale data.

---

# Ops milestones

| Milestone | Route | API | Key UI requirements |
| :--- | :--- | :--- | :--- |
| `ADMIN_OPS_MILESTONE_HOME` | `/ops` | dashboard summary, metrics, stream | incident tiles; optional SSE |
| `ADMIN_OPS_MILESTONE_SHARDS` | `/ops/shards` | `ops/shards`, catchup | slot map; catchup action with confirm |
| `ADMIN_OPS_MILESTONE_DLQ` | `/ops/dlq` | dlq, inbox, retry | two grids; retry per row |
| `ADMIN_OPS_MILESTONE_DOMAINS` | `/ops/domains` | rotation, tls-allowed | policy form + host list |
| `ADMIN_OPS_MILESTONE_BLACKLIST` | `/ops/blacklist` | blacklist CRUD | add IP; TTL display |
| `ADMIN_OPS_MILESTONE_RECON` | `/ops/recon` | `recon/runs` | run history directory |
| `ADMIN_OPS_MILESTONE_CONSENT` | `/ops/consent` | consent proofs | proof list |
| `ADMIN_OPS_MILESTONE_ML` | `/ops/ml-model` | ml-model, eval, labels | status panel; eval trigger |
| `ADMIN_OPS_MILESTONE_EDGE_PARITY` | `/ops/edge-parity` | `reports/edge-parity` | parity metrics grid |

**Embed gaps into HOME or follow-up:** `ops/doctor`, `ops/outbox`, `ops/incidents`, `ops/rum`.

---

# Self-serve and publisher

| Milestone | Route | API | Requirements |
| :--- | :--- | :--- | :--- |
| `ADMIN_SELFERVE_MILESTONE_HOME` | `/selfserve` | selfserve portfolio | buyer landing; Bearer note in docs only |
| `ADMIN_SELFERVE_MILESTONE_BILLING` | `/selfserve/billing` | statement, invoices | read-only money from API |
| `ADMIN_SELFERVE_MILESTONE_KEYS` | `/selfserve/api-keys` | api-keys CRUD | show `raw_key` once on create only |
| `ADMIN_SELFERVE_MILESTONE_CREATE` | `/selfserve/campaigns/new` | selfserve campaigns | minimal create form |
| `ADMIN_PUBLISHER_MILESTONE` | `/publisher` | publisher dashboard, statements | scoped publisher KPIs |

---

# Fraud admin

| Milestone | Route | APIs | Requirements |
| :--- | :--- | :--- | :--- |
| `ADMIN_FRAUD_MILESTONE_ADMIN` | gap: `/fraud/decisions` (proposed) | fraud decisions, labels, overrides, presets, integrations | directory + detail; link from ops nav |
| (embedded) | `/campaigns/:id` fraud tab | `campaigns/{id}/fraud`, preview | part of `ADMIN_DETAIL_MILESTONE_CAMPAIGN` |

---

# Settings gaps (API ready, milestone TBD)

| Milestone | Route | API |
| :--- | :--- | :--- |
| `ADMIN_SETTINGS_MILESTONE_DISPUTES` | `/settings/disputes` | `GET /api/v1/disputes` |
| `ADMIN_SETTINGS_MILESTONE_REPORT_SCHEDULES` | `/settings/report-schedules` | `GET/POST /api/v1/report-schedules` |
| `ADMIN_DIRECTORY_MILESTONE_BRANDS` | `/brands` | `GET/POST /api/v1/brands`, creatives |

---

# Definition of done (all milestones)

- [ ] `ADMIN_<SLUG>_MILESTONE.md` sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] Global requirements G1–G10 satisfied
- [ ] Verification commands pasted in PR with exit code 0
- [ ] No `live: true` without backend route
- [ ] Legacy page replaced or thin-wrapper delegating to `ui/<domain>/`
- [ ] Commit title names concrete surface (`core.mdc`)

---

# Suggested ship order (full stack)

1. `admin_contract_gate`  
2. `admin_tokens`  
3. `admin_shell`  
4. `admin_page_chrome`  
5. `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS` (reference directory)  
6. `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS`  
7. `ADMIN_DETAIL_MILESTONE_CAMPAIGN`  
8. `ADMIN_DETAIL_MILESTONE_CUSTOMER`  
9. `ADMIN_DASHBOARD_MILESTONE_BUYER` + operator  
10. `ADMIN_INTEGRATIONS_MILESTONE_COST_SYNC` (first integration)  
11. `ADMIN_REPORT_MILESTONE_PLACEMENTS` (reference report)  
12. Remaining reports in catalog order  
13. Ops block (`ADMIN_OPS_MILESTONE_HOME` first)  
14. Settings + team + audit  
15. Self-serve block  
16. `admin_campaigns_migrate`  
17. `ADMIN_FRAUD_MILESTONE_ADMIN`  
18. Gap pages (disputes, schedules, brands, platform-campaigns, ML reports)

---

## Related

- `admin_ui_redesign_backlog.md` — short index  
- `admin_web_pages_backlog.md` — per-route content  
- `ui-backlog.mdc` — spec section schema  
- `MILESTONE_TEMPLATE.md` — copy for each `ADMIN_*_MILESTONE.md` file  

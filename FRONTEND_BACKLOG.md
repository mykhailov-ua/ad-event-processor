# Admin frontend backlog (dashboard refactor)

Scope: `web/` SPA aligned with tracker-operator analytics (Keitaro-style buyer home), then adjacent surfaces.

Baseline (2026-09-01): ~90 routes in `web/src/app_routes.tsx`, buyer dashboard at `/dashboards/buyer` (`DashboardPage` -> `RoleDashboardView` -> `BuyerDashboardView`). Chart today: hand-rolled SVG `DashboardSeriesChart` (3 series, per-series normalization, no dual axis). API: `GET /api/v1/dashboards/buyer` returns `BuyerPortfolioDTO` (`internal/dashboardadmin/portfolio.go`).

Contracts: `ui.mdc`, `frontend-modular.mdc`, `boundaries.mdc`, `react.mdc`, `web/DESIGN.md`. Ship order: **OpenAPI + handler -> API helper -> domain UI -> thin page -> e2e**.

Visual target: **Grok surfaces + Geist density** (dark theme). Do not copy Keitaro light/green chrome; copy **information architecture** (KPI strip, multi-metric chart, breakdown grid, recent events).

Out of scope: hot path (`/track`, `/click`, `/openrtb/bid`), inbound webhooks, hosted lander static (`/lp/*`).

---

## Already shipped (do not re-open unless regression)

| Area | State |
| :--- | :--- |
| Shell | Auth, grouped nav, collapsible sidebar, command palette, skip link |
| Directory pattern | Customers/Campaigns reference; waves A/B/C largely done |
| Billing, fraud, team, ops breadth | Routes + core CRUD/read patterns |
| Integrations, creative write, campaign editor depth | Phases from prior backlog closed |
| RTB admin, automation, portals, onboarding | Closed |

Prior backlog files (`FRONTEND_UX_BACKLOG.md`, phases 0-17) are retired. Open polish only where listed under **Deferred polish** below.

---

## Coverage model (honest)

| Layer | Dashboard refactor | Rest of admin |
| :--- | :--- | :--- |
| Route exists | `/dashboards/buyer` yes | Mostly yes |
| API contract | `BuyerPortfolioDTO` incomplete vs target | Reports catalog broad |
| UI calls API | Yes; client sort on top campaigns | Core paths wired |
| Operator UX | MVP dashboard, not Keitaro parity | Directory/CRUD usable |

---

## Execution order (mandatory)

1. **dashboard_contract** — OpenAPI + Go series/breakdown fields
2. **dashboard_chart** — dual-axis multi-series chart (Recharts/shadcn chart)
3. **dashboard_home** — buyer page layout + toolbar + KPI alignment
4. **dashboard_breakdowns** — 2x2 tables + server totals
5. **dashboard_recent_clicks** — embedded block + API fields
6. **dashboard_click_log** — full `/reports/click-log` page
7. **dashboard_campaign** — campaign-scoped dashboard parity
8. **deferred_polish** — a11y manual pass, orphan cleanup (as needed)

---

## Phase dashboard_contract — Buyer dashboard API

**Goal:** One cold-path response powers KPI strip, chart, breakdown tables, and recent clicks preview. No client aggregation.

**Gap today**

| Target field | Today |
| :--- | :--- |
| `series[].clicks`, `conversions`, `revenue_micro`, `profit_micro` | Only `impressions`, `blocks`, `spend_micros` (`chart_series.go`) |
| `breakdowns.{campaigns,sources,landers,offers}` | Not in `BuyerPortfolioDTO` |
| `breakdowns.*.totals` | Not server-side |
| `recent_clicks[]` | Not on dashboard; `ClickLogEvent` lacks `ip`, `os_label`, `browser_label`, `destination_url` |
| `top_sources` on `BuyerDashboardDTO` | Dead struct; live handler uses `BuyerPortfolioDTO` only |
| Client `topCampaigns()` sort | `buyer_dashboard_view.tsx` sorts in browser — remove |

**OpenAPI / Go tasks**

| Item | Detail |
| :--- | :--- |
| Extend `DashboardSeriesPoint` schema | `clicks`, `conversions`, `spend_micro`, `revenue_micro`, `profit_micro`; optional `roi_pct`, `unique_clicks` (product decision) |
| `DashboardBreakdownTable` schema | `rows[]`, `totals`, `truncated`, `total` |
| `BuyerPortfolioDTO` (or `BuyerDashboardHomeDTO`) | `breakdowns`, `recent_clicks`, `table_sections_meta` |
| `QueryCustomerDashboardSeries` | PG/CH joins for clicks, conversions, money per bucket (`chartBucketWidth` rules unchanged) |
| Breakdown queries | Campaign top N + totals; sources via `sub1`/`sub2` or reuse `traffic-sources` SQL; landers/offers P1/P2 |
| Recent clicks | Last 20 from CH browse; cap in handler |
| Payload cap | Reuse `writeRoleDashboardJSON` / `payload_cap` patterns (100 rows) |

**Product decisions (blockers)**

| Question | Default if unset |
| :--- | :--- |
| UC (unique clicks) metric | Omit MVP; use impressions or add CH dedupe later |
| ROI on chart | Tooltip only in MVP; not third Y-axis |
| `group_by` query param | Defer; dashboard always shows all four breakdown sections |

**DoD**

- [ ] `api/openapi/components/schemas/ops_reports.yaml` updated; `bash scripts/ci/admin/openapi.sh` green
- [ ] `internal/dashboardadmin/portfolio.go` + `internal/reports/chart_series.go` populate new fields
- [ ] Holdout or handler test: breakdown totals match sum of visible rows when not truncated
- [ ] `web/src/types/generated/openapi.d.ts` regenerated
- [ ] `buyer_dashboard_types.ts` mirrors OpenAPI (no invented fields)

---

## Phase dashboard_chart — Multi-axis time series

**Goal:** Replace `DashboardSeriesChart` SVG with Keitaro-equivalent behavior: multiple metrics, real Y scales, legend, tooltip.

**Gap today:** Each series normalized to its own max; Y labels are `%`; no markers; no hover; 3 metrics only.

**UI tasks**

| Item | Detail |
| :--- | :--- |
| Add shadcn chart | `npx shadcn@latest add chart` -> `web/src/components/ui/chart.tsx` (+ Recharts dep) |
| `dashboard_multi_axis_chart.tsx` | `ComposedChart`: left axis volume (clicks, conversions), right axis money (spend, revenue, profit) |
| Shared metric config | `DASHBOARD_METRICS` shared with `DashboardKpiStrip` (`id`, `label`, `color` chart-1..5, `axis`) |
| Tooltip | `ChartTooltip`; format via `displayCount` / `displayMicro` |
| Area vs line | Line for volume; area fill for 1-2 primary series only (readability on dark) |
| Empty / loading | `PanelSection` + muted copy; no fake series |
| Retire or wrap | `dashboard_series_chart.tsx` -> thin wrapper or delete after migration |

**Runtime class:** Cold (regime S). Recharts on fetch snapshot is OK (`react.mdc`). No client KPI math.

**DoD**

- [ ] Chart renders 5+ series from API without normalizing each to 100%
- [ ] Left/right axis labels show real units (count / currency display)
- [ ] Legend colors match KPI strip accents
- [ ] `read_lints` clean on touched TSX
- [ ] e2e: `dashboards.spec.js` — chart region visible after Load

---

## Phase dashboard_home — Buyer page layout

**Goal:** Keitaro-shaped home: filters -> KPI strip -> chart -> breakdown grid -> recent clicks.

**Route:** `/dashboards/buyer?customer_id=&from=&to=`

**Layout (ui.mdc macro grid)**

```
PageChrome title="Dashboard"
  PageToolbar (customer, range preset, from/to, Apply, Refresh)
  DashboardKpiStrip
  DashboardMultiAxisChart
  grid xl:grid-cols-2 gap-6
    PanelSection x4 (breakdown tables)
  PanelSection "Recent clicks"
    DashboardRecentClicks + Link to full log
```

**Tasks**

| Item | Detail |
| :--- | :--- |
| `dashboard_home_toolbar.tsx` | Extract filters from `role_dashboard_view.tsx`; draft/applied via URL |
| Remove client sort | Delete `topCampaigns()`; render `breakdowns.campaigns.rows` from API |
| Replace bottom 2x2 | Drop Portfolio status / Attention / Alerts from **primary** grid (move Attention to sidebar panel or P2) |
| Freshness | Keep `badge` on `PageChrome` from `kpis.freshness` |
| Role selector | Keep for non-buyer roles; buyer default landing stays `/dashboards/buyer` |

**DoD**

- [ ] Single `getRoleDashboard('buyer')` fetch on Apply (no N+1 breakdown fetches)
- [ ] URL is source of truth for `customer_id`, `from`, `to`
- [ ] Stale-while-revalidate on refresh error when snapshot present
- [ ] No `useMemo` filter/sort over full campaign list

---

## Phase dashboard_breakdowns — Four tables

**Goal:** Campaign, Source, Lander, Offer tables with Clicks / UC? / Conversions + **total row** from server.

**Component:** `dashboard_breakdown_table.tsx` — props: `title`, `rows`, `totals`, `truncated`, `emptyLabel`, column links.

| Table | Row link | MVP |
| :--- | :--- | :--- |
| Campaign | `/campaigns/{id}/edit` | P0 |
| Source | filter drill or `/reports/traffic-sources` | P1 |
| Lander | `/creative/landers/{id}/editor` | P2 |
| Offer | `/creative/offers` | P2 |

**DoD**

- [ ] Totals row from `breakdowns.*.totals` only
- [ ] `tabular-nums` on numeric columns
- [ ] Truncated sections show meta (`table_sections_meta` or section `truncated` flag)
- [ ] Empty state per table (not silent blank)

---

## Phase dashboard_recent_clicks — Embedded event log

**Goal:** Keitaro "Recent clicks" strip on dashboard (last ~20 rows).

**MVP columns (API permitting):** `click_id`, `created_at`, `campaign_id` (name via join or label), `country`, `sub1`, `event_type`.

**Full columns (after CH fields):** `ip`, `os_label`, `browser_label`, `destination_url` with icons.

**DoD**

- [ ] Rows from dashboard response OR dedicated sub-endpoint — not client merge
- [ ] `click_id` links to `/reports/click-log?click_id=` (or detail route)
- [ ] "View all" -> click log page
- [ ] Server pagination not required for embedded block

---

## Phase dashboard_click_log — Clicks log directory

**Goal:** First-class operator page for `GET /api/v1/reports/click-log` (alias `/reports/clicks`).

**Gap today:** OpenAPI typed `ClickLogReportResponse`; no `reports_api.ts` wrapper; `ReportRunner` expects `ReportMapRow[]`.

**Tasks**

| Item | Detail |
| :--- | :--- |
| `getClickLogReport` | `web/src/api/reports_api.ts` |
| `click_log_directory.tsx` | `DirectoryTable`, server `limit`/`offset`/`next_cursor` |
| Route | `/reports/click-log` in `app_routes.tsx` + catalog entry |
| Filters | `customer_id`, `from`, `to`, `campaign_id`, `click_id` per OpenAPI |
| Freshness chip | From `freshness` DTO |

**DoD**

- [ ] Canonical directory stack (`FilterPanel`, Apply, pagination)
- [ ] e2e: `web/e2e/click_log.spec.js` smoke
- [ ] No client-side event filtering

---

## Phase dashboard_campaign — Campaign dashboard parity

**Goal:** `/dashboards/campaign/:id` matches buyer visual language (KPI + chart + placement/source breakdown).

**API:** Extend `GET /api/v1/dashboards/campaign/{id}` series/breakdowns same shape as buyer (scoped to one campaign).

**DoD**

- [ ] Reuse `DashboardMultiAxisChart`, `DashboardBreakdownTable`, `DashboardKpiStrip`
- [ ] `campaign_dashboard_view.tsx` refactored off one-off SVG if any
- [ ] Link from campaign editor retained

---

## Deferred polish (non-blocking)

| Item | Notes |
| :--- | :--- |
| Manual a11y keyboard pass | Campaigns, Customers, Team, Dashboard after chart |
| `ux_cleanup` verification | `bash scripts/ci/admin/ui_surface.sh` on touch |
| Attention / alerts on buyer home | P2 panel below fold or ops-style `PanelSection` |
| Light theme | Out of scope |
| `domains/` vs `ui/` rename | Large churn; no dedicated phase |
| Fraud overview block on buyer | Keep optional `portfolio.fraud` section below grid |

---

## Keitaro parity matrix (tracking)

| Keitaro surface | Target phase | Status |
| :--- | :--- | :--- |
| 7 KPI cards | dashboard_home | done |
| Dual-axis chart | dashboard_chart | done |
| Campaign breakdown table | dashboard_breakdowns | done |
| Source breakdown | dashboard_breakdowns | done |
| Lander breakdown | dashboard_breakdowns | placeholder |
| Offer breakdown | dashboard_breakdowns | placeholder |
| Recent clicks table | dashboard_recent_clicks | done |
| Clicks log (full) | dashboard_click_log | done |
| Global date filter | dashboard_home | exists |
| Group-by dimension dropdown | deferred | — |

---

## Verification (per dashboard PR)

Paste in PR when closing a phase:

```bash
cd web && npm run typecheck
bash scripts/ci/admin/ui_surface.sh
bash scripts/ci/admin/web.sh
```

Go handler phase also:

```bash
bash scripts/ci/admin/openapi.sh
go test ./internal/dashboardadmin/ -short -run TestGetBuyerDashboard -count=1
```

Name e2e spec added or extended. If not run locally, state **not run** and list `read_lints` / `grep` used.

---

## Tracking

| Phase | Status | Notes |
| :--- | :--- | :--- |
| dashboard_contract | done | Series, breakdowns, recent_clicks on BuyerPortfolioDTO |
| dashboard_chart | done | Recharts dual-axis `DashboardMultiAxisChart` |
| dashboard_home | done | KPI strip, chart, breakdown grid, recent clicks |
| dashboard_breakdowns | done | Campaign + source; lander/offer placeholders |
| dashboard_recent_clicks | done | Embedded block + View all link |
| dashboard_click_log | done | `/reports/click-log` directory page |
| dashboard_campaign | open | Campaign-scoped dashboard parity |
| deferred_polish | open | a11y, orphan cleanup |

Update status when every DoD checkbox in a phase is `[x]`.

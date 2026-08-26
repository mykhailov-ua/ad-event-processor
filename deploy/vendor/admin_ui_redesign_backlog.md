# Admin UI redesign backlog (internal)

Shippable admin web work to fix information architecture drift, design-system adoption gaps, and responsive debt in `web/`. Derived from the 2026 admin UI audit (route inventory, component usage scan, `ui.mdc` contract).

**Not in scope:** marketing copy refresh, competitor positioning, new report metrics without backend handlers, Tailwind migration, `@grafana/ui` dependency.

**Canonical implementation truth:** `.cursor/rules/ui.mdc`, `web/src/styles/tokens.css`, `web/src/pages/dev_components_page.tsx`.

Cross-reference slugs by name in PRs and docs. Do not close a slug until every applicable gate below is checked.

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `ui_p0` | Blocks IA clarity or hides live features; ship first |
| `ui_p1` | Design-system contract or god-page split; high maintenance cost |
| `ui_p2` | Responsive polish, dedup, visual rhythm |
| `ui_p3` | Nice-to-have consolidation after p0-p2 |

---

## Global close checklist (every slug)

Rule: `ui.mdc`, `anti-slop.mdc` admin UI section

- [ ] Read Go handler DTO before UI fields (`internal/controlplane/*_handlers.go`)
- [ ] `json` tags match `web/src/types/` and `helpers/*_api.ts`
- [ ] `live: true` only when `/api/v1` backend exists (`bash scripts/ci/report_live_routes_gate.sh`)
- [ ] `StubBanner` on 501; `ErrorBlock` on errors; no silent `catch` -> empty table
- [ ] Mutations use `apiConfirmed`; no "Saved" toast before 2xx
- [ ] Money via `formatUsdDecimal` / `formatAmountMicro`
- [ ] No hard-coded hex in components (tokens only)
- [ ] No `(skeleton)` / demo KPIs / user-visible "coming soon" without `StubBanner`
- [ ] JSDoc on every function and block-scoped arrow in touched `web/**` files (`quality.mdc`)
- [ ] `cd web && npm run typecheck`
- [ ] `bash scripts/ci/admin_web.sh` on touched admin surface
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched vendor docs (`naming.mdc`)

Rule: `core.mdc` commit policy (when landing code)

- [ ] Imperative commit title names concrete surface (route, page, nav group, component)
- [ ] Docs-only UI claims ship in the same commit as code

---

## Target information architecture (north star)

Seven top-level areas. Self-serve uses the same shell and tokens, not a parallel product chrome.

| Area | Audience | Replaces / absorbs |
| :--- | :--- | :--- |
| Home | All roles | `/` overview; ops strip for operators only |
| Campaigns | Buyer, ad ops | list, detail, flows, wizard, portfolio |
| Analytics | Buyer, finance, fraud | dashboards, reports hub, telegram reports |
| Integrations | Ad ops | cost sync, postbacks, alerts, automation, margin guard, supply, schemas |
| Finance | Finance, admin | customers, billing, invoices, disputes |
| Trust & Safety | Fraud analyst | blacklist, ML model, fraud dashboards, fraud reports |
| Platform | Operator | ops, settings, audit, license, domains |

---

## Summary

| Slug | Priority | Outcome | Rough surface | Est. effort |
| :--- | :--- | :--- | :--- | :--- |
| `page_chrome_contract` | ui_p0 | One header, stack, skeleton, empty pattern on all pages | `page_header.tsx`, shared loaders | S |
| `nav_integrations_hub` | ui_p0 | Cost sync, automation, margin guard discoverable | `nav_config.ts`, hub page | S |
| `nav_reports_sidebar_groups` | ui_p0 | Live reports in sidebar groups, not Cmd+K only | `nav_reports.ts`, `report_hub_page.tsx` | M |
| `context_bar_customer_role` | ui_p0 | Global buyer/customer context visible | `shell_layout.tsx` | S |
| `campaign_detail_zones` | ui_p1 | Split 11-tab god-page into 3 zones or sub-routes | `campaign_detail_page.tsx` | L |
| `report_engine_unify` | ui_p1 | One config-driven report page | `report_*_page.tsx` | L |
| `integrations_hub_page` | ui_p1 | Card hub for all integration surfaces | new page + nav | M |
| `trust_safety_section` | ui_p1 | Fraud tools under one nav area | nav + landing page | M |
| `home_role_split` | ui_p1 | Buyer home vs operator home; no duplicate ops on `/` | `overview_page.tsx`, `ops_home_page.tsx` | M |
| `selfserve_main_shell` | ui_p1 | Self-serve on main shell; drop inline width hack | `selfserve_shell_layout.tsx` | M |
| `panel_primitive_section_card` | ui_p1 | Retire `section-block` / `settings-panel` drift | styles + pages | M |
| `table_skeleton_shared` | ui_p2 | One `TableSkeleton` export | ~20 pages | S |
| `empty_state_shared` | ui_p2 | `EmptyState` component everywhere | list/report pages | S |
| `responsive_breakpoints` | ui_p2 | 640 / 1024 / 1280 layout rules | `shell.css`, `page.css` | M |
| `responsive_table_cards` | ui_p2 | Card fallback for key lists under 768px | campaigns, customers, billing | M |
| `margin_guard_canonical_url` | ui_p2 | One URL for margin guard | `app_routes.tsx`, redirects | S |
| `role_dashboards_tabs` | ui_p2 | adops / cfo / accountant / fraud as tabs | `role_dashboard_page.tsx` | M |
| `kpi_grid_component` | ui_p3 | Shared KPI grid replaces ad-hoc `metric-card` | overview, dashboards | S |
| `typography_token_cleanup` | ui_p3 | Remove `page-title`, map headings to token classes | pages + CSS | S |
| `dev_components_public` | ui_p3 | Styleguide reachable for designers (not role A only) | `app_routes.tsx`, `route_guard.ts` | S |

---

## `page_chrome_contract`

**Priority:** ui_p0

**Gap:** `PageHeader` exists but production uses hand-built `page-header` divs, legacy `page-title`, or `Breadcrumbs` without a title row. `PageStack` and `PageSkeleton` are documented in `ui.mdc` but barely adopted.

**Current state:** `PageHeader` imported only in `dev_components_page.tsx`. ~40 pages use `page-header__title`; 5 use `page-title`.

**Target:**

- Every authenticated page: `PageHeader` (title, optional desc, breadcrumbs, actions) -> `PageStack` body.
- Loading: `PageSkeleton` for full-page; shared `TableSkeleton` for table bodies.
- Empty: `EmptyState` component, not raw `empty-state` div copy-paste.

**Surfaces:**

- `web/src/components/page_header.tsx`
- `web/src/components/page_skeleton.tsx`
- `web/src/components/empty_state.tsx` (new or extend `table_skeleton.tsx`)
- Pilot migrate: `campaigns_page.tsx`, `billing_page.tsx`, `integrations_postbacks_page.tsx`, `report_hub_page.tsx`

**Close gates:**

- [ ] `grep -r "page-title" web/src/pages` returns zero (except login/bootstrap)
- [ ] `grep -r "from '../components/page_header" web/src/pages` covers all list/detail/settings templates
- [ ] `dev_components_page.tsx` documents the four templates (Overview, List, Detail, Settings)

**Depends on:** none

---

## `nav_integrations_hub`

**Priority:** ui_p0

**Gap:** `/integrations/automation` is routed but not in sidebar or overflow. Cost sync, margin guard, smart alerts live in `NAV_OVERFLOW_LINKS` only. Integrations sidebar shows one link ("Integrations" -> postbacks).

**Current state:** `nav_config.ts` Integrations group has a single postbacks link. Automation invisible without direct URL.

**Target:**

- Integrations nav group lists: Hub (new or postbacks as index), Cost sync, Postbacks, Smart alerts, Automation, Margin guard, Supply, Schemas.
- Overflow retains deep links for command palette parity only.

**Surfaces:**

- `web/src/helpers/nav_config.ts`
- `web/src/helpers/route_guard.ts` (automation perm if missing)

**Close gates:**

- [ ] `/integrations/automation` appears in sidebar for `campaigns:read`
- [ ] `bash scripts/ci/report_live_routes_gate.sh` unchanged (no report impact)
- [ ] Command palette still indexes integration routes

**Depends on:** none (hub page optional in `integrations_hub_page`)

---

## `nav_reports_sidebar_groups`

**Priority:** ui_p0

**Gap:** 25+ live reports; sidebar shows only Reports hub + Telegram. Individual reports require Cmd+K (`reportCommandPaletteLinks()`).

**Current state:** `sidebarReportsNavLinks()` returns two links. Full catalog in hub table only.

**Target:**

- Sidebar Analytics subsection with collapsible groups:
  - Performance (placements, keywords, pacing, spend velocity, campaign overview, ...)
  - Quality & fraud (filter rejects, fraud breakdown, silent reject funnel, ivt-by-source, data quality, ...)
  - Financial (true ROI, discrepancy, cost sync coverage, customer portfolio, ...)
  - RTB (overview, no-bid, geo-device)
  - Telegram (existing sub-links)
- Hub remains catalog + saved views + export queue.

**Surfaces:**

- `web/src/helpers/nav_reports.ts`
- `web/src/models/report.ts` (add `group` field on `ReportCardDTO`)
- `web/src/helpers/nav_config.ts`

**Close gates:**

- [ ] Every `live: true` report reachable from sidebar without Cmd+K
- [ ] Retired reports stay hub-only with alt links
- [ ] Nav does not exceed practical height on 1080p (collapsible groups)

**Depends on:** none

---

## `context_bar_customer_role`

**Priority:** ui_p0

**Gap:** Buyer-scoped sessions reuse admin routes (`/billing`, `/campaigns`) without persistent UI context. Operator cannot see which customer a masked buyer views.

**Current state:** `boundCustomerId()` used in page logic; no global chrome indicator.

**Target:**

- Shell toolbar or header strip: role label (Buyer / Admin / Publisher) + customer name/UUID when session-scoped.
- Link to switch customer for admin; read-only badge for masked buyer.

**Surfaces:**

- `web/src/components/shell_layout.tsx`
- `web/src/styles/shell.css`
- `web/src/helpers/buyer_session.ts`

**Close gates:**

- [ ] Buyer on `/campaigns` sees customer context without opening filters
- [ ] Publisher-only role still gets publisher-only nav (`visibleNavGroups`)

**Depends on:** none

---

## `campaign_detail_zones`

**Priority:** ui_p1

**Gap:** `campaign_detail_page.tsx` is 1475 lines with 11 horizontal tabs (overview, stats, config, tracking, postbacks, fraud, filters, margin, platform, events, creative, telegram). Unusable on laptop; high merge conflict surface.

**Current state:** `TabBar` with `?tab=` query param. Section components already extracted under `web/src/components/campaign_*_section.tsx`.

**Target (pick one, document in PR):**

- **Option A:** Vertical sub-nav (Performance | Setup | Monetization & risk) with nested links to current sections.
- **Option B:** Sub-routes `/campaigns/:id/setup/tracking`, etc., with shared layout shell.

| Zone | Sections |
| :--- | :--- |
| Performance | overview, stats, events |
| Setup | config, tracking, filters, creative, telegram |
| Monetization & risk | postbacks, fraud, margin guard, platform sync |

**Surfaces:**

- `web/src/pages/campaign_detail_page.tsx` (split into layout + zone pages)
- `web/src/app_routes.tsx` if sub-routes
- `web/src/components/campaign_*_section.tsx` (unchanged contracts)

**Close gates:**

- [ ] No file over 600 lines in campaign detail tree
- [ ] Deep links `?tab=postbacks` redirect or alias to new paths for one release
- [ ] Masked buyer still sees reduced tab set
- [ ] `bash scripts/ci/admin_web.sh`

**Depends on:** `page_chrome_contract` (recommended)

---

## `report_engine_unify`

**Priority:** ui_p1

**Gap:** Three parallel report implementations: `ReportQueryPage`, `ReportCustomerRangePage`, `ReportSimplePage`, plus custom click log and telegram shell.

**Current state:** Shared helpers (`report_rows.ts`, `report_api.ts`) but duplicated filter UI, skeleton rows, export wiring.

**Target:**

- Single `ReportPage` driven by `report_configs.ts` (endpoint, columns, filters, financial mask).
- `report_route_pages.tsx` becomes thin wrappers or route table only.
- One date/customer filter toolbar component.

**Surfaces:**

- `web/src/pages/report_query_page.tsx`
- `web/src/pages/report_customer_range_page.tsx`
- `web/src/pages/report_simple_page.tsx`
- `web/src/pages/report_configs.ts`
- New `web/src/pages/report_page.tsx`

**Close gates:**

- [ ] Line count of report pages drops by >= 40% vs baseline
- [ ] All `live: true` reports pass existing openapi/report tests
- [ ] `report_live_routes_gate.sh` green
- [ ] Saved views + export still work on pilot reports (placements, true-roi)

**Depends on:** `page_chrome_contract`

---

## `integrations_hub_page`

**Priority:** ui_p1

**Gap:** No landing page explaining integration surfaces; users land on postbacks or hunt via Cmd+K.

**Target:**

- `/integrations` hub with cards: status summary, link, one-line description per integration.
- Postbacks page moves to `/integrations/postbacks`; nav "Integrations" points to hub.

**Surfaces:**

- New `web/src/pages/integrations_hub_page.tsx`
- `web/src/app_routes.tsx`
- `web/src/helpers/nav_config.ts`

**Close gates:**

- [ ] Hub cards match real routes (no dead links)
- [ ] Each card shows error state if status API fails (not empty fake OK)
- [ ] Works for buyer masked perm (`campaigns:read:masked`)

**Depends on:** `nav_integrations_hub`

---

## `trust_safety_section`

**Priority:** ui_p1

**Gap:** Fraud spread across campaign tab, `/ops/blacklist`, `/ops/ml-model`, `/dashboards/fraud`, and multiple reports.

**Target:**

- New nav area **Trust & Safety** with landing page linking:
  - Fraud dashboard (`/dashboards/fraud` or moved)
  - Blacklist (`/ops/blacklist`)
  - ML model (`/ops/ml-model`)
  - Decision lookup (from fraud dashboard panel)
  - Reports: fraud breakdown, silent reject funnel, ivt-by-source
- Ops nav keeps infra-only items (shards, DLQ, edge parity).

**Surfaces:**

- `web/src/helpers/nav_config.ts`
- New `web/src/pages/trust_safety_home_page.tsx` (or extend `role_dashboard_page.tsx`)
- `web/src/pages/ops_blacklist_page.tsx`, `ops_ml_model_page.tsx` (chrome only)

**Close gates:**

- [ ] Fraud analyst can complete daily review without visiting Campaigns or generic Ops home
- [ ] Permissions: `audit:read` / `blacklist:read` respected
- [ ] No duplicate nav entries for same URL

**Depends on:** `nav_reports_sidebar_groups` (fraud report links)

---

## `home_role_split`

**Priority:** ui_p1

**Gap:** `/` Overview loads ops doctor, incidents, summary for `shards:read` AND buyer portfolio for buyer role on one page. `ops_home_page.tsx` duplicates much of the same data plus outbox/DLQ.

**Target:**

- **Home `/`:** role-specific widgets only.
  - Buyer: portfolio, recommendations, quick links (no doctor panel).
  - Operator: compact health strip + link to Platform > Operations (not full ops console).
- **Platform > Operations:** full ops console (current `ops_home_page` content).

**Surfaces:**

- `web/src/pages/overview_page.tsx`
- `web/src/pages/ops_home_page.tsx`
- `web/src/helpers/home_alerts.ts`

**Close gates:**

- [ ] Buyer home does not call `/api/v1/ops/doctor` unless `shards:read`
- [ ] Operator can reach outbox/DLQ from ops home in <= 2 clicks from `/`
- [ ] Live feed polling unchanged for ops role

**Depends on:** `context_bar_customer_role` (optional)

---

## `selfserve_main_shell`

**Priority:** ui_p1

**Gap:** `selfserve_shell_layout.tsx` uses inline `width: 220`, `sidebar-nav` classes, `main-content`, always-open sidebar, no mobile drawer.

**Target:**

- Reuse `ShellLayout` with `navVariant="selfserve"` and filtered link set.
- Same tokens, collapse, mobile overlay as main admin.
- Remove duplicate CSS hooks (`sidebar-nav__link` vs `sidebar__link`).

**Surfaces:**

- `web/src/components/selfserve_shell_layout.tsx`
- `web/src/components/shell_layout.tsx`
- `web/src/app_shell.tsx`
- `web/src/styles/shell.css`

**Close gates:**

- [ ] No inline `style={{ width:` on self-serve sidebar
- [ ] `@media (max-width: 48rem)` applies to self-serve
- [ ] `data-testid="selfserve-shell"` preserved or updated in e2e

**Depends on:** `page_chrome_contract`

---

## `panel_primitive_section_card`

**Priority:** ui_p1

**Gap:** Three panel dialects: `section-card`, `section-block`, `settings-panel` with different padding, headers (`subsection-title`, `settings-panel__title`).

**Target:**

- `SectionCard` as the only panel primitive for page body sections.
- `settings-panel` retained only on settings-style forms or aliased to `SectionCard` variant.
- Deprecate `section-block` in new code; migrate high-traffic pages.

**Surfaces:**

- `web/src/components/section_card.tsx`
- `web/src/styles/cards.css`, `page.css`
- `campaign_detail_page.tsx`, `ops_home_page.tsx`, `report_hub_page.tsx`

**Close gates:**

- [ ] `dev_components_page` shows SectionCard variants (default, flush, danger)
- [ ] No new `section-block` in touched files
- [ ] Visual diff acceptable on billing + campaign detail (screenshot or manual sign-off)

**Depends on:** `page_chrome_contract`

---

## `table_skeleton_shared`

**Priority:** ui_p2

**Gap:** Identical `TableSkeleton` function copy-pasted in ~20 page files.

**Target:**

- `web/src/components/table_skeleton.tsx` exported; pages import it.
- Optional `cols` / `rows` props only.

**Surfaces:** all pages with local `function TableSkeleton`

**Close gates:**

- [ ] `rg "function TableSkeleton" web/src/pages` returns zero
- [ ] Shimmer uses `.data-table__row--skeleton` from `tables.css`

**Depends on:** none

---

## `empty_state_shared`

**Priority:** ui_p2

**Gap:** `EmptyState` component unused in production; pages inline `empty-state` markup with inconsistent CTAs.

**Target:** Replace inline empty blocks on list pages with `EmptyState` (icon, title, desc, optional action).

**Pilot pages:** `campaigns_page.tsx`, `customers_page.tsx`, `billing_page.tsx`, `report_hub_page.tsx`, `audit_page.tsx`

**Close gates:**

- [ ] `EmptyState` used on all pilot pages
- [ ] No table row with sole `text-muted` "No X yet" without `EmptyState` on pilot pages

**Depends on:** `page_chrome_contract`

---

## `responsive_breakpoints`

**Priority:** ui_p2

**Gap:** Single mobile breakpoint at 48rem in `shell.css`. Filter toolbars and page headers lack documented narrow behavior.

**Target:**

- Document breakpoints: 640 (sm), 1024 (md), 1280 (lg) in `tokens.css` or `system.css`.
- Page header actions stack below title on sm.
- Filter toolbar wraps with `cluster` rules.

**Surfaces:**

- `web/src/styles/shell.css`
- `web/src/styles/page.css`
- `web/src/styles/forms.css`
- `web/src/components/filter_toolbar.tsx`

**Close gates:**

- [ ] 375px wide: sidebar drawer works; no horizontal page overflow on campaigns list
- [ ] No new inline `style={{` for layout in touched files

**Depends on:** `selfserve_main_shell` (for self-serve parity)

---

## `responsive_table_cards`

**Priority:** ui_p2

**Gap:** Wide tables only scroll horizontally on mobile; poor scanability.

**Target:** Below 768px, render card list for campaigns, customers, billing ledger (pilot). Keep table at md+.

**Surfaces:**

- `web/src/components/data_table.tsx` or new `responsive_table.tsx`
- `campaigns_page.tsx`, `customers_page.tsx`, `billing_page.tsx`

**Close gates:**

- [ ] Sort and pagination work in card mode
- [ ] Masked columns still hidden per `report_mask` / permissions

**Depends on:** `responsive_breakpoints`

---

## `margin_guard_canonical_url`

**Priority:** ui_p2

**Gap:** Duplicate routes `/margin-guard` and `/integrations/margin-guard` to the same page.

**Target:** Canonical `/integrations/margin-guard`; other path 301 or client redirect for one release.

**Surfaces:**

- `web/src/app_routes.tsx`
- `web/src/helpers/nav_config.ts` (overflow links)
- `web/src/pages/overview_page.tsx` (quick link)

**Close gates:**

- [ ] Single nav entry
- [ ] Old URL still resolves (redirect)

**Depends on:** `nav_integrations_hub`

---

## `role_dashboards_tabs`

**Priority:** ui_p2

**Gap:** `/dashboards/adops` in sidebar; cfo, accountant, fraud only in overflow / Cmd+K.

**Target:** `/dashboards` with tab param or sub-route per role; sidebar one "Dashboards" entry.

**Surfaces:**

- `web/src/pages/role_dashboard_page.tsx`
- `web/src/app_routes.tsx`
- `web/src/helpers/nav_config.ts`

**Close gates:**

- [ ] Role tab hidden when user lacks perm
- [ ] Deep links `/dashboards/fraud` still work

**Depends on:** `nav_reports_sidebar_groups` (Analytics grouping)

---

## `kpi_grid_component`

**Priority:** ui_p3

**Gap:** `metric-card` markup duplicated on overview, ops, role dashboards.

**Target:** `KpiGrid` + `KpiCard` components with icon, label, value, optional delta.

**Surfaces:**

- New `web/src/components/kpi_grid.tsx`
- `overview_page.tsx`, `ops_home_page.tsx`, `role_dashboard_page.tsx`

**Depends on:** `page_chrome_contract`

---

## `typography_token_cleanup`

**Priority:** ui_p3

**Gap:** `page-title`, `subsection-title` bypass `.text-heading-24` / `.text-heading-20` from `ui.mdc`.

**Target:** Map all page titles to `page-header__title` or typography tokens; remove unused CSS classes.

**Close gates:**

- [ ] `rg "page-title|subsection-title" web/src` zero in pages

**Depends on:** `page_chrome_contract`, `panel_primitive_section_card`

---

## `dev_components_public`

**Priority:** ui_p3

**Gap:** Living styleguide at `/dev/components` restricted to role `A` only.

**Target:** Available to `settings:read` or dedicated `ui:read` perm for internal operators; documents all primitives post-redesign.

**Depends on:** `page_chrome_contract`, `panel_primitive_section_card`

---

## Suggested delivery order

```text
Wave 1 (ui_p0, ~1-2 weeks)
  page_chrome_contract
  nav_integrations_hub
  nav_reports_sidebar_groups
  context_bar_customer_role
  table_skeleton_shared          # quick win, parallel

Wave 2 (ui_p1, ~2-3 weeks)
  integrations_hub_page
  home_role_split
  selfserve_main_shell
  panel_primitive_section_card
  campaign_detail_zones          # start vertical nav; sub-routes optional follow-up
  report_engine_unify            # can span multiple PRs
  trust_safety_section

Wave 3 (ui_p2/p3)
  responsive_breakpoints
  responsive_table_cards
  margin_guard_canonical_url
  role_dashboards_tabs
  empty_state_shared
  kpi_grid_component
  typography_token_cleanup
  dev_components_public
```

---

## Explicit non-goals (do not open slugs without new decision)

| Item | Reason |
| :--- | :--- |
| Tailwind or CSS-in-JS migration | Tokens + modular CSS are canonical (`ui.mdc`) |
| Rewrite flow builder UX | Separate product surface; only chrome alignment |
| Merge admin + self-serve products | Same shell, different nav - not single route tree |
| New reports without CH + handler | Backend first (`control-plane.mdc`) |
| Light theme as default | Dark default is documented; light tokens exist for optional toggle |

---

## References

| Doc | Role |
| :--- | :--- |
| `.cursor/rules/ui.mdc` | Templates, tokens, honesty gates |
| `.cursor/rules/anti-slop.mdc` | Admin slop patterns |
| `web/src/pages/dev_components_page.tsx` | Component catalog |
| `web/src/helpers/nav_config.ts` | Sidebar source of truth |
| `web/src/models/report.ts` | Report catalog and `live` flags |
| `scripts/ci/admin_web.sh` | Merge gate orchestrator |

# web

Operator admin SPA served by control plane (`:8188`, `/api/v1/*`). Layout and anti-slop policy: `.cursor/rules/ui.mdc`, `.cursor/rules/frontend-modular.mdc`. This file is the **visual and component reference** for `web/src/`.

Cross-ref: `FRONTEND.md` (route status), `docs/DEVELOPMENT.md` (OpenAPI types, CI).

Theme: **surface depth + Inter density** — hierarchy from OKLCH shade steps, hairline borders, compact radii (`--admin-radius-sm`).

---

## Tree

| Path | Role |
| :--- | :--- |
| `web/src/api/` | HTTP client, resource hooks, dev mock (`?admin_dev=1`) |
| `web/src/domains/` | Route-owned screens (`campaigns/list`, `ops`, …) |
| `web/src/shell/` | Page chrome, directory frames, empty/error states |
| `web/src/components/ui/` | shadcn primitives (`Button`, `Table`, …) |
| `web/src/styles/globals.css` | Surface ladder, admin utilities, typography |
| `web/src/lib/control_size.ts` | Control height contract (no manual `h-7`/`h-8`/`h-9`) |

Embed: `npm run build` → `sync_embed.mjs` → `internal/controlplane/admin_static_stub/`.

---

## Surface hierarchy

Defined in `web/src/styles/globals.css` (`:root`, `.dark`).

| Level | Variable | Utility | Nesting role |
| :--- | :--- | :--- | :--- |
| 0 | `--admin-surface-0` | `.admin-surface-0` | App canvas (`admin-main`, `--admin-bg`) |
| 1 | `--admin-surface-1` | `.admin-surface-1` | Chrome on canvas: sidebar, header, control panel, table wrap |
| 2 | `--admin-surface-2` | `.admin-surface-2` | Controls inside L1: buttons, inputs, table head/foot, dialogs |
| 3 | `--admin-surface-3` | `.admin-surface-3` | Deepest inset: kbd chip, column-menu presets |

Each step multiplies OKLCH lightness by `--admin-surface-step: 0.98` from `--admin-surface-base` (`#f4f4f5` light, `#13151a` dark).

**Rule:** child surface = parent + 1 (max 3). Toolbars bump control fills via descendant selectors (`.admin-control-panel .admin-btn` → L2).

Semantic aliases (`--admin-surface`, `--admin-input-bg`, `--admin-btn-bg`, …) map onto the ladder for backward compatibility.

Borders: `--admin-border` = 50% mix of `--admin-border-strong` toward transparent. Do not drop borders for elevation; use surface step + `1px solid var(--admin-border)`.

---

## Typography

Canonical roles: `web/src/lib/admin_typography.ts`. Fonts: `@fontsource-variable/inter`, `@fontsource-variable/jetbrains-mono` in `globals.css`.

| Role | Face | Classes | Use for |
| :--- | :--- | :--- | :--- |
| UI prose | Inter | `font-sans` | Labels, nav, names |
| Numeric | Inter + tnum | `tabular-nums`, `admin-tabular-nums`, `.num` | Money, counts, rates, KPIs, chart ticks |
| Wire / code | JetBrains Mono | `font-mono`, `ui-code-block` | UUID, JSON editor, cron, secrets |

Table metrics use Inter tnum, not mono. Headings: `tracking-tight`, `-0.02em` letter-spacing.

---

## Controls and primitives

Interactive controls use `@/components/ui/button` (wraps `.admin-btn`), not raw `<button className="admin-btn">`. CI: `bash scripts/ci/admin/ui_slop.sh`.

| Primitive | Default radius | Notes |
| :--- | :--- | :--- |
| `Button`, `Input`, `Select` | `--admin-radius-sm` | Default size = `--admin-control-height` (32px) |
| `Card`, panels, tables | `--admin-radius` | `admin-table-wrap`, `ui-surface*` |
| Nav pills, chips | `rounded-full` | `SectionNav`, `ToggleChipGroup` only |
| `Button` `shape="pill"` | `rounded-full` | Opt-in; not default |

Depth: no drop shadows on panels. Motion: `transition-colors` on interactive nodes; Radix enter/exit on overlays; `prefers-reduced-motion` in globals.

shadcn HSL tokens (`--background`, `--card`, `--muted`, …) mirror the ladder for `bg-muted`, command palette, etc. Prefer `--admin-surface-*` for new admin chrome.

---

## Shell components (`web/src/shell/`)

| Component | File | Role |
| :--- | :--- | :--- |
| `PageChrome` | `page_chrome.tsx` | Title, description, badge, actions |
| `PageBreadcrumbs` | `page_breadcrumbs.tsx` | Route trail |
| `SectionNav` | `section_nav.tsx` | Pill nav for domain sections |
| `PageToolbar` | `page_toolbar.tsx` | Action row on `ui-surface` |
| `FilterPanel` / `DirectoryFilterForm` | `filter_panel.tsx` | Directory filter grid |
| `ToggleChipGroup` | `toggle_chip_group.tsx` | Server-driven status chips with counts |
| `DirectoryListMeta` | `directory_list_meta.tsx` | Showing X–Y of Z |
| `DirectoryTable` | `directory_table.tsx` | Scrollable table frame (`admin-table-wrap`) |
| `PaginationPrevNext` | `pagination_prev_next.tsx` | Prev/Next in filter row |
| `EmptyState` / `ErrorBlock` | `empty_state.tsx`, `error_block.tsx` | Blank and blocking errors |
| `HubLinkGrid` / `StatPanel` | `hub_link_card.tsx`, `stat_panel.tsx` | Hubs and KPI tiles |

**Banned in new JSX:** raw `rounded-md border` wrappers; raw `<table>`; raw `fetch()` in domains/shell/pages.

---

## Page patterns

### Directory list (canonical)

Reference: `web/src/domains/campaigns/list/campaigns_directory.tsx`.

Stack: `PageChrome` → optional `AppliedCustomerBanner` → `FilterPanel` / `DirectoryFilterForm` → `DirectoryListMeta` → optional `ToggleChipGroup` → `DirectoryTable` | `EmptyState`.

Filter layouts: `layout="directory"` (campaigns matrix) or `layout="auto-fill"` (`minmax(12rem, 1fr)`). Apply and pagination are sibling grid children, not nested.

### Domain hub

`PageChrome` → `SectionNav` → `HubLinkGrid`. Examples: `fraud_hub.tsx`, `integrations_hub.tsx`.

### Ops / JSON dashboard

`PageChrome` → `SectionNav` → `PageToolbar` → `StatPanel` grid → `PanelSection` tables. Reference: `ops_home.tsx`, `json_dashboard_view.tsx`.

### Campaigns metrics disclosure

| Tier | Trigger | Surface |
| :--- | :--- | :--- |
| 0 | Table row | Column cells |
| 1 | Budget used cell | `Popover` + trend chart |
| 2 | Row menu / link | `Sheet` overview |

Shared: `campaign_metrics_shared.tsx`.

---

## Charts

Colors: `hsl(var(--primary))`, `hsl(var(--muted-foreground) / 0.7)`, `hsl(var(--border))` for grid. Container: `rounded-xl bg-muted/20`. Empty copy when series has no points.

---

## Verification

```bash
bash scripts/ci/admin/ui_slop.sh
bash scripts/ci/admin/web.sh
cd web && npm test
cd web && npm run typecheck    # when changing types across many files
rg 'className="admin-btn' web/src/domains web/src/shell web/src/pages   # expect 0
rg 'rounded-md border' web/src --glob '*.tsx'   # expect 0 in new JSX
```

---

## Related docs

| Doc | Content |
| :--- | :--- |
| `FRONTEND.md` | Route maturity, backlog, e2e tiers |
| `.cursor/rules/ui.mdc` | Layout grid, error honesty, fixture ban |
| `.cursor/rules/frontend-hot-path.mdc` | Client sort/filter regimes (S/L/F) |

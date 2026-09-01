# Admin UI design system (Grok surfaces)

Operator-facing admin SPA (`web/`). Policy and agent rules: `.cursor/rules/ui.mdc`. This file is the **visual and component reference**.

Theme direction: **Grok surfaces + Geist density** — true-black canvas, depth from surface shades (not shadows), pill navigation, soft hairline borders.

## Tokens

Defined in `web/src/styles/globals.css` and `web/tailwind.config.ts`. HSL components without `hsl()` wrapper in CSS variables.

| Token | Default | Role |
| :--- | :--- | :--- |
| `--background` | `0 0% 0%` | Page canvas |
| `--foreground` | `0 0% 98%` | Primary text |
| `--card` | `0 0% 4%` | Sidebar, elevated panels |
| `--popover` | `0 0% 7%` | Menus, dialogs, dropdowns |
| `--muted` | `0 0% 6%` | Input fills, table hover, chips |
| `--secondary` | `0 0% 10%` | Active nav pill, unselected chip variant |
| `--border` | `0 0% 14%` | Hairline separators |
| `--primary` | `0 0% 98%` | Inverted CTA (white on black) |
| `--radius` | `1rem` | Base corner radius |
| `--chart-1` … `--chart-5` | grayscale steps | Sparklines and charts |

Semantic surface ladder (conceptual):

| Level | Token / utility | Use |
| :--- | :--- | :--- |
| 0 | `bg-background` | App canvas, main scroll area |
| 1 | `ui-surface` | Filter panels, toolbars, inline grouping (no border) |
| 2 | `ui-surface-raised` | Hub cards, KPI tiles, clickable panels |
| 3 | `ui-table-frame` | Data tables, directory lists |
| 4 | `ui-shell` / `bg-popover` | Popover, select menu, dialog shell |

Typography:

- Font: Geist Variable (`font-sans`), Geist Mono for code (`font-mono`, `ui-code-block`).
- Headings: `tracking-tight`, `letter-spacing: -0.02em` (globals).
- Metrics: `tabular-nums` on counts, money, dates.

Motion:

- `transition-colors` on interactive surfaces only.
- Radix enter/exit on overlays (dialog, popover, sheet).
- `prefers-reduced-motion` respected in globals.

## Utility classes (`globals.css` `@layer components`)

| Class | Maps to | When to use |
| :--- | :--- | :--- |
| `ui-surface` | `rounded-2xl bg-muted/30` | Filter/toolbar background |
| `ui-surface-raised` | `rounded-2xl border border-border/40 bg-card/80` | Hub link cards, stat tiles |
| `ui-table-frame` | `rounded-2xl border border-border/40 bg-card/40` | Table wrappers |
| `ui-filter-panel` | `ui-surface` + `grid gap-4 p-4 md:p-5` | Prefer `<FilterPanel>` component |
| `ui-shell` | popover/dialog outer chrome | Used inside shadcn primitives |
| `ui-code-block` | mono pre blocks | JSON preview, logs |
| `ui-scrollbar` | thin scrollbar | Long nav, tables, popovers |

**Banned in new JSX:** raw `rounded-md border` wrappers. Use utilities or system components below.

## System components (`web/src/components/system/`)

### Page shell

| Component | File | Role |
| :--- | :--- | :--- |
| `PageChrome` | `page_chrome.tsx` | Title, description, badge, actions |
| `PageBreadcrumbs` | `page_breadcrumbs.tsx` | Route trail |
| `SectionNav` | `section_nav.tsx` | Pill nav for domain sections (Ops, Integrations, …) |
| `PageToolbar` | `page_toolbar.tsx` | Action row (`ui-surface` flex bar) |

### Directory pattern (reference: Campaigns)

| Component | File | Role |
| :--- | :--- | :--- |
| `FilterPanel` | `filter_panel.tsx` | Filter section wrapper (`ui-filter-panel`) |
| `DirectoryFilterForm` | `filter_panel.tsx` | Grid form: `layout="directory"` or `"auto-fill"` |
| `FilterField` | `filter_panel.tsx` | Label + control cell |
| `ToggleChipGroup` | `toggle_chip_group.tsx` | Server-driven status/filter chips with counts |
| `DirectoryListMeta` | `directory_list_meta.tsx` | "Showing X–Y of Z" line |
| `DirectoryTable` | `directory_table.tsx` | Scrollable semantic table frame |
| `PaginationPrevNext` | `pagination_prev_next.tsx` | Prev/Next in filter row |
| `FilterApplyButton` / `FilterResetButton` | `action_buttons.tsx` | Submit actions (`h-9`, pill) |
| `EmptyState` | `empty_state.tsx` | Blank / no-results |
| `ErrorBlock` | `error_block.tsx` | Blocking errors |
| `AppliedCustomerBanner` | `applied_customer_banner.tsx` | Customer scope chip bar |

### Hub and KPI

| Component | File | Role |
| :--- | :--- | :--- |
| `HubLinkGrid` / `HubLinkCard` | `hub_link_card.tsx` | Domain hub landing cards |
| `StatPanel` / `StatRow` | `stat_panel.tsx` | KPI metric tiles |
| `PanelSection` | `stat_panel.tsx` | Titled block + table (Ops, JSON dashboard) |

### shadcn primitives (`web/src/components/ui/`)

Interactive controls use **pill shape** by default (`Button`, `Input`, `Select` trigger). Cards use `rounded-2xl border border-border/40`.

## Page patterns

### A. Directory list (canonical)

Stack:

```
PageChrome
  [AppliedCustomerBanner]
  FilterPanel
    DirectoryFilterForm
      FilterField × N
      FilterApplyButton
      [FilterResetButton]
      PaginationPrevNext
  DirectoryListMeta
  ToggleChipGroup          (optional quick filters)
  DirectoryTable | EmptyState
```

Reference implementation: `web/src/domains/campaigns/campaigns_directory.tsx`.

Filter grid layouts:

- `layout="directory"` — campaigns-shaped wide matrix (customer, status, search, actions).
- `layout="auto-fill"` — `minmax(12rem, 1fr)` columns (customers, audit, billing).

Apply and pagination are **sibling** grid children of the filter form, not wrapped together.

### B. Domain hub

```
PageChrome
  SectionNav
  HubLinkGrid
    HubLinkCard × N
```

Examples: `fraud_hub.tsx`, `integrations_hub.tsx`, `ops_home.tsx` (nav only; home uses KPI pattern).

### C. Ops / JSON dashboard

```
PageChrome
  SectionNav
  PageToolbar
  StatPanel × N  (grid)
  PanelSection   (tables)
```

Reference: `web/src/domains/ops/ops_home.tsx`, `json_dashboard_view.tsx`.

### D. Settings / forms

```
PageChrome
  JsonDashboardView | read-only payload
  PageToolbar
  Dialog / Sheet for mutations
```

### E. Progressive disclosure (Campaigns metrics)

| Tier | Trigger | Surface |
| :--- | :--- | :--- |
| 0 | Table row | Slim columns only |
| 1 | Budget used cell | `Popover` + `HourlyTrendChart` |
| 2 | Row menu / popover link | `Sheet` overview |

Shared metrics UI: `campaign_metrics_shared.tsx`.

## Shape rules

| Element | Radius | Notes |
| :--- | :--- | :--- |
| Buttons, inputs, selects | `rounded-full` | Default `Button` shape |
| Panels, tables, cards | `rounded-2xl` | `ui-surface*`, `Card` |
| Code blocks | `rounded-xl` | `ui-code-block` |
| Nav pills, status chips | `rounded-full` | `SectionNav`, `ToggleChipGroup` |
| Dropdown/select items | `rounded-lg` | Inside `ui-shell` |

Depth: **no drop shadows** on panels. Elevation = background step + optional `border-border/40`.

## Charts

- Colors: `hsl(var(--primary))`, `hsl(var(--muted-foreground) / 0.7)`, `hsl(var(--border))` for grid lines.
- Container: `rounded-xl bg-muted/20` (no heavy border).
- Empty state copy when series has no activity.

## Verification

```bash
cd web && npm test
cd web && npm run typecheck    # when changing types or many files
rg 'rounded-md border' web/src --glob '*.tsx'   # expect 0
```

## Roadmap (not required for every PR)

| Phase | Item |
| :--- | :--- |
| A | Migrate remaining `ui-filter-panel` class strings to `<FilterPanel>` |
| B | `/design` dev route showcasing all system components |
| C | Semantic `--surface-0..3` aliases in CSS |
| D | Playwright screenshot smoke on Campaigns, Ops, Settings |

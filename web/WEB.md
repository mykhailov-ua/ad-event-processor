# web

Operator admin SPA. Production: control plane embed (`:8188`). Local dev: Vite-style dev server on `:5173` (`bash scripts/dev/aed-admin up`).

Cross-ref: `FRONTEND.md` (route status), `docs/DEVELOPMENT.md` (OpenAPI types, CI, full compose profiles).

Theme: **flat white / zinc dark** — first-party primitives in `web/src/components/ui/`, hairline borders, compact radii (4px controls, 6px panels).

**No shadcn.** Do not add `components.json`, `@radix-ui/*`, or `class-variance-authority`. Overlays use native `<dialog>` / portal + `web/src/lib/overlay_*`.

---

## Local dev

First clone: `make gen`, copy `.env.example` → `.env`, `cd web && npm ci`.

```bash
bash scripts/dev/aed-admin up
# or UI only: cd web && npm run dev
```

| | |
| :--- | :--- |
| **Admin UI** | [http://localhost:5173](http://localhost:5173) |
| **API** | [http://localhost:8188](http://localhost:8188) |
| **Login** | `admin@test.local` |
| **Password** | `Password123!` |

Credentials from `seed_admin.sh` (override in `.env`: `ADMIN_BOOTSTRAP_EMAIL`, `ADMIN_BOOTSTRAP_PASSWORD`).

**Mock API (UI without control):** if `:8188` is down, boot probes `/api/v1/meta` and enables mock automatically. Force mock: [http://localhost:5173/?admin_dev=1](http://localhost:5173/?admin_dev=1). Live API: [http://localhost:5173/?admin_dev=0](http://localhost:5173/?admin_dev=0). Mock responses are real HTTP on `:5173` (`X-Admin-Dev-Mock: 1`); unimplemented mock routes return **501** (`X-API-Stub: true`), not fake 200 empty lists. Dev server proxies `/health`, `/healthz`, `/readyz`, `/metrics` to `:8188` (never SPA `index.html`). Banner: *Dev mode - mock API responses*.

### Non-prod verification tiers (admin SPA)

Do not treat these modes as production wiring proof (`anti-slop.mdc` tier honesty).

| Tier | Trigger | What runs | Proves |
| :--- | :--- | :--- | :--- |
| **Live API** | `admin_dev=0` or control `:8188` healthy | Real `/api/v1/*` handlers | Operator UX against Go/OpenAPI contract |
| **Dev mock** | `?admin_dev=1`, localStorage, or meta probe when control is down | `web/src/api/dev_mock/*` intercept in `api/client.ts` | UI layout and fetch lifecycle only; partial API parity |
| **Chart mock** | `?chart_mock=1` on dashboard | `dashboard_series_mock.ts` synthetic series/KPIs | Chart component preview only; not `GET /api/v1/dashboards/*` data |

CI/e2e and merge claims must use **live API** against control (or documented integration tier). Green UI under dev mock or chart mock does not prove handler wiring.

```bash
bash scripts/dev/aed-admin status
bash scripts/dev/aed-admin down
```

Full stack: `docs/DEVELOPMENT.md`.

---

## Tree

| Path | Role |
| :--- | :--- |
| `web/src/api/` | HTTP client, resource hooks, dev mock (`?admin_dev=1`) |
| `web/src/domains/` | Route-owned screens (`campaigns/list`, `ops`, …) |
| `web/src/shell/` | Page chrome, directory frames, empty/error states |
| `web/src/components/ui/` | First-party primitives (`Button`, `Table`, `Dialog`, …) |
| `web/src/lib/admin_chrome.ts` | Shared zinc class tokens for primitives |
| `web/src/styles/app.css` | Tailwind entry (`@tailwind` only + minimal base) |
| `web/src/lib/control_size.ts` | Control height contract (`h-8` = 32px) |

Embed: `npm run build` → `sync_embed.mjs` → `internal/controlplane/admin_static_stub/`.

---

## Theme

Styling: Tailwind utility classes in TSX. Entry: `web/src/styles/app.css`. Palette: zinc via `tailwind.config.ts` and `dark:` classes.

| Role | Light | Dark | Tailwind |
| :--- | :--- | :--- | :--- |
| Canvas | `#ffffff` | `#09090b` | `bg-white` / `dark:bg-zinc-950` |
| Muted surface | `#fafafa` | `#18181b` | `bg-zinc-50` / `dark:bg-zinc-900` |
| Inset | `#f4f4f5` | `#27272a` | `bg-zinc-100` / `dark:bg-zinc-800` |
| Border | `#e4e4e7` | `white/10` | `border-zinc-200` / `dark:border-zinc-800` |
| Body text | `#09090b` | `#fafafa` | `text-zinc-900` / `dark:text-zinc-100` |
| Secondary text | `#71717a` | `#a1a1aa` | `text-zinc-500` / `dark:text-zinc-400` |

**Rule:** do not stack progressively darker gray panels. Canvas + `1px` border; `bg-zinc-50` only for table headers, hovers, filter wells.

---

## Typography

Canonical roles: `web/src/lib/admin_typography.ts`. Fonts: Inter Variable + JetBrains Mono in `app.css`.

| Role | Size | Weight | Classes |
| :--- | :--- | :--- | :--- |
| Page title | 20px | 600 | `text-xl font-semibold tracking-tight` |
| Section title | 15px | 600 | `text-[15px] font-semibold` |
| Body / UI | 13px | 400 | `text-sm` (14px default; table uses `text-[13px]` where dense) |
| Table header | 12px | 600 | `text-xs font-semibold text-zinc-500` |
| Numeric | 13px | 500 | `text-sm font-medium tabular-nums` |
| Code / ID | 12px | 400 | `font-mono text-xs` |

---

## Controls and primitives

Import from `@/components/ui/*`. Token source: `web/src/lib/admin_chrome.ts`. CI: `bash scripts/ci/admin/ui_slop.sh`.

| Primitive | Height | Radius | Notes |
| :--- | :--- | :--- | :--- |
| `Button`, `Input`, `SelectTrigger` | 32px (`h-8`) | `rounded-sm` (4px) | `variant`: default / secondary / outline / ghost / destructive |
| Panels, tables | — | `rounded-md` (6px) | `adminChrome.panel` |
| Nav pills, status chips | — | `rounded-full` | `ToggleChipGroup`, `SectionNav` only |

**Button hierarchy:** primary = `default` (zinc-950 fill); bulk toolbar = `secondary`; tertiary = `ghost`; destructive actions = `destructive`. Avoid `outline` on white toolbars (too low contrast).

Depth: no drop shadows on page panels. Overlays (`Dialog`, `DropdownMenu`, `Popover`) may use `shadow-lg`.

Motion: `transition-colors` on interactive nodes; `prefers-reduced-motion` respected in globals.

**Banned:** `components.json`, `npx shadcn`, `@radix-ui/*`, `class-variance-authority`, raw `<button>` in domains (use `Button`).

---

## Shell components (`web/src/shell/`)

| Component | File | Role |
| :--- | :--- | :--- |
| `PageChrome` | `page_chrome.tsx` | Title, description, badge, actions |
| `PageBreadcrumbs` | `page_breadcrumbs.tsx` | Route trail |
| `SectionNav` | `section_nav.tsx` | Pill nav for domain sections |
| `PageToolbar` | `page_toolbar.tsx` | Action row |
| `FilterPanel` / `DirectoryFilterForm` | `filter_panel.tsx` | Directory filter grid |
| `ToggleChipGroup` | `toggle_chip_group.tsx` | Server-driven status chips with counts |
| `DirectoryListMeta` | `directory_list_meta.tsx` | Showing X–Y of Z |
| `DirectoryTable` | `directory_table.tsx` | Scrollable table frame (replaces legacy `ui-table-frame`) |
| `DirectoryPaginationFooter` | `directory_pagination_footer.tsx` | Prev/Next + page size + range meta |
| `PaginationPrevNext` | `pagination_prev_next.tsx` | Prev/Next only (used inside footer shell) |
| `EmptyState` / `ErrorBlock` | `empty_state.tsx`, `error_block.tsx` | Blank and blocking errors |
| `HubLinkGrid` / `StatPanel` | `hub_link_card.tsx`, `stat_panel.tsx` | Hubs and KPI tiles |

**Banned in new JSX:** ad-hoc `rounded-md border` panel wrappers in domains; raw `fetch()` in domains/shell/pages; `ui-table-frame` (use `DirectoryTable`).

Directory tables use semantic `<table>` inside `DirectoryTable` / `Table` — not CSS grid matrices.

---

## Page patterns

### Directory list (canonical)

Reference: `web/src/domains/campaigns/list/campaigns_directory.tsx`.

Stack: `PageLayout` → `CampaignsListToolbar` (command + scope + filter well) → `CampaignsListTable` → footer pagination.

Use `DirectoryFilterForm layout="directory"` and `ToggleChipGroup` for status chips.

### Domain hub

`PageChrome` → `SectionNav` → `HubLinkGrid`.

### Ops / JSON dashboard

`OpsPageShell` → `OpsStatGrid` → `OpsBlock` tables.

### Campaigns metrics disclosure

| Tier | Trigger | Surface |
| :--- | :--- | :--- |
| 0 | Table row | Column cells |
| 1 | Budget used cell | `Popover` + trend chart |
| 2 | Row menu / link | `Sheet` overview |

---

## Charts

Colors: `text-zinc-900`, `text-zinc-500`, `border-zinc-200` for grid. Container: `adminChrome.panel`. Empty copy when series has no points.

---

## Verification

```bash
bash scripts/ci/admin/ui_slop.sh
bash scripts/ci/admin/web.sh
cd web && npm test
cd web && npm run typecheck
cd web && npm run test:e2e
rg '@radix-ui|class-variance-authority|shadcn' web/   # expect 0
rg 'className="admin-btn' web/src/domains web/src/shell web/src/pages   # expect 0
```

---

## Related docs

| Doc | Content |
| :--- | :--- |
| `FRONTEND.md` | Route maturity, backlog, e2e tiers |
| `docs/DEVELOPMENT.md` | Codegen, compose profiles, stack.sh |
| `.cursor/rules/ui.mdc` | Layout grid, error honesty, fixture ban |
| `.cursor/rules/frontend-modular.mdc` | Layer map |

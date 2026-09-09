# Admin design system

Visual reference: Admin Control Plane UI kit (buttons, inputs, badges, tables, sidebar, pagination, feedback).

Implementation map:

| Kit section | Tokens | Primitives | Shell |
| :--- | :--- | :--- | :--- |
| Spacing / typography | `admin_spacing.ts` | — | `page_layout.tsx`, `filter_panel_classes.ts` |
| Buttons | `admin_chrome.ts` | `components/ui/button.tsx` | `action_buttons.tsx` |
| Form inputs | `admin_kit.ts`, `admin_chrome.ts` | `input.tsx`, `textarea.tsx`, `select.tsx` | `filter_panel.tsx`, `search_input.tsx` |
| Checkboxes / toggles | `admin_kit.ts` | `checkbox.tsx`, `switch.tsx` | — |
| Status badges | `admin_kit.ts` | `badge.tsx` | `status_badge.tsx` |
| Tabs | — | `tabs.tsx` (`segmented`, `pill`, `underline`) | `filter_chip_group.tsx` |
| Tables | `admin_kit.ts`, `admin_chrome.ts` | `table.tsx`, `directory_table.tsx` | — |
| Pagination | — | — | `pagination_pages.tsx`, `directory_pagination_footer.tsx` |
| Sidebar | `shell_chrome.ts`, `app.css` | — | `app_sidebar.tsx` |
| Feedback / errors | `admin_kit.ts`, `admin_error.ts` | `sonner.tsx` | `error_block.tsx`, `admin_error_details.tsx`, `stub_banner.tsx` |
| Empty state | `admin_spacing.ts` | — | `empty_state.tsx` |
| Metric cards | `app.css` `.admin-metric-card` | `card.tsx` | `metric_card.tsx` |
| Progress | `app.css` `.admin-progress` | — | `progress_bar.tsx` |

Agent rules (full tables, layout contract, error catalog IDs): `.cursor/rules/ui.mdc`, `.cursor/rules/frontend-primitives.mdc`, `.cursor/rules/frontend-slop.mdc`.

## Color system: AEP Muted Cool

Visual anchor: dark auth card on VPS (Sign in button ~`#2b5278`, canvas ~`#141414`, card ~`#1c1c1c`, secondary link ~`#405d7e`). Palette is **desaturated, low-glare, operator-grade** -- not Material neon, not pgAdmin candy.

**Rules**

| Rule | Detail |
| :--- | :--- |
| Default theme | Dark (`html.dark`); light is parity, not a separate brand |
| Saturation cap | Brand hues stay below ~50% S; semantic hues below ~45% S |
| Color = meaning | State, metric sign, chart series, status only -- never decoration |
| No inline color | CSP: Tailwind tokens / `CHART_SWATCH_CLASS` only (`frontend-slop.mdc` **CSP-***) |
| Gradients | Banned on chrome and data surfaces |

Implementation: HSL components in `web/src/styles/tailwind.css` (`:root` light, `.dark` dark). Hex below is reference for design review; **code uses CSS vars**.

### Layer stack (surfaces)

Dark (canonical):

| Token | CSS var | Hex ref | Use |
| :--- | :--- | :--- | :--- |
| Canvas 0 | `--background` | `#141414` | Page shell, filter band backdrop |
| Canvas 1 | `--card` | `#1e1e1e` | Cards, table host, auth card, dialogs |
| Canvas 2 | `--popover` | `#222222` | Dropdowns, command palette, date popover |
| Canvas 3 | `--admin-control-bg` | `#1a1a1a` | Inputs, selects, idle buttons on card |
| Muted fill | `--muted` | `#333333` | Disabled chips, archived badge bg |
| Filters band | `--admin-filters-bg` | `#141414` | Directory filter panel (same as canvas) |

Light (muted cool parity):

| Token | CSS var | Hex ref | Use |
| :--- | :--- | :--- | :--- |
| Canvas 0 | `--background` | `#ebeef3` | Cool gray page (pgAdmin lineage, desaturated) |
| Canvas 1 | `--card` | `#ffffff` | Panels |
| Canvas 3 | `--admin-control-bg` | `#fafafa` | Inputs on white card |
| Muted fill | `--muted` | `#f3f5f9` | Subtle bands |

Elevation: **border + 1 step lighter bg**, not drop shadow. Shadow allowed only on toasts (`adminKit.toastSurface`).

### Borders and dividers

| Role | Dark `--border` | Light | Use |
| :--- | :--- | :--- | :--- |
| Subtle | `#2a2a2a` (16% L) | `#dde0e6` | Control outline (`border-border/40`), table grid |
| Default | same | same | Card outline, section split |
| Strong | `#3a3a3a` (23% L) | `#c8ccd4` | Focus-adjacent, pinned column edge |
| Scrollbar | `--scrollbar-thumb` | muted blue-gray | `.scrollbar-admin` |

### Brand (primary blue)

Muted navy -- the Sign in button family.

| Step | Dark hex | HSL (dark target) | Use |
| :--- | :--- | :--- | :--- |
| Brand 700 | `#254a6e` | `206 52% 22%` | Primary hover, `--admin-brand-hover` |
| Brand 600 | `#2b5278` | `206 52% 28%` | Primary button, `--primary`, `--admin-brand` |
| Brand 500 | `#3a6288` | `206 46% 32%` | Link default, `--ring` focus |
| Brand 400 | `#4d7399` | `206 40% 38%` | Link hover, ghost button hover text |
| Brand tint | `#1e2f42` | `206 36% 19%` | Selection wash, `--admin-selection` (dark) |
| Brand mist | `#2b5278` @ 10% | `primary/10` | Summary band, filter chip active bg |
| On-brand text | `#ffffff` | `0 0% 100%` | `--primary-foreground`, `--admin-brand-fg` |

Light brand center: `#326690` (`206 48% 38%`) -- same hue family, slightly more chroma for white backgrounds.

### Text hierarchy

| Role | Dark CSS | Dark hex | Light CSS | Use |
| :--- | :--- | :--- | :--- | :--- |
| Primary | `--foreground` | `#d4d4d4` | `#222222` | Body, table cells, control text |
| Strong | `--admin-fg-strong` | `#ededed` | `#171717` | Page title, emphasized KPI |
| Secondary | `--admin-fg-secondary` | `#b8b8b8` | `#5c6370` | Descriptions, meta links band |
| Muted | `--muted-foreground` | `#8a8a8a` | `#6b7280` | Placeholders, captions, zero metrics |
| Disabled | `--admin-muted` | `#666666` | `#9ca3af` | Disabled control text |
| Link | `text-primary` | Brand 500 | Brand 600 | Inline links, activate license |
| Link hover | underline + Brand 400 | | | Auth secondary actions |

Numeric data: `tabular-nums` + `--foreground`; money/KPI never mono (`admin_typography.ts`).

### Semantic (success, warning, error, info)

All semantic colors are **muted** -- readable on dark without glowing.

| Semantic | Dark fg token | Dark hex | Dark bg (alert/row) | Light fg | Use |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Success | `--admin-positive-fg` | `#4a8f58` (`124 32% 42%`) | `success/10` + border `active/25` | `#2e7d3e` | Active status, positive ROI, ops healthy |
| Warning | `--admin-warn-fg` | `#b8924a` (`38 42% 50%`) | `--admin-warn-bg` `#2a2418` | `#9a7340` | Paused, pacing, rate benchmarks |
| Error | `--destructive` | `#c4706a` (`6 35% 58%`) | `--admin-row-critical-bg` | `#c0392b` | Destructive btn, errors, negative delta |
| Info | `--admin-metric-conversion-fg` | `#5a85a8` (`206 38% 48%`) | `primary/10` | `#326690` | Hints, conversion metrics, stub banners |

**Metric sign mapping** (`admin_metric_tone.ts`): positive -> `text-admin-positive`; negative -> `text-destructive`; zero -> `text-muted-foreground`; stale -> `text-muted-foreground`.

### Status badges (entity state)

| Tone | Token pair | Appearance |
| :--- | :--- | :--- |
| Active | `--admin-status-active` | Muted green fg + `bg-admin-status-active/10` |
| Paused / scheduled | `--admin-status-paused` | Muted amber fg + `/10` wash |
| Draft | `--admin-status-draft` | Neutral gray fg + `/10` wash |
| Archived | `muted` | Border + muted bg |
| Error | `destructive` | Muted red fg + `destructive/10` |

Base chrome: `adminStatusBadgeBase` -- never per-row random hues.

### Table and selection

| State | Dark tokens | Use |
| :--- | :--- | :--- |
| Header | `--admin-table-header-bg` `#292929` | Column labels (caption typography) |
| Row hover | `--admin-table-row-hover` `#333333` | Directory tbody |
| Totals row | `--admin-table-totals-bg` | Footer aggregates |
| Selected row | `--admin-row-selected-bg` + `--admin-row-selected-edge` | Brand tint + left rail |
| Pinned col bg | `--admin-table-pin-bg` | Horizontal scroll pin |
| Warning row | `--admin-row-warning-*` | Margin breach, soft alerts |
| Critical row | `--admin-row-critical-*` | Hard failures in grid |

Tree/sidebar selection: same wash as `--admin-selection` (brand tint, not saturated blue block).

### Chart series (data viz)

Five **muted** series for dark mode; share hue with brand/semantic but lower saturation:

| Slot | CSS | Dark character | Typical metric |
| :--- | :--- | :--- | :--- |
| `--chart-1` | Brand navy | Volume, impressions, primary series |
| `--chart-2` | Muted green | Conversions, approved, positive |
| `--chart-3` | Muted amber | Rates, pacing, secondary |
| `--chart-4` | Cool gray-blue | CPC, neutral money |
| `--chart-5` | Deep slate | CPA, tertiary / compare |

Swatches: `adminKpiAccent*Class` and `DASHBOARD_*_AXIS_COLOR` at 78-95% alpha -- never raw hex in TSX.

Light charts: same slot order; `--chart-5` lifts to airy blue `#d6effc` for contrast on white.

### UI components (color recipes)

Use primitives + tokens only.

| Component | Default | Hover / focus | Disabled | Error |
| :--- | :--- | :--- | :--- | :--- |
| **Primary button** | `bg-primary text-primary-foreground` | `--admin-brand-hover` | opacity 50%, `--admin-input-disabled` | n/a |
| **Secondary button** | `bg-admin-control border-border/40` | `bg-accent` | muted text | n/a |
| **Ghost / link button** | transparent | `text-primary`, `bg-primary/10` | muted | n/a |
| **Destructive button** | `bg-destructive text-destructive-foreground` | darker destructive | opacity 50% | n/a |
| **Input / textarea** | `bg-admin-control border-border/40` | `border-border`, ring `--ring` | `--admin-input-disabled` | `border-destructive/50` |
| **Select / combobox** | same as input | popover `--popover` | same | same |
| **Checkbox / switch** | border `--border`; checked `primary` | focus ring | muted | n/a |
| **Card** | `bg-card border-border` | n/a | n/a | n/a |
| **Dialog / sheet** | card + overlay `bg-black/60` | n/a | n/a | n/a |
| **Tabs (segmented)** | idle `muted`; active `bg-card border-primary/20` | accent | muted | n/a |
| **Filter chips** | idle border; active `border-primary/40 bg-primary/15` | hover `primary/10` | n/a | n/a |
| **Toast** | `adminKit.toastSurface` | n/a | n/a | destructive tint for error toast |
| **ErrorBlock** | `uiSurfaces.messageError` | n/a | n/a | always |
| **StubBanner** | `messageMuted` or info tint | n/a | n/a | n/a |
| **EmptyState** | muted fg, no fake chart colors | n/a | n/a | n/a |
| **Pagination** | control chrome; current page `primary/10` | hover accent | disabled muted | n/a |
| **Sidebar** | canvas 0; active `bg-admin-selection font-semibold` | hover `bg-accent` | n/a | n/a |
| **Header** | fixed `bg-background border-border` | n/a | n/a | n/a |
| **Metric card** | `.admin-metric-card`; accent top bar `adminKpiAccentTopBarClass` | n/a | n/a | n/a |
| **Progress bar** | track `muted`; fill `primary` or semantic | n/a | n/a | n/a |
| **Calendar** | range middle `--admin-selection`; endpoints `primary` | | | |
| **Command palette** | popover bg; selection `--admin-selection` | | | |

Auth card (login / activate): `Card` on canvas 0; primary full-width button Brand 600; secondary action `text-primary` centered (Brand 500).

### Focus and interaction

| Pattern | Token |
| :--- | :--- |
| Focus visible | `ring-0` policy + border shift; calendar/select use `--ring` |
| Hover (controls) | `bg-accent` or `border-primary/40` -- never brighten saturation |
| Active press | 1 step darker than hover (brand-700 family) |
| Selected (not focus) | `--admin-selection` wash, not solid primary fill |

### Opacity scale (Tailwind)

| Alpha | Use |
| :--- | :--- |
| `/5` | Summary band bg |
| `/10` | Badge wash, message bg, ghost hover |
| `/15` | Active filter chip |
| `/25` | Message border, status border |
| `/40` | Control border default, chip hover border |
| `/60` | Dialog overlay companion borders |
| `/78` | Chart fill default |
| `/95` | Chart axis lines |

### Light/dark parity checklist

When adding a new surface color:

1. Define **both** `:root` and `.dark` HSL in `tailwind.css`.
2. Wire `--admin-*` if used in domains via `admin_metric_tone` or `adminStatusBadgeClass`.
3. Verify contrast: body text >= 4.5:1 on card; primary button text white on Brand 600.
4. No new hue outside brand / semantic / chart slots without DESIGN.md update.

### Quick reference: CSS var index

| Family | Vars |
| :--- | :--- |
| shadcn core | `--background`, `--foreground`, `--card`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`, `--popover` |
| Admin brand | `--admin-brand`, `--admin-brand-hover`, `--admin-brand-fg` |
| Admin text | `--admin-fg`, `--admin-fg-strong`, `--admin-fg-secondary`, `--admin-muted` |
| Admin metrics | `--admin-positive-fg`, `--admin-negative-fg`, `--admin-warn-*`, `--admin-metric-*` |
| Admin status | `--admin-status-active`, `--admin-status-paused`, `--admin-status-draft`, `--admin-status-scheduled` |
| Admin table | `--admin-table-*`, `--admin-row-*`, `--admin-selection` |
| Charts | `--chart-1` .. `--chart-5` |

Legacy note: pgAdmin hex names in old PRs map to this palette; canonical doc is **AEP Muted Cool** above.

## Spacing and typography (canonical)

**Single source of truth:** `web/src/lib/admin_spacing.ts` (`adminSpacing`, `adminTypography`). Re-exported from `admin_kit.ts`. Domains use shell roles and kit tokens -- not ad-hoc Tailwind scale in page files.

### Gap scale (only five steps)

| Token | Tailwind | px | Use |
| :--- | :--- | ---: | :--- |
| `adminSpacing.gap.xs` | `gap-1` | 4 | Icon + label inside one control |
| `adminSpacing.gap.sm` | `gap-1.5` | 6 | Chip inner, tight label row |
| `adminSpacing.gap.md` | `gap-2` | 8 | Button groups, toolbar peers, label-to-control in `FilterField` |
| `adminSpacing.gap.lg` | `gap-3` | 12 | Section stacks, panel interiors, main column |
| `adminSpacing.gap.xl` | `gap-4` | 16 | Workspace bands, filter field matrix, main + aside grid |

**Banned in `web/src/domains/**` and `web/src/pages/**`:** `gap-5`, `gap-6`, `gap-7`, `gap-8` (enforced by `bash scripts/ci/admin/ui_slop.sh`).

**Rule:** `gap` only between siblings in one semantic group at the same level. Space between page bands (control panel / main / footer) uses `PageLayout` wrappers and `adminSpacing.flex.workspaceFlat`, not a large `gap` on a flat parent. See `frontend-slop.mdc` **Layout contract**.

### Padding (inset)

| Token | Class | Use |
| :--- | :--- | :--- |
| `adminSpacing.inset.canvas` | `px-6 py-4` | Route canvas (`PageCanvasInset`) |
| `adminSpacing.inset.panel` | `p-4` | Cards, selection panel, section panels |
| `adminSpacing.inset.band` | `px-4 py-2` | Dialog section header, table caption |
| `adminSpacing.inset.bandLg` | `px-4 py-3` | Dialog footer |
| `adminSpacing.inset.tableCellX` | `px-4` | Table `th` / `td` horizontal |
| `adminSpacing.inset.footer` | `px-6 pt-3` | Page footer band |
| `adminSpacing.inset.emptyState` | `px-6 py-16` | `EmptyState` only |
| `adminKit.controlPaddingX` | `px-3` | Buttons, inputs, selects |

Prefer grid tracks and band wrappers over ad-hoc `margin-*` between unrelated blocks.

### Typography roles

| Token | Size / weight | Use |
| :--- | :--- | :--- |
| `adminTypography.pageTitle` | 18px bold | Page `h1` only |
| `adminTypography.sectionTitle` | 13px semibold | Section `h2`, empty state title |
| `adminTypography.panelTitle` | 13px semibold | Aside control panel title |
| `adminTypography.body` | 13px regular | Body, controls, table cells |
| `adminTypography.bodyMuted` | 13px muted | Secondary copy, list meta, page description |
| `adminTypography.label` | 13px medium | Form labels (`FilterField`, `adminKit.fieldLabelClass`) |
| `adminTypography.labelMuted` | 13px muted | Definition-list labels |
| `adminTypography.caption` | 11px semibold caps | Table headers, column labels |
| `adminTypography.badge` | 12px (`text-xs`) | Status badges only |
| `adminTypography.monoData` | 12px mono | UUID, deployment id, secrets |

**Banned in domains/pages:** `text-sm`, `text-base`, `text-xs`, arbitrary `text-[Npx]`. **Banned:** `font-bold` outside page title (use role token).

Numeric columns: Inter + `tabular-nums` (`admin_typography.ts`, `.num`). Money and KPI counts never use `font-mono`; mono is for opaque identifiers and JSON only.

### Layout presets (shell)

| Export | Use |
| :--- | :--- |
| `adminSpacing.flex.buttonGroup` | Refresh + Export + Create cluster |
| `adminSpacing.flex.actionsRowEnd` | Dialog footer actions |
| `adminSpacing.flex.pageHeader` | Title row + header actions |
| `adminSpacing.grid.filterMatrix` / `CAMPAIGNS_FILTER_ROW` | Directory filter grids |
| `COMPACT_TOOLBAR_ROW_CLASS` | Pagination + page size row |
| `page_layout.tsx` exports | Canvas inset, workspace fill, section stack, footer band |

## Radii

| Surface | Radius |
| :--- | :--- |
| Controls (button, input) | `8px` (`adminKit.controlRadius`) |
| Cards / table shell | `8px` (`adminKit.panelRadius`) |
| Nested chips / menu rows | `4px` (`adminKit.nestedRadius`) |
| Pill actions / status badge | `rounded-full` (`adminKit.pillRadius`) |

## Explicit error handling

Operators must always know when a request **failed**. The UI must not pretend success, hide failures behind empty tables, or dump stack traces into the main surface.

**Definitions:**

| Term | Meaning |
| :--- | :--- |
| Explicit failure | Non-2xx or thrown `ApiError` becomes visible UI (`ErrorBlock`, `StubBanner`, mutation error slot) |
| Safe message | Short operator text from `userErrorMessage()` or server `error.message` when it is already operator-safe |
| Technical details | Stack, status code dump, raw payload -- **never** in default production UI |
| Error slop | Silent `catch`, fake empty data, toast before `await`, `200` on validation failure |

### Operator-visible copy (hide technical details)

Canonical module: `web/src/lib/admin_error.ts`.

| Function / component | Role |
| :--- | :--- |
| `userErrorMessage(error, fallback?)` | Maps `ApiError` status/code to safe text; generic message for `5xx` and unknown `Error` |
| `adminErrorTitle` / `adminErrorUserMessage` | Route-level pages (404, forbidden, load failure) |
| `ErrorBlock` | Inline / full-section alert: **title + safe message** |
| `formatAdminErrorDetails` | Builds developer dump (status, code, stack) |
| `AdminErrorDetails` | Renders dump only when `shouldShowAdminErrorDetails()` is true (local dev; **false** in shipped builds) |
| `panelError(error, title)` | `403` / `501` -> `StubBanner`; other errors -> `ErrorBlock` with safe message |
| `DirectoryMutationError` | Directory overlay mutation slot |

**Required mapping (client):**

| HTTP / condition | Operator sees |
| :--- | :--- |
| `401` | Session expired; sign in again |
| `403` | No permission (or `StubBanner` when route is license-gated) |
| `404` | Resource or page not found |
| `5xx` | Generic server error; try again later (not raw exception text) |
| `501` / feature stub | `StubBanner` with server message -- explicit degraded mode |
| Validation `4xx` | Server `error.message` when present |
| Abort / cancelled fetch | No error UI (`isAbortError`) |

**Banned in operator-visible strings:** SQL fragments, Redis keys, file paths, `panic:` lines, internal service names, raw JSON payloads. Pass `error` into `ErrorBlock` for the details panel in dev; pass **`userErrorMessage(error)`** (or let `ErrorBlock` resolve it) for the main paragraph -- not `error.stack` in production copy.

```tsx
// Explicit -- operator-safe message, optional dev details
<ErrorBlock title="Could not save campaign" error={actionError} />

// Slop -- never
catch { /* ignored */ }
catch { setItems([]); }
toast.success('Saved');
<p>{err.stack}</p>
```

### Fetch lifecycle (lists)

| Phase | UI |
| :--- | :--- |
| First load, `error && !hasSnapshot` | Blocking `ErrorBlock` or `panelError` -- not `EmptyState` |
| Refresh, `error && hasSnapshot` | Keep last rows; inline refresh error or toast |
| Success, zero rows | `EmptyState` (filters active vs blank slate) |
| Missing scope (no customer id) | Disable action or scope prompt -- not fake `[]` |

### Mutations (forms, dialogs)

1. `await` the API call inside `try/catch`.
2. On failure: `setActionError` / `ErrorBlock`; **keep dialog open** on `4xx`.
3. Success toast, close, or navigate **only after** `2xx` response.
4. Never `toast.success` before `await` completes.

### Server parity (admin API)

Errors use `{"error":{"code","message"}}` only. `5xx` responses: generic operator message in JSON; full error logged server-side (`slog.Error`). Status must match semantics (`400` validation, `404` missing, `409` conflict). See `frontend-slop.mdc` **ES1-ES7**.

### Banned patterns (error slop)

| Pattern | Why |
| :--- | :--- |
| Empty `catch {}` / `catch { return }` | Operator cannot tell the action failed |
| `catch` -> empty table / `setItems([])` | Failure masquerades as "no data" |
| `data ?? []` when fetch errored | Masks shape errors and HTTP failures |
| `Promise.resolve([])` as fake API | Silent stub without `StubBanner` |
| `genericOk()` / fake `200` on unimplemented route | False success |
| Toast before `await` | Phantom success |
| `console.error` only, no UI | Debug theater |
| `EmptyState` when `error && !hasSnapshot` | Confuses error with zero rows |
| Raw stack / status dump in main UI | Leaks implementation detail |

Catalog IDs for review and CI: `frontend-slop.mdc` **EH-***, **EC1-EC9**, **ES1-ES7**.

## Verification

```bash
cd web && npm run typecheck
bash scripts/ci/admin/ui_slop.sh
bash scripts/ci/admin/web.sh
```

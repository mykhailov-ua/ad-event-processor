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

## Colors

Palette aligned with [pgAdmin standard theme](https://www.pgadmin.org/styleguide/themes/color_palettes/) (`#326690` primary, `#EBEEF3` hover, `#D6EFFC` selection).

| Role | Light token |
| :--- | :--- |
| Primary action | `--admin-brand` `#326690`, hover `--admin-brand-hover` |
| Danger / success / warn | `--destructive`, `--admin-positive-fg`, `--admin-warn-fg` |
| Focus ring | `--ring` (primary blue) |
| Canvas | `--background` `#EBEEF3`, panels `--card` white |
| Controls | `--input` border `#DDE0E6`, disabled `--admin-input-disabled` `#F3F5F9` |
| Menu / tree selection | `--admin-selection` `#D6EFFC` |
| Table header | `--admin-table-header-bg` `#F3F5F9` |

Use color only to encode data or state. Decorative gradients and per-row random pill colors are banned.

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

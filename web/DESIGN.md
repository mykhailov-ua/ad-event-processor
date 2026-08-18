# BidShard Admin — Design System

Spec for the embedded admin UI (`web/`).

| Layer | Source of truth | Adopt |
|-------|-----------------|--------|
| **Visual** | Neutral zinc dark (Linear / Cursor / shadcn) + Geist type | Near-black canvas, subtle borders, inverted primary CTA |
| **UX / interaction** | [Grafana Saga](https://grafana.com/developers/saga/about/overview/) | Principles, alerts, forms, friction, page templates — adapt |
| **UI chrome** | Zinc tokens + selective Saga patterns | Flat cards, 14px body, minimal sidebar chrome |

**Not adopted:** `@grafana/ui`, Grafana Storybook components, Grafana palette/typography as a full replacement for Geist tokens.

**Stack:** React 19 + react-router-dom + TypeScript (`web/src/`). Routing: `app_routes.tsx`. Layout: `shell_layout.tsx`. Styling stays CSS tokens — not CSS-in-JS.

Implementation: `web/src/styles/tokens.css` + `main.css` + `components.css` + `a11y.css`. Shared controls: `web/src/components/`. Confirm flows: `helpers/confirm_catalog.ts` / `confirm_registry.ts`. Toasts: `helpers/toast_ui.ts`.

---

## 1. Dual foundation

### 1.1 Visual (Geist → BidShard)

| Geist concept | BidShard token | Usage |
|---------------|----------------|--------|
| Background 1 | `--bg-canvas` | Page shell, inputs |
| Gray 100–300 | `--bg-surface`, `--bg-surface-hover` | Panels, hover/active fills |
| Gray 400–600 | `--border-color`, `--border-strong` | Dividers, control borders |
| Gray 900 | `--text-secondary`, `--text-muted` | Labels, hints |
| Gray 1000 | `--text-main` | Headings, body |
| Blue 1000 | `--accent` | Primary CTA, links, focus |
| Materials base (6px) | `--radius-sm` | Buttons, inputs, chips |
| Materials menu/modal (12px) | `--radius-lg`, `--radius-modal` | Modals, menus |

Reference: [Geist colors](https://vercel.com/geist/colors), [materials](https://vercel.com/geist/materials).

### 1.2 UX principles (Saga → BidShard)

Adapted from [Saga design principles](https://grafana.com/developers/saga/foundations/design-principles/):

1. **Task-at-hand** — each view optimizes the operator’s primary job (pace campaign, clear backlog, read doctor). Secondary chrome stays quiet.
2. **Tasteful friction** — UI resistance matches blast radius. Rename / filter tweak: low friction. Pause campaign, revoke API key, budget overwrite, blacklist: confirm via catalog. Never surprise-destructive.
3. **System carries complexity** — prefer smart defaults, progressive disclosure, and clear errors over dumping ops jargon on the user.
4. **Default to reusability** — same page chrome, panels, tables, empty states, alerts. New layout only when an existing template cannot express the task.

Visual Geist rules that still apply:

5. **Border over shadow** — 1px `--border-color` on surfaces; shadows only on floating layers (modal, toast, menu).
6. **Typography hierarchy** — one dominant metric per view; secondary text `--text-muted`.
7. **Color = meaning** — green/amber/red for status only; charts monochrome + accent line.
8. **4px spacing grid** — `--spacing-4` … `--spacing-48`; default gaps `--spacing-16` / `--spacing-24`.
9. **Dark default** — `storage.js` sets `data-theme="dark"`; light is opt-in.

---

## 2. Typography

Geist type scale (CSS classes in `system.css`):

| Class | Size / weight | Use |
|-------|----------------|-----|
| `.text-heading-24` | 24px / 600 | Page titles (`page-header__title`) |
| `.text-heading-20` | 20px / 600 | Section titles |
| `.text-label-14` | 14px / 500 | Nav, buttons, table headers |
| `.text-label-13` | 13px / 500 | Sidebar groups, compact labels |
| `.text-label-12` | 12px / 500 | Captions, badges |
| `.text-copy-14` | 14px / 400 | Default body |
| `.text-copy-13` | 13px / 400 | Dense tables, hints |
| `.font-mono` | Geist Mono stack | IDs, money, KPI values |

Type-scale tokens: `--text-2xs` (0.6875rem) through `--text-2xl` (1.5rem) in `tokens.css`.

**Heading semantics (Saga Text pattern):** one `h1` per view; ranks continuous (`h1` → `h2` → `h3`). Change appearance with classes, not by skipping heading levels. Do not use color alone for emphasis — color is status.

**Font loading:** `build.mjs` links Geist Sans/Mono from `cdn.jsdelivr.net`. Air-gapped installs may remove those `<link>` tags; `tokens.css` falls back to `system-ui`.

Letter-spacing: `-0.02em` on headings ≥20px.

---

## 3. Layout & page templates

### Shell

- Fixed sidebar (`--sidebar-width`) + scrollable main (`main` / `main__content`, max-width 1600px, padding `--spacing-24`).
- Grid: `grid-stats` / `settings-grid` → `repeat(auto-fit, minmax(200px, 1fr))`, gap `--spacing-16`.
- Card-less sections: `page-header` + flat `settings-panel` with border — not stacked shadows.

### Templates (reuse before inventing)

Inspired by [Saga templates](https://grafana.com/developers/saga/templates/overview/) — same jobs look the same:

| Template | Structure | Examples |
|----------|-----------|----------|
| **Overview / hub** | `PageHeader` → alerts strip → `KpiGrid` → sections | Overview, Ops home, Reports hub |
| **List** | `PageHeader` + filters + `data-table` + `EmptyState` | Campaigns, customers, blacklist |
| **Detail** | `PageHeader` + tabs/panels + primary actions in header | Campaign detail, customer detail |
| **Settings / form** | `PageHeader` + stacked `SectionCard` bodies + action bar | Settings, margin guard, postback, filters |
| **Wizard** | Stepper + one-column body + single primary CTA | Campaign wizard |

Empty states: short title, one sentence why empty, optional single primary action — no decorative empty illustrations. Use `<EmptyState>` component.

### 3.1 Component library (Phase 0)

Standard layout components under `web/src/components/`:

| Component | File | Purpose |
|-----------|------|---------|
| `PageHeader` | `page_header.tsx` | h1 + breadcrumbs + desc + actions slot. Required on every authenticated page. |
| `PageStack` | `page_stack.tsx` | Root wrapper that enforces `--space-lg` gap between sections. Prevents card collapse. |
| `PageSkeleton` | `page_skeleton.tsx` | Shimmer loading state. Replaces bare "Loading…" text. |
| `EmptyState` | `empty_state.tsx` | Standardized empty state: icon + title + desc + one CTA. |
| `SectionCard` | `section_card.tsx` | Raised surface with optional header (wraps `.settings-panel`). |
| `FormField` | `form_field.tsx` | Labeled field with hint/error slot and aria-invalid wiring. |

### 3.2 CSS module split (Phase 0)

`main.css` is now a pure `@import` entry. Domain CSS lives in:

| Module | Contents |
|--------|---------|
| `shell.css` | Sidebar, main, #app-outlet, banner-slot, mobile |
| `page.css` | page-header, breadcrumbs, error-page, login, dev-gallery |
| `cards.css` | section-card, metric-card, settings-panel, doctor-panel, definition-list |
| `tables.css` | data-table, skeleton-bar, empty-state |
| `forms.css` | inputs, labels, checkboxes, date-picker, select, check |
| `controls.css` | buttons, segmented-control, chips, filter-toolbar, pagination |
| `feedback.css` | badges, banners, modals, toasts, cmd-palette, status-hint |
| `charts.css` | chart-root, uPlot, ops-kpi-strip, metric-chart-card |
| `components.css` | Size tokens + loading spinner overrides (canonical btn--primary) |

Import order = cascade order. `components.css` must remain last.

---

## 4. Feedback: alerts, toasts, banners

Saga [Alert pattern](https://grafana.com/developers/saga/patterns/alert/) mapped to BidShard widgets:

| Kind | When | Placement | BidShard |
|------|------|-----------|----------|
| **Inline** | Task / page context (doctor fail, validation, policy) | Top or bottom of main content / panel | `alert-banner`, `alert-feed__item` |
| **Toast** | Short result of a user action (saved, copied, blocked) | Corner stack; auto-dismiss or × | `pushToastMessage` (`helpers/toast_ui.js`) |
| **Banner** | Persistent system state until resolved or dismissed | Page-level strip | `alert-banner` / home alerts |

| Status | Use | Token / badge |
|--------|-----|----------------|
| Info | Non-blocking information | `--info` / `badge--info` |
| Success | Action completed as expected | `--success` / `badge--success` |
| Warning | Undesirable but recoverable | `--warning` / `badge--warning` |
| Error | Failure; user should act | `--danger` / `badge--danger` |

**Copy rules**

- Title required: short, human, no stack traces or raw codes as the only text.
- Details optional: what happened + path forward.
- CTAs / links only on persistent (inline/banner) alerts — not on fleeting toasts unless essential.
- Prefer toast for “Saved” / “Copied”; prefer inline for form/field failures still on screen.

---

## 5. Forms

Adapted from [Saga Forms](https://grafana.com/developers/saga/patterns/forms/), spaced with BidShard tokens:

### Structure

- **Header** — title + short description (`settings-panel__header` / `page-header`).
- **Body** — left-aligned, **one column**, vertical stack.
- **Action bar** — primary submit; secondary/cancel distinct.

### Spacing (map Saga 16 / 40 → tokens)

| Gap | Token | Where |
|-----|-------|--------|
| Between fields / controls | `--spacing-16` | Form stack |
| Title ↔ body, body ↔ actions, between sections | `--spacing-24` or `--spacing-32` | Prefer `--spacing-24` at BidShard density; use `--spacing-32` for heavy settings sections |

### Behavior

- One **primary** button per form (`btn btn--primary`). Left-align with the field column.
- Destructive / discard: away from primary (header or secondary); confirm via catalog when irreversible.
- Labels associated with controls; required only when truly required (`*` sparingly).
- Inline validation; error on the field + how to fix. Never disable the submit button as the only error signal — explain why submit failed.
- Progressive disclosure: advanced options behind expander / “Show advanced”; wizards for long creates.
- Dialog (modal): ≤ ~5 fields, stays in context. Longer side flows: drawer/panel if already in shell patterns; else full-page settings template.
- Smart defaults over empty required fields when a safe majority choice exists.

---

## 6. Components (visual)

| Pattern | Classes | Notes |
|---------|---------|--------|
| Primary button | `btn btn--primary` | `--accent` fill, 6px radius; one per form/view action cluster |
| Secondary | `btn btn--secondary` | `--bg-surface-hover` + border |
| Nav item | `sidebar__link` | Active: `--bg-surface-hover`, not accent wash |
| KPI tile | `metric-card` | Label 12px uppercase muted; value 24px mono |
| Doctor row | `doctor-stack__row` | Two columns, border-separated |
| Panel | `settings-panel` | Header + body; flat border |
| Modal | `modal` | 12px radius, `--shadow-lg` |
| Banner / inline alert | `alert-banner`, `alert-feed__item` | Semantic border + subtle fill |
| Table | `data-table` | 13px, row hover `--bg-surface-hover` |

---

## 7. Status colors

| State | Token | Badge |
|-------|-------|-------|
| OK | `--success` | `badge--success` |
| Warning | `--warning` | `badge--warning` |
| Error | `--danger` | `badge--danger` |
| Info | `--info` | `badge--info` |

---

## 8. Accessibility

Target: **WCAG 2.1 AA** intent (Saga accessibility posture), implemented with Geist contrast.

- Focus: `:focus-visible` ring in `a11y.css` + `--focus-ring` on inputs.
- Contrast: primary text on canvas ≥ 7:1 (Geist gray 1000 on dark canvas).
- Forms: visible labels tied to controls; errors announced with the field.
- Live regions: prefer polite toasts; assertive only for critical, user-triggered failures.
- Motion: respect `prefers-reduced-motion`.
- E2E: prefer role-based queries (`web/e2e/`).

---

## 9. Do not

- Adopt Grafana visual language (colors, type, denseness) as a replacement for Geist tokens.
- Drop a full npm UI kit (MUI, Ant, Chakra) as the default layout system — it fights token-based CSS and duplicates primitives under `web/src/components/`.
- Box-shadow on every card (Stripe/legacy dashboard style).
- Accent-colored inactive nav items.
- Hard-coded hex in page/components — CSS variables only.
- Multiple primary buttons competing in one form.
- Toast for persistent system faults that need an on-page path (use inline/banner).
- Disable submit with no explanation.
- New one-off page layouts when a §3 template fits.

---

## 10. Dependencies, charts, and bundle

Admin is embedded in `control` (`go:embed` on `web/dist/`). **Dev** (`npm run dev`, port 5173) serves a fresh build; **production** serves whatever was built before `go build` — rebuild `web/dist` after UI changes.

### npm dependencies

- **Allowed:** chart libraries (Chart.js, ECharts, uPlot, Recharts, etc.), date/math utilities, small focused hooks — prefer lazy `import()` on route or tab activation.
- **Discouraged:** monolithic UI kits that replace `tokens.css` + shared components; duplicate icon sets.
- **Process:** add to `web/package.json`, run `npm ci`, keep types green (`npm run typecheck`).

### Charts

- Today: canvas helpers in `web/src/charts/` + **uPlot** for ops/campaign series.
- New work: pick any maintained chart lib; **lazy-load** on the page that needs it (ops dashboards, reports, campaign detail). Match Geist/Saga chart rules (§1.2 rule 7): status colors sparingly; default series monochrome + one accent.
- Do not fetch chart data the UI never renders (anti-slop §11).

### Bundle posture (CI)

`scripts/ci/admin_bundle_gate.sh` enforces **generous** uncompressed limits so chart deps are not blocked:

| Check | Limit (approx.) |
|-------|-----------------|
| Total JS under `dist/src/` | 5 MB |
| `main.js` entry | 1 MB |
| Each lazy chunk | 2 MB |

Goals: keep `main.js` tiny (shell + router), push heavy pages and chart vendors into lazy chunks, avoid duplicate chart stacks on the same route. If a feature needs headroom, raise the gate with a one-line comment in the script — not a DESIGN.md essay.

**Not a goal:** sub‑1 MB total bundle at the expense of operator charts or maintainability.

---

## 11. Anti-slop and honesty (2026 admin UI)

Operators and buyers must never think a screen works when it does not. This section is the **product honesty** bar for `web/`.

### 11.1 What counts as slop / lying

| Class | Example (forbidden) | Why |
|-------|---------------------|-----|
| **False live** | `live: true` in `REPORT_CATALOG` but route mounts `report_stub_page` | User pays for analytics that 501 |
| **Dead fetch** | `api('/rtb/deals')` then always render “No rows — connect API” | Looks wired; data discarded |
| **Skeleton in prod copy** | Page desc “(skeleton)” while linked from `nav_config.ts` | Admits placeholder in GA chrome |
| **Silent failure** | `catch` → empty table with no `renderErrorBlock` / toast | Hides outage as “no data” |
| **Fake save** | Toast “Saved” before `apiConfirmed` resolves 2xx | User believes mutation persisted |
| **Phantom fields** | Form submits `budget_limit` when `PatchCampaignRequest` has no such field | UI lies about server contract |
| **Demo KPIs** | Hardcoded `metric-card` numbers not from API | Fraudulent dashboard |
| **Docs ≠ routes** | DESIGN claims shipped without `app_routes.tsx` + e2e | Agent/human marketing drift |
| **Mock e2e overclaim** | Playwright mock cited as backend/CH/PG proof | Spec `harness=mock_api`; use stack smoke or Go integration test |
| **Marketing filler** | “Seamless”, “cutting-edge”, “revolutionize” in operator UI | AI boilerplate; no signal |
| **Secret echo** | Show `api_token`, JWT, webhook secret after save | Security + fake “configured” state |

**Not slop:** loading `tableSkeletonRows` during fetch; `stub-banner` on real 501/404; `retired: true` reports with redirect; honest empty state **after** successful API returning `items: []`.

### 11.2 Required patterns (how to avoid)

| Situation | Do this |
|-----------|---------|
| API not ready | **No nav link**; or `report_stub_page` + hub card without `live: true` |
| API 501 / stub backend | `StubBanner` + link to live alternative (`components/stub_banner.tsx`) |
| API error / timeout | `renderErrorBlock` or `mapServiceError` toast — never blank table |
| Partial fan-out (`503` + items) | Yellow `stub-banner` listing `errors[]` (ops outbox pattern) |
| Report is GA | `live: true` + dedicated route in `app_routes.tsx` + `report_live_routes_gate.sh` + handler returns rows or documented empty-state |
| Mutation | `apiConfirmed` + `confirm_registry` level matches blast radius |
| Money | `formatUsdDecimal` / `formatAmountMicro` — never string concat `$` |
| Types | `web/src/types/*` matches Go `json` tags — grep handler DTO before form |
| Agent claims “done” | Must cite green command: `bash scripts/ci/admin_web.sh` (includes slop gate) |

### 11.3 Verification (CI + manual)

**Automated** (in `admin_web.sh`):

```bash
bash scripts/ci/report_live_routes_gate.sh   # live reports ≠ stub route
bash scripts/ci/check_ui_literals.sh         # EN copy, money helpers
bash scripts/ci/check_ui_slop.sh           # skeleton / fake-wiring phrases
bash scripts/ci/check_web_security.sh      # no console.log, secrets in URLs
```

Stack finance path (Go + Postgres testcontainer, not Playwright mock): `bash scripts/test/billing_export_smoke.sh` — see `docs/DEVELOPMENT.md` skip matrix.

**Per-feature before merge:**

1. Playwright: mock API returns rows → table shows them; mock 500 → error visible.
2. Manual: Network tab — no fetch on hidden tabs until selected (lazy ops tabs).
3. `grep` view for `TODO` / `skeleton` / `FIXME` in user-visible strings.

### 11.4 Agent / LLM checklist

When implementing or reviewing `web/` changes:

- [ ] Read Go handler + DTO before building form fields.
- [ ] Run `npm run typecheck` and targeted e2e — do not claim green without output.
- [ ] If API response is unused, **delete the fetch** or wire the table — never “pretend shipped”.
- [ ] Do not set `live: true` without route + gate.
- [ ] Copy: imperative, short, English — title + what to do next (§4 copy rules).
- [ ] No new nav entry until happy path works with mocked 200 in Playwright.

---

## 12. Changing the theme or UX rules

1. Visual tokens: edit semantic mappings in `tokens.css` (not one-off component CSS).
2. UX rules: edit this doc first; then align `confirm_catalog`, toast helpers, and shared panel/form classes.
3. Run `node web/scripts/build.mjs` and verify Overview, Settings, Login.
4. Check light + dark via sidebar theme toggle.

### External references

- Geist: https://vercel.com/geist/introduction
- Saga overview: https://grafana.com/developers/saga/about/overview/
- Saga principles: https://grafana.com/developers/saga/foundations/design-principles/
- Saga alerts: https://grafana.com/developers/saga/patterns/alert/
- Saga forms: https://grafana.com/developers/saga/patterns/forms/

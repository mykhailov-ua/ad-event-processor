# BidShard Admin — Design System

Spec for the embedded admin UI (`web/`).

| Layer | Source of truth | Adopt |
|-------|-----------------|--------|
| **Visual** | [Vercel Geist](https://vercel.com/geist/introduction) | Tokens, type, surfaces, chrome — keep |
| **UX / interaction** | [Grafana Saga](https://grafana.com/developers/saga/about/overview/) | Principles, alerts, forms, friction, page templates — adapt |
| **UI chrome** | Geist + selective Saga patterns | Forms spacing, alert kinds, one primary CTA — where it fits BidShard density |

**Not adopted:** `@grafana/ui`, Grafana Storybook components, Grafana palette/typography. **React:** allowed under [`.cursor/WEB_REACT_MIGRATION.md`](../.cursor/WEB_REACT_MIGRATION.md) (Phase 0+); legacy `mount()` views coexist until ported. No Geist React or npm UI kits.

Implementation: `web/src/styles/tokens.css` + `main.css` + `components.css` + `a11y.css`. Shared controls: `web/src/ui/button.ts`, `web/src/ui/form_field.ts`. Confirm flows: `helpers/confirm_catalog.js` / `confirm_registry.js`. Toasts: `helpers/toast_ui.js`.

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

Geist type scale (CSS classes in `main.css`):

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
| **Overview / hub** | `page-header` → alerts strip → KPI row → sections | Overview, Ops home, Reports hub |
| **List** | `page-header` + filters + `data-table` + empty state | Campaigns, customers, blacklist |
| **Detail** | `page-header` + tabs/panels + primary actions in header | Campaign detail, customer detail |
| **Settings / form** | `page-header` + one or more `settings-panel` form bodies + action bar | Settings, margin guard, postback, filters |
| **Wizard** | Stepper + one-column body + single primary CTA | Campaign wizard |

Empty states: short title, one sentence why empty, optional single primary action — no decorative empty illustrations.

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

- Adopt Grafana visual language (colors, type, denseness) as a replacement for Geist.
- Import `@grafana/ui`, Geist React, or any npm UI kit into `web/src` (React view layer in `web/src/react/` only — see `WEB_REACT_MIGRATION.md`).
- Box-shadow on every card (Stripe/legacy dashboard style).
- Accent-colored inactive nav items.
- Hard-coded hex in `views/` — CSS variables only.
- Multiple primary buttons competing in one form.
- Toast for persistent system faults that need an on-page path (use inline/banner).
- Disable submit with no explanation.
- New one-off page layouts when a §3 template fits.

---

## 11. Anti-slop and honesty (2026 admin UI)

Operators and buyers must never think a screen works when it does not. This section is the **product honesty** bar for `web/` — complementary to engineering DoD in [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md) §1.0. **React migration does not relax this bar** (see `WEB_REACT_MIGRATION.md` §4).

### 11.1 What counts as slop / lying

| Class | Example (forbidden) | Why |
|-------|---------------------|-----|
| **False live** | `live: true` in `REPORT_CATALOG` but route mounts `report_stub.js` | User pays for analytics that 501 |
| **Dead fetch** | `api('/rtb/deals')` then always render “No rows — connect API” | Looks wired; data discarded — see `rtb_deals.ts` (allowlisted debt until §1.3.1) |
| **Skeleton in prod copy** | Page desc “(skeleton)” while linked from `nav_config.ts` | Admits placeholder in GA chrome |
| **Silent failure** | `catch` → empty table with no `renderErrorBlock` / toast | Hides outage as “no data” |
| **Fake save** | Toast “Saved” before `apiConfirmed` resolves 2xx | User believes mutation persisted |
| **Phantom fields** | Form submits `budget_limit` when `PatchCampaignRequest` has no such field | UI lies about server contract — see MILESTONE §1.2.4 |
| **Demo KPIs** | Hardcoded `metric-card` numbers not from API | Fraudulent dashboard |
| **Docs ≠ routes** | MILESTONE/DESIGN “shipped” without `routes.ts` + e2e | Agent/human marketing drift |
| **Mock e2e overclaim** | Playwright mock cited as backend/CH/PG proof | Spec `harness=mock_api`; use stack smoke or Go integration test |
| **Marketing filler** | “Seamless”, “cutting-edge”, “revolutionize” in operator UI | AI boilerplate; no signal |
| **Secret echo** | Show `api_token`, JWT, webhook secret after save | Security + fake “configured” state |

**Not slop:** loading `tableSkeletonRows` during fetch; `stub-banner` on real 501/404; `retired: true` reports with redirect; honest empty state **after** successful API returning `items: []`.

### 11.2 Required patterns (how to avoid)

| Situation | Do this |
|-----------|---------|
| API not ready | **No nav link**; or `report_stub.js` + hub card without `live: true`; or `placeholder.ts` pattern (not in `nav_config`) |
| API 501 / stub backend | `renderStubBanner` + link to live alternative ([`stub_banner.ts`](src/ui/stub_banner.ts)) |
| API error / timeout | `renderErrorBlock` or `mapServiceError` toast — never blank table |
| Partial fan-out (`503` + items) | Yellow `stub-banner` listing `errors[]` (ops outbox pattern) |
| Report is GA | `live: true` + dedicated route in `routes.ts` + `report_live_routes_gate.sh` + handler returns rows or documented empty-state |
| Mutation | `apiConfirmed` + `confirm_registry` level matches blast radius |
| Money | `formatUsdDecimal` / `formatAmountMicro` — never string concat `$` |
| Types | `web/src/types/api/*` matches Go `json` tags — grep handler DTO before form |
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

**Known debt (remove allowlist when fixed):**

- `web/src/views/rtb_deals.ts` — §1.3.1 RTB deals CRUD ([MILESTONE](../.cursor/MILESTONE.md) §1.3.1).

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

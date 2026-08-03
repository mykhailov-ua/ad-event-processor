# BidShard Admin — Design System

Visual spec for the embedded admin UI (`web/`). Based on [Vercel Geist](https://vercel.com/geist/introduction): high-contrast neutrals, minimal chrome, color reserved for status and primary actions.

Implementation: `web/src/styles/tokens.css` (variables) + `main.css` (components). No React/Geist npm packages — vanilla CSS only (`admin-web.mdc`).

---

## 1. Foundations (Geist → BidShard)

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

---

## 2. Principles

1. **Border over shadow** — surfaces use 1px `--border-color`; shadows only on floating layers (modal, toast, menu).
2. **Typography hierarchy** — one dominant metric per view; secondary text uses `--text-muted`.
3. **Color = meaning** — green/amber/red for status only; charts monochrome + accent line.
4. **4px spacing grid** — `--spacing-4` … `--spacing-48`; default gaps `--spacing-16` / `--spacing-24`.
5. **Dark default** — `storage.js` sets `data-theme="dark"`; light is opt-in.

---

## 3. Typography

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

**Font loading:** `build.mjs` links Geist Sans/Mono from `cdn.jsdelivr.net`. Air-gapped installs may remove those `<link>` tags; `tokens.css` falls back to `system-ui`.

Letter-spacing: `-0.02em` on headings ≥20px.

---

## 4. Layout

- **Shell:** fixed sidebar (`--sidebar-width`) + scrollable `main__content` (max-width 1600px, padding `--spacing-24`).
- **Grid:** `grid-stats` / `settings-grid` use `repeat(auto-fit, minmax(200px, 1fr))` with `--spacing-16` gap.
- **Card-less sections:** prefer `page-header` + flat `settings-panel` with border, not stacked shadows.

---

## 5. Components

| Pattern | Classes | Notes |
|---------|---------|--------|
| Primary button | `btn btn--primary` | `--accent` fill, 6px radius |
| Secondary | `btn btn--secondary` | `--bg-surface-hover` + border |
| Nav item | `sidebar__link` | Active: `--bg-surface-hover`, not accent wash |
| KPI tile | `metric-card` | Label 12px uppercase muted; value 24px mono |
| Doctor row | `doctor-stack__row` | Two columns, border-separated |
| Modal | `modal` | 12px radius, `--shadow-lg` |
| Banner | `alert-banner` | Semantic border + subtle fill |
| Table | `data-table` | 13px, row hover `--bg-surface-hover` |

---

## 6. Status colors

| State | Token | Badge |
|-------|-------|-------|
| OK | `--success` | `badge--success` |
| Warning | `--warning` | `badge--warning` |
| Error | `--danger` | `badge--danger` |
| Info | `--info` | `badge--info` |

---

## 7. Accessibility

- Focus: `:focus-visible` ring in `a11y.css` + `--focus-ring` on inputs.
- Contrast: primary text on canvas ≥ 7:1 (Geist gray 1000 on black).
- Motion: respect `prefers-reduced-motion` for segmented-control pill transitions.

---

## 8. Do not

- Box-shadow on every card (Stripe/legacy dashboard style).
- Accent-colored inactive nav items.
- Hard-coded hex in `views/` — use CSS variables only.
- npm UI libraries or Geist React components in `web/src`.

---

## 9. Changing the theme

1. Edit semantic mappings in `tokens.css` (not component CSS).
2. Run `node web/scripts/build.mjs` and verify Overview, Settings, Login.
3. Check light + dark via sidebar theme toggle.

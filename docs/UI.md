# UI

Admin design system (`web/`). Visual: Geist zinc dark tokens. UX patterns: Grafana Saga (principles, not `@grafana/ui`).

Sources: `web/src/styles/tokens.css`, `main.css`, `components.css`, `a11y.css`. Stack: React 19 + TypeScript; CSS tokens only.

## Tokens

| Token | Usage |
| :--- | :--- |
| `--bg-canvas` | Page shell, inputs |
| `--bg-surface`, `--bg-surface-hover` | Panels, hover |
| `--border-color`, `--border-strong` | Dividers, controls |
| `--text-main`, `--text-secondary`, `--text-muted` | Body, labels, hints |
| `--accent` | Primary CTA, links, focus |
| `--radius-sm` (6px) | Buttons, inputs |
| `--radius-lg`, `--radius-modal` (12px) | Modals, menus |
| `--spacing-4` … `--spacing-48` | 4px grid; default gaps 16/24 |
| `--success`, `--warning`, `--danger`, `--info` | Status only |

**Rules:** border over shadow (shadows on modal/toast/menu only); dark default; no hard-coded hex in components.

## Typography

| Class | Use |
| :--- | :--- |
| `.text-heading-24` | Page titles |
| `.text-heading-20` | Section titles |
| `.text-label-14` / `.text-copy-14` | Nav, buttons, body |
| `.font-mono` | IDs, money, KPI values |

One `h1` per view; continuous heading ranks; color = status, not emphasis.

## Page templates

Reuse before inventing:

| Template | Structure | Examples |
| :--- | :--- | :--- |
| **Overview** | `PageHeader` → alerts → `KpiGrid` → sections | Overview, Ops |
| **List** | header + filters + `data-table` + `EmptyState` | Campaigns, blacklist |
| **Detail** | header + tabs/panels + header actions | Campaign detail |
| **Settings** | header + `SectionCard` stack + action bar | Settings, filters |
| **Wizard** | stepper + one column + single primary CTA | Campaign wizard |

Required chrome: `PageHeader` on every authenticated page; `PageStack` for section gaps; `PageSkeleton` for loading; `EmptyState` for empty (title + one sentence + optional CTA).

**Confirm friction:** match blast radius via `confirm_catalog` — low for renames; confirm for pause, revoke key, budget overwrite.

## Anti-slop (product honesty)

Operators must never think a screen works when it does not.

**Forbidden:**

| Class | Example |
| :--- | :--- |
| False live | `live: true` but route is stub |
| Dead fetch | API call then always "No rows" |
| Silent failure | `catch` → empty table, no error |
| Fake save | "Saved" toast before 2xx |
| Phantom fields | Form field not in Go DTO |
| Demo KPIs | Hardcoded metrics |
| Secret echo | Show token/JWT after save |

**Required:**

| Situation | Do |
| :--- | :--- |
| API not ready | No nav link, or stub without `live: true` |
| 501 / stub | `StubBanner` + link to alternative |
| Error | `renderErrorBlock` or error toast — never blank table |
| GA report | `live: true` + route + gate + real or documented empty data |
| Mutation | `apiConfirmed`; money via `formatUsdDecimal` / `formatAmountMicro` |
| Types | Match Go `json` tags before building forms |

**CI gates** (in `admin_web.sh`): `report_live_routes_gate.sh`, `check_ui_literals.sh`, `check_ui_slop.sh`, `check_web_security.sh`.

**Agent checklist:** read handler DTO first; run `npm run typecheck`; no `live: true` without route; no fetch unless data is rendered; imperative English copy.

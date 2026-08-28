# ADMIN_TOKENS_MILESTONE

Design tokens in `rem` only. No page layout; establishes variables consumed by `ui/system/` and domain modules.

**Status:** DRAFT  
**Slug:** `admin_tokens`  
**Depends on:** admin_contract_gate  
**Blocks:** admin_shell  
**Pattern:** shell (design tokens)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| px in tokens | Breaks rem scale | rg 'px' web/src/styles/tokens.css — spacing rows empty |
| Hex in domain CSS | Bypasses semantic palette | rg '#[0-9a-fA-F]{3,8}' web/src/ui --glob '*.module.css' |
| Token sprawl | Per-page one-off vars | Add to tokens.css with name in 4.4 table |
| Shell without --sidebar-width | Magic number sidebar | T4 shell module uses var |
| Asymmetric chip padding | FreshnessBadge layout bug | T5 symmetric --space-1 |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Inline hex in new module | Quick color fix | var(--color-*) from tokens |
| px gaps in grid | Copy from legacy global CSS | rem tokens only |
| Duplicate token names | --gap vs --space | Single spacing scale |
| tokens in pages/ | Page imports tokens.css | Modules only per frontend-modular.mdc |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Skip token doc in 4.4 | Undocumented magic vars | Table every new token name |
| Leave legacy hex | Only new files matter | New modules gate-clean |
| admin_web.sh not run | Assume tokens fine | Paste exit 0 in section 7 |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Tokens shipped" with px in tokens.css spacing scale
- "Chip contract met" without symmetric padding token
- New domain module with hard-coded hex

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `web/src/styles/tokens.css` — spacing, typography, color semantic tokens
- Document token names in milestone section 4.4
- Remove hard-coded hex from **new** CSS modules (legacy globals may remain until migrated)
- `--sidebar-width`, chip padding tokens for shell and badges

### Out of scope

- Components, routes, API
- Domain-specific grid templates
- Page layout CSS

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| T1 | `--space-*` in tokens.css | rem only — no px spacing in tokens file |
| T2 | `--text-*-leading`, font size tokens | Referenced by system components |
| T3 | Semantic colors | surface, border, text, danger, success — no product hex in new domain CSS |
| T4 | `--sidebar-width` | shell.css or shell module uses var |
| T5 | Chip/badge tokens | symmetric padding variable per ui.mdc |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| spacing | tokens.css --space-* | Vertical rhythm scale | static rem |
| typography | tokens.css --text-* | Font sizes + leading | static rem |
| color | tokens.css --color-* | Semantic surfaces and text | no raw hex in new modules |
| layout | tokens.css --sidebar-width, --page-max-width | Shell + page width caps | static |
| chip | tokens.css chip padding | Symmetric badge inset | ui.mdc |
| row_hover | tokens.css --color-row-hover | Directory row hover | static |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
(No page — tokens.css only)
┌ tokens.css ─────────────────────────────┐
│ --space-*  --text-*  --color-*         │
│ --sidebar-width  --page-max-width      │
│ chip / badge padding tokens            │
└────────────────────────────────────────┘
         │
         ▼ consumed by ui/shell/ + ui/system/ + ui/<domain>/*.module.css
```

| Invariant | Value |
| :--- | :--- |
| rem only | Spacing and type tokens use rem, not px |
| Semantic colors | Domain modules reference var(--color-*), not #hex |
| No tokens in pages | pages/ never import tokens.css directly — via modules |

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/<domain>/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Primary fetch | OpenAPI | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/styles/tokens.css | Canonical design tokens |
| web/src/ui/shell/shell.module.css | First consumer: sidebar width token |
| web/src/ui/system/*.module.css | Button, ErrorBlock, chips use tokens |
| scripts/ci/admin_web.sh | Orchestrator includes UI gates when web/ exists |


**Remove from this route (legacy):** Hard-coded hex in new `web/src/ui/**/*.module.css`, Ad-hoc px spacing in new domain modules.

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| px spacing tokens | Breaks ui.mdc rem rhythm |
| Hex in *.module.css | Fails literal scan when wired |
| --sidebar-width missing | Shell uses magic 240px |
| Nested chip frames | Token exists but component double-wraps |
| Page imports tokens | pages/ must not import CSS modules or tokens |
| Global style leak | New work duplicates tokens in styles/*.css |
| Asymmetric badge padding | ui.mdc violation — one layer chip |
| Token rename without consumers | Grep all var(--old) before delete |
| Dark mode fiction | No dark theme in SKU — do not add unprefixed dark tokens |
| Typography px | Use rem for --text-* sizes |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | web/src/styles/tokens.css | T1 spacing scale in rem | No px in --space-* |
| 2 | tokens.css | T2 typography tokens | shell/system reference vars |
| 3 | tokens.css | T3 semantic color palette | document in 4.4 |
| 4 | tokens.css + shell.module.css | T4 --sidebar-width | sidebar uses var |
| 5 | tokens.css | T5 chip padding token | FreshnessBadge uses symmetric var |
| 6 | web/src/ui/system/ | Wire tokens into Button, ErrorBlock | typecheck |
| 7 | bash scripts/ci/admin_web.sh | Admin orchestrator | exit 0 |
| 8 | rg hex scan | New modules only | no # in new *.module.css |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| N/A | Tokens | — | No runtime SLA |

## 7. Verification (paste in PR)

```bash
bash scripts/ci/admin_web.sh
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| admin_web | bash scripts/ci/admin_web.sh | exit 0 |
| px ban | rg '\d+px' web/src/styles/tokens.css | no px in spacing rows |
| hex scan | rg '#[0-9a-fA-F]{3,8}' web/src/ui --glob '*.module.css' \|\| true | only legacy allowed |
| sidebar var | rg 'sidebar-width' web/src/ui/shell/ | uses var(--sidebar-width) |
| typecheck | cd web && npm run typecheck | exit 0 |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied

- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/<domain>/`, helpers, and page compose in one slice; no half migration.


# ADMIN_PAGE_CHROME_MILESTONE

Reusable page frame: PageChrome, Toolbar slot, Footer slot, system primitives for all domain pages.

**Status:** DRAFT  
**Slug:** `admin_page_chrome`  
**Depends on:** admin_shell  
**Blocks:** directory, detail, report, dashboard pages  
**Pattern:** shell (shared chrome)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Nested chip chrome | PageHeader wraps badge | P2 one layer only |
| Portal filter listbox | Narrow/wide menu bug | P3 inline drop |
| PageChrome missing | Domain builds own h1 stack | test -f page_chrome.tsx |
| Pagination emits page not offset | API expects offset | PaginationBar contract |
| ErrorBlock skipped | Silent empty tables | G5 ErrorBlock on blocking fail |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Wrapper PageChrome | *DirectoryChrome around PageChrome | Flat section compose |
| Freshness without API field | Invented lag chip | Omit until envelope has freshness |
| Flex section stack | flex-col page root | CSS Grid sections |
| btn--primary in new code | Legacy global BEM | ui/system/Button |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Reuse components/page_header | Skip system/ | New ui/system primitives |
| Skip surface gate | typecheck only | check_ui_surface_gate.sh P4 |
| Copy ErrorBlock from components/ | Leave duplicate | Single system ErrorBlock |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Page chrome done" without check_ui_surface_gate.sh exit 0
- Nested chip borders on FreshnessBadge
- pages/*.tsx importing *.module.css
- Portal listbox on filter Select fields

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `web/src/ui/system/` — PageChrome, FreshnessBadge, ErrorBlock, PaginationBar, Button, EmptyState, PageSkeleton
- Section stack documented: ContextBar → PageChrome → Toolbar → Content → Footer
- Chip: symmetric padding; one layer — no nested frame
- Field Select/listbox: inline drop, wrapper width 100%
- Sample directory page consumes chrome (dogfood with customers or stub)

### Out of scope

- Domain grids
- Business API helpers
- Route registration

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| P1 | Section stack | ContextBar → PageChrome → Toolbar → Content → Footer documented in 4.3 |
| P2 | Chip contract | symmetric padding var(--space-1); no nested frame |
| P3 | Field listbox | inline .drop; wrapper width 100% |
| P4 | Surface gate | bash scripts/ci/check_ui_surface_gate.sh pass on sample page |
| PageChrome | title slot + optional badge slot direct child | no wrapper chrome |
| PaginationBar | prev/next emits offset only | parent refetches |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | optional tenant/customer scope breadcrumb | URL + session |
| page_chrome | PageChrome | title + optional badge slot (direct child) | API title or static |
| freshness | FreshnessBadge | freshness_label, stale from API | envelope — omit if absent |
| toolbar_slot | Toolbar slot | primary actions region below chrome | permissions |
| error | ErrorBlock | API error message + retry | fetch error |
| skeleton | PageSkeleton | grid-shaped loading placeholder | loading state |
| empty | EmptyState | one sentence + action when no rows | items.length === 0 |
| pagination | PaginationBar | prev/next + range label | limit, offset, total |
| button | Button | replaces btn--* BEM on pages | variants.ts + cn() |
| select | Select / FilterDropdown | inline drop listbox | draft filter fields |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
ContextBar (optional)
┌ PageChrome ─────────────────────────────────────────────┐
│ Title                              [FreshnessBadge?]    │
├ Toolbar (slot) ─────────────────────────────────────────┤
│ primary actions                                         │
├ FilterPanel (domain — optional) ────────────────────────┤
├ Content ────────────────────────────────────────────────┤
│ role=grid | form | report body                          │
├ Footer ─────────────────────────────────────────────────┤
│ PaginationBar | save bar                                │
└─────────────────────────────────────────────────────────┘
ErrorBlock replaces content on blocking fetch failure
```

| Invariant | Value |
| :--- | :--- |
| Section stack | P1 order: ContextBar → PageChrome → Toolbar → Content → Footer |
| One chrome per region | No nested PageChrome wrappers |
| Chip single layer | P2 symmetric padding; parent renders badge directly |
| Listbox inline | P3 field dropdowns not portal-measured for filter fields |
| Pages compose only | pages/ import domain + system components; no *.module.css on pages |

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
| web/src/ui/system/page_chrome.tsx | Title + badge slot |
| web/src/ui/system/page_chrome.module.css | Chrome grid |
| web/src/ui/system/freshness_badge.tsx | API-driven chip |
| web/src/ui/system/error_block.tsx | Blocking error UI |
| web/src/ui/system/pagination_bar.tsx | Offset pagination control |
| web/src/ui/system/button.tsx | Button + variants.ts |
| web/src/ui/system/empty_state.tsx | Empty copy + action |
| web/src/ui/system/page_skeleton.tsx | Loading skeleton |
| web/src/ui/system/select.tsx | Inline drop listbox |
| web/src/ui/shell/context_bar.tsx | Optional scope breadcrumb |
| scripts/ci/check_ui_surface_gate.sh | P4 gate |


**Remove from this route (legacy):** web/src/components/page_header.js patterns, btn--* BEM on new pages, Nested FreshnessBadge wrapper chrome.

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Double chip chrome | Gray box around colored pill |
| Asymmetric chip padding | ui.mdc violation |
| Portal filter dropdown | Width measure bug — use inline drop |
| Page imports CSS module | surface gate fail |
| Pagination sets page index only | Must sync offset URL param |
| ErrorBlock inside grid only | Should replace content region on blocking error |
| EmptyState on error | Differentiate empty vs ErrorBlock |
| Freshness invented | No API field — omit badge |
| Toolbar flex escape | Toolbar is grid section leaf |
| Button BEM leak | New pages use system Button only |
| Skeleton wrong shape | PageSkeleton must match grid columns |
| ContextBar duplicates PageChrome title | Scope vs title separation |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | web/src/ui/system/page_chrome.tsx | P1 title + badge slot | no nested wrapper |
| 2 | freshness_badge.tsx | P2 chip tokens | symmetric padding visual |
| 3 | select.tsx | P3 inline listbox | wrapper width 100% |
| 4 | error_block.tsx + pagination_bar.tsx | G5 error + offset footer | compose test page |
| 5 | button.tsx + empty_state.tsx + page_skeleton.tsx | Replace btn--* / empty patterns | typecheck |
| 6 | context_bar.tsx | Optional scope bar | shell integration |
| 7 | bash scripts/ci/check_ui_surface_gate.sh | P4 on sample page | exit 0 |
| 8 | cd web && npm run typecheck | S5 parity | exit 0 |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| N/A | Chrome | — | No hot path |

## 7. Verification (paste in PR)

```bash
bash scripts/ci/check_ui_surface_gate.sh
cd web && npm run typecheck
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Surface gate | bash scripts/ci/check_ui_surface_gate.sh | exit 0 |
| typecheck | cd web && npm run typecheck | exit 0 |
| Chip visual | Manual: FreshnessBadge | symmetric inset, one border |
| Listbox width | Manual: filter Select open | drop matches field width |
| No page CSS | rg '\.module\.css' web/src/pages/ | no imports |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied

- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/<domain>/`, helpers, and page compose in one slice; no half migration.


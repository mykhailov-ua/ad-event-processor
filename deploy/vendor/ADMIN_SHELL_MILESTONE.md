# ADMIN_SHELL_MILESTONE

Login, bootstrap, app shell (sidebar, nav, session guard), embed `web/dist` in control binary.

**Status:** DRAFT  
**Slug:** `admin_shell`  
**Depends on:** admin_tokens  
**Blocks:** all authenticated pages  
**Pattern:** shell  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| #root placeholder text | Breaks boot inject | TestAdminStaticRoutes S6 |
| Session guard bypass | Domain route without guard | Manual unauthenticated fetch |
| Nav shows forbidden routes | Missing permission check | S4 manual role matrix |
| Stub embed left on | dist not wired | embed.go points to web/dist |
| EULA skipped | App loads without accept | POST eula/accept enforced |
| Login error swallowed | Silent fail | Show API error on login form |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Non-empty #root | Helpful placeholder copy | Empty div only |
| Nav live without permission | All items visible | nav_config permission filter |
| Bootstrap fields invented | TS-only platform fields | OpenAPI bootstrap schema only |
| Double shell wrap | Layout inside layout | One AppShell outlet |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Keep admin_static_stub | Avoid npm build | Ship dist + embed in same PR |
| Skip EULA gate | Faster dev | S2/S3 manual checks |
| Copy legacy login CSS | Global btn--* | ui/shell modules + tokens |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Shell shipped" without TestAdminStaticRoutes pass
- "Login works" without POST session manual check
- Text inside `<div id="root">`
- Domain PageChrome inside shell milestone PR

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/login`, `/bootstrap`, `/install/done` routes
- Session guard: loading → authenticated / login redirect / ForbiddenPage
- Sidebar nav from `nav_config.ts`; permission-gated items
- EULA gate: modal or gate page before app
- Embed `web/dist` via `web/embed.go`; remove stub when dist ships
- Empty `<div id="root"></div>` in boot HTML

### Out of scope

- Domain pages
- PageChrome / directory grids
- Customer/campaign routes

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| S1 | TestAdminStaticRoutes | go test ./internal/controlplane/ -run TestAdminStaticRoutes |
| S2 | Login | POST /api/v1/session; cookie set; redirect `/` |
| S3 | Auth guard | Unauthenticated → `/login`; RBAC fail → forbidden page |
| S4 | Nav RBAC | Nav items respect session permissions |
| S5 | typecheck | cd web && npm run typecheck |
| S6 | Empty #root | No text inside `<div id="root">` in boot HTML |
| Login API | POST /api/v1/session | OpenAPI platform.yaml |
| Meta | GET /api/v1/meta | Role hints for overview |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| login | LoginForm | email, password, submit, API error text | POST /api/v1/session |
| bootstrap | BootstrapForm | platform bootstrap fields from OpenAPI only | bootstrap APIs |
| install_done | InstallDonePage | static post-install | static |
| shell_sidebar | AppSidebar | nav groups; collapse; user footer | nav_config + session |
| shell_main | AppOutlet | route outlet for authenticated pages | react-router |
| guard | SessionGuard | loading → auth / login redirect | GET session |
| forbidden | ForbiddenPage | RBAC failure surface | static |
| eula | EulaGate | POST accept before app | POST /api/v1/eula/accept |
| root | #root | empty mount point | index.html / login.html |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/login` |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ AppShell (grid: sidebar | main) ────────────────────────┐
│ ┌ Sidebar ───┐ ┌ Main outlet ─────────────────────────┐ │
│ │ nav groups │ │  (domain routes render here)          │ │
│ │ user footer│ │                                       │ │
│ └────────────┘ └───────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘

/login, /bootstrap: full-width forms (no sidebar)
SessionGuard wraps authenticated /*
```

| Invariant | Value |
| :--- | :--- |
| Shell grid | sidebar \| main — sidebar fixed width token |
| No flex page sections | Domain pages use PageChrome milestone grid inside main |
| Empty root | #root has no child text before boot inject |
| Nav permission | Hide nav item when session lacks permission |

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
| Primary fetch | POST /api/v1/session; GET /api/v1/session; GET /api/v1/meta; GET/POST /api/v1/settings/platform/bootstrap; POST /api/v1/eula/accept | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/ui/shell/ | sidebar, layout, login form modules |
| web/src/ui/shell/shell.module.css | sidebar \| main grid |
| web/src/pages/login_page.tsx | thin compose |
| web/src/pages/bootstrap_page.tsx | bootstrap compose |
| web/src/pages/install_done_page.tsx | static done page |
| web/src/pages/forbidden_page.tsx | 403 surface |
| web/src/app_boot.tsx | guard + router boot |
| web/src/app_routes.tsx | route table + lazy imports |
| web/src/helpers/nav_config.ts | nav groups + permissions |
| web/src/helpers/auth.ts | session read/write |
| web/embed.go | serve web/dist |
| internal/controlplane/admin_static_stub/ | removed when dist ships |


**Remove from this route (legacy):** internal/controlplane/admin_static_stub/ when web/dist embed ships, Text inside #root in stub HTML, Global flex layout on app root (migrate to shell grid).

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| #root text | Static route test fails — empty root only |
| Guard on wrong route | Login page redirect loop |
| Permission nav leak | Buyer sees operator-only links |
| EULA bypass via deep link | Guard must wrap all authenticated routes |
| embed.go still stub | Production serves placeholder |
| Session cookie not set | Login 2xx but no cookie — manual S2 |
| Bootstrap OpenAPI drift | Form fields not on PATCH body |
| Flex shell layout | Use grid sidebar \| main |
| Nav config duplicate | app_routes and nav_config drift |
| Forbidden page missing | 403 shows blank — use ForbiddenPage |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | web/src/ui/shell/ | Shell grid sidebar \| main | typecheck |
| 2 | web/src/pages/login_page.tsx | LoginForm POST session | manual S2 redirect |
| 3 | web/src/app_boot.tsx | SessionGuard + EulaGate | manual S3 |
| 4 | web/src/helpers/nav_config.ts | Permission-filtered nav | manual S4 |
| 5 | web/src/app_routes.tsx | Login, bootstrap, guarded /* | routes resolve |
| 6 | web/embed.go | Embed web/dist | TestAdminStaticRoutes S1 |
| 7 | index.html / login.html | Empty #root S6 | static test pass |
| 8 | Remove admin_static_stub | When dist ships | single embed target |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| N/A | Shell | — | Not hot path; not /track |

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/admin_web.sh
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| S1 static routes | go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1 | pass |
| S2 login | Manual: valid credentials | redirect `/`; session cookie set |
| S3 guard | Manual: unauthenticated /customers | redirect `/login` |
| S4 nav RBAC | Manual: buyer session | operator-only nav hidden |
| S5 typecheck | cd web && npm run typecheck | exit 0 |
| S6 empty root | curl -s boot HTML \| rg 'id="root"' | no inner text |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied

- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/<domain>/`, helpers, and page compose in one slice; no half migration.


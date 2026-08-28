# ADMIN_SELFERVE_MILESTONE_KEYS

API keys CRUD; raw_key shown once in ApiKeyOnceModal on POST create

**Status:** DRAFT  
**Slug:** `admin_selfserve_keys`  
**Depends on:** admin_detail_customer or buyer dashboard  
**Blocks:** —  
**Pattern:** self-serve  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| raw_key shown twice | Key in table after create | ApiKeyOnceModal only; list id/name |
| Key in toast | pushToastMessage with secret | Toast title only; modal holds key |
| Operator smuggle | Admin nav links to selfserve | Role guard on route group |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per self-serve |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- raw_key column in grid
- console.log(raw_key)
- Toast body containing raw_key

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/selfserve/api-keys` — API keys CRUD; raw_key shown once in ApiKeyOnceModal on POST create
- API: GET/POST /api/v1/selfserve/api-keys
- Confirm modal: POST /api/v1/selfserve/api-keys
- POST create returns raw_key once; list GET must not echo secret

### Out of scope

- Operator-only admin routes
- Persisting raw_key client-side

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| List | GET /api/v1/selfserve/api-keys | no raw_key in list response |
| Create | POST /api/v1/selfserve/api-keys | returns raw_key once (OpenAPI required) |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | API keys | static |
| form | ApiKeyCreateForm | name input + create | POST api-keys |
| modal | ApiKeyOnceModal | raw_key copy-once UI | create response only |
| list | ApiKeysList | id, name, expires_at — no secret column | GET api-keys |
| warning | AlertBanner | Keys shown once at creation | static copy |
| error | ErrorBlock | Create/list failure | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/selfserve/api-keys` |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/selfserve/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Primary fetch | GET/POST /api/v1/selfserve/api-keys | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/selfserve_api_keys_page.tsx | Compose |
| web/src/ui/selfserve/* | Self-serve sections |
| web/src/helpers/selfserve_api.ts | API helpers |
| web/src/ui/selfserve/selfserve_shell_layout.tsx | Buyer chrome |

**Legacy page:** `selfserve_api_keys_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| raw_key replay | Show raw_key once in modal; never in list grid or logs |
| Operator route leak | Self-serve pages buyer-scoped only; no admin shortcuts |
| Toast before 2xx | apiConfirmed on key create and campaign pause/resume |
| Client portfolio filter | visiblePortfolioRows is legacy; rebuild with server params |
| Flex page root | CSS Grid via selfserve_shell_layout sections |
| Silent catch → empty | ErrorBlock on portfolio/billing fetch fail |
| Demo billing KPIs | Statement fields from selfserve billing API only |
| Key in localStorage | Never persist raw_key beyond session modal |
| Invented selfserve fields | OpenAPI selfserve paths only |
| Copy models/buyer.js | Delete on migrate; OpenAPI types |
| Bulk pause without confirm | Confirm modal on bulk campaign actions |
| Payment intent secrets | No echo of client_secret after create |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi paths selfserve | Confirm portfolio/billing/keys ops | openapi_gate.sh |
| 2 | make openapi-types | Regenerate types | typecheck |
| 3 | web/src/helpers/selfserve_api.ts | Bearer selfserve fetch helpers | Compiles |
| 4 | web/src/ui/selfserve/* | Sections per milestone | surface gate |
| 5 | web/src/pages/selfserve_*_page.tsx | Compose under selfserve shell | Route resolves |
| 6 | web/src/app_routes.tsx | Self-serve route group + session guard | Buyer role only |
| 7 | ApiKeyOnceModal | raw_key one-time display component | List route never returns raw_key |
| 8 | Manual key create | POST → modal → dismiss clears state | Reload list shows id/name only |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| raw_key once | Manual: create API key | modal shows secret; list has no raw_key column |
| No localStorage key | DevTools: localStorage/sessionStorage | no raw_key persisted |
| Billing error | Manual: block statement API | ErrorBlock visible |
| Buyer scope | Manual: operator session | redirect or 403 on /selfserve/* |
| Confirm bulk pause | Manual: bulk pause → cancel | no POST |
| Campaign create | Manual: wizard completes | campaign appears in portfolio refetch |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `selfserve_api_keys_page.tsx` replaced or delegates to `ui/selfserve/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/selfserve/`, helpers, and page compose in one slice; no half migration.


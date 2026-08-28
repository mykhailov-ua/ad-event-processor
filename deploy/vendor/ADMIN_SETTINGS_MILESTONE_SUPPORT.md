# ADMIN_SETTINGS_MILESTONE_SUPPORT

Settings: /support/feedback.

**Status:** DRAFT  
**Slug:** `admin_settings_support`  
**Depends on:** admin_page_chrome  
**Blocks:** —  
**Pattern:** detail  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Secret echo | Re-show stripe_secret after PATCH | Clear inputs; show masked metadata only |
| Toast before 2xx | Optimistic save | apiConfirmed |
| Gap nav lie | live link without route | gap_route + nav live=false |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per detail |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Saved" toast before PATCH 2xx
- Display full license JWT or stripe secret after save
- Form fields not on Go PATCH struct

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/support/feedback` — Operator feedback form with deployment meta
- API: GET /api/v1/support/feedback/meta; POST /api/v1/support/feedback

### Out of scope

- Platform hot-path config reads on tracker
- Client-side license JWT parsing

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Meta | GET /api/v1/support/feedback/meta | SupportFeedbackMeta |
| Submit | POST /api/v1/support/feedback | SupportFeedbackRequest |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Support feedback | static |
| meta | FeedbackMeta | deployment_id, binary_version | supportFeedbackMeta |
| form | FeedbackForm | type, message, attach_bundle | SupportFeedbackRequest |
| success | FeedbackSuccess | Recorded id after 201 | SupportFeedbackResponse |
| toolbar | FeedbackSubmit | POST feedback | supportFeedbackCreate |
| error | ErrorBlock | POST failure | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/support/feedback` |
| Permission | settings:read / settings:write per OpenAPI x-permissions |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome → [TabBar] → Toolbar → Content → ErrorBlock

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/support/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Primary fetch | GET /api/v1/support/feedback/meta; POST /api/v1/support/feedback | Server owns business rules |
| Display | `*_display`, `status_label` | Render only |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/support_feedback_page.tsx | Compose |
| web/src/ui/support/* | Settings sections + CSS modules |
| web/src/helpers/support_api.ts | API helpers |
| web/src/types/generated/openapi.d.ts | Generated types |

**Legacy page:** `support_feedback_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Toast before 2xx | apiConfirmed after PATCH/POST apply |
| Echo secrets after save | Stripe/license JWT cleared from input after 2xx; never re-display |
| live without route | Register app_routes + nav before catalog live |
| TS-only PATCH fields | PlatformSettingsPatch ⊆ pkg/platformconfig.Patch |
| Flex page root | CSS Grid sections per ui.mdc |
| Silent catch → empty | ErrorBlock on blocking GET |
| Gap route nav lie | gap_route milestones: nav live=false until route registered |
| Client license validation | POST /api/v1/license/apply validates server-side only |
| Invented team fields | TeamMember DTO from OpenAPI only |
| Double chip chrome | status_label chips one layer |
| Disputes client filter | customer_id query param + server pagination |
| Report schedule without customer | customer_id required query on list |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/platform.yaml | Confirm settings/team/license/disputes ops | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/support_api.ts | Typed GET/PATCH/POST helpers | Compiles |
| 4 | web/src/ui/support/* | Form/grid sections per 4.1 | check_ui_surface_gate.sh |
| 5 | web/src/pages/support_feedback_page.tsx | Compose; no CSS modules on page | ≤120 lines |
| 6 | web/src/app_routes.tsx | Route registration (gap routes: step 6 mandatory) | Lazy import resolves |
| 7 | web/src/ui/shell/nav | Settings nav entry; gap routes stay live=false | RBAC |
| 8 | Secret handling | Clear JWT/stripe inputs after 2xx | rg no value={savedSecret} |

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
| PATCH parity | Cross-check OpenAPI Patch body vs form | No extra TS fields |
| No secret echo | Manual: save stripe key → reload page | masked placeholder only |
| License apply | Manual: paste JWT → apply → reload | token input cleared; status updates |
| ErrorBlock | Manual: block GET | ErrorBlock visible |
| Gap route honest | When gap_route: nav hidden or live=false | No dead nav link |
| Team tab URL | Manual: ?tab=approvals | Approvals queue loads |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `support_feedback_page.tsx` replaced or delegates to `ui/support/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/support/`, helpers, and page compose in one slice; no half migration.


# ADMIN_INTEGRATIONS_MILESTONE_SMART_ALERTS

Integration: /integrations/smart-alerts.

**Status:** DRAFT  
**Slug:** `admin_integrations_smart_alerts`  
**Depends on:** admin_shell, admin_page_chrome  
**Blocks:** —  
**Pattern:** admin_integrations_hub  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| 501 as success | Fake toast on not-implemented | StubBanner on 501 |
| Invented form fields | TS-only PATCH keys | OpenAPI request body only |
| Secret leak | Echo api_token after save | Blank secret fields on edit; extra_config_set hints |
| N+1 client fanout | Parallel per-row GET in browser | Prefer list endpoint or document api_gaps |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Legacy table layout | <table> data grids | role=grid + CSS subgrid |
| Monolith page | 600+ line page TSX | ui/<domain>/ sections |
| Silent catch | catch -> empty table | ErrorBlock |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_integrations_hub |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- Toast before mutating 2xx
- Form fields not in OpenAPI request schema
- live:true without registered route

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/integrations/smart-alerts` — Rule CRUD; firing history; acknowledge events
- APIs: GET/POST /api/v1/smart-alerts/rules; PATCH/DELETE .../rules/{id}; GET .../history; POST .../events/{id}/ack
- Forms ⊆ OpenAPI; StubBanner on 501
- Confirm modal on destructive/retry/run actions
- RBAC: `campaigns:read; campaigns:write` on write actions

### Out of scope

- Hot-path ingest wiring
- Client-side validation beyond OpenAPI
- Invented query params not on handler

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Rules | GET/POST /api/v1/smart-alerts/rules | SmartAlertRule; customer_id required |
| Update | PATCH .../rules/{id} | UpsertSmartAlertRuleRequest |
| Delete | DELETE .../rules/{id} | 204 |
| History | GET /api/v1/smart-alerts/history | SmartAlertEvent[] |
| Ack | POST /api/v1/smart-alerts/events/{id}/ack | 204 |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Smart alerts | static |
| scope | CustomerScopeBar | customer_id | URL + session |
| rules | RulesGrid | metric, operator, threshold, enabled | GET rules |
| rule_form | RuleForm | UpsertSmartAlertRuleRequest | POST/PATCH rules |
| history | AlertHistoryGrid | fired events | GET history |
| ack | AckToolbar | ack per event | POST events/{id}/ack |
| error | ErrorBlock | Failures | errors |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/integrations/smart-alerts` |
| Permission | campaigns:read; campaigns:write |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

PageChrome
  -> CustomerScopeBar
  -> RulesGrid + RuleForm
  -> AlertHistoryGrid + AckToolbar

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/smart_alerts/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| customer_id | customer_id |  | required query on rules/history |
| limit | limit | 50 | history pagination |
| prefill | prefill |  | optional rule form prefill from URL |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/smart-alerts/rules?customer_id={uuid}
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/integrations_smart_alerts_page.tsx | Compose |
| web/src/ui/smart_alerts/* | Sections + CSS modules |
| web/src/helpers/smart_alerts_api.ts | Typed API helpers |
| web/src/types/generated/openapi.d.ts | Generated types |

**Legacy page:** `integrations_smart_alerts_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| 501 fake success | StubBanner on 501; no Saved toast |
| Silent catch → empty | ErrorBlock on blocking failure |
| live without route | Register app_routes before nav live:true |
| TS-only form fields | PATCH/POST body ⊆ OpenAPI schema |
| Toast before 2xx | apiConfirmed on mutations |
| Secret echo | Never re-display api_token or stored secrets on GET |
| Client list filter | Server query params; no useMemo over full arrays |
| Destructive no confirm | Confirm modal on delete/retry/run |
| Flex page layout | Grid sections; no flex page root |
| Portal filter listbox | Inline field dropdown per ui.mdc |
| Permission lie | Hide write actions when x-permissions missing |
| Demo table rows | EmptyState when API returns []; no fixture data |
| Metric enum drift | TS metrics not in OpenAPI | Use schema enum only |
| Prefill slop | URL prefill bypasses validation | Validate prefill like form submit |
| Ack without confirm | Mis-click ack | Optional confirm for ack |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | GET/POST /api/v1/smart-alerts/rules | Confirm OpenAPI ops + x-permissions | openapi_gate.sh |
| 2 | internal/*/handlers.go | Handler DTO matches schema | handler unit test |
| 3 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 4 | web/src/helpers/smart_alerts_api.ts | Typed API helpers; secrets never logged | Compiles |
| 5 | web/src/ui/smart_alerts/* | Sections per regions + grid_ascii | check_ui_surface_gate.sh |
| 6 | web/src/pages/integrations_smart_alerts_page.tsx | Compose + URL params | No client filter on items[] |
| 7 | web/src/app_routes.tsx | Route + integrations nav | Resolves; live flag honest |

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
| No client sort/filter | rg 'useMemo.*filter\|table_sort' web/src/pages/integrations_smart_alerts_page.tsx web/src/ui/smart_alerts/ | no matches |
| PATCH fields | Cross-check OpenAPI request bodies | No extra TS fields |
| Error | Manual: block primary GET | ErrorBlock visible |
| 501 path | Manual: force 501 if applicable | StubBanner; no success toast |
| Permissions | Manual: read-only session | Write buttons hidden/disabled |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `integrations_smart_alerts_page.tsx` replaced or delegates to `ui/smart_alerts/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/smart_alerts/`, helpers, and page compose in one slice; no half migration.


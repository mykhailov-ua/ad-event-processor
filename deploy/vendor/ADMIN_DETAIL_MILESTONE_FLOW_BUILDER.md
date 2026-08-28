# ADMIN_DETAIL_MILESTONE_FLOW_BUILDER

Detail/editor: /campaigns/flows/:id/builder.

**Status:** DRAFT  
**Slug:** `admin_detail_flow_builder`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_FLOWS`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Toast before 2xx | Save toast on click | apiConfirmed after PUT 2xx |
| TS-only path keys | Extra keys on paths[] | FlowPath schema only |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client KPI math | Sum sub-resources in browser | Stats endpoints only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_detail_pattern |
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
- Form fields not on Go PATCH struct

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/campaigns/flows/:id/builder` — Path editor: lander/offer weights; PUT UpdateFlowRequest; validate via campaign flow endpoint
- Legacy: `flow_builder_page.tsx`

### Out of scope

- Client-derived KPIs
- TS-only PATCH fields

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| PUT name | UpdateFlowRequest | name |
| PUT paths | UpdateFlowRequest | paths[] |
| PUT path.weight | FlowPath | weight |
| PUT path.landers | FlowPath.landers | lander_id, weight per FlowPathLanderRef |
| PUT path.offers | FlowPath.offers | offer_id, weight per FlowPathOfferRef |
| GET Flow | GET /api/v1/flows/{id} | id, name, paths, created_at |
| POST validate | POST .../campaigns/{id}/flow/validate | optional paths[] body |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Flows → flow name | GET flow.name |
| chrome | PageChrome | Flow builder title | GET flow.name |
| toolbar | FlowBuilderToolbar | Save paths; Add path; Validate | PUT flow; POST validate |
| tabs | TabBar | graph \| validate \| catalog | URL ?tab= |
| tab_graph | FlowPathEditor | paths[] rows: weight, landers[], offers[] | GET/PUT flow |
| path_row | FlowPathRow | lander_id+weight, offer_id+weight selects | UpdateFlowRequest.paths |
| tab_validate | FlowValidatePanel | validation errors from POST | POST .../flow/validate |
| tab_catalog | FlowCatalogSidebar | lander/offer pick lists | GET landers/offers |
| error | ErrorBlock | GET/PUT/validate failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| graph | Paths | GET/PUT /api/v1/flows/{id} |
| validate | Validate | POST /api/v1/campaigns/{campaign_id}/flow/validate |
| catalog | Landers & offers | GET /api/v1/landers; GET /api/v1/offers |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/flows/:id/builder` |
| Nav group | Commercial → Campaigns → Flows → Builder |
| Permission | campaigns:read (view); campaigns:write (save) |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/flows/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Tabs | Separate GET per tab | No client merge of tab payloads |
| GET detail | Full DTO from handler | Render as returned |
| PATCH | Fields ⊆ OpenAPI Patch*Request | apiConfirmed after 2xx |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/flow_builder_page.tsx | Thin compose; tabs route optional |
| web/src/ui/flows/flows_detail.tsx | Detail shell |
| web/src/ui/flows/*.module.css | Section CSS modules |
| web/src/helpers/flows_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../helpers/flow_builder_model.ts imports from page.

**Legacy page:** `flow_builder_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Success toast before 2xx | apiConfirmed after PUT flow |
| TS-only PUT fields | UpdateFlowRequest keys only |
| Client validate only | Use POST flow/validate for server errors |
| PATCH instead of PUT | flowsUpdate is PUT not PATCH |
| Wrapper stack | No FlowBuilderChrome |
| Flex page root | CSS Grid sections |
| Copy flow_builder_model slop | Move helpers to flows_api |
| Silent save error | ErrorBlock on PUT failure |
| Empty paths PUT | Require >=1 path before save |
| Weight cap fiction | No invented max weight |
| Catalog N+1 | Batch landers/offers fetch once |
| Piecemeal edit | Atomic ui/flows/ folder |
| Navigate without id | ErrorBlock when :id missing |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm UpdateFlowRequest + flowsUpdate PUT | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/flows_api.ts | getFlow, updateFlow, validateFlow | Compiles |
| 4 | web/src/ui/flows/flow_builder.tsx | Path editor grid | surface gate |
| 5 | web/src/pages/flow_builder_page.tsx | Compose; breadcrumbs | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /campaigns/flows/:id/builder | Resolves |
| 7 | Directory link | ADMIN_DIRECTORY_MILESTONE_FLOWS row → builder | id from items[] |
| 8 | Legacy cleanup | Remove components/ imports from flow_builder_page | typecheck |

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
| PUT body | Manual: save flow | network PUT matches UpdateFlowRequest |
| apiConfirmed | Manual: save | toast after 2xx |
| Validate tab | Manual: invalid path | server errors displayed |
| ErrorBlock | Manual: block GET flow | ErrorBlock visible |
| No PATCH | rg 'PATCH.*flows' web/src/helpers/flows_api.ts | no matches |
| Positive weight | Manual: weight 0 | blocked before PUT or server 400 |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `flow_builder_page.tsx` replaced or delegates to `ui/flows/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/flows/`, helpers, and page compose in one slice; no half migration.


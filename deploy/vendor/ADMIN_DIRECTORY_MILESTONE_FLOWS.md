# ADMIN_DIRECTORY_MILESTONE_FLOWS

Campaign flows hub at /campaigns/flows — landers, offers, flows tabs with create rows; flow row links to builder.

**Status:** DRAFT  
**Slug:** `admin_directory_flows`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** `admin_detail_flow_builder`  
**Pattern:** admin_directory_pattern (hub tabs)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Client sort on arrays | Legacy none but agents add sortRows | No sort until OpenAPI param |
| Fake pagination | PaginationBar without total | Omit footer — document api_gap |
| Partial error swallow | Promise.all one tab fails silently | ErrorBlock on any tab failure |
| Create toast before 2xx | Optimistic create | apiConfirmed after POST 201 |
| Path validation in browser | parseFlowPaths business rules | Server flows validate endpoint on builder |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Flows-only grid | Drops landers/offers tabs | Preserve 3-tab hub per legacy route |
| Copy campaign_flows_page | 700-line monolith | ui/flows/* sections |
| Portal tab listbox | Flex tab bar | TabBar grid section |
| ZIP upload without lander id | Dead upload button | Select lander row first |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Keep components/ TabBar | Skip ui/flows | Domain folder per frontend-modular.mdc |
| Single fetch flows only | Ignore landers/offers | Parallel fetch all three tabs |
| Skip URL tab param | useState tab only | ?tab= sync on refresh |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Directory paginated" when APIs return full arrays
- "Server sort" without OpenAPI sort param on list ops
- Create form fields not on Create*Request schemas
- Builder link using wrong id field

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Rebuild `/campaigns/flows` under `web/src/ui/flows/`
- Tab hub: landers | offers | flows (URL `?tab=` sync)
- List tables per tab from GET landers/offers/flows (full array — no pagination)
- Inline create forms per tab (POST) when campaigns:write
- Flow row → `/campaigns/flows/{id}/builder`
- Lander ZIP upload → POST landers/{id}/hosted-upload when on landers tab

### Out of scope

- Flow builder editor (`ADMIN_DETAIL_MILESTONE_FLOW_BUILDER.md`)
- Hosted editor file tree (`ADMIN_DETAIL_MILESTONE_LANDER.md`)
- Client-side path validation (server validate on save)
- Server pagination (APIs return full arrays today)

**Not on page (explicit):** `Client pagination`, `sortRows on arrays`, `Monolithic campaign_flows_page copy`

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| No ListEnvelope | landersList, offersList, flowsList return bare arrays | No PaginationBar; document full-fetch; window if N>500 (react.mdc G9) |
| No list query params | No limit/offset/sort/q on flow/lander/offer list ops | No filter panel until OpenAPI adds params |
| Flow paths shape | Flow.paths is array or string in schema | Use summarizeFlowPaths helper; do not invent path DSL |
| Tab hub vs pure directory | Legacy is 3-tab hub not single grid | Milestone ships tab hub; do not collapse to flows-only grid without operator ask |

### Stop triggers (revert slice; do not compensate)

- Wrapper stack (`FlowsHubChrome`) — revert
- Invented list filters without OpenAPI query params

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Landers list | GET /api/v1/landers — landersList | array of Lander |
| Offers list | GET /api/v1/offers — offersList | array of Offer |
| Flows list | GET /api/v1/flows — flowsList | array of Flow |
| Create lander | POST /api/v1/landers — landersCreate | CreateLanderRequest |
| Create offer | POST /api/v1/offers — offersCreate | CreateOfferRequest |
| Create flow | POST /api/v1/flows — flowsCreate | CreateFlowRequest |
| Hosted upload | POST /api/v1/landers/{id}/hosted-upload | ZIP upload on landers tab |
| RBAC | x-permissions | campaigns:read / campaigns:write |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Title "Campaign flows" | static |
| tabs | FlowsTabBar | landers \| offers \| flows | URL ?tab= |
| toolbar_landers | LandersToolbar | Create lander + ZIP upload | POST landers + hosted-upload |
| toolbar_offers | OffersToolbar | Create offer | POST offers |
| toolbar_flows | FlowsToolbar | Create flow | POST flows |
| content_landers | LandersGrid | name, url, id, created_at | GET landers[] |
| content_offers | OffersGrid | name, url, id | GET offers[] |
| content_flows | FlowsGrid | name, paths summary, created_at, builder link | GET flows[] |
| col_builder | grid cell | Open builder → /campaigns/flows/{id}/builder | item.id |
| create_forms | InlineCreateRow | Per-tab POST forms | OpenAPI create bodies |
| error | ErrorBlock | Any tab fetch/create failure | fetch error |
| loading | tab skeleton | placeholder rows per active tab | loading state |
| empty | EmptyState | per-tab empty copy | array length 0 |


**Not on page (explicit):** Client pagination, sortRows on arrays, Monolithic campaign_flows_page copy.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/flows` |
| Nav group | Commercial → Campaigns → Flows |
| Icon | `git-branch` |
| Permission | campaigns:read |
| `live` | true |
| Handler | internal/flow/handlers.go — landersList, offersList, flowsList |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
┌ PageChrome ─────────────────────────────────────────────┐
│ Campaign flows                                          │
├ TabBar ─────────────────────────────────────────────────┤
│ [Landers] [Offers] [Flows]                              │
├ Toolbar (per tab) ──────────────────────────────────────┤
│ [Create …] [Upload ZIP] (landers tab only)              │
├ Content (role=grid) ────────────────────────────────────┤
│ --flows-cols: minmax(12rem,2fr) 10rem 8rem 8rem         │
│ Name | URL/Paths | Created | Actions                    │
├ (no footer — full array fetch) ─────────────────────────┤
└─────────────────────────────────────────────────────────┘
```

| Invariant | Value |
| :--- | :--- |
| Page grid | `grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content |
| Row subgrid | Each row uses `--flows-cols` |
| Max width | `max-width: var(--page-max-width)` on page root |
| Scroll | Content region scrolls; chrome/toolbar/filter/footer in page grid |
| No pagination footer | APIs return full arrays — no fake PaginationBar |
| Tab in URL | ?tab=landers\|offers\|flows |
| No client sort | No sort headers until OpenAPI sort param |


**Non-sortable headers:** name, url, paths, created_at

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/flows/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |
| Grid token | `--flows-cols`: minmax(12rem,2fr) 10rem 8rem 8rem |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| tab | tab | landers | landers \| offers \| flows — UI only until API tab scope |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |


Fetch example:

```
GET /api/v1/flows (parallel: GET /api/v1/landers, GET /api/v1/offers)
```

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/campaign_flows_page.tsx | Compose: tab URL + parallel fetch |
| web/src/ui/flows/flows_hub.tsx | Tab hub section assembly |
| web/src/ui/flows/flows_hub.module.css | Hub section grid |
| web/src/ui/flows/landers_panel.tsx | Landers grid + create + upload |
| web/src/ui/flows/offers_panel.tsx | Offers grid + create |
| web/src/ui/flows/flows_panel.tsx | Flows grid + create + builder link |
| web/src/ui/flows/*.module.css | Per-panel column templates |
| web/src/helpers/flows_api.ts | fetchLanders/Offers/Flows + create helpers |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../components/tab_bar.js, ../components/breadcrumbs.js on page, Monolithic inline forms in campaign_flows_page.tsx.

**Legacy page:** `campaign_flows_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Fake pagination | No limit/offset in OpenAPI — no PaginationBar |
| Client sort on landers[] | Forbidden — no sort param |
| Tab state not in URL | Refresh loses tab — use ?tab= |
| Monolith copy | Reuse 500-line campaign_flows_page |
| Toast before create 2xx | apiConfirmed pattern |
| ZIP upload wrong lander | Require selected lander id |
| Flow paths editor on hub | Belongs in builder milestone |
| Parallel fetch partial fail | ErrorBlock not empty tab |
| Wrapper FlowsHubChrome | Flat ui/flows/ sections |
| Flex tab layout | CSS Grid sections only |
| Invented offer URL rules | Only http(s) check if mirrored from legacy — server validates |
| Builder link typo | Use item.id UUID not array index |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm landers/offers/flows ops + create bodies | openapi_gate |
| 2 | make openapi-types | Regenerate openapi.d.ts | typecheck |
| 3 | web/src/helpers/flows_api.ts | Typed list + create helpers | no hand DTO |
| 4 | web/src/ui/flows/* | Tab panels per 4.1 | check_ui_surface_gate.sh |
| 5 | web/src/pages/campaign_flows_page.tsx | URL ?tab= + compose | no components/ imports |
| 6 | web/src/app_routes.tsx | Confirm /campaigns/flows route | lazy import resolves |
| 7 | Builder link | Row → /campaigns/flows/{id}/builder | manual navigation |
| 8 | Legacy cleanup | Remove inline tables from old page | rg TabBar web/src/pages/campaign_flows_page.tsx |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |
| Full array fetch | Admin cold path | Window rows if N>500 | react.mdc G9 threshold |

## 7. Verification (paste in PR)

```bash
rg 'table_sort' web/src/pages/campaign_flows_page.tsx web/src/ui/flows/
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No client sort | rg 'table_sort\|sortRows' web/src/pages/campaign_flows_page.tsx web/src/ui/flows/ | no matches |
| Pagination | Manual: change offset in URL | refetch; total stable |
| Error | Manual: block API | ErrorBlock visible |
| Tab URL | Manual: ?tab=flows refresh | stays on flows tab |
| Create lander | Manual: POST create | 201 + list refetch |
| Builder link | Manual: open flow row | navigates to builder route |
| Error | Manual: block API | ErrorBlock visible |
| No pagination | rg PaginationBar web/src/ui/flows/ | no matches |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `campaign_flows_page.tsx` replaced or delegates to `ui/flows/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/flows/`, helpers, and page compose in one slice; no half migration.


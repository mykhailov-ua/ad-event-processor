# ADMIN_DETAIL_MILESTONE_LANDER

Detail/editor: /campaigns/landers/:id/editor.

**Status:** DRAFT  
**Slug:** `admin_detail_lander`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_FLOWS`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| TS-only form fields | Invented PATCH keys | OpenAPI Patch*Request only |
| Toast before 2xx | Optimistic save | apiConfirmed after response |
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

- `/campaigns/landers/:id/editor` — Hosted file tree; monaco/plain editor; draft save + publish
- Legacy: `lander_editor_page.tsx`

### Out of scope

- Client-derived KPIs
- TS-only PATCH fields

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET state | GET /api/v1/landers/{id}/hosted-editor | lander_id, name, draft_version, published_version, has_unpublished_draft, files[], preview_url |
| GET file | GET .../hosted-editor/files/{path} | HostedEditorFileBody.content |
| PUT content | HostedEditorFileBody | content |
| POST publish version | HostedEditorPublishRequest | version |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Landers → lander name | HostedEditorState.name |
| chrome | PageChrome | Hosted lander editor | GET state.name |
| toolbar | LanderEditorToolbar | Save draft; Publish; dirty indicator | PUT file; POST publish |
| sidebar | HostedFileTree | files[] paths; editable flag | GET hosted-editor state |
| tab_editor | HostedFileEditor | content textarea/monaco | GET/PUT file body |
| tab_files | HostedFileList | file size, editable | HostedEditorState.files |
| tab_publish | LanderPublishPanel | draft_version, published_version | state + POST publish |
| preview | PreviewLink | preview_url when API provides | HostedEditorState.preview_url |
| error | ErrorBlock | GET/PUT/publish failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| editor | Editor | GET/PUT /api/v1/landers/{id}/hosted-editor/files/{path} |
| files | Files | GET /api/v1/landers/{id}/hosted-editor |
| publish | Publish | POST /api/v1/landers/{id}/hosted-editor/publish |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/landers/:id/editor` |
| Nav group | Commercial → Campaigns → Landers → Editor |
| Permission | campaigns:read (view); campaigns:write (save/publish) |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/landers/*.module.css` only |
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
| web/src/pages/lander_editor_page.tsx | Thin compose; tabs route optional |
| web/src/ui/landers/landers_detail.tsx | Detail shell |
| web/src/ui/landers/*.module.css | Section CSS modules |
| web/src/helpers/landers_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../helpers/flows_api.ts lander editor fns → landers_api.

**Legacy page:** `lander_editor_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Success toast before 2xx | apiConfirmed on save/publish |
| TS-only PUT fields | content only on file PUT |
| Publish version drift | Pass state.draft_version to publish body |
| Dirty navigate away | Prompt when dirty=true |
| Wrapper stack | No LanderEditorChrome |
| Flex page root | CSS Grid: sidebar + editor |
| Edit non-editable file | Disable editor when editable=false |
| Silent file load error | ErrorBlock on GET file fail |
| Binary file edit | Text editor for editable paths only |
| Copy components/ | ui/landers/ only |
| Phantom PATCH lander | PUT file content only |
| Preview URL fiction | preview_url from API only |
| Large file client validate | Server enforces size |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm hosted-editor schemas | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/landers_api.ts | fetchEditorState, getFile, saveFile, publish | Compiles |
| 4 | web/src/ui/landers/lander_editor.tsx | Tree + editor split | surface gate |
| 5 | web/src/pages/lander_editor_page.tsx | Compose | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /campaigns/landers/:id/editor | Resolves |
| 7 | Unsaved guard | dirty flag before route change | manual |
| 8 | Legacy cleanup | Split lander helpers from flows_api | typecheck |

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
| apiConfirmed save | Manual: save file | toast after PUT 2xx |
| apiConfirmed publish | Manual: publish draft | toast after POST 2xx |
| Dirty guard | Manual: edit + navigate away | confirm prompt |
| Non-editable file | Manual: select binary path | editor disabled |
| ErrorBlock | Manual: block GET state | ErrorBlock visible |
| draft_version bump | Manual: save twice | version increments in UI |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `lander_editor_page.tsx` replaced or delegates to `ui/landers/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/landers/`, helpers, and page compose in one slice; no half migration.


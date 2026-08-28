# ADMIN_DETAIL_MILESTONE_RTB

Detail/editor: /rtb/integration.

**Status:** DRAFT  
**Slug:** `admin_detail_rtb`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| PATCH fiction | Agents add profile Save | rg patch: api/openapi/paths/rtb.yaml under integration-profile — empty |
| Toast before 2xx | Floors apply toast on click | apiConfirmed after POST 2xx |
| TS-only profile keys | Extend RtbIntegrationProfile in types/rtb.js | OpenAPI additionalProperties only |
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

- Rebuild `/rtb/integration` under `web/src/ui/rtb/`
- Copy endpoint URLs from profile DTO
- Validate tab uses POST validate-bid-request + fixture from openrtb_endpoint helper
- Floors apply respects dry_run query + confirm when dry_run=false

### Out of scope

- PATCH integration profile (not in OpenAPI)
- Client shadow diff math
- Hand-written RtbIntegrationProfile TS diverging from OpenAPI

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| Profile PATCH | integration-profile GET only | No edit form; display handler fields only |

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| GET profile | GET /api/v1/rtb/integration-profile | RtbIntegrationProfile (additionalProperties) — read-only |
| GET shadow | GET /api/v1/rtb/shadow-diff | shadow_evals, parity_rate, mismatch_rate, window |
| POST validate | POST /api/v1/rtb/validate-bid-request | raw OpenRTB JSON body → OpenRtbValidationResult |
| POST floors dry_run | POST /api/v1/rtb/floors/apply?dry_run=true | RtbFloorsApplyRequest.placement_ids |
| PATCH gap | No PatchRtbIntegrationProfileRequest | Profile tab read-only; no Save button |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | RTB integration | static |
| tabs | TabBar | profile \| shadow \| validate \| floors | URL ?tab= |
| tab_profile | RtbProfilePanel | openrtb_version, endpoints, runtime hints | GET integration-profile |
| tab_shadow | RtbShadowDiffPanel | parity_rate, mismatch_rate, shadow_no_bid | GET shadow-diff |
| tab_validate | RtbBidValidator | Paste JSON; POST validate; errors[] | POST validate-bid-request |
| tab_floors | RtbFloorsApply | dry_run toggle; suggestions table | POST floors/apply |
| copy_rows | IntegrationCopyRow | openrtb_bid_url copy | profile.endpoints |
| error | ErrorBlock | Tab fetch/POST failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| profile | Profile | GET /api/v1/rtb/integration-profile |
| shadow | Shadow diff | GET /api/v1/rtb/shadow-diff?window= |
| validate | Bid validator | POST /api/v1/rtb/validate-bid-request |
| floors | Floors apply | POST /api/v1/rtb/floors/apply?dry_run= |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/rtb/integration` |
| Nav group | Integrations → RTB |
| Permission | rtb:read (view); rtb:write / settings:write (floors apply) |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/rtb/*.module.css` only |
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
| web/src/pages/rtb_integration_page.tsx | Thin compose; tabs route optional |
| web/src/ui/rtb/rtb_detail.tsx | Detail shell |
| web/src/ui/rtb/*.module.css | Section CSS modules |
| web/src/helpers/rtb_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../types/rtb.js where duplicated.

**Legacy page:** `rtb_integration_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Phantom PATCH profile | GET-only integration-profile |
| Client parity math | Display shadow-diff fields as returned |
| Toast before 2xx | apiConfirmed on floors apply |
| TS-only fields | No keys beyond OpenAPI schema |
| Validate fixture drift | Use VALIDATE_BID_FIXTURE from helper; not invented JSON |
| Floors apply no confirm | Confirm when dry_run=false |
| Wrapper stack | No RtbIntegrationChrome |
| Flex page root | CSS Grid sections |
| Copy legacy types/rtb.js | openapi.d.ts after make openapi-types |
| Silent validate error | ErrorBlock when validate POST fails |
| live RTB claim | Profile runtime.rtb_enabled from API only |
| Double chip chrome | One StatusBadge on shadow summary |
| 501 as success | StubBanner if handler 501 |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/rtb.yaml | Confirm profile GET + validate + floors ops | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/rtb_api.ts | fetchProfile, shadowDiff, validate, floorsApply | Compiles |
| 4 | web/src/ui/rtb/rtb_integration.tsx | TabBar + profile/shadow/validate panels | surface gate |
| 5 | web/src/pages/rtb_integration_page.tsx | Compose; ?tab= | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /rtb/integration | Resolves |
| 7 | Nav | Integrations hub link | RBAC rtb:read |
| 8 | Legacy cleanup | Drop hand-written types/rtb.js where generated types suffice | typecheck |

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
| No PATCH UI | rg 'PATCH.*integration-profile' web/src/ui/rtb/ | no matches |
| Validate POST | Manual: run fixture validate | valid/errors from API |
| apiConfirmed floors | Manual: floors apply dry_run=0 | toast after 2xx |
| Copy URL | Manual: copy openrtb URL | clipboard from profile field |
| ErrorBlock | Manual: block profile GET | ErrorBlock visible |
| Shadow window | Manual: ?window=2h | refetch shadow-diff |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `rtb_integration_page.tsx` replaced or delegates to `ui/rtb/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/rtb/`, helpers, and page compose in one slice; no half migration.


# ADMIN_CONTRACT_MILESTONE_GATE

OpenAPI is the single wire contract for admin UI types and route catalog. Handlers and YAML stay in sync before any TS field work.

**Status:** DRAFT  
**Slug:** `admin_contract_gate`  
**Depends on:** —  
**Blocks:** admin_tokens, all page milestones  
**Pattern:** gate (no UI page)  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Handlers undocumented | Agent assumes route exists | `bash scripts/ci/openapi_gate.sh` |
| ListEnvelope missing | Copy partial list schema | C3 — grep ListEnvelope / items+total in OpenAPI |
| Handler-only JSON tags | Go struct field not in YAML | C2 spectral + catalog test |
| Report path drift | Backlog slug ≠ OpenAPI path | C4 manual cross-check |
| Missing x-permissions | UI shows action handler rejects | C5 OpenAPI review |
| Types not regenerated | Stale openapi.d.ts | `make openapi-types` + typecheck |
| live: true without handler | Nav lies | report_live_routes_gate when web/ exists |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| YAML-only PR | OpenAPI without handler | Handler + YAML same PR or gap table |
| Hand-written TS DTO | CustomerDTO in types/ | Generated openapi.d.ts only |
| Partial ListEnvelope | items[] without total | Full envelope on directory ops |
| Skip catalog test | Green unit tests only | openapi_gate.sh exit 0 |
| Invented query param | UI filter with no handler | Param in OpenAPI or api_gaps row |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Skip openapi_gate | Assumes CI will catch | Paste gate output in section 7 |
| Copy old types file | Fastest compile | Delete hand DTOs; use generated |
| Allowlist grow silently | Hide missing docs | Named allowlist entry per route |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Contract gate done" without `bash scripts/ci/openapi_gate.sh` exit 0 pasted
- "Types updated" without `make openapi-types` run
- New admin query param in milestone 4.5 without OpenAPI row
- Directory milestone before C1 green

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `api/openapi/openapi.yaml` and path fragments match live `cmd/control` handlers
- `make openapi-types` → `web/src/types/generated/openapi.d.ts`
- Catalog test: every registered `/api/v1` admin route documented or allowlisted
- `report_live_routes_gate.sh` inputs: route registry vs OpenAPI report keys
- Cross-check `deploy/vendor/admin_web_pages_backlog.md` report paths vs `GET /api/v1/reports/*`

### Out of scope

- React components, embed, Docker
- New handler behavior (separate backend milestone)
- Spectral rule changes without operator ask

**Not on page (explicit):** `UI pages`, `web/dist embed`, `Domain CSS modules`

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| C1 | `bash scripts/ci/openapi_gate.sh` | exit 0 — export, catalog test, spectral when available |
| C2 | `internal/*/handlers.go`, `*_bridge.go` | No handler-only DTO fields missing from OpenAPI on touched routes |
| C3 | ListEnvelope shape | `items`, `total`, `limit`, `offset`, optional `freshness`, `filters_applied`, `sort` on directory handlers |
| C4 | Report catalog | `GET /api/v1/reports/*` paths match `admin_web_pages_backlog.md` |
| C5 | RBAC wire | `x-permissions` on mutating operations where handler enforces RBAC |
| OpenAPI root | `api/openapi/openapi.yaml` | TS types, route catalog, spectral |
| Types output | `web/src/types/generated/openapi.d.ts` | Regenerated via `make openapi-types` |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| openapi | api/openapi/ | Wire contract source of truth | YAML fragments + bundled spec |
| handlers | internal/<domain>/handlers.go | Go DTOs returned on `/api/v1` | must match schemas |
| bridges | internal/controlplane/*_bridge.go | Route registration only | no undocumented paths |
| catalog_test | openapi_gate catalog | Every admin route in OpenAPI or allowlist | CI |
| live_routes | report_live_routes_gate.sh | SPA `live` flags vs handler registry | when web/ exists |
| types_gen | make openapi-types | openapi.d.ts for admin SPA | npm run typecheck |


**Not on page (explicit):** UI pages, web/dist embed, Domain CSS modules.

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| `live` | true |

### 4.3 Layout and placement (grid contract)

Section stack (CSS Grid on page root; no flex on page/section):

```
(No UI page — contract gate only)
OpenAPI paths ──► handlers ──► openapi_gate.sh ──► openapi.d.ts
                      │
                      └── admin_web_pages_backlog.md cross-check
```

| Invariant | Value |
| :--- | :--- |
| No TS before contract | Page milestones blocked until C1 green |
| ListEnvelope | Directory handlers document envelope fields in OpenAPI |
| Allowlist honesty | Undocumented routes must be explicit allowlist row, not silent skip |

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
| api/openapi/openapi.yaml | Root spec + component refs |
| api/openapi/paths/*.yaml | Path fragments per domain |
| api/openapi/components/schemas/*.yaml | DTO schemas |
| scripts/ci/openapi_gate.sh | C1 gate orchestrator |
| scripts/ci/report_live_routes_gate.sh | Route live vs API registry |
| web/src/types/generated/openapi.d.ts | Generated TS (after make openapi-types) |
| deploy/vendor/admin_web_pages_backlog.md | Route/API index cross-check |


**Remove from this route (legacy):** Hand-written `web/src/types/campaign.js` style DTOs, Undocumented handler fields consumed by legacy pages.

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| YAML-only change | Handler returns extra fields — add to schema or stop exposing |
| Breaking rename without UI milestone | Coordinate OpenAPI + generated types PR |
| Array list + invented total | Audit-style headers must be documented, not fake total field |
| Report key typo | Catalog card 404 — match handler path exactly |
| x-permissions omitted on POST | RBAC surprise in UI — document on mutating ops |
| Bundled spec drift | Fragment updated but bundle stale — run openapi_gate export |
| Allowlist as junk drawer | Every allowlisted route needs expiry note in PR |
| C3 waived for one page | Breaks directory pattern — fix schema first |
| Spectral skipped locally | npx unavailable — gate documents fallback |
| Handler enum ≠ OpenAPI enum | Sort/filter silently ignored — align enums |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/ | Diff touched routes vs handler registration | Every path has operationId |
| 2 | internal/<domain>/handlers.go | DTO fields ⊆ OpenAPI schemas (C2) | No orphan json tags |
| 3 | ListEnvelope schemas | Directory ops document items/total/limit/offset | C3 grep green |
| 4 | x-permissions | Mutating ops annotated (C5) | OpenAPI review checklist |
| 5 | bash scripts/ci/openapi_gate.sh | C1 gate | exit 0 |
| 6 | make openapi-types | Regenerate openapi.d.ts | cd web && npm run typecheck |
| 7 | admin_web_pages_backlog.md | C4 report path cross-check | No orphan report slugs |
| 8 | PR section 7 | Paste commands + exit codes | quality.mdc honesty |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| N/A | Contract gate | — | No hot-path SLA; not /track |

## 7. Verification (paste in PR)

```bash
bash scripts/ci/openapi_gate.sh
make openapi-types
cd web && npm run typecheck
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| C1 openapi_gate | bash scripts/ci/openapi_gate.sh | exit 0 |
| Types | make openapi-types && cd web && npm run typecheck | exit 0 |
| ListEnvelope | rg 'CampaignListResponse\|InvoiceListResponse' api/openapi/ | directory schemas found |
| Report paths | Manual: spot-check one report slug vs OpenAPI | paths match backlog |
| No hand DTO | rg 'interface CustomerDTO' web/src/types/ | no matches outside generated/ |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied

- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/<domain>/`, helpers, and page compose in one slice; no half migration.


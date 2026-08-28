# ADMIN_FRAUD_MILESTONE_ADMIN

Fraud admin suite at /fraud/* — decisions, labels, overrides, presets, integrations (gap routes).

**Route gap:** register `/fraud/decisions` in `app_routes.tsx` before `live: true`.

**Status:** DRAFT  
**Slug:** `admin_fraud_admin`  
**Depends on:** admin_detail_campaign  
**Blocks:** —  
**Pattern:** directory + detail  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per directory + detail |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- ghost_* column headers; use silent_reject_* / fraud_reason canonical names
- UI implying per-IP silent reject when ML only enqueues blacklist
- Auto-enable silent_reject_enabled from fraud admin UI
- live: true on /fraud/* before routes registered

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- gap_route suite: register `/fraud/decisions`, `/fraud/labels`, `/fraud/overrides`, `/fraud/presets`, `/fraud/integrations`
- Directory: fraud decisions log grid (tier, reason, silent_reject_event semantics)
- Labels: manual ML labels + bulk import
- Overrides: per-campaign fraud override create
- Presets: global sensitivity presets list
- Integrations: third-party fraud integration status
- Campaign fraud tab remains in ADMIN_DETAIL_MILESTONE_CAMPAIGN
- ANTIFRAUD.md: silent_reject_* canonical; ML action does not auto-enable campaign flag

### Out of scope

- Per-IP ghosting UI (ML marketing lie)
- PATCH silent_reject_enabled from ML decisions UI
- ghost_* column headers in grids

### API gaps (block or stub)

| Gap | Current state | UI behavior until fixed |
| :--- | :--- | :--- |
| SPA routes | OpenAPI handlers exist under internal/fraudadmin/ | Register /fraud/* in app_routes first |

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Decisions | GET /api/v1/fraud/decisions | FraudDecision — explain tier for IP hash |
| Labels | GET/POST /api/v1/fraud/labels | FraudManualLabelRequest |
| Bulk labels | POST /api/v1/fraud/labels/bulk | FraudManualLabelBulkRequest |
| Overrides | POST /api/v1/fraud/overrides | per-campaign override |
| Presets | GET /api/v1/fraud/presets | FraudPolicyPreset[] |
| Integrations | GET /api/v1/fraud/integrations | integration status rows |
| Campaign fraud | GET/PATCH .../campaigns/{id}/fraud | ADMIN_DETAIL_MILESTONE_CAMPAIGN tab |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Fraud admin | static |
| nav | FraudSubNav | decisions \| labels \| overrides \| presets \| integrations | routes under /fraud/ |
| decisions | FraudDecisionsGrid | IP hash tier explain rows | fraud/decisions |
| labels | FraudLabelsGrid | Manual label CRUD + bulk | fraud/labels |
| overrides | FraudOverridesForm | Per-campaign override create | POST fraud/overrides |
| presets | FraudPresetsList | Policy preset cards | fraud/presets |
| integrations | FraudIntegrationsGrid | Vendor integration status | fraud/integrations |
| error | ErrorBlock | Fetch/mutation failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| decisions | Decisions | GET /api/v1/fraud/decisions |
| labels | Labels | GET/POST /api/v1/fraud/labels; POST .../bulk |
| overrides | Overrides | POST /api/v1/fraud/overrides |
| presets | Presets | GET /api/v1/fraud/presets |
| integrations | Integrations | GET /api/v1/fraud/integrations |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/fraud/decisions` |
| Permission | fraud-admin / campaigns:read + shards:read per OpenAPI x-permissions |
| `live` | false |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/fraud/*.module.css` only |
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
| web/src/pages/gap (no legacy page; campaign fraud tab in campaign_detail_page.tsx) | Compose |
| web/src/ui/fraud/* | Sections + CSS modules |
| web/src/helpers/fraud_api.ts | API helpers |

**Legacy page:** `gap (no legacy page; campaign fraud tab in campaign_detail_page.tsx)`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| ghost_* headers | silent_reject_* and fraud_reason from API |
| ML ghosting lie | ANTIFRAUD.md: ML silent_reject adds IP to blacklist; no campaign flag flip |
| silent_reject_enabled toggle lie | Campaign fraud tab documents flag; admin suite does not fake per-IP |
| Decisions client aggregation | Display handler rows only |
| Bulk label no confirm | Confirm before bulk POST |
| Override without campaign id | OpenAPI required fields only |
| Integration status invented | fraud/integrations DTO only |
| Flex fraud layout | Grid sections per ui.mdc |
| Report link ghost funnel | Link silent-reject-impression-funnel report key |
| Preset name drift | Use preset name from API; ops fraud presets separate path |
| Duplicate campaign fraud | Document boundary vs campaign detail fraud tab |
| CH column ghost_event | Analytics silent_reject_event per ANTIFRAUD.md |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/ops_reports.yaml fraud/* | Confirm fraud opIds | openapi_gate.sh |
| 2 | deploy/vendor/ANTIFRAUD.md | Read silent_reject semantics | Doc matches UI copy |
| 3 | make openapi-types | Regenerate types | typecheck |
| 4 | web/src/app_routes.tsx | Register /fraud/decisions, /fraud/labels, /fraud/overrides, /fraud/presets, /fraud/integrations | All resolve |
| 5 | web/src/ui/shell/nav | Fraud admin nav; live=false until REVIEW | No dead links in prod |
| 6 | web/src/helpers/fraud_api.ts | Typed fraud admin helpers | Compiles |
| 7 | web/src/ui/fraud/* | Grids per tab | check_ui_surface_gate.sh |
| 8 | Campaign fraud tab | Cross-link only; no duplicate PATCH logic | Detail milestone owns campaign fraud |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
deploy/vendor/ANTIFRAUD.md silent_reject_enabled section
rg 'ghost_' web/src/ui/fraud/ — no matches
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| No ghost labels | rg 'ghost_' web/src/ui/fraud/ | no user-visible ghost_* |
| Decisions load | Manual: open /fraud/decisions | grid or ErrorBlock |
| Label create | Manual: POST label | apiConfirmed; list refetch |
| ANTIFRAUD copy | Manual: read override help text | no per-IP ghosting claim |
| Nav honest | Catalog live flag | false until routes ship |
| Campaign tab boundary | Manual: campaign fraud tab | still in campaign detail milestone |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `gap (no legacy page; campaign fraud tab in campaign_detail_page.tsx)` replaced or delegates to `ui/fraud/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/fraud/`, helpers, and page compose in one slice; no half migration.


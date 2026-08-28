# ADMIN_CAMPAIGNS_MIGRATE_MILESTONE

Campaign import wizard at /campaigns/migrate — Keitaro/Binom JSON preview and import.

**Status:** DRAFT  
**Slug:** `admin_campaigns_migrate`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS`  
**Blocks:** —  
**Pattern:** wizard  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Client import rules | Validate rows in TS | Display API failed[] only |
| Toast before 2xx | Migration complete on click | apiConfirmed after import 2xx |
| TS-only import fields | Extra keys on import body | MigrateImportRequest only |
| Idempotency skip | Repeat import duplicates | Idempotency-Key header |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per wizard |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- "Import complete" before POST 2xx
- Invented source_kind beyond OpenAPI enum

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- Source picker from GET .../migrate/sources
- Preview table from POST .../migrate/preview
- Import POST .../migrate/import with Idempotency-Key + apiConfirmed
- customer_id in URL; name_prefix + budget_limit_micro per MigrateImportRequest
- Replace legacy campaigns_migrate_page.tsx

### Out of scope

- Client validation of import rows beyond JSON parse
- TS-only import body fields

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| POST source_kind preview | MigratePreviewRequest | source_kind, payload |
| POST customer_id import | MigrateImportRequest | customer_id |
| POST source_kind import | MigrateImportRequest | source_kind |
| POST payload import | MigrateImportRequest | payload |
| POST name_prefix | MigrateImportRequest | name_prefix |
| POST budget_limit_micro | MigrateImportRequest | budget_limit_micro |
| POST pull source_kind | MigratePullRequest | source_kind |
| POST pull base_url | MigratePullRequest | base_url |
| POST pull api_token | MigratePullRequest | api_token |
| POST pull customer_id | MigratePullRequest | customer_id |
| GET sources | GET .../migrate/sources | source kinds + max_payload_bytes |
| Header Idempotency-Key | POST .../migrate/import | required per OpenAPI |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| chrome | PageChrome | Migrate campaigns | static |
| steps | WizardSteps | source → payload → preview → confirm → result | URL ?step= |
| tab_source | MigrateSourcePick | source_kind select | GET migrate/sources |
| tab_payload | MigratePayloadEditor | JSON textarea + file pick | sources.max_payload_bytes |
| tab_preview | PreviewGrid | preview rows + warnings[] | POST migrate/preview |
| tab_import | MigrateImportConfirm | customer_id, name_prefix, budget_limit_micro | MigrateImportRequest |
| tab_result | MigrateResultPanel | imported[], failed[] | POST migrate/import response |
| pull_panel | MigratePullPanel | base_url, api_token for pull/* | MigratePullRequest |
| error | ErrorBlock | Preview/import failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| source | Source | GET /api/v1/campaigns/migrate/sources |
| payload | Payload | JSON paste or file upload |
| preview | Preview | POST /api/v1/campaigns/migrate/preview |
| import | Import | POST /api/v1/campaigns/migrate/import |
| pull | Pull import | POST .../migrate/pull/preview; POST .../migrate/pull/import |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/migrate` |
| Nav group | Commercial → Campaigns → Migrate |
| Permission | campaigns:write |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/campaigns/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

**URL param mapping:**

| URL param | API query | Default | Notes |
| :--- | :--- | :--- | :--- |
| step | — | source | source \| payload \| preview \| import |
| customer_id | customer_id |  | required for import |

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Pagination | `limit`, `offset`, `total` | URL → refetch on page change |
| Sort | OpenAPI `sort`/`order` when present | URL → refetch on Apply / header click |
| Tabs | Separate GET per tab | No client merge of tab payloads |
| GET detail | Full DTO from handler | Render as returned |
| PATCH | Fields ⊆ OpenAPI Patch*Request | apiConfirmed after 2xx |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/campaigns_migrate_page.tsx | Thin compose |
| web/src/ui/campaigns/migrate_wizard.tsx | Wizard sections |
| web/src/helpers/migration_api.ts | sources, preview, import helpers |


**Remove from this route (legacy):** ../components/page_header.js.

**Legacy page:** `campaigns_migrate_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| Import without confirm | Confirm modal before import POST |
| Success toast before 2xx | apiConfirmed on import |
| TS-only import fields | MigrateImportRequest keys only |
| Skip Idempotency-Key | Header required on import |
| Client preview fiction | Must POST preview |
| budget_limit_micro fiction | Send micros from ParseDecimal |
| customer_id missing | Block import until UUID valid |
| max_payload_bytes | Error when paste exceeds limit |
| Wrapper stack | No MigrateChrome |
| Flex page root | CSS Grid wizard |
| Pull token echo | Never echo api_token after submit |
| Silent catch empty preview | ErrorBlock on preview fail |
| Navigate without result | Show imported[]/failed[] from API |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/campaigns.yaml | Confirm Migrate*Request + idempotency | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/migration_api.ts | sources, preview, import with Idempotency-Key | Compiles |
| 4 | web/src/ui/campaigns/migrate_wizard.tsx | Step sections | surface gate |
| 5 | web/src/pages/campaigns_migrate_page.tsx | Compose; customer_id URL | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /campaigns/migrate | Resolves |
| 7 | Campaigns directory | Migrate toolbar link | nav |
| 8 | Legacy cleanup | Remove PageHeader/components imports | rg PageHeader campaigns_migrate_page |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
rg 'Idempotency-Key' web/src/helpers/migration_api.ts
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Idempotency | Manual: repeat import same key | no duplicate campaigns |
| apiConfirmed import | Manual: import | toast after 2xx only |
| Preview POST | Manual: preview | network POST preview |
| max bytes | Manual: oversized JSON | UI blocks before POST |
| customer_id | Manual: import without customer | blocked |
| failed rows | Manual: bad payload | failed[] from API |
| Pull optional | Manual: pull tab if shipped | token not shown after submit |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `campaigns_migrate_page.tsx` replaced or delegates to `ui/campaigns/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/campaigns/`, helpers, and page compose in one slice; no half migration.


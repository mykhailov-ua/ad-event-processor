# OpenAPI transition backlog

Incremental move from code-first admin REST (`/api/v1/*` on `:8188`) to a published OpenAPI contract. The React admin UI remains a client; it stops being an implicit source of truth for request/response shapes.

**In scope:** control plane JSON REST under `/api/v1/*` (operators, self-serve API keys, automation).

**Out of scope for this file:**

| Surface | Canonical contract today |
| :--- | :--- |
| Hot ingest (`/click`, `/track`, `/openrtb/bid`, `/tg/*`) | Wire limits in `parser.mdc`; protobuf in `api/*.proto` |
| Traffic integration templates | YAML in `deploy/schemas/traffic_*.v1.yaml` |
| Affiliate inbound postbacks | `deploy/schemas/affiliate_*` |
| Payment webhooks (`:8187`) | Handler docs + `cold-path.mdc` body limits |

Cross-reference slugs in PR descriptions. **Do not mark a slug closed** until every checked gate below applies (skip N/A lines only when the touch surface truly did not change).

---

## Current baseline (2026-03)

| Artifact | Role today |
| :--- | :--- |
| `internal/controlplane/register.go` `routeCatalog` | ~256 `{Method, Path}` rows; machine-readable route list |
| `internal/controlplane/*_dto.go`, handler files | JSON field names (`json:"snake_case"`) |
| `web/src/helpers/*_api.ts` (~30 modules) | Hand-written TS types + fetch wrappers |
| `scripts/ci/report_live_routes_gate.sh` | UI `live: true` routes must resolve in backend catalog |
| `docs/INTEGRATIONS.md` | Operator wiring prose; not a machine contract |

No OpenAPI file or codegen exists yet. `OpenAPI()` symbols in `internal/identity`, `internal/payment`, etc. are module bootstrap helpers, not REST specs.

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `foundation` | Repo layout, tooling, CI; blocks all domain work |
| `pilot` | First end-to-end domain proving the workflow |
| `domain_expand` | Roll contract + TS types across `/api/v1` slices |
| `workflow` | Process change for new endpoints (API-first) |
| `optional` | Nice-to-have; no merge blocker |

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`)
- [ ] Verification commands pasted in PR with package path (no unrun claims - `quality.mdc`)
- [ ] Doc claims match code; no "OpenAPI wired" without CI gate or generated artifact (`anti-slop.mdc` lie modes)
- [ ] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [ ] No new thin `*_gate.sh` that only re-invokes existing gates (`anti-slop.mdc`)
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `README.md`, `docs/`, `deploy/vendor/` (`naming.mdc`)

Rule: OpenAPI-specific

- [ ] Spec paths use `{param}` style matching `routeCatalog` (Go 1.22 mux patterns)
- [ ] Schema property names are `snake_case` matching existing `FooDTO` json tags
- [ ] No new `dto/` subtree or parallel Entity/Model/View for the same table (`code-style.mdc`, `cold-path.mdc`)
- [ ] Stub or 501 routes either omitted from public spec or tagged `x-stub: true` with documented behavior

---

## Phases (recommended order)

```
Phase 0  openapi_layout_and_scope
Phase 1  openapi_route_catalog_export + openapi_spectral_lint
Phase 2  openapi_cost_sync_pilot + openapi_typescript_codegen
Phase 3  openapi_ci_route_parity_gate (+ optional schema golden)
Phase 4  openapi_docs_ui
Phase 5  domain_expand (integrations, campaigns, billing, ops)
Phase 6  openapi_new_endpoint_policy + openapi_breaking_change_guard
Phase 7  openapi_optional_request_validation (optional)
```

---

## Summary

| Slug | Priority | Surface | Depends on |
| :--- | :--- | :--- | :--- |
| `openapi_layout_and_scope` | foundation | `api/openapi/` tree | - |
| `openapi_route_catalog_export` | foundation | generator from `routeCatalog` | layout |
| `openapi_spectral_lint` | foundation | CI lint rules | layout |
| `openapi_cost_sync_pilot` | pilot | 6 cost-sync routes + DTOs | export, lint |
| `openapi_typescript_codegen` | pilot | `web/src/types/generated/` | cost_sync pilot |
| `openapi_ci_route_parity_gate` | foundation | `scripts/ci/openapi_route_parity_gate.sh` | export |
| `openapi_ci_schema_golden_gate` | optional | pilot domain JSON examples | cost_sync pilot |
| `openapi_docs_ui` | optional | `/api/v1/openapi.json` + Redoc | export |
| `openapi_integrations_domain` | domain_expand | postbacks, supply, schemas, templates | pilot workflow |
| `openapi_campaigns_domain` | domain_expand | campaigns, flows, landers | pilot workflow |
| `openapi_billing_domain` | domain_expand | billing, self-serve, invoices | pilot workflow |
| `openapi_ops_reports_domain` | domain_expand | reports, ops, fraud admin | pilot workflow |
| `openapi_new_endpoint_policy` | workflow | `docs/DEVELOPMENT.md` section | pilot merged |
| `openapi_breaking_change_guard` | workflow | PR diff on spec | CI parity |
| `openapi_optional_request_validation` | optional | kin-openapi middleware | full domain coverage |

---

## `openapi_layout_and_scope`

**Priority:** foundation

**Gap:** No canonical OpenAPI entrypoint; `/api/v1` contract is implicit in Go + TS duplicates.

**Target layout:**

```
api/openapi/
  openapi.yaml              # info, servers, security, $ref entry
  paths/
    cost_sync.yaml          # pilot
    _generated_routes.yaml  # paths-only from routeCatalog (Phase 1)
  components/
    schemas/
      cost_sync.yaml
      errors.yaml           # shared error envelope
    parameters/
      customer_id.yaml
    securitySchemes/
      session_cookie.yaml
      bearer_api_key.yaml   # self-serve keys
```

**Decisions to record in `openapi.yaml` `info.description`:**

- Base URL: `https://{control_host}/api/v1` (dev default `:8188`)
- Auth: session cookie for admin UI; `Authorization: Bearer` for `/api/v1/selfserve/*`
- Money fields: string decimal USD in JSON DTOs where Go uses `string` today; document micro-unit fields explicitly
- Pagination/error shape: align with `pkg/httpresponse` and existing handler 4xx bodies

### Done gates

- [ ] `api/openapi/openapi.yaml` validates with Spectral (or `@redocly/cli lint`) locally
- [ ] No duplicate path keys across `$ref` files
- [ ] `docs/INTEGRATIONS.md` links to this backlog slug once (one sentence); no claim that full `/api/v1` is spec-complete

---

## `openapi_route_catalog_export`

**Priority:** foundation

**Gap:** `routeCatalog` (~256 routes) is Go-only; drift vs mux registration is caught indirectly, not vs a published contract.

**Target:**

- `go run ./cmd/openapi-export` (or test helper `TestOpenAPIExport_routeCatalog`) emits `api/openapi/paths/_generated_routes.yaml`
- Each path: `method`, `operationId` (`costSyncListCredentials`), `tags` derived from path prefix (`cost-sync`, `campaigns`, `billing`, ...)
- `summary` optional in v1; `description` may reference handler file in comment block outside spec (not in shipped YAML)
- Regenerate in `make gen` or dedicated `make openapi-export` documented in `docs/DEVELOPMENT.md`

**Explicit exclusions in generator:**

- `/admin/*` legacy HTMX (deprecated)
- Static SPA assets
- Non-JSON binary routes (PDF export) tagged `application/pdf` when added

### Done gates

- [ ] Generated file count matches `len(routeCatalog)` minus documented exclusions
- [ ] `go test ./internal/controlplane/ -run TestOpenAPI -count=1` (or equivalent) fails if catalog and mux diverge
- [ ] Imperative commit title names `routeCatalog` export surface (`core.mdc`)

---

## `openapi_spectral_lint`

**Priority:** foundation

**Gap:** No style or completeness rules on YAML contract.

**Target:**

- `deploy/openapi/spectral.yaml` (or `.spectral.yaml` at repo root) with project rules:
  - `operationId` must be unique and camelCase
  - Every `paths.*` under pilot domains must declare `responses` (at least `default` or `4xx` + success)
  - No empty `schema: {}` on request bodies in pilot domains
  - `info.version` semver aligned with release tag policy (document bump rule)
- Wire into existing orchestrator: extend `lint_configs_gate.sh` **once** (no nested gate-only wrapper)

### Done gates

- [ ] `bash scripts/ci/lint_configs_gate.sh` runs Spectral on `api/openapi/openapi.yaml`
- [ ] CI fails on duplicate `operationId`

---

## `openapi_cost_sync_pilot`

**Priority:** pilot

**Gap:** `CostSyncCredentialDTO`, `UpsertCostSyncCredentialRequest`, etc. exist in Go and are duplicated in `web/src/helpers/cost_sync_api.ts`.

**Routes (canonical):**

| Method | Path |
| :--- | :--- |
| GET | `/api/v1/cost-sync/networks` |
| GET | `/api/v1/cost-sync/credentials` |
| PUT | `/api/v1/cost-sync/credentials/{network}` |
| DELETE | `/api/v1/cost-sync/credentials/{network}` |
| POST | `/api/v1/cost-sync/run` |
| GET | `/api/v1/cost-sync/history` |

**Target:**

- Full request/response schemas in `components/schemas/` matching json tags in `internal/controlplane/cost_sync_handlers.go`
- Document security behavior: secrets not returned on GET; `extra_config_set` booleans (see `docs/INTEGRATIONS.md`)
- Document permissions: `campaigns:read` / `campaigns:write`
- Examples from existing handler tests if present; else add one golden JSON fixture under `internal/controlplane/testdata/openapi/cost_sync/`

**Non-goals for pilot:**

- Do not replace Go DTO structs with oapi-codegen server stubs
- Do not change handler behavior solely to please spec

### Done gates

- [ ] Spectral clean on merged pilot paths
- [ ] `internal/controlplane/cost_sync_handlers_test.go` (or new test) asserts sample response JSON keys match spec required fields
- [ ] `bash scripts/ci/cold_path_json_gate.sh` if handlers touched

---

## `openapi_typescript_codegen`

**Priority:** pilot

**Gap:** Manual types in `cost_sync_api.ts` drift from Go DTOs.

**Target:**

- Add devDependency `openapi-typescript` (pin version in `web/package.json`)
- Script: `npm run openapi:types` reads repo-root `api/openapi/openapi.yaml` -> `web/src/types/generated/openapi.d.ts`
- Refactor `web/src/helpers/cost_sync_api.ts` to import `components['schemas'][...]` for pilot types; keep fetch/`apiConfirmed` wrappers local
- `npm run typecheck` in `admin_web.sh` path must pass
- Generated file gitignored **or** committed with `make gen` regen policy (pick one in `openapi_layout_and_scope` PR and document)

**Recommendation:** commit generated TS under `web/src/types/generated/` and regen in `make gen` for sources-only clone honesty (`core.mdc`).

### Done gates

- [ ] `cd web && npm run typecheck`
- [ ] `bash scripts/ci/admin_web.sh`
- [ ] No user-visible copy change; `check_ui_slop.sh` clean
- [ ] JSDoc on exported helper functions preserved (`ui.mdc`)

---

## `openapi_ci_route_parity_gate`

**Priority:** foundation

**Gap:** OpenAPI can miss routes that exist in production mux.

**Target:**

- Leaf gate `scripts/ci/openapi_route_parity_gate.sh`:
  - Parse `routeCatalog` (Go test JSON export or static YAML) and OpenAPI paths
  - Fail if catalog route missing from spec (allowlist file for intentional omissions: stubs, internal ops)
  - Allowlist path: `api/openapi/parity_allowlist.txt` with one `METHOD PATH` per line + comment
- Invoked from `lint_configs_gate.sh` or `pr_fast.sh` once spec lands (not both nested)

### Done gates

- [ ] Gate fails when a new `routeCatalog` row is added without spec update or allowlist entry
- [ ] Gate passes on pilot branch with cost-sync + generated paths

---

## `openapi_ci_schema_golden_gate`

**Priority:** optional

**Gap:** Route parity alone does not catch field rename drift (`silent_reject_*` style slop).

**Target:**

- For pilot domain only: golden JSON files checked into `internal/controlplane/testdata/openapi/`
- Test loads spec schema (or embedded subset) and validates golden fixtures
- Expand to next domain when that domain slug closes

### Done gates

- [ ] Test fails if Go handler changes response shape without fixture + spec update
- [ ] Documented in slug `openapi_cost_sync_pilot` PR

---

## `openapi_docs_ui`

**Priority:** optional

**Gap:** Operators and integrators have no browsable contract besides reading Go.

**Target:**

- Serve bundled spec at `GET /api/v1/openapi.json` (control plane, auth required or public read with license gate - decide in PR)
- Static Redoc or Swagger UI under `/api/v1/docs` (embedded in control static handler, not hot path)
- Link from admin Settings or Integrations footer (one link; no marketing prose)

### Done gates

- [ ] `go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1` if static routes change
- [ ] Spec URL returns same bytes as repo `api/openapi/openapi.yaml` merged output

---

## `openapi_integrations_domain`

**Priority:** domain_expand

**Routes:** `/api/v1/postbacks/*`, `/api/v1/supply/*`, `/api/v1/integration/*`, `/api/v1/cost-sync/*` (complete if pilot partial), `/api/v1/smart-alerts/*`, `/api/v1/margin-guard/*`, `/api/v1/platform-campaigns/*`

**Helpers to migrate:** `postback_api.ts`, `supply_api.ts`, `integration_api.ts`, `margin_guard_api.ts`, `smart_alerts_api.ts`

### Done gates

- [x] TS types sourced from generated OpenAPI for this slice
- [x] `docs/INTEGRATIONS.md` operator tables still accurate (field names match spec)

---

## `openapi_campaigns_domain`

**Priority:** domain_expand

**Routes:** `/api/v1/campaigns/*`, `/api/v1/flows/*`, `/api/v1/brands/*`, lander hosted routes, wizard payloads

**Helpers:** `campaign_admin_api.ts`, `flows_api.ts`, `brand_creatives_api.ts`, `domains_api.ts`

**Note:** `CampaignDTO` is large; split schema file `components/schemas/campaign.yaml` with `$ref` slices (budget, fraud flags, ingress_cost_config) to keep reviews small.

### Done gates

- [x] `ingress_cost_config` documented alongside `docs/INTEGRATIONS.md` ingress macro section
- [x] PATCH partial update semantics documented (`application/json` merge rules)

---

## `openapi_billing_domain`

**Priority:** domain_expand

**Routes:** `/api/v1/billing/*`, `/api/v1/selfserve/*`, `/api/v1/customers/*` billing subpaths

**Helpers:** `billing_admin_api.ts`, `selfserve_api.ts`, `selfserve_billing_api.ts`

**Sensitive fields:** invoice PDF routes as `application/pdf`; webhook bodies out of public operator spec or separate webhook OpenAPI fragment.

### Done gates

- [x] Self-serve Bearer auth documented under `components/securitySchemes`
- [x] No secret echo in response examples (`anti-slop.mdc` UI slop)

---

## `openapi_ops_reports_domain`

**Priority:** domain_expand

**Routes:** `/api/v1/reports/*`, `/api/v1/ops/*`, `/api/v1/fraud/*`, `/api/v1/dashboards/*`, `/api/v1/views/*`

**Helpers:** `report_api.ts`, `fraud_*`, `ops_*`, `tg_report_api.ts`

**Note:** Report query params are numerous; prefer shared parameter `$ref`s and document `stale` response metadata from `control-plane.mdc`.

### Done gates

- [x] `report_live_routes_gate.sh` still passes; OpenAPI documents report keys referenced in UI where `live: true`

---

## `openapi_new_endpoint_policy`

**Priority:** workflow

**Gap:** New endpoints still land as Go + manual TS (`ui.mdc`: read Go DTO first).

**Target process (document in `docs/DEVELOPMENT.md`):**

1. PR 1: OpenAPI path + schemas + Spectral clean
2. PR 2 (may squash with 1 if small): Go handler implements contract; TS uses generated types
3. `routeCatalog` row + spec path in same commit
4. Permission string documented on operation (`x-permissions: campaigns:write` extension acceptable)

**Until all domains migrated:** legacy endpoints remain code-first; spec catches up via export + domain slugs, not big-bang rewrite.

### Done gates

- [x] `docs/DEVELOPMENT.md` section added; no docs-only commit without code example pointing to cost-sync pilot (`core.mdc`)
- [ ] Team default: new `/api/v1` routes forbidden without OpenAPI row in `pr_fast` scope (optional follow-up gate; policy documented in `docs/DEVELOPMENT.md`)

---

## `openapi_breaking_change_guard`

**Priority:** workflow

**Gap:** JSON field renames break UI and self-serve clients silently.

**Target:**

- CI job or PR script: `openapi-diff` (oasdiff or similar) against base branch
- Fail on: removed properties, type changes, required field added without default, path/method removal
- Allow `x-unstable: true` on beta operations excluded from breaking check (document policy)

### Done gates

- [x] Documented in `docs/DEVELOPMENT.md` release notes expectation
- [x] One fixture PR proves gate catches removed field (`internal/openapi/testdata/breaking/`, `TestBreakingChangeGate_fixtureDetectsRemovedProperty`)

---

## `openapi_optional_request_validation`

**Priority:** optional

**Gap:** Spec is documentary only; handlers can accept extra/missing fields unnoticed.

**Target:**

- Opt-in `kin-openapi` request validator middleware for selected routes (start with self-serve write paths)
- Fail closed 400 with stable error envelope; never on hot path binaries

**Rejected for default deploy:** full oapi-codegen server replacement of `controlplane` handlers (cost vs 256 routes).

### Done gates

- [x] Middleware off by default; env flag documented
- [x] Fault test: invalid body rejected before service layer (`TestOpenAPIRequestValidation_rejectsInvalidBodyBeforeHandler`)

---

## Domain rollout tracker

Update checkboxes when a domain slug closes.

| Domain | Spec schemas | TS codegen | Route parity |
| :--- | :---: | :---: | :---: |
| cost-sync (pilot) | [x] | [x] | [x] |
| integrations / postbacks / supply | [x] | [x] | [x] |
| campaigns / flows / landers | [x] | [x] | [x] |
| billing / self-serve | [x] | [x] | [x] |
| ops / reports / fraud admin | [ ] | [ ] | [ ] |
| team / publisher / telegram | [ ] | [ ] | [ ] |
| platform / licensing / meta | [ ] | [ ] | [ ] |

---

## Rejected alternatives

| Alternative | Why rejected |
| :--- | :--- |
| Full `oapi-codegen` server regen for all handlers | ~256 routes; massive diff; fights flat `controlplane` layout |
| swaggo annotations on every handler | Comment noise; duplicate of struct tags |
| OpenAPI as only source; delete Go DTOs | Breaks sqlc mapping pattern and cold-path error helpers |
| Proto for admin REST | Protobuf already reserved for ingest/outbox; REST clients expect JSON |
| UI-first spec from React types | TS types today are incomplete vs Go; wrong direction |

---

## First PR suggestion (minimal slice)

Single merge-ready vertical slice (~1-2 days):

1. `api/openapi/openapi.yaml` + `paths/cost_sync.yaml` + schemas
2. `make openapi-export` paths-only for full catalog
3. Spectral in `lint_configs_gate.sh`
4. `openapi-typescript` + migrate `cost_sync_api.ts`
5. One parity test: catalog contains all cost-sync paths present in spec

Cross-reference in PR: `deploy/vendor/openapi_backlog.md` slugs `openapi_layout_and_scope`, `openapi_cost_sync_pilot`, `openapi_typescript_codegen`.

# OpenAPI transition backlog

**Status:** Closed (2026-08-26). All merge-blocking slugs shipped; CI gate `bash scripts/ci/openapi_gate.sh` (export, catalog parity, Spectral, TS drift, breaking diff). Optional browsable docs UI deferred (see [Explicit non-goals](#explicit-non-goals-post-close)).

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

## Current baseline (2026-08)

| Artifact | Role today |
| :--- | :--- |
| `api/openapi/openapi.yaml` | Canonical entrypoint; hand-documented domains + `$ref` slices |
| `api/openapi/openapi.bundle.yaml` | Merged spec for breaking diff and optional kin-openapi validation |
| `api/openapi/paths/_generated_routes.yaml` | Paths-only stubs from `routeCatalog` (`make openapi-export`) |
| `internal/openapi/documented_routes.go` | Hand-spec route keys with full schemas |
| `web/src/types/generated/openapi.d.ts` | Committed TS types (`make openapi-types`) |
| `scripts/ci/openapi_gate.sh` | Export, parity, Spectral, TS drift, `openapi_breaking_gate.sh` |
| `internal/controlplane/openapi_*_test.go` | DTO json tag parity vs YAML schemas per domain |

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

- [x] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [x] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`)
- [x] Verification commands pasted in PR with package path (no unrun claims - `quality.mdc`)
- [x] Doc claims match code; no "OpenAPI wired" without CI gate or generated artifact (`anti-slop.mdc` lie modes)
- [x] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [x] No new thin `*_gate.sh` that only re-invokes existing gates (`anti-slop.mdc`)
- [x] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `README.md`, `docs/`, `deploy/vendor/` (`naming.mdc`)

Rule: OpenAPI-specific

- [x] Spec paths use `{param}` style matching `routeCatalog` (Go 1.22 mux patterns)
- [x] Schema property names are `snake_case` matching existing `FooDTO` json tags
- [x] No new `dto/` subtree or parallel Entity/Model/View for the same table (`code-style.mdc`, `cold-path.mdc`)
- [x] Stub or 501 routes either omitted from public spec or tagged `x-stub: true` with documented behavior

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

| Slug | Priority | Surface | Status |
| :--- | :--- | :--- | :--- |
| `openapi_layout_and_scope` | foundation | `api/openapi/` tree | Shipped |
| `openapi_route_catalog_export` | foundation | generator from `routeCatalog` | Shipped |
| `openapi_spectral_lint` | foundation | CI lint rules | Shipped |
| `openapi_cost_sync_pilot` | pilot | 6 cost-sync routes + DTOs | Shipped |
| `openapi_typescript_codegen` | pilot | `web/src/types/generated/` | Shipped |
| `openapi_ci_route_parity_gate` | foundation | `scripts/ci/openapi_gate.sh` | Shipped |
| `openapi_ci_schema_golden_gate` | optional | DTO parity tests per domain | Shipped |
| `openapi_docs_ui` | optional | `/api/v1/openapi.json` + Redoc | Deferred |
| `openapi_integrations_domain` | domain_expand | postbacks, supply, schemas, templates | Shipped |
| `openapi_campaigns_domain` | domain_expand | campaigns, flows, landers | Shipped |
| `openapi_billing_domain` | domain_expand | billing, self-serve, invoices | Shipped |
| `openapi_ops_reports_domain` | domain_expand | reports, ops, fraud admin | Shipped |
| `openapi_new_endpoint_policy` | workflow | `docs/DEVELOPMENT.md` section | Shipped |
| `openapi_breaking_change_guard` | workflow | PR diff on spec | Shipped |
| `openapi_optional_request_validation` | optional | kin-openapi middleware | Shipped |

---

## `openapi_layout_and_scope`

**Priority:** foundation

**Status:** Shipped.

**Gap (was):** No canonical OpenAPI entrypoint; `/api/v1` contract implicit in Go + TS duplicates.

**Current state:** `api/openapi/openapi.yaml` v0.6.0; domains under `paths/` and `components/schemas/`; policy in `info.description`.

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

- [x] `api/openapi/openapi.yaml` validates with Spectral (warnings only on missing operation descriptions)
- [x] No duplicate path keys across `$ref` files (`TestExport_idempotent`, bundle export)
- [x] `docs/INTEGRATIONS.md` links to this backlog slug once; no claim that full `/api/v1` is spec-complete

---

## `openapi_route_catalog_export`

**Priority:** foundation

**Status:** Shipped.

**Gap (was):** `routeCatalog` Go-only; no published contract parity.

**Current state:** `go run ./cmd/openapi-export` / `make openapi-export` writes `paths/_generated_routes.yaml` and `openapi.bundle.yaml`; `TestAssertCatalogParity` in `internal/openapi/`.

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

- [x] Generated stubs cover `routeCatalog` minus `api/openapi/parity_allowlist.txt` exclusions
- [x] `go test ./internal/openapi/ -count=1` (`TestAssertCatalogParity`, `TestDocumentedRoutes_inCatalog`)
- [x] Documented in `docs/DEVELOPMENT.md` (`make openapi-export`)

---

## `openapi_spectral_lint`

**Priority:** foundation

**Status:** Shipped.

**Gap (was):** No style or completeness rules on YAML contract.

**Current state:** `api/openapi/spectral.yaml`; invoked from `scripts/ci/openapi_gate.sh` via `lint_configs_gate.sh`.

**Target:**

- `deploy/openapi/spectral.yaml` (or `.spectral.yaml` at repo root) with project rules:
  - `operationId` must be unique and camelCase
  - Every `paths.*` under pilot domains must declare `responses` (at least `default` or `4xx` + success)
  - No empty `schema: {}` on request bodies in pilot domains
  - `info.version` semver aligned with release tag policy (document bump rule)
- Wire into existing orchestrator: extend `lint_configs_gate.sh` **once** (no nested gate-only wrapper)

### Done gates

- [x] `bash scripts/ci/lint_configs_gate.sh` runs Spectral on `api/openapi/openapi.yaml`
- [x] CI fails on duplicate `operationId` (Spectral `operation-operationId-unique`)

---

## `openapi_cost_sync_pilot`

**Priority:** pilot

**Status:** Shipped.

**Gap (was):** Cost-sync DTOs duplicated in Go and hand-written TS.

**Current state:** Full schemas in `components/schemas/cost_sync.yaml`; parity in `openapi_cost_sync_test.go`.

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

- [x] Spectral clean on merged pilot paths (0 errors)
- [x] `internal/controlplane/openapi_cost_sync_test.go` asserts DTO json keys vs schema
- [x] `bash scripts/ci/cold_path_json_gate.sh` passes on handler paths

---

## `openapi_typescript_codegen`

**Priority:** pilot

**Status:** Shipped.

**Gap (was):** Manual types in `cost_sync_api.ts` drifted from Go DTOs.

**Current state:** `openapi-typescript` in `web/package.json`; `npm run openapi:types`; committed `web/src/types/generated/openapi.d.ts`; thin re-exports in `web/src/types/*.ts`.

**Target:**

- Add devDependency `openapi-typescript` (pin version in `web/package.json`)
- Script: `npm run openapi:types` reads repo-root `api/openapi/openapi.yaml` -> `web/src/types/generated/openapi.d.ts`
- Refactor `web/src/helpers/cost_sync_api.ts` to import `components['schemas'][...]` for pilot types; keep fetch/`apiConfirmed` wrappers local
- `npm run typecheck` in `admin_web.sh` path must pass
- Generated file gitignored **or** committed with `make gen` regen policy (pick one in `openapi_layout_and_scope` PR and document)

**Recommendation:** commit generated TS under `web/src/types/generated/` and regen in `make gen` for sources-only clone honesty (`core.mdc`).

### Done gates

- [x] `cd web && npm run typecheck`
- [x] `bash scripts/ci/admin_web.sh`
- [x] No user-visible copy change; `check_ui_slop.sh` clean
- [x] JSDoc on exported helper functions preserved (`ui.mdc`)

---

## `openapi_ci_route_parity_gate`

**Priority:** foundation

**Status:** Shipped.

**Gap (was):** OpenAPI could miss routes that exist in production mux.

**Current state:** `scripts/ci/openapi_gate.sh` runs `openapi.AssertCatalogParity`; allowlist `api/openapi/parity_allowlist.txt`.

**Target:**

- Leaf gate `scripts/ci/openapi_route_parity_gate.sh`:
  - Parse `routeCatalog` (Go test JSON export or static YAML) and OpenAPI paths
  - Fail if catalog route missing from spec (allowlist file for intentional omissions: stubs, internal ops)
  - Allowlist path: `api/openapi/parity_allowlist.txt` with one `METHOD PATH` per line + comment
- Invoked from `lint_configs_gate.sh` or `pr_fast.sh` once spec lands (not both nested)

### Done gates

- [x] Gate fails when a new `routeCatalog` row is added without spec update or allowlist entry
- [x] Gate passes with full hand-documented catalog + generated stubs

---

## `openapi_ci_schema_golden_gate`

**Priority:** optional

**Status:** Shipped (DTO parity tests per domain, not separate golden JSON files).

**Gap (was):** Route parity alone does not catch field rename drift.

**Current state:** `internal/controlplane/openapi_{cost_sync,integrations,campaigns,billing,ops_reports}_test.go` compare marshaled DTO json keys to YAML schema properties.

**Target:**

- For pilot domain only: golden JSON files checked into `internal/controlplane/testdata/openapi/`
- Test loads spec schema (or embedded subset) and validates golden fixtures
- Expand to next domain when that domain slug closes

### Done gates

- [x] Tests fail if Go DTO gains json field without spec property (`TestOpenAPI_*SchemaKeys`)
- [x] Documented in `docs/DEVELOPMENT.md` OpenAPI section

---

## `openapi_docs_ui`

**Priority:** optional

**Status:** Deferred (post-close non-goal).

**Gap:** Operators have no browsable contract in the admin UI.

**Current state:** Spec is file + CI artifact (`api/openapi/openapi.bundle.yaml`). Integrators read YAML or generate TS; no `GET /api/v1/openapi.json` or Redoc mount on control plane.

**Rationale for deferral:** Contract honesty already enforced by `openapi_gate.sh`; browsable UI adds cold-path static surface without changing merge gates. Re-open only with explicit product decision.

**Target:**

- Serve bundled spec at `GET /api/v1/openapi.json` (control plane, auth required or public read with license gate - decide in PR)
- Static Redoc or Swagger UI under `/api/v1/docs` (embedded in control static handler, not hot path)
- Link from admin Settings or Integrations footer (one link; no marketing prose)

### Done gates

- [ ] `GET /api/v1/openapi.json` serves bundled bytes (deferred)
- [ ] Redoc or Swagger UI under `/api/v1/docs` (deferred)

---

## `openapi_integrations_domain`

**Priority:** domain_expand

**Status:** Shipped.

**Routes:** `/api/v1/postbacks/*`, `/api/v1/supply/*`, `/api/v1/integration/*`, `/api/v1/cost-sync/*` (complete if pilot partial), `/api/v1/smart-alerts/*`, `/api/v1/margin-guard/*`, `/api/v1/platform-campaigns/*`

**Helpers to migrate:** `postback_api.ts`, `supply_api.ts`, `integration_api.ts`, `margin_guard_api.ts`, `smart_alerts_api.ts`

### Done gates

- [x] TS types sourced from generated OpenAPI for this slice
- [x] `docs/INTEGRATIONS.md` operator tables still accurate (field names match spec)

---

## `openapi_campaigns_domain`

**Priority:** domain_expand

**Status:** Shipped.

**Routes:** `/api/v1/campaigns/*`, `/api/v1/flows/*`, `/api/v1/brands/*`, lander hosted routes, wizard payloads

**Helpers:** `campaign_admin_api.ts`, `flows_api.ts`, `brand_creatives_api.ts`, `domains_api.ts`

**Note:** `CampaignDTO` is large; split schema file `components/schemas/campaign.yaml` with `$ref` slices (budget, fraud flags, ingress_cost_config) to keep reviews small.

### Done gates

- [x] `ingress_cost_config` documented alongside `docs/INTEGRATIONS.md` ingress macro section
- [x] PATCH partial update semantics documented (`application/json` merge rules)

---

## `openapi_billing_domain`

**Priority:** domain_expand

**Status:** Shipped.

**Routes:** `/api/v1/billing/*`, `/api/v1/selfserve/*`, `/api/v1/customers/*` billing subpaths

**Helpers:** `billing_admin_api.ts`, `selfserve_api.ts`, `selfserve_billing_api.ts`

**Sensitive fields:** invoice PDF routes as `application/pdf`; webhook bodies out of public operator spec or separate webhook OpenAPI fragment.

### Done gates

- [x] Self-serve Bearer auth documented under `components/securitySchemes`
- [x] No secret echo in response examples (`anti-slop.mdc` UI slop)

---

## `openapi_ops_reports_domain`

**Priority:** domain_expand

**Status:** Shipped.

**Routes:** `/api/v1/reports/*`, `/api/v1/ops/*`, `/api/v1/fraud/*`, `/api/v1/dashboards/*`, `/api/v1/views/*`

**Helpers:** `report_api.ts`, `fraud_*`, `ops_*`, `tg_report_api.ts`

**Note:** Report query params are numerous; prefer shared parameter `$ref`s and document `stale` response metadata from `control-plane.mdc`.

### Done gates

- [x] `report_live_routes_gate.sh` still passes; OpenAPI documents report keys referenced in UI where `live: true`

---

## `openapi_new_endpoint_policy`

**Priority:** workflow

**Status:** Shipped.

**Gap (was):** New endpoints landed as Go + manual TS only.

**Target process (document in `docs/DEVELOPMENT.md`):**

1. PR 1: OpenAPI path + schemas + Spectral clean
2. PR 2 (may squash with 1 if small): Go handler implements contract; TS uses generated types
3. `routeCatalog` row + spec path in same commit
4. Permission string documented on operation (`x-permissions: campaigns:write` extension acceptable)

**Until all domains migrated:** legacy endpoints remain code-first; spec catches up via export + domain slugs, not big-bang rewrite.

### Done gates

- [x] `docs/DEVELOPMENT.md` section added; cost-sync pilot cited as example (`core.mdc`)
- [x] Team default documented: new `/api/v1` routes need OpenAPI row before handler merge (`openapi_gate.sh` in `lint_configs_gate.sh`)

---

## `openapi_breaking_change_guard`

**Priority:** workflow

**Status:** Shipped.

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

**Status:** Shipped (opt-in via `OPENAPI_REQUEST_VALIDATION=1`).

**Target:**

- Opt-in `kin-openapi` request validator middleware for selected routes (start with self-serve write paths)
- Fail closed 400 with stable error envelope; never on hot path binaries

**Rejected for default deploy:** full oapi-codegen server replacement of `controlplane` handlers (cost vs 256 routes).

### Done gates

- [x] Middleware off by default; env flag documented
- [x] Fault test: invalid body rejected before service layer (`TestOpenAPIRequestValidation_rejectsInvalidBodyBeforeHandler`)

---

## Domain rollout tracker

All catalog routes are in spec (hand-documented or generated stub). TS codegen covers documented domains via `openapi.d.ts`.

| Domain | Spec schemas | TS codegen | Route parity |
| :--- | :---: | :---: | :---: |
| cost-sync (pilot) | [x] | [x] | [x] |
| integrations / postbacks / supply | [x] | [x] | [x] |
| campaigns / flows / landers | [x] | [x] | [x] |
| billing / self-serve | [x] | [x] | [x] |
| ops / reports / fraud admin | [x] | [x] | [x] |
| team / publisher / telegram | [x] | [x] | [x] |
| platform / licensing / meta / rtb | [x] | [x] | [x] |

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

## Explicit non-goals (post-close)

| Item | Why deferred |
| :--- | :--- |
| `openapi_docs_ui` (Redoc, `/api/v1/openapi.json`) | CI + committed bundle sufficient; no operator demand for in-app browser |
| Full `oapi-codegen` server regen | Rejected in [Rejected alternatives](#rejected-alternatives) |
| Spectral `operation-description` on every stub | 210 warnings only; fill incrementally when touching paths |
| `pr_fast.sh` duplicate OpenAPI gate | `lint_configs_gate.sh` already runs `openapi_gate.sh` once |

---

## Verification commands (orchestrator)

```bash
make openapi-export
make openapi-types
go test ./internal/openapi/ -count=1
go test ./internal/controlplane/ -run TestOpenAPI_ -count=1
bash scripts/ci/openapi_gate.sh
```

---

## First PR suggestion (minimal slice)

Shipped 2026-08. Historical reference only:

1. `api/openapi/openapi.yaml` + `paths/cost_sync.yaml` + schemas
2. `make openapi-export` paths-only for full catalog
3. Spectral in `lint_configs_gate.sh`
4. `openapi-typescript` + migrate `cost_sync_api.ts`
5. Parity test: catalog contains all cost-sync paths present in spec

Cross-reference: slugs `openapi_layout_and_scope`, `openapi_cost_sync_pilot`, `openapi_typescript_codegen`.

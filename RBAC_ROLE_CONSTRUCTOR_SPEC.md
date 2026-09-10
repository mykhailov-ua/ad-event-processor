# RBAC Role Constructor — Technical Specification

Status: **spec only** (not implemented).  
Owner: control plane / admin UI.  
Canonical enforcement: `.cursor/rules/control-plane.mdc` (RBAC), `deploy/operator/roles.yaml`.

---

## 1. Purpose

Deliver a **production-ready role constructor** so an operator can define who sees which admin pages and which mutations are allowed, without redeploying binaries.

Requirements from product:

- Simple mental model: **page visible / hidden**, **can edit campaigns**, **can delete/archive campaigns**, **can configure postbacks**, etc.
- **Single source of truth**: YAML file on disk (`OPERATOR_ROLES_YAML`, default `deploy/operator/roles.yaml`).
- **Two edit surfaces** with identical semantics:
  1. Admin UI (Settings → Access → Roles).
  2. HTTP API (curl / Postman); optional raw YAML upload.
- **Server-authoritative** RBAC: UI/nav is UX only; every `/api/v1/*` route stays behind `RequirePermission` / `RequireAnyPermission`.

Non-goals:

- Per-request dynamic policy from external SaaS.
- RBAC on tracker hot path (`/track`, `/click`) — out of scope (`hot-path.mdc`).
- Replacing Postgres `users.role` with a separate identity provider.
- Full ABAC (attribute-based) expressions in v1.

---

## 2. Current state (honest baseline)

| Area | Today | Gap |
| :--- | :--- | :--- |
| Role matrix | Embedded `internal/control/http/rbac.go` + merge from `roles.yaml` at startup | No CRUD API; custom roles require manual YAML + `POST /api/v1/ops/roles/reload` |
| Reload | `POST /api/v1/ops/roles/reload` (`settings:write`) reloads file into `authz.Store` | No validation API, no dry-run, no version/etag |
| Team UI | `/team` — free-text `role` field | Server accepts only `TL`, `MB`, `B`; looks like a constructor but is not |
| Session bootstrap | `GET /api/v1/session/bootstrap` returns `permissions[]` from static matrix fallback | Drift vs live `authz.Store` when YAML customized |
| DB grants | `auth.user_roles` query in `policy_db.go` | **No migrations**; path is non-operational |
| Page gating | `web/src/lib/nav_config.ts`, `route_permissions.ts`, `PermissionGate` | Partial route coverage; ops link historically under-gated in nav |
| Postbacks / delete | OpenAPI uses `campaigns:read` / `campaigns:write` | Cannot deny postbacks while allowing campaign edits without new permissions |
| Audit | `admin_audit_log` for many admin mutations | Role YAML changes not audited today |

This spec closes those gaps in one coherent delivery, not a UI stub over the existing reload button.

---

## 3. Goals and success criteria

### 3.1 Functional

1. Operator creates role `ANALYST` with: sees Campaigns + Reports, read-only campaigns, no Settings, no Ops.
2. Operator assigns `ANALYST` to a user on `/team` via dropdown (not free text).
3. User with `ANALYST` gets `403 FORBIDDEN` on `PATCH /api/v1/campaigns/{id}` and sees `ErrorBlock` / forbidden state in UI.
4. Same role matrix editable via API `PUT /api/v1/access/roles` (JSON) or `PUT /api/v1/access/roles.yaml` (raw body).
5. Invalid YAML (unknown permission, reserved code, cyclic inheritance if used) returns `400` with field errors; **disk file unchanged**.
6. Successful apply updates in-memory `authz.Store` without process restart; active sessions pick up new permissions on next bootstrap refresh (or explicit policy refresh hook).

### 3.2 Non-functional

| Requirement | Target |
| :--- | :--- |
| Apply latency | p95 < 200 ms for YAML up to 256 KiB (cold path) |
| Availability | Failed apply must not corrupt last good YAML (atomic write + backup) |
| Concurrency | Last-write-wins with `If-Match` / `revision` conflict `409` |
| Security | Only `A` or role with `access:write` may mutate roles; never expose `*` assignment through UI without break-glass confirm |
| Honesty | OpenAPI `x-permissions` == mux wrappers; CI gate fails on drift |

### 3.3 Definition of done (production-ready)

- [ ] All acceptance scenarios in section 12 pass on T1+ stack (`curl :8188/health`).
- [ ] Holdout tests: deny path for custom role on operator-only route; apply invalid YAML does not change effective policy.
- [ ] `bash scripts/ci/pr_fast.sh` green; new static gate `roles_contract_gate.sh` green.
- [ ] Playwright **L2** for role apply + team assign (not heading-only).
- [ ] Operator runbook section in `deploy/vendor/VENDOR_OPS_RUNBOOK.md` (same PR as code).

---

## 4. Concepts

### 4.1 Layers

```
capabilities (UI toggles, operator-friendly)
        | compile
permissions (wire slugs, handler checks)
        | group
roles (code -> scope + permissions[+capabilities])
        | assign
users.role (PG column users.role)
```

- **Permission** — atomic string checked by middleware/handlers (`campaigns:write`). Canonical list in section 5.
- **Capability** — stable UI toggle id (`page.campaigns`, `action.campaigns.archive`). Compiles to one or more permissions. Defined in repo catalog, not invented per install.
- **Role** — named bundle: `code`, `scope`, `permissions[]`, optional `capabilities[]` (compiler expands capabilities into permissions on save).
- **Built-in role** — shipped in repo (`A`, `M`, `U`, `TL`, `MB`, `B`, `S`, `P`). May be **extended** by operator YAML but not **deleted** while users reference the code.
- **Custom role** — operator-defined code matching `^[A-Z][A-Z0-9_]{0,15}$`, not in reserved set.

### 4.2 Scope (tenant isolation)

Unchanged from today (`authz.Scope`):

| Scope | Meaning | List/mutation filters |
| :--- | :--- | :--- |
| `global` | Operator (`A`, `S`) | Cross-customer where handler allows |
| `customer` | Customer admin (`M`, `U`) | `customer_id` bound to session |
| `team` | Media buyers (`TL`, `MB`, `B`) | Campaign ownership / team filters |

Custom roles must declare `scope`. Constructor UI defaults new custom roles to `team` or `customer`; only `access:write` holders may create `global` roles.

### 4.3 Masking

Mask level remains derived from permissions (`MaskLevelFromPermissions`):

- `campaigns:read` or `*` → full campaign fields.
- Only `campaigns:read:masked` → masked DTO; `IsMaskedMutation` rejects budget/url writes.

Capability `data.campaigns.unmasked` maps to `campaigns:read`; `data.campaigns.masked_only` maps to `campaigns:read:masked` (mutually exclusive in UI).

---

## 5. Permission catalog

### 5.1 Existing permissions (v1 baseline — must remain valid)

| Permission | Typical use |
| :--- | :--- |
| `*` | Admin superuser only |
| `customers:read` / `customers:write` | Customer directory, wallet, tax |
| `campaigns:read` / `campaigns:read:masked` | Campaign list/detail |
| `campaigns:write` / `campaigns:write:masked` | Create/update campaign, flows, landers |
| `campaigns:pause` | Pause/resume |
| `brands:read` / `brands:write` | Brand/creative admin |
| `billing:read` / `billing:write` | Invoices, ledger views |
| `settings:read` / `settings:write` | Platform settings, EULA, license panels |
| `blacklist:read` / `blacklist:write` | Fraud blacklist |
| `audit:read` | Audit log |
| `users:write` | User/team admin |
| `shards:read` / `shards:write` | Ops console |
| `ops:write` | Ops mutations |
| `rtb:read` / `rtb:write` | RTB admin |
| `supply:read:scoped` | Publisher supply |
| `access:read` | Read role matrix (new) |
| `access:write` | Mutate role matrix (new) |

### 5.2 New fine-grained permissions (v1 — required for honest toggles)

Split mutations that today share `campaigns:write`:

| Permission | Routes / behavior | Replaces (initially) |
| :--- | :--- | :--- |
| `campaigns:archive` | Bulk archive, archive endpoints | part of `campaigns:write` |
| `campaigns:delete` | Hard delete if exposed | part of `campaigns:write` |
| `postbacks:read` | `GET /api/v1/postbacks/*` read family | `campaigns:read` on those routes |
| `postbacks:write` | `PUT/POST` postback config, DLQ retry | `campaigns:write` on those routes |
| `exports:read` | Report jobs list/download metadata | `campaigns:read` on export hub |
| `exports:run` | Create async export jobs | `campaigns:write` on export mutations |
| `team:read` | Team roster list | `campaigns:read` on team overview |
| `team:write` | Invite, patch member | `users:write` / team handlers |

**Migration rule:** handlers accept **either** legacy or new permission during one release (`RequireAnyPermission`), then OpenAPI documents the new slug; gate fails if route still checks only legacy after cutover date.

Source of truth for catalog: new file `deploy/operator/permissions.yaml` (generated constants in `internal/controlplane/authz/permission_catalog.go` via `make gen` or hand-maintained with CI diff gate).

---

## 6. Capability catalog (UI)

Capabilities are **not** checked at runtime directly; they compile to permissions on save/load.

### 6.1 Pages (nav visibility)

Maps to `web/src/lib/nav_config.ts` and `internal/commandpalette/routes.go`.

| Capability ID | Label | Compiles to (minimum) |
| :--- | :--- | :--- |
| `page.customers` | Customers | `customers:read` |
| `page.campaigns` | Campaigns | `campaigns:read` OR `campaigns:read:masked` |
| `page.team` | Team | `team:read` |
| `page.settings` | Settings | `settings:read` |
| `page.exports` | Exports | `exports:read` |
| `page.ops` | Ops | `shards:read` |
| `page.audit` | Audit | `audit:read` |
| `page.integrations` | Integrations hub | `campaigns:read` OR `postbacks:read` |
| `page.integrations.postbacks` | Postbacks / CAPI | `postbacks:read` |
| `page.billing` | Billing | `billing:read` |
| `page.rtb` | RTB | `rtb:read` |

Server returns this catalog from `GET /api/v1/access/catalog` so UI does not hardcode labels.

### 6.2 Actions (mutations)

| Capability ID | Label | Compiles to |
| :--- | :--- | :--- |
| `action.campaigns.create` | Create campaigns | `campaigns:write` |
| `action.campaigns.edit` | Edit campaigns / flows | `campaigns:write` |
| `action.campaigns.pause` | Pause / resume | `campaigns:pause` |
| `action.campaigns.archive` | Archive campaigns | `campaigns:archive` |
| `action.campaigns.delete` | Delete campaigns | `campaigns:delete` |
| `action.postbacks.view` | View postback config | `postbacks:read` |
| `action.postbacks.edit` | Edit postback config | `postbacks:write` |
| `action.exports.run` | Run exports | `exports:run` |
| `action.team.invite` | Invite team members | `team:write` |
| `action.team.manage` | Block / spend cap | `team:write` |
| `action.settings.edit` | Platform settings | `settings:write` |
| `action.blacklist.edit` | Fraud blacklist | `blacklist:write` |
| `action.billing.edit` | Billing mutations | `billing:write` |

UI groups capabilities under Pages / Campaigns / Integrations / Admin. Advanced tab shows raw `permissions[]` for break-glass (requires `access:write`).

---

## 7. YAML schema (versioned)

File: `deploy/operator/roles.yaml` (or `OPERATOR_ROLES_YAML`).

```yaml
version: 1
revision: 4          # monotonic int; server increments on each successful apply
updated_at: "2026-09-10T15:00:00Z"
updated_by: "uuid"   # last editor user_id

roles:
  MB:
    scope: team
    builtin: true
    label: Media buyer
    capabilities:
      - page.campaigns
      - action.campaigns.create
      - action.campaigns.edit
      - action.campaigns.pause
    # permissions optional when capabilities present; compiler fills permissions:

  ANALYST:
    scope: team
    builtin: false
    label: Reporting only
    capabilities:
      - page.campaigns
      - page.exports
      - action.exports.run
    permissions: []   # explicit extras allowed

  POSTBACK_OPS:
    scope: customer
    label: Postback manager
    capabilities:
      - page.integrations
      - page.integrations.postbacks
      - action.postbacks.view
      - action.postbacks.edit
```

### 7.1 Validation rules

| Rule | Error |
| :--- | :--- |
| `version` must be `1` | `400 BAD_REQUEST` |
| Role code unique, uppercase | `400` |
| Reserved codes `A` delete forbidden | `403` |
| Unknown capability / permission | `400` with `unknown_capability` |
| `capabilities` + `permissions` merged; duplicates deduped | — |
| `scope` in `global|customer|team` | `400` |
| Role referenced by `users.role` cannot be removed | `409 CONFLICT` |
| At least one permission after compile | `400` |
| Max roles 128, max permissions per role 64, file max 256 KiB | `400` |

### 7.2 Built-in defaults

Repo ships canonical `deploy/operator/roles.yaml` matching current matrix. On first boot, if operator file missing, copy embedded default. `builtin: true` marks rows that upgrade merge must preserve (installer may overlay operator edits but not drop `A`).

### 7.3 Atomic persistence

Apply pipeline (`internal/controlplane/access/roles_store.go`):

1. Validate parsed doc + compile capabilities.
2. Write `roles.yaml.new` in same directory.
3. `fsync`; rename to `roles.yaml` (atomic on same filesystem).
4. Copy previous to `roles.yaml.bak` (keep last 1 backup).
5. `authz.LoadRolesYAML` into store; `store.Reload()` clears user snapshot cache.
6. Append `admin_audit_log` row `action=access_roles_apply`.
7. Return new `revision` + compiled permission list per role.

Failed step 1–3: **no** change to live file or memory store.

---

## 8. HTTP API

Prefix: `/api/v1/access/*`. All routes require session auth. Mutations require `access:write` (and `settings:write` for backward compat during migration — deprecate in release+1).

| Method | Path | Permission | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/access/catalog` | `access:read` | Capabilities + permissions metadata |
| `GET` | `/api/v1/access/roles` | `access:read` | JSON view of roles (compiled permissions included) |
| `GET` | `/api/v1/access/roles.yaml` | `access:read` | Raw YAML bytes (`Content-Type: application/yaml`) |
| `PUT` | `/api/v1/access/roles` | `access:write` | Replace matrix from JSON body |
| `PUT` | `/api/v1/access/roles.yaml` | `access:write` | Replace matrix from raw YAML (`If-Match: revision`) |
| `POST` | `/api/v1/access/roles/validate` | `access:read` | Dry-run validate; no write |
| `POST` | `/api/v1/access/roles/reload` | `access:write` | Re-read file from disk (replaces legacy ops-only reload) |

Legacy `POST /api/v1/ops/roles/reload` remains as alias calling same service for one release.

### 8.1 Example: curl apply YAML

```bash
curl -sf -X PUT \
  -H "Cookie: $SESSION" \
  -H "Content-Type: application/yaml" \
  -H "If-Match: 4" \
  --data-binary @deploy/operator/roles.yaml \
  http://127.0.0.1:8188/api/v1/access/roles.yaml
```

### 8.2 Example: JSON patch single role

```bash
curl -sf -X PUT \
  -H "Cookie: $SESSION" \
  -H "Content-Type: application/json" \
  -d '{"version":1,"revision":4,"roles":{"ANALYST":{"scope":"team","label":"Analyst","capabilities":["page.campaigns","page.exports"]}}}' \
  http://127.0.0.1:8188/api/v1/access/roles
```

### 8.3 OpenAPI

Add `api/openapi/paths/access.yaml`; register in `openapi.yaml`. Every route documents `x-permissions`. Run `bash scripts/ci/admin/openapi.sh`.

### 8.4 Error envelope

Standard `{"error":{"code":"...","message":"..."}}`. Validation returns `400` with optional `details: [{field, code}]` (same pattern as admin mutations, `frontend-slop.mdc` SV*).

---

## 9. Server implementation

### 9.1 Package layout (`modular-monolith.mdc`)

New domain package:

```
internal/access/
  doc.go
  catalog.go          # capabilities + permissions metadata
  compile.go            # capabilities -> permissions
  validate.go
  roles_store.go        # read/write YAML, atomic apply
  handlers.go           # HTTP
  handlers_test.go
internal/controlplane/access_bridge.go   # wire only
```

Forbidden: logic in `controlplane` god file; new logic in bridge + `internal/access/`.

### 9.2 Policy store integration

- Extend `authz.LoadRolesYAML` to parse `version`, `capabilities`, call `access.CompileRole`.
- `InitPolicyStore()` load order unchanged: embed → YAML file.
- `SessionPermissions` / bootstrap must use **`authz.Store` only**, not `GetPermissionsForRole` static fallback when store loaded (`control-plane.mdc` known gap).
- On apply: `policy.RefreshAllUsers()` or bump generation counter so cached `userSnaps` invalidate.

### 9.3 Handler permission migration

For each route family in section 5.2:

1. Add `RequireAnyPermission` with new + legacy permission.
2. Update OpenAPI `x-permissions`.
3. Add httest: custom role with `postbacks:read` only → `GET /api/v1/postbacks/config` 200, `PUT` 403.

Static gate `scripts/ci/static/roles_contract_gate.sh`:

- Every permission in `permissions.yaml` appears in catalog GET schema or is marked `internal`.
- Every `x-permissions` entry exists in catalog.
- No handler registers without `Require*` (extend existing rbac scan).

### 9.4 User assignment

**Team invite/update** (`internal/platformadmin/governance.go`):

- Replace free-text role with validated role **code** from current matrix (`GET /api/v1/access/roles` keys).
- `NormalizeTeamRole` deprecated; new `ValidateAssignableRole(code)` allows any role with `scope=team` OR operator override.
- Admin role `A` assignment only via CLI `cmd/admin/user` or `users:write` + break-glass.

**Users table:** keep `users.role` string column; no FK until `auth.roles` table ships (optional v1.1).

### 9.5 Optional v1.1: PG-backed overrides (not blocking v1)

If per-user grants are required later:

- Migration `auth.roles`, `auth.permissions`, `auth.role_permissions`, `auth.user_roles` (activate `policy_db.go`).
- YAML remains **role definition**; PG stores user→role mapping overrides only.
- v1 explicitly **does not** ship half-migrated `auth.*` tables.

---

## 10. Admin UI

Route: `/settings/access` (tab under Settings). Permission: `access:read` (view), `access:write` (edit).

### 10.1 Screens

| Screen | Behavior |
| :--- | :--- |
| Role list | Table: code, label, scope, member count, builtin flag |
| Role editor | Two columns: capability checkboxes (Pages / Actions) + Advanced permissions |
| YAML panel | Read-only view with copy; edit mode for `access:write` with Monaco-less textarea + validate button |
| Apply | `POST validate` then `PUT` with revision; `ErrorBlock` on 4xx; success toast after 2xx only |
| Assign | Link to `/team` — role dropdown populated from `GET /access/roles` |

### 10.2 UX rules (`frontend-slop.mdc`)

- Server owns catalog; no client-side permission invention.
- `PermissionGate` on `/settings/access` with `access:read`.
- Mutation buttons disabled without `access:write`.
- No `Promise.resolve` stubs; T1+ live API.
- E2E **L2**: apply role denying `campaigns:write`, login as test user, assert `403` on PATCH.

### 10.3 Team page fix (same PR)

Replace `Input` for role in `team_overview.tsx` with `Select` options from `GET /api/v1/access/roles?scope=team`.

---

## 11. Security

| Threat | Mitigation |
| :--- | :--- |
| Privilege escalation via YAML | `access:write` required; audit log; validate no self-escalation without `A` |
| `*` in custom role | UI forbids; API rejects unless actor has `A` + explicit `allow_wildcard: true` in body |
| Path traversal on YAML path | `OPERATOR_ROLES_YAML` must be absolute path under allowlist (`/etc/ad-event-processor/`, `deploy/operator/`) |
| Stale session after deny | Bootstrap refresh on visibility + after role apply for affected users |
| API key scopes | Self-serve API keys cannot call `/access/*`; existing `RestrictSnapshotForAPIKeyScopes` unchanged |
| CSRF | Existing CSRF on mutating `/api/v1` |

---

## 12. Acceptance scenarios

| ID | Scenario | Expected |
| :--- | :--- | :--- |
| AC-1 | Apply YAML adding role `VIEWER` with only `page.campaigns` + masked read | User sees Campaigns nav; campaign URLs/budgets masked; PATCH 403 |
| AC-2 | Remove `action.postbacks.edit` from role | `PUT /api/v1/postbacks/config/{id}` 403; GET 200 if read granted |
| AC-3 | Invalid permission `foo:bar` in YAML | 400; file on disk unchanged; store unchanged |
| AC-4 | Concurrent apply revision mismatch | Second writer gets 409 |
| AC-5 | Delete role in use | 409 `role_in_use` |
| AC-6 | MB deep-link `/ops` | Nav may show or hide; `GET /api/v1/ops/home` 403 |
| AC-7 | curl YAML apply + UI reload | Same effective permissions |
| AC-8 | `POST validate` only | No audit row; no file change |
| AC-9 | Reload after manual file edit on disk | `POST .../reload` picks up external edit |
| AC-10 | Bootstrap `permissions` matches store after apply | No static fallback drift |

---

## 13. Testing and CI

| Tier | Command | Scope |
| :--- | :--- | :--- |
| Unit | `go test ./internal/access/ -short -count=1` | compile, validate, conflict |
| Authz | `go test ./internal/controlplane/authz/ -short -run TestPolicyPermissionMatrix` | mask/scope |
| Handler | `go test ./internal/controlplane/ -short -run 'RBAC|Role|Access'` | httest 403 matrix |
| Holdout | `TestAccessApply_invalidYAMLDoesNotMutateStore_holdout` | revert safety |
| Holdout | `TestAccess_customRole_deniedPostbackWrite_holdout` | fine-grained perm |
| Static | `bash scripts/ci/static/roles_contract_gate.sh` | OpenAPI + catalog parity |
| Admin | `bash scripts/ci/admin/web.sh` | UI routes |
| E2E | `ADMIN_WEB_E2E_SMOKE=1` + new `access_roles_write.spec.ts` | L2 mutate |

Do not claim production RBAC from mocks only (`anti-slop.mdc` RB-S*).

---

## 14. Rollout plan

| Phase | Deliverable |
| :--- | :--- |
| **P0** | `permissions.yaml`, compile/validate, atomic store, API GET/PUT/validate, audit log, bootstrap fix |
| **P1** | Handler migration for `postbacks:*`, `campaigns:archive`, `exports:*`, `team:*`; contract gate |
| **P2** | Settings UI + Team role Select; deprecate ops reload path in docs |
| **P3** | (Optional) PG `auth.*` user grants; not required for v1 GA |

Feature flag: none required; incomplete migration uses `RequireAnyPermission` dual checks until P1 completes.

---

## 15. Documentation updates (same PR as code)

| File | Change |
| :--- | :--- |
| `deploy/vendor/VENDOR_OPS_RUNBOOK.md` | curl recipes, backup path, break-glass |
| `deploy/operator/roles.yaml` | Add `version`, `capabilities` to builtins |
| `.cursor/rules/control-plane.mdc` | Role constructor section; remove "no constructor" gap |
| `README.md` | Link to this spec until feature shipped |

---

## 16. Open decisions (operator sign-off)

| # | Question | Recommendation |
| :--- | :--- | :--- |
| D1 | Custom role codes: allow lowercase? | **No** — uppercase only, matches `NormalizeRole` |
| D2 | Per-customer role namespaces? | **v1 no** — global YAML per appliance; multi-tenant SaaS not default model |
| D3 | Split `campaigns:write` immediately? | **Yes** for archive/postbacks/export; keep dual-check one release |
| D4 | Who may grant `access:write`? | Only `A` and explicit `settings:write` + `access:write` on bootstrap admin |
| D5 | Embed YAML in binary for air-gap? | Optional `//go:embed` default; file on disk wins when present |

---

## 17. References

| Artifact | Path |
| :--- | :--- |
| Current role matrix | `deploy/operator/roles.yaml` |
| Permission constants | `internal/control/http/rbac.go`, `internal/controlplane/authz/policy.go` |
| Policy load | `internal/controlplane/authz/roles_yaml.go`, `internal/control/http/policy_init.go` |
| Middleware | `internal/controlplane/adminauth/middleware.go` |
| Nav permissions | `web/src/lib/nav_config.ts`, `web/src/lib/route_permissions.ts` |
| Command palette | `internal/commandpalette/routes.go` |
| RBAC rules | `.cursor/rules/control-plane.mdc` |

---

*End of specification.*

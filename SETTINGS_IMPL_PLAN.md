# Settings implementation plan

Operator-facing Settings for tracker and control-plane configuration **without manual `.env` / `.yaml` edits**. Configuration is stored in Postgres via HTTP API; persistence to tracker-facing env uses `POST /api/v1/settings/platform/apply` (writes `install.compose.env` on the server).

Sources: `api/openapi/paths/platform.yaml`, `internal/platformadmin/*`, `internal/licensingadmin/*`, `pkg/platformconfig/*`, `deploy/operator/roles.yaml`.

Cross-ref: `docs/CONTROL_PLANE_UI_SCOPE.md`, `web/src/api/settings_api.ts`.

---

## 1. Goals and non-goals

### Goals

- Single route `/settings` for Admin role (`settings:read` / `settings:write`).
- Configure tracker-relevant platform fields through PATCH + optional Apply.
- License lifecycle (view status, apply JWT).
- Explicit UX for **draft (PG) vs disk (compose env) vs running processes**.
- Every mutation: confirm where needed, `ErrorBlock` on failure, toast only after 2xx.

### Non-goals (stay env / ops / other routes)

| Surface | Why not Settings |
| :--- | :--- |
| `FILTER_TIMEOUT_MS`, Redis shard URLs (except profile-derived defaults on apply) | Env / compose only |
| `INSTALL_BOOTSTRAP_TOKEN`, license file path env | Bootstrap / installer |
| Cloudflare API tokens | Env (`domains` package) |
| Emergency breaker toggle | No HTTP API (read via `GET /api/v1/ops/shards`) |
| Arbitrary `system_settings` KV (RTB mode, fraud epochs) | Internal admin only |
| First-run bootstrap form | `/activate`, `/setup` + `X-Install-Token` |
| Domain SSL park/probe/bulk | `/ops/domains` (link from Settings) |
| Supply ads.txt / sellers | `/integrations` (same permissions) |
| Team / RBAC | `/team`, `POST /ops/roles/reload` |

---

## 2. Configuration pipeline (operator mental model)

```
┌─────────────┐    PATCH      ┌──────────────────┐
│ Settings UI │ ────────────► │ Postgres         │
│  (draft)    │               │ platform_config  │
└─────────────┘               │ restart_pending  │
       │                      └────────┬─────────┘
       │ POST apply                     │
       ▼                                │ edge_expose_* also
┌──────────────────┐                    ▼ SyncEdgeExpose (Redis)
│ install.compose  │            ┌──────────────────┐
│ .env on disk     │            │ Tracker/edge     │
└──────────────────┘            │ (after restart)  │
       │                        └──────────────────┘
       └── requires service restart for restart_required fields
```

| Layer | What changes | Operator action |
| :--- | :--- | :--- |
| Draft | `platform_config` in PG | **Save** (`PATCH`) |
| Edge flags (partial) | Redis `edge_expose_*` | Automatic on PATCH |
| Disk | `install.compose.env` | **Write to disk** (`POST apply`) |
| Running tracker | Process env | Restart tracker/edge (runbook; no UI button) |

**Fields in `restart_required`** (`pkg/platformconfig/validate.go` `RestartRequiredFields`):

- `ingress_schema`, `telemetry_enabled`, `edge_xdp`, `edge_expose_click`, `edge_expose_openrtb`
- `profile`, `network_interface`, `stripe` (any stripe sub-field change)

**Not in `restart_required`:** `tracking_domain`, `default_currency`, `timezone` — still need **Apply** for disk/env sync.

---

## 3. RBAC and access

| Role | `settings:read` | `settings:write` | Settings UI |
| :--- | :---: | :---: | :--- |
| A (Admin) | yes | yes | Full |
| M, U, TL, MB, B, S, P | no | no | Hidden / 403 |

| Endpoint | OpenAPI permission | Handler permission | Notes |
| :--- | :--- | :--- | :--- |
| `GET/PATCH /settings/platform` | `settings:read` / `settings:write` | same | Core |
| `POST /settings/platform/apply` | `settings:write` | same | Destructive disk write |
| `GET /license/status` | *(customers:read in code)* | `customers:read` | MB/TL can read |
| `POST /license/apply` | `settings:write` | same | Admin only |
| `GET /meta` | public | rate limit | Boot shell |
| `POST /eula/accept` | `settings:write` in OpenAPI | **auth only** | Usually EulaGate modal |

**Corner case:** non-Admin with `customers:read` can call license status API but must not see Settings nav (no `settings:read`). License diagnostics belong on Settings only for Admin.

---

## 4. Page information architecture

Route: `/settings` (aliases redirect: `/license`, `/licence`, `/settings/license`, `/settings/licence` → `/settings`).

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│ Settings                                    [License badge] │
├─────────────────────────────────────────────────────────────┤
│ [Global banners: restart required | PG≠disk hint | errors]  │
├─────────────────────────────────────────────────────────────┤
│ § Deployment summary (read-only)                            │
│ § License                                                   │
│ § Tracker endpoints                                         │
│ § Edge & ingress                                            │
│ § Regional defaults                                         │
│ § Payments (if meta.payment_enabled)                        │
│ § Persist to disk                                           │
├─────────────────────────────────────────────────────────────┤
│ Related: Manage domains → /ops/domains                      │
└─────────────────────────────────────────────────────────────┘
```

### Section → API mapping

| Section | Primary APIs | Editable |
| :--- | :--- | :---: |
| Deployment summary | `GET /meta`, `GET /settings/platform` | no |
| License | `GET /license/status`, `POST /license/apply`, `meta.license` | apply only |
| Tracker endpoints | `PATCH` `tracking_domain`; read templates from GET | yes |
| Edge & ingress | `PATCH` edge flags, `ingress_schema` | yes |
| Regional defaults | `PATCH` `default_currency`, `timezone` | yes |
| Payments | `PATCH` `stripe.*`; read `secrets.*` masked | yes |
| Persist | `POST /settings/platform/apply` | action |

---

## 5. UI components and buttons

| Control | Type | API | Enabled when |
| :--- | :--- | :--- | :--- |
| **Save changes** | Primary | `PATCH /settings/platform` | draft ≠ server, not saving, `settings:write` |
| **Discard** | Ghost | — | draft ≠ server |
| **Write install.compose.env** | Primary (destructive confirm) | `POST .../apply` | bootstrapped, not applying |
| **Apply license** | Primary | `POST /license/apply` | token non-empty |
| **Copy deployment ID** | Icon button | — | deployment_id present |
| **Manage domains** | Link | — | always (Admin) |

**Banned patterns** (`frontend-slop.mdc`): raw JSON dump, duplicate patch forms, toast before 2xx, silent `catch` → empty state.

---

## 6. UX scenarios (step-by-step)

### S1 — Open Settings (happy path)

**Pre:** User role A, `bootstrap_complete=true`, session valid.

1. User opens Account menu → **Settings**.
2. UI loads in parallel:
   - `GET /api/v1/meta` (from `MetaProvider`, refresh after mutations)
   - `GET /api/v1/settings/platform`
   - `GET /api/v1/license/status` (license section)
3. Page renders all sections; forms initialized from `platform.config`.
4. If `restart_required.length > 0`, show **Restart required** banner listing field keys (human labels).

**Success:** All GET 200; no blocking `ErrorBlock`.

**Errors:**

| Condition | UI |
| :--- | :--- |
| `GET /settings/platform` 403 | Full-page forbidden or redirect `/forbidden` |
| 502 (control down) | `ErrorBlock` "Could not load platform settings" |
| Timeout (`TIMEOUT`) | `ErrorBlock` with retry hint |
| `bootstrap_complete=false` | Redirect to `/activate` (shell gate; not Settings content) |

---

### S2 — View deployment summary (read-only)

**Fields displayed:**

| Label | Source |
| :--- | :--- |
| Product / version | `meta.product_name`, `meta.version` |
| Deployment ID | `meta.deployment_id` or `license.deployment_id` |
| Bootstrap | `meta.bootstrap_complete` |
| Ingress schemas (allowed) | `meta.ingress_schemas[]` |
| Payment module | `meta.payment_enabled` |
| Click URL template | `platform.click_url_template` |
| OpenRTB endpoint template | `platform.openrtb_endpoint_template` |

**Actions:** Copy deployment ID.

**Corner cases:**

- Empty `tracking_domain` → templates empty; show hint "Set tracking domain below".
- `payment_enabled=false` → hide Payments section entirely (not disabled stub).

---

### S3 — Replace license token

**API:** `POST /api/v1/license/apply` `{ "token": "<JWT>" }`

1. User pastes JWT in textarea **License token**.
2. Clicks **Apply license**.
3. UI sets `applying=true`, clears prior field error.
4. On 200: clear textarea, toast "License applied", `refreshMeta()`, refetch license status.
5. Update badge from `meta.license.state` / status response.

**Client validation:**

- Empty token → button disabled (no request).

**Server errors:**

| HTTP | Code | UI |
| :--- | :--- | :--- |
| 400 | `BAD_REQUEST` | `ErrorBlock` under form; show `error.message` |
| 403 | `FORBIDDEN` | `ErrorBlock` "Insufficient permissions" |
| 429 | rate limit | `ErrorBlock` "Too many attempts; wait and retry" |
| 5xx | `INTERNAL_ERROR` | `ErrorBlock` + support link from `license.support_url` if present |

**Corner cases:**

- Valid JWT but HWID mismatch (`hwid_match=false`) → show warning in summary; apply may still 200; display diagnostics read-only.
- `state=UNCONFIGURED` before apply → summary shows Unconfigured; after apply → Active/Trial/etc.
- Double-click Apply → `applying` guard; single in-flight request.
- User navigates away mid-request → abort via workspace cleanup if using `AbortSignal`.

---

### S4 — Change tracking domain

**API:** `PATCH { "tracking_domain": "trk.example.com" }`

1. User edits **Tracking domain** input.
2. Clicks **Save changes**.
3. PATCH returns updated view; UI updates templates from response.
4. Show info banner: "Saved to control plane. Write to disk and restart tracker for edge containers."

**Validation (server, `platformconfig.Validate`):**

- Strips `http://`, `https://`, path suffix.
- Rejects empty after normalize, spaces, invalid host → 400 `BAD_REQUEST` `invalid tracking_domain`.

**Client validation (before PATCH):**

- Trim whitespace.
- Optional: warn if value contains `/` or scheme (strip client-side to match server).

**Corner cases:**

- Domain set in platform but not in `/domains` registry → templates work in admin; SSL/health may fail until domain ops (link **Manage domains**).
- PATCH ok but user never Apply → tracker still old `TRACKING_DOMAIN` in env.
- Concurrent edit: on PATCH 409/validation error, refetch GET and reset draft.

---

### S5 — Change ingress schema

**API:** `PATCH { "ingress_schema": "ad_event_processor_native" | "openrtb_3" }`

1. User selects schema from select (options = `meta.ingress_schemas`).
2. Save → PATCH.
3. Response includes `restart_required` containing `ingress_schema`.
4. Banner: "Restart required: Ingress schema".

**Corner cases:**

- Invalid enum → 400 `invalid ingress_schema: ...`.
- Switch schema with traffic live → copy warns restart + apply; no auto restart.

---

### S6 — Toggle edge exposure and XDP

**API:** `PATCH` with `edge_expose_click`, `edge_expose_openrtb`, `edge_xdp` booleans.

1. User toggles switches in **Edge & ingress**.
2. Save → PATCH (Redis sync for expose flags runs server-side).
3. All three fields may appear in `restart_required`.

**Corner cases:**

| Case | Behavior |
| :--- | :--- |
| `profile=compose_dev` + `edge_xdp=true` | Server 400: `edge_xdp is not supported in compose_dev profile` — disable XDP switch when profile is compose_dev |
| Enterprise XDP not licensed | Flag may save to PG; edge binary may ignore — show doc link, not fake "enabled" on edge |
| Expose click without tracking domain | Allowed in PG; edge may serve nothing useful — warning if `tracking_domain` empty |

---

### S7 — Regional defaults (currency, timezone)

**API:** `PATCH { "default_currency": "USD", "timezone": "Europe/Berlin" }`

1. User edits currency (3 letters) and timezone (IANA string).
2. Save → PATCH.

**Validation:**

- `default_currency` must be 3-letter ISO when non-empty → else 400.

**Corner cases:**

- Invalid timezone not validated server-side today — optional client check; wrong values surface in downstream reporting only.
- Not in `restart_required` but still in compose env only after Apply.

---

### S8 — Stripe billing block (conditional)

**Visible when:** `meta.payment_enabled === true`.

**API:** `PATCH` with `stripe.enabled`, URLs, `secret_key`, `webhook_secret`.

**Fields:**

| Field | Write | Read |
| :--- | :--- | :--- |
| Enabled | PATCH | `config.stripe.enabled` |
| Success / cancel URLs | PATCH | `config.stripe.*` |
| Secret key | PATCH (non-empty only updates) | `secrets.stripe_secret_key` masked |
| Webhook secret | PATCH (non-empty only) | `secrets.stripe_webhook_secret` masked |

1. User enables Stripe, fills URLs.
2. To rotate secret: paste new value in password input → Save.
3. Empty secret fields on PATCH must **not** clear existing secrets (server preserves).

**Validation:**

- `stripe.enabled=true` without `secret_key` in resulting config → 400 `stripe.secret_key is required when stripe is enabled`.

**Corner cases:**

- Disable Stripe → restart_required includes `stripe`; secrets remain in PG until overwritten.
- `payment_enabled=false` → hide section (backend may still have stripe in config).

---

### S9 — Write configuration to disk (Apply)

**API:** `POST /api/v1/settings/platform/apply` body optional `{ "install_root": "/opt/..." }`

1. User optionally fills **Install root** (empty = server default from `platformconfig.FormatInstallRoot`).
2. Clicks **Write install.compose.env**.
3. Confirm dialog: "Overwrite install.compose.env on the server?"
4. POST apply.
5. On 200: show `written_path`, toast success, clear `restart_required` in PG (server clears pending list on apply — refetch GET).

**Compose env keys written** (`pkg/platformconfig/render.go`):

- `TRACKER_INGRESS_SCHEMA`, `AD_EVENT_PROCESSOR_TELEMETRY_ENABLED`
- `TRACKING_DOMAIN`, `DEFAULT_CURRENCY`, `PLATFORM_TIMEZONE`
- `REDIS_ADDRS` (from profile), `EDGE_EXPOSE_CLICK`, `EDGE_EXPOSE_OPENRTB`
- Stripe `STRIPE_*` when enabled

**Not written:** `edge_xdp`, `network_interface` (PG only today).

**Errors:**

| HTTP | Cause | UI |
| :--- | :--- | :--- |
| 400 | `platform config not bootstrapped` | `ErrorBlock`; redirect hint to `/activate` |
| 403 | FORBIDDEN | Permission error |
| 500 | disk permission / mkdir fail | `ErrorBlock` with message; do not toast success |

**Corner cases:**

- Apply without prior PATCH → writes **current PG config** (still valid).
- Apply clears restart pending in DB but **does not restart processes** — banner switches to "Restart services to activate".
- `install_root` path invalid → 500; show error, do not clear form.
- Double Apply → `applying` guard.

---

### S10 — Discard local draft

1. User edited fields but clicks **Discard**.
2. Reset draft from last successful GET snapshot.
3. No API call.

**Corner case:** Discard while PATCH in flight → disable Discard during `patching`.

---

### S11 — Pending restart banner workflow

**Trigger:** `GET /settings/platform` → `restart_required.length > 0`.

**Banner content:**

- Title: "Service restart required"
- List human-readable labels for each key (map `ingress_schema` → "Ingress schema", etc.)
- Actions: **Write to disk** (shortcut to S9), link to runbook anchor in `/docs`

**After successful Apply:**

- Refetch GET; if tracker not restarted, show secondary note: "Configuration on disk updated. Restart tracker and edge."

**Corner case:** User PATCHes multiple times — pending list replaced by server merge logic; always show latest GET.

---

### S12 — EULA (usually not on Settings page)

**When:** `meta.eula_required && !meta.eula_accepted` — handled by `EulaGate` modal, not Settings form.

If shown on Settings (optional fallback):

- `GET /api/v1/eula` → display text
- **Accept** → `POST /api/v1/eula/accept` `{ "version": "<legal.Version>" }`
- Mismatch → 400 `EULA version mismatch`

---

### S13 — Navigate to related surfaces

| Link | Target | Permission |
| :--- | :--- | :--- |
| Manage domains | `/ops/domains` | `settings:read` |
| Integrations / supply | `/integrations` | `settings:read` |
| Documentation | `/docs` | auth |

No embedded domain table on Settings (avoid duplicate ops UI).

---

## 7. Error handling contract (explicit)

### HTTP envelope (frontend `parseApiError`)

```json
{ "error": { "code": "BAD_REQUEST", "message": "..." } }
```

### Mutation matrix

| Operation | On success | On failure |
| :--- | :--- | :--- |
| PATCH platform | Toast "Settings saved"; refetch GET; update draft | `ErrorBlock` on section; keep draft; no toast |
| POST apply | Toast "Written to …"; show path; refetch GET | `ErrorBlock` in Persist section |
| POST license apply | Toast "License applied"; refresh meta | `ErrorBlock` under license form |
| GET platform | Render snapshot | Page-level `ErrorBlock` if blocking |

### Status code playbook

| Status | Code | Operator message (template) |
| :---: | :--- | :--- |
| 400 | `BAD_REQUEST` | Show server `message` (validation) |
| 403 | `FORBIDDEN` | "You don't have permission to change platform settings." |
| 409 | `CONFLICT` | Rare on PATCH; show message |
| 429 | `LIMIT_EXCEEDED` | "Rate limited. Retry later." |
| 0 | `TIMEOUT` | "Request timed out. Check control plane and retry." |
| 502 | proxy | "API unavailable. Is control plane running?" |
| 5xx | `INTERNAL_ERROR` | "Server error. Check control logs." |

### Loading / concurrency

- `useResource` for GET with abort on unmount.
- `patching` / `applying` / `licenseApplying` flags — disable submit buttons.
- `useCoalescedBumpRefresh` after mutations (no refresh hammer).
- No optimistic UI for PATCH (wait for 2xx before updating "saved" baseline).

---

## 8. Field reference (PATCH body)

| Field | Type | Restart? | Apply env var |
| :--- | :--- | :---: | :--- |
| `tracking_domain` | string | no | `TRACKING_DOMAIN` |
| `default_currency` | string (3) | no | `DEFAULT_CURRENCY` |
| `timezone` | string | no | `PLATFORM_TIMEZONE` |
| `ingress_schema` | enum | yes | `TRACKER_INGRESS_SCHEMA` |
| `telemetry_enabled` | bool | yes | `AD_EVENT_PROCESSOR_TELEMETRY_ENABLED` |
| `profile` | `single_vps` \| `compose_dev` | yes | `REDIS_ADDRS` (derived) |
| `edge_xdp` | bool | yes | — |
| `edge_expose_click` | bool | yes | `EDGE_EXPOSE_CLICK` |
| `edge_expose_openrtb` | bool | yes | `EDGE_EXPOSE_OPENRTB` |
| `network_interface` | string | yes | — |
| `stripe.*` | object | yes | `STRIPE_*` |

---

## 9. API gaps (do not build UI for)

| Item | Detail |
| :--- | :--- |
| Bootstrap `license_key`, `license_server`, `deployment_id` | OpenAPI only; handler ignores |
| `install.yaml` generation | `RenderInstallYAML` exists; apply handler does not call it |
| `POST /settings/platform/bootstrap` | First-run + `X-Install-Token`; belongs on `/activate`, not Settings |
| Emergency breaker | Read `emergency_breaker` on ops shards only |
| `system_settings` KV editor | No public API |

---

## 10. Frontend implementation plan

### Files (target)

| File | Role |
| :--- | :--- |
| `web/src/pages/settings_page.tsx` | Page entry |
| `web/src/domains/settings/settings_license.tsx` | License section (exists) |
| `web/src/domains/settings/settings_platform_form.tsx` | Tracker / edge / regional / stripe fields |
| `web/src/domains/settings/settings_persist_section.tsx` | Apply + install root |
| `web/src/domains/settings/settings_summary.tsx` | Deployment read-only band |
| `web/src/domains/settings/use_settings_page_workspace.ts` | GET + PATCH + apply orchestration |
| `web/src/domains/settings/settings_field_labels.ts` | Human labels for `restart_required` keys |
| `web/src/api/settings_api.ts` | Already wired |

### Workspace state

```ts
// Draft vs server snapshot pattern (single owner)
platformSnapshot: PlatformSettingsView | undefined
draft: PlatformConfigPatch fields
patching, applying, patchError, applyError
draftDirty: deep compare draft vs snapshot.config
```

### Phases

| Phase | Deliverable |
| :---: | :--- |
| **P0** | License section (done) + deployment summary + restart banner |
| **P1** | Tracker domain + templates preview; Save/PATCH/Discard |
| **P2** | Edge toggles + ingress schema |
| **P3** | Persist section (apply + confirm) |
| **P4** | Regional defaults + stripe block (gated) |
| **P5** | E2E: `settings.spec.js`, `settings_platform_patch.spec.js`, `settings_apply.spec.js` |

---

## 11. Testing plan

### E2E (live `:8188`)

| Spec | Scenario |
| :--- | :--- |
| `settings.spec.js` | Load `/settings`, heading, license apply visible |
| `settings_license.spec.js` | Legacy `/settings/license` → `/settings` |
| `settings_platform_patch.spec.js` | PATCH tracking_domain; assert templates in GET |
| `settings_apply.spec.js` | POST apply; assert `written_path` in response |

### Unit

- `settings_field_labels.ts` maps all `RestartRequiredFields` keys.
- Draft dirty detection vs snapshot.
- Client strip/normalize tracking domain input.

### RBAC

- MB role: `GET /settings/platform` → 403 (audit in `rbac_smoke_route_audit_test.go`).
- Admin: full flow.

### Manual QA checklist

1. PATCH tracking domain → templates update without restart flag.
2. PATCH ingress_schema → `restart_required` includes field.
3. Apply → `written_path` shown; file exists on server.
4. License apply invalid JWT → 400 with message, form preserved.
5. `edge_xdp` + `compose_dev` → 400, user-friendly error.
6. Control plane stopped → `ErrorBlock`, not empty page.

---

## 12. Copy deck (operator-facing)

| Element | Copy |
| :--- | :--- |
| Page title | Settings |
| Save | Save changes |
| Discard | Discard changes |
| Apply button | Write install.compose.env |
| Apply confirm | Overwrite install.compose.env on the server? This does not restart services. |
| Restart banner | Some changes need a tracker or edge restart after writing to disk. |
| PG vs disk hint | Changes are saved to the control plane. Write to disk and restart services to apply on the tracker. |
| License apply | Apply license |
| Domains link | Manage tracking domains |

---

## 13. Current vs target snapshot

| Area | Current (`web/`) | Target |
| :--- | :--- | :--- |
| Route | `/settings` | unchanged |
| License | Minimal apply form | + status diagnostics from `/license/status` |
| Platform config | Removed (was bento/bootstrap/raw JSON) | Structured sections per this doc |
| Apply to disk | Removed | Restored with confirm + path display |
| Bootstrap form | Removed | Stays on `/activate` only |

---

## 14. Verification commands

```bash
# UI gates
cd web && npm run typecheck
bash scripts/ci/admin/web.sh

# Backend platform handlers
go test ./internal/controlplane/ -short -run TestPlatform -count=1
go test ./internal/platformadmin/ -short -count=1
go test ./pkg/platformconfig/ -short -run TestRestartRequiredFields -count=1

# Live smoke (stack up)
curl -sf http://127.0.0.1:8188/health
curl -sf -b cookie.jar http://127.0.0.1:8188/api/v1/settings/platform
```

---

*Last updated: implementation spec for Settings rebuild after platform UI removal. Backend contracts authoritative over this doc when they diverge; update this file when handlers or OpenAPI change.*

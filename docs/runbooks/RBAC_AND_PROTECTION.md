# Runbook: RBAC and svyazka protection

How operators control access to sensitive ad-tech assets (creatives, URLs, sources) within a self-hosted install.

**GAP:** GAP-PROD-11 — full acceptance criteria: [.cursor/GAP_SPECS.md](../../.cursor/GAP_SPECS.md#gap-prod-11--rbac--field-masking).

Related: [ARCHITECTURE.md](../ARCHITECTURE.md), [PROTECTION.md](../PROTECTION.md), [CUSTOMER_ENTITY_MODEL.md](./CUSTOMER_ENTITY_MODEL.md).

---

## Role constructor

Roles are permission sets mapped to `auth` users.

| Role | Scope | Permissions |
| :--- | :---: | :--- |
| Admin | Global | `*` |
| Manager | Customer | `campaigns:read`, `campaigns:write`, `finance:read` |
| Buyer | Team | `campaigns:read:masked`, `campaigns:pause` |
| Support | Global | `system:read`, `campaigns:read:masked`, `audit:read` |

### Permission strings

| Permission | Effect |
| :--- | :--- |
| `campaigns:read` | Full campaign DTO including `target_url`, creative |
| `campaigns:read:masked` | List/status/spend only; URL and creative scrubbed |
| `campaigns:write:masked` | Pause/budget; cannot read or change URL after create |
| `campaigns:pause` | Pause/resume without write access to creatives |

---

## Svyazka masking

A **svyazka** is source + geo + creative + landing. Masking runs at the **service boundary** before JSON serialization.

| Field | Masked value |
| :--- | :--- |
| `target_url` | `https://***.masked/path` |
| `creative_payload` | `{"type":"image","hash":"sha256:..."}` |
| `referrer_filter` | omitted |

Masked DTOs never carry plaintext in memory past the scrub step.

---

## Visibility scopes

| Scope | Boundary |
| :--- | :--- |
| Global | All `customers` |
| Customer-only | `user.customer_id` FK |
| Team-only | Future: `team_tag` on campaigns |

Self-serve API (`/api/v1/selfserve/*`) always enforces **customer-only** from API key session.

---

## Audit trail

| Field | Value |
| :--- | :--- |
| Table | `admin_audit_log` |
| Flag | `is_masked = true` for masked-role mutations |
| Required | actor_id, action, resource_id, payload hash |

---

## Configuration (GAP-PROD-11)

Target file: `deploy/operator/roles.yaml` — reload via management on startup or `POST /api/v1/ops/roles/reload`.

Until shipped: SQL seed or JSON API.

---

## SLA

| Metric | Target |
| :--- | :--- |
| AuthZ check p99 | < 5 ms (cached role snapshot) |
| Mask scrub per DTO | < 50 µs (cold path only) |

---

## SQL plans

Detail: [GAP_SPECS § SQL — GAP-PROD-11](../../.cursor/GAP_SPECS.md#sql--gap-prod-11).

| Query | Index target |
| :--- | :--- |
| User permissions join | `user_roles(user_id)` — nested loop < 1 ms |
| Audit insert | PK insert < 5 ms |

---

## Patterns

| Pattern | Use |
| :--- | :--- |
| Policy engine | Permission string → allow/mask map |
| Snapshot cache | Roles loaded on login; atomic refresh on reload |
| Deny by default | Missing permission → 403 |
| Boundary scrub | Service layer, not handler string replace |

---

## Tests

| Layer | Requirement |
| :--- | :--- |
| Unit | Table: masked user never receives `target_url` in JSON |
| Integration | PG auth schema; role reload → API behavior change |
| Fault | `fault_proof fault=rbac_mask_enforced` |
| Audit | Masked mutation creates `is_masked=true` row |

---

## FAQ

**Can Admin see everything?** Yes — operator admin is ultimate trust holder.

**Can Buyer change URL?** Only via `PENDING_APPROVAL` workflow (Manager/Admin approves).

**Is masking reversible via API?** No — scrub happens before marshal; plaintext not sent.

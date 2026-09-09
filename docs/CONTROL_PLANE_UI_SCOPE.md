# Control Plane UI scope

Operator decision (2026-03): admin SPA is **Control Plane only** — configure tenants, campaigns, team, platform settings, ops controls, and integration wiring. **Not** in scope for now: reports/BI, dashboards, RTB console, creative hubs, fraud/automation portals, buyer-facing polish, and directory UX chrome.

This document is the source of truth for **KEEP vs FREEZE** and a **backend parity audit** (`/api/v1` on `:8188` vs `web/`). Routes stay registered when frozen; freeze means **no active product work** and **sidebar deprioritization**, not deletion.

Related: `docs/DEVELOPMENT.md` (admin dev), `web/WEB.md` (T1 live tier), `.cursor/rules/frontend-patterns.mdc` (паттерны фронтенда), `.cursor/rules/ui.mdc`, `.cursor/rules/frontend-slop.mdc`, `.cursor/rules/frontend-primitives.mdc`.

---

## KEEP (active Control Plane)

| Surface | UI routes | Backend packages | Why CP |
| :--- | :--- | :--- | :--- |
| Auth / session | `/login`, `/activate`, `/invite/accept` | `internal/control/http`, `platformadmin` session | Bootstrap |
| Customers | `/customers`, `/customers/:id` | `platformadmin`, `billingadmin` (tabs) | Tenant config |
| Campaigns list | `/campaigns` | `internal/campaign` list/facets/metrics/bulk/clone | Operate campaigns |
| Campaign editor | `/campaigns/:id/edit` | `campaign` editor/wizard/publish/fraud/integration | Core CP workflow |
| Team | `/team` | `platformadmin` team | RBAC, invites, budget approvals |
| Settings | `/settings`, `/settings/license` | `platformadmin`, `licensingadmin` | Platform + license |
| Customer billing tabs | customer detail: Balance, Ledger, Statement, Wallet, Tax | `billingadmin` per-customer | Money gates clone/spend |
| Billing ops (minimal) | `/billing/invoices/:id`, void/retry from customer context | `billingadmin` invoice detail | Operator correction |
| Ops | `/ops/*` (home, dlq, blacklist, outbox, shards, domains, ml-model, recon, consent, rum, metrics) | `opsadmin`, `doctor` | Stack control |
| Audit | `/audit` | `opsadmin` audit | Read-only trail |
| Integrations | `/integrations/*` (cost-sync, postbacks, schemas, platform-campaigns, affiliate-presets) | `billingadmin`, `campaign`, `platformadmin` | External wiring |
| Export Hub | `/exports` (legacy: `/reports/jobs`, `/billing/exports`) | `internal/reports`, `reportjob`, `billingadmin` exports | Async BI export; no in-browser report tables in CP |

### KEEP adjuncts (not full frozen hubs)

These use APIs from frozen **sections** but are required for the campaign CP path:

| Need | UI location | APIs | Note |
| :--- | :--- | :--- | :--- |
| Flow pick / routing | campaign editor (`use_campaign_editor_load`, advanced routing) | `GET /api/v1/flows/:id`, list flows | No `/creative` hub required |
| Campaign fraud policy | campaign editor fraud panel | `GET/PATCH /api/v1/campaigns/:id/fraud`, `POST .../fraud/preview` | Not `/fraud` hub |
| Publish / validate | campaign editor actions | `POST .../validate`, `publish`, `publish-check`, `smoke` | |
| Clone | list + editor | `POST .../clone`, `bulk-clone`, `clone-preview` | Needs customer balance (seed or ledger) |
| Integration panel | campaign editor | `GET .../integration-panel`, `integration-health`, schema apply | |
| Postbacks / cost-sync | integrations section + editor deps | `/api/v1/postbacks/*`, `/api/v1/cost-sync/*` | |

---

## FREEZE (no active work)

| Surface | UI routes | Backend | Freeze reason |
| :--- | :--- | :--- | :--- |
| Reports hub + runners | `/reports`, `/reports/*`, `/report-schedules`, `/views` | `internal/reports`, `reportjob` (~40+ CH reports) | BI, not CP |
| Dashboards | `/dashboards/:role`, `/dashboards/campaign/:id` | `internal/dashboardadmin` | Charts/KPIs |
| RTB | `/rtb/*` | `internal/rtbadmin` | Auction ops, license-gated |
| Creative hub | `/creative`, `/flows`, `/landers`, `/offers`, `/brands`, `/supply`, `/domains` | `internal/flow`, `brand`, `supply`, `platformadmin/domains` | Content studio |
| Fraud hub | `/fraud/*` | `internal/fraudadmin` | Policy tools beyond per-campaign panel |
| Automation | `/automation/*`, traffic-optimizer, smart-alerts, margin-guard | respective packages | Optimization product |
| Portals | `/portals`, `/selfserve`, `/publisher/*`, `/telegram/*`, `/forecast/campaign` | `campaign/selfserve`, `dashboardadmin`, `telegram` | Buyer/publisher portals |
| Billing directory | `/billing` (invoice list hub), `/billing/exports` | `billingadmin` summary/list/exports | Operator BI; keep per-customer + invoice detail |
| Directory polish | column resize, export UX, overview sheet depth, campaign dashboard links from list | n/a | UX chrome |
| Disputes / support (low) | `/disputes`, `/support/feedback` | `billingadmin`, `platformadmin` | Secondary; no nav priority |

**Remove from default nav (proposal, not yet coded):** Dashboard, Billing (hub), Reports, RTB, Fraud, Creative, Automation, Portals. **Core nav:** Customers, Campaigns, Team, Settings. **Operations:** Ops, Audit, Integrations, **Exports** (`/exports`).

---

## Export Hub

**Purpose:** BI and report surfaces do not render heavy browser tables in Control Plane mode. The operator configures filters and runs async export jobs (or sync CSV when the result set is small). Data stays on the server / ClickHouse; the SPA submits jobs and polls for download URLs.

**Canonical route:** `/exports` in **Operations** nav -- unified catalog, job form, and job status/history. Legacy deep-link aliases remain registered: `/reports/jobs`, `/billing/exports` (redirect or query-preserving alias to `/exports` in Phase C).

### KEEP tables vs export-only

| Pattern | Routes / surfaces | CP behavior |
| :--- | :--- | :--- |
| **KEEP tables** | Customers, campaigns list, ops, audit list, integrations | Operational CRUD; **Export** is an adjunct toolbar action (CSV where handler supports sync export) |
| **Export-only** | Customer-scoped reports (`CustomerReportPage` keys), telegram / ml / evidence / campaign-stats runners, fraud reasons table view | No `DirectoryTable` body in CP mode; `ExportOnlyReportStub` + link to Export Hub with pre-filled `report_key` |

Export-only catalog keys today include typed sets in `web/src/lib/report_paths.ts` (`TYPED_CUSTOMER_REPORT_KEYS`, `TYPED_TELEGRAM_REPORT_KEYS`, `TYPED_ML_REPORT_KEYS`, `TYPED_CAMPAIGN_STATS_REPORT_KEYS`, `TYPED_EVIDENCE_PACK_REPORT_KEYS`, `TYPED_EXPORT_ONLY_REPORT_KEYS`). Reference implementation: `web/src/pages/fraud_evidence_pack_bulk_page.tsx` (stub + `buildReportJobsHref`).

### Job flow

```
customer_id + from + to + report_key + format
  -> POST report job (or billing export when ledger-scoped)
  -> poll GET job status
  -> download via job download URL (or billing export download)
```

Reuse existing APIs:

| Job kind | API | UI today |
| :--- | :--- | :--- |
| Report catalog jobs | `POST/GET /api/v1/reports/jobs`, download on completed job | `/reports/jobs` (`use_report_jobs_page_workspace.ts`) |
| Billing ledger exports | `POST/GET /api/v1/billing/exports`, download | `/billing/exports` (`billing_api.ts`) |

Export Hub form maps one UX to both backends by `report_key` / export kind (catalog entry metadata).

### Catalog sources

| Source | Entries |
| :--- | :--- |
| `GET /api/v1/reports/catalog` | Dynamic report keys, titles, async-capable flags |
| Static CP entries (Phase C) | Audit CSV (`GET /api/v1/audit/export`), campaign list CSV export, billing ledger export |

Pre-fill deep links via `buildReportJobsHref` in `web/src/lib/report_paths.ts` (`report_key`, `customer_id`, `from`, `to`, `format`, `job_id`). Phase C adds `buildExportHubHref` targeting `/exports?...` with the same query shape.

### Phase rollout

| Phase | Deliverable | Status |
| :--- | :--- | :--- |
| **A** | This spec + cross-ref in `frontend-primitives.mdc` | **done** (this section) |
| **B** | Primitives catalog (`ExportOnlyReportStub`, date range / searchable select shell roles) | pending |
| **C** | MVP `/exports` route, Operations nav, legacy alias redirects | pending |
| **D** | Stub report pages when `CONTROL_PLANE_NAV_ENABLED`; frozen report routes show export stub instead of table | pending |

Cross-ref: `web/src/lib/report_paths.ts` (`buildReportJobsHref`), `web/src/pages/fraud_evidence_pack_bulk_page.tsx`, `.cursor/rules/frontend-primitives.mdc` (**Export Hub UI**).

---

## Backend vs frontend parity

Legend: **match** = UI calls live handler; **partial** = conditional (CH, license, nil dep, 501); **backend-only** = API exists, no admin page; **ui-only** = page exists, backend edge case.

### Auth, session, platform

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `POST /api/v1/auth/{login,logout,refresh}` | live | login flow | match |
| `GET /api/v1/auth/me`, `GET /api/v1/session/bootstrap` | live | app shell | match |
| `GET /api/v1/meta` | live | `MetaProvider` | match |
| `GET/PATCH /api/v1/settings/platform` | live | `/settings` | match |
| `GET /api/v1/license/status`, `POST .../apply` | live | `/settings/license` | match |
| `POST /api/v1/public/{activate,invite/accept}` | live | onboarding pages | match |
| `POST /api/v1/ops/roles/reload` | live | ops home action | match |

### Customers

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/customers`, `GET/PATCH .../{id}` | live | directory + profile tab | match |
| `GET .../balance`, `ledger`, `balance/export` | live | customer tabs | match |
| `GET .../billing/statement`, `forecast` | live / partial (CH) | customer tabs | partial if CH down |
| `GET .../wallet`, `payments` | live | wallet tab (gated on `payment_enabled`) | match |
| `GET/PUT .../tax-profile` | live | tax tab | match |
| `GET /api/v1/disputes` | live | `/disputes` page | backend-only priority (page exists, freeze) |

### Campaigns

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/campaigns`, `list-facets`, metrics batch | live | `/campaigns` | match |
| `POST bulk`, `bulk-clone`, `clone` | live | list actions | match (balance prerequisite) |
| `GET/PATCH /api/v1/campaigns/{id}` | live | editor | match |
| Editor bundle: `editor`, `validate`, `publish*`, `diff`, `macro-preview` | live | editor | match |
| `GET/PATCH .../fraud`, `POST .../fraud/preview` | live | fraud panel | match |
| `GET .../integration-panel`, `integration-health` | live | integration panel | match |
| Wizard `GET/POST /api/v1/campaigns/wizard/session` | live | wizard | match |
| Migrate/import ` /api/v1/campaigns/migrate/*`, `/import/*` | live | editor import flows | match |
| `GET /api/v1/campaigns/{id}/stats` | live (CH) | no dedicated page | backend-only (use reports freeze) |
| `GET /api/v1/dashboards/campaign/{id}` | live / 501 if reader nil | `/dashboards/campaign/:id` | match but **FREEZE** |
| `POST /api/v1/selfserve/campaigns` (create from list) | live | campaigns list create | match (selfserve API used from admin) |

### Team

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/team/overview`, `members`, `budget-approvals` | live | `/team` | match |
| `POST members`, `PATCH .../{id}`, approve/deny | live | team actions | match |

### Billing

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Per-customer ledger/statement/wallet | live | customer tabs | **KEEP** |
| `GET /api/v1/billing/invoices/{id}`, void, deliveries | live | `/billing/invoices/:id` | **KEEP** |
| `GET /api/v1/billing/summary`, `invoices` list, `invariant`, exports | live | `/billing`, `/billing/exports` | match but **FREEZE** hub |
| `POST /api/v1/billing/crypto/webhook` | live | no UI (ingress) | backend-only |
| `GET /api/v1/cost-sync/*` | live | integrations cost-sync | **KEEP** |

### Ops

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `GET /api/v1/ops/home`, dlq, outbox, shards, blacklist | live | ops subpages | match |
| `GET /api/v1/audit`, `audit/export` | live | `/audit` | match |
| `GET /api/v1/ops/domains/*` | live | `/ops/domains` | match |
| `GET /api/v1/recon/runs` | live | `/ops/recon` | match |
| `GET/POST /api/v1/ops/rum` | live | `/ops/rum` (GET only) | partial (no ingest UI; POST is telemetry ingress) |
| `GET /api/v1/ops/dashboard/{metrics,stream}` | live | `/ops/metrics` | match |
| `PATCH /api/v1/ops/fraud/presets/{name}` | live | `/fraud/presets` only | backend-only in KEEP path (use fraud hub or add later) |

### Integrations

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| `/api/v1/postbacks/*` | live | integrations postbacks | match |
| `/api/v1/integration/schemas`, templates, affiliate-presets | live | integrations pages | match |
| `/api/v1/platform-campaigns/*` | live | platform-campaigns page | match |
| `/api/v1/domains/*` (admin) | live / SSL 501 if script missing | `/domains` (creative) | match but **FREEZE** hub; ops has rotation subset |

### Fraud (hub vs campaign)

| API | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Campaign `.../fraud` | live | editor panel | **KEEP** |
| `GET /api/v1/fraud/labels`, overrides, presets, corpus, decisions | live | `/fraud/*` | match but **FREEZE** |
| `GET /api/v1/fraud/probe-clusters/{id}` | live | not in UI (openapi + docs only) | **backend-only** |
| `GET /api/v1/fraud/crowd-waves/{campaign_id}` | live | docs reference only | **backend-only** |
| `POST /api/v1/fraud/wasm-attest/dry-run` | partial (501 disabled) | no page | backend-only |

### Reports, dashboards, RTB (all FREEZE)

| Domain | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Reports catalog + ~40 report keys | live / CH partial | `/reports/*` | match (freeze) |
| Report jobs, schedules, saved views | live | jobs + portals pages | match (freeze) |
| Dashboards buyer/adops/cfo/... | live | `/dashboards/:role` | match (freeze) |
| RTB deals, shadow, floors, validate | live / license | `/rtb/*` | match (freeze) |

### Creative, automation, portals (FREEZE)

| Domain | Backend | UI | Parity |
| :--- | :--- | :--- | :--- |
| Flows, landers, offers, brands, supply | live | `/creative/*` | match (freeze); editor uses flows API only |
| Automation, traffic-optimizer, smart-alerts, margin-guard | live | `/automation/*` | match (freeze) |
| Selfserve, publisher, telegram | live | `/portals/*` | match (freeze) |

---

## Gap summary

### Backend ahead of UI (KEEP backlog candidates)

| API | Suggested CP surface |
| :--- | :--- |
| `GET /api/v1/fraud/probe-clusters/{id}` | Ops or campaign debug drawer (not fraud hub) |
| `GET /api/v1/fraud/crowd-waves/{campaign_id}` | Campaign editor ops strip |
| `PATCH /api/v1/ops/fraud/presets/{name}` | Settings or ops, not full presets browser |

### UI ahead of or wider than CP scope

| UI | Issue |
| :--- | :--- |
| Campaign list → `/dashboards/campaign/:id` | Links into FREEZE dashboards |
| Campaign list metrics columns | Depends on CH batch metrics (degraded without CH) |
| `/campaigns` create via selfserve API | Works but blurs admin vs portal boundary |
| Buyer dashboard in Core nav | Should move to FREEZE / remove from nav |

### Conditional / degraded (not stub catalog)

| Condition | Symptom | Correct UI behavior |
| :--- | :--- | :--- |
| ClickHouse down | reports + some forecasts/metrics 503 or `stale` | `ErrorBlock`; no fake rows |
| `CampaignDashboard` reader nil | `GET /dashboards/campaign/{id}` 501 | StubBanner on frozen page |
| Domain SSL script missing | domains SSL 501 | StubBanner (integrations/creative) |
| RTB not in license | 403 on `/api/v1/rtb/*` | StubBanner on frozen RTB |
| Stripe disabled | `payment_enabled: false` | Hide wallet/payments tab |
| Insufficient customer balance | clone 400 | Show server error; seed `db seed-clone-balance` in dev |

`stubRouteCatalog` in `internal/controlplane/admin_static.go` is **empty** — no global 501 stubs. Failures are real handler responses.

---

## Inventory scale

| Layer | Count | Source |
| :--- | :--- | :--- |
| Catalogued `/api/v1` routes | ~362 | `internal/controlplane/routecatalog/catalog.go` |
| Admin SPA top-level nav items | 14 | `web/src/lib/nav_config.ts` |
| Report runner routes | ~40 | `web/src/app_routes.tsx` |
| Ops subroutes | 13 | `web/src/domains/ops/ops_nav.tsx` |

OpenAPI parity: `api/openapi/` + `internal/openapi/documented_routes.go`. Regenerate TS: `make openapi-types`.

---

## Verification (scope audit)

```bash
# Control healthy (T1)
curl -sf http://127.0.0.1:8188/health

# Session bootstrap (auth)
curl -sf -c /tmp/aed.cookie -b /tmp/aed.cookie \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@test.local","password":"Password123!"}' \
  http://127.0.0.1:8188/api/v1/auth/login
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/session/bootstrap | head -c 200

# KEEP samples
curl -sf -b /tmp/aed.cookie 'http://127.0.0.1:8188/api/v1/customers?limit=1'
curl -sf -b /tmp/aed.cookie 'http://127.0.0.1:8188/api/v1/campaigns?limit=1'
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/team/overview
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/settings/platform
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/ops/home

# FREEZE samples (still live on backend)
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/reports/catalog
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/dashboards/buyer
curl -sf -b /tmp/aed.cookie http://127.0.0.1:8188/api/v1/rtb/deals
```

Admin static gates (when touching `web/`): `npm run typecheck`, `bash scripts/ci/admin/web.sh`.

---

## Implementation status

| Item | Status |
| :--- | :--- |
| This scope doc | **done** |
| Export Hub spec (Phase A) | **done** (this doc) |
| Nav trim (remove FREEZE from sidebar) | **done** (`nav_config.ts`, `tracker_nav.ts`) |
| `ControlPlaneFrozenGate` on outlet | **done** |
| Unstyled HTML frames (`control_plane_*_frame.tsx`) | **done** |
| Remove campaign list → dashboard links | **done** |
| Link from `DEVELOPMENT.md` | done |
| Export Hub route `/exports` (Phase C) | pending |
| Export-only report stubs (Phase D) | pending |

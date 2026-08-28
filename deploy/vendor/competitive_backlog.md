# Competitive parity backlog

Open gaps vs Keitaro, Binom, and BeMob-class trackers. Shipped parity work removed 2026-08-27; see git history for done gates.

**Canonical:** `docs/INTEGRATIONS.md`, `deploy/vendor/ANTIFRAUD.md`, [marketing tracker parity](./marketing_tracker_parity_backlog.md), [admin UI rebuild](./admin_ui_redesign_backlog.md).

**Not in scope:** antifraud ML signals (`ANTIFRAUD.md`); OpenAPI contract workflow (`docs/DEVELOPMENT.md`); compliance offensive tooling (`compliance.mdc`).

Cross-reference slugs in PR descriptions. Do not mark a slug closed until done gates pass in the same commit as code.

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`, `hot-path.mdc`)
- [ ] Verification commands pasted in PR with package path (`quality.mdc`)
- [ ] Holdout or fault test when behavior is non-obvious (`testing.mdc`)
- [ ] Doc claims match code; no microbench cited as prod SLA (`anti-slop.mdc`)
- [ ] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `deploy/vendor/` prose (`naming.mdc`)

---

## Open slugs

No open competitive-parity slugs. Admin migrate wizard shipped at `/campaigns/migrate`. Deferred Cost Sync networks: table below.

---

## `migration_live_tracker_pull`

**Priority:** P1

**Gap:** Migration today is file upload or paste JSON only. Keitaro/Binom operators expect HTTP pull from admin API credentials.

**Surface:** cold worker or controlplane job; reuse existing preview/import TX path.

**Target:**

- Operator supplies base URL + API token; job fetches export and runs preview/import.
- RBAC `campaigns:write`; audit row per job; no secrets in logs.
- File upload remains default when pull credentials absent.

### Done gates

- [x] Pull failure does not partial-import (holdout or integration test)
- [x] `ReadLimitedBody` / timeout on outbound HTTP; cold path only
- [x] `deploy/vendor/migration/README.md` documents pull vs file upload

---

## `migration_streams_flows_import`

**Priority:** P1

**Gap:** v1 importer maps flat campaign rows only. Keitaro streams, offers, and rotation weights are dropped.

**Surface:** `internal/migrationsource/` adapters + import TX.

**Target:**

- Keitaro stream JSON fixture imports path weights and offer refs into flow snapshot.
- Warnings for unsupported constructs; no silent drop.
- Flat campaign import path unchanged.

### Done gates

- [x] Integration test: preview + import round-trip for multi-path flow fixture
- [x] Holdout: unmapped stream node produces warning slug, not silent omit
- [x] No `internal/ingestion` import from adapters (`boundaries.mdc`)

---

## `admin_campaigns_migrate`

**Priority:** P4 (UI)

**Gap:** `POST /api/v1/campaigns/migrate/preview|import` and `GET .../sources` are live; no admin wizard (`web/` removed 2026-08-27).

**Backlog:** [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md) slug `admin_campaigns_migrate`.

**Spec:** create `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE.md` from `MILESTONE_TEMPLATE.md` before implementation.

**Route:** `/campaigns/migrate` (Upload -> Preview -> Confirm import).

### Done gates

- [ ] Milestone sections 1, 4, 5, 6, 7 filled (`ui.mdc`)
- [ ] Preview warnings match server; `apiConfirmed` on import
- [ ] No `live: true` route without preview API (`anti-slop.mdc`)

---

## Deferred (no public advertiser spend API)

Reopen Cost Sync adapter only when a network publishes stable **advertiser** spend API with documented auth:

| Network | Blocker |
| :--- | :--- |
| `zeropark` | Campaign API only; spend via panel |
| `rollerads` | No public API |
| `pushground` | Private API docs |
| `clickadilla` | Voluum token integration only |
| `ezmob` | Account-gated API docs |

See `docs/INTEGRATIONS.md` Cost Sync section.

---

## Antifraud extensions (not competitive parity)

| Slug | Surface |
| :--- | :--- |
| `mobile_biometrics_pipeline` | Client gyro/touch ingest + `ivt-detector` cold rule (**done**) |
| `fraud_evidence_pack_export` | `GET /api/v1/reports/fraud-evidence-pack` signed CPA bundle (**done**) |

Residential proxy and L7 desync coverage: [residential_proxy_detection_backlog.md](./residential_proxy_detection_backlog.md) (**CLOSED**).

Canonical antifraud behavior: `deploy/vendor/ANTIFRAUD.md`.

---

## Explicit non-goals

| Competitor claim | Our stance |
| :--- | :--- |
| Guaranteed no ban on FB gambling | Not verifiable; do not document |
| Unlimited RPS on cheap license | `max_rps` enforced in JWT |
| ML on hot path | Batch sidecars only (`ANTIFRAUD.md`) |
| XDP stops residential fraud | L3/L4 only; docs stay honest |
| Pay-per-click cloud packages | On-prem unlimited events; price by RPS/hosts |

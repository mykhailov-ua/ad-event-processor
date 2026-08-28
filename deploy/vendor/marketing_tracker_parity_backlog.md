# Marketing tracker parity backlog

Gap between ad-event-processor as a **self-hosted engine** and Keitaro / Binom / Voluum / RedTrack-class **buyer-ready SaaS**. Shipped surfaces stay in git history; this file tracks what remains.

**Status:** SHIPPED (2026-08-28). P0-P4 API and admin UI parity complete except deferred items below.

**Policy:** Server contracts and cold-path workers first. Admin UI (`web/`) last — thin client over APIs already proven in section 7 of each milestone.

**Canonical:**

| Doc | Role |
| :--- | :--- |
| [cold_path_admin_api_backlog.md](./cold_path_admin_api_backlog.md) | Campaign editor, reports, RBAC slugs (detail lives here) |
| [competitive_backlog.md](./competitive_backlog.md) | Keitaro/Binom migration pull and streams import |
| [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md) | UI milestones after cold contracts |
| `docs/INTEGRATIONS.md` | Shipped integrations truth table |

**Not in scope:** antifraud signal depth (`ANTIFRAUD.md`); hot-path ingest (`internal/ingestion`); ML training (`cmd/fraud-scorer`).

Cross-reference slugs in PR descriptions. Do not mark a slug closed until done gates pass in the same commit as code.

---

## Problem (one paragraph)

Buyers expect a cockpit: create campaign in 30 seconds, Automizer saves budget overnight, attribution explains long funnels. Today the engine is strong on ingest, fraud, and cost sync, but **operators without a developer** cannot safely run campaigns: `web/` is removed, flows are JSON-heavy, errors surface on prod traffic, and multi-touch marketing attribution is absent. Closing this backlog means **API-first buyer workflows** with validation gates and automation presets — not a React rewrite ahead of server contracts.

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build -o /dev/null ./cmd/control/`)
- [ ] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`)
- [ ] OpenAPI fragment + handler registered together (`openapi_gate.sh`)
- [ ] Verification commands pasted in PR with package path (`quality.mdc`)
- [ ] Holdout or integration test when behavior is non-obvious (`testing.mdc`)
- [ ] Doc claims match code; no microbench cited as prod SLA (`anti-slop.mdc`)
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `deploy/vendor/` prose (`naming.mdc`)

---

## Priority tiers

| Tier | Meaning | Ship before |
| :--- | :--- | :--- |
| **P0** | Production safety — invalid config reaches live traffic or budget leaks | Anything else in this backlog |
| **P1** | Buyer can operate via API without hand-built JSON; migration off SaaS | P2+ UI |
| **P2** | Automizer comfort, bulk ops, ops visibility | P4 UI |
| **P3** | Async ergonomics, TCO docs, nice-to-have scale | P4 UI |
| **P4** | Admin UI (`web/`) — thin client over P0–P2 APIs | — |
| **Deferred** | Explicit non-goal for marketing parity v1 | Reopen on new SKU only |

Within a tier, ship top-to-bottom in the table below.

### P0 — production safety

| Order | Slug | Owner | Status | Why P0 |
| :---: | :--- | :--- | :--- | :--- |
| 1 | `campaign_flow_validation` | cold_path | shipped | Cyclic redirects and dead paths hit prod without gate |
| 2 | `campaign_macro_preview` | cold_path | shipped | Broken tokens invisible until traffic fails |
| 3 | `campaign_publish_gate` | this file | shipped | Hard block on `active` without validation bundle |

### P1 — buyer workflow (API)

| Order | Slug | Owner | Status | Why P1 |
| :---: | :--- | :--- | :--- | :--- |
| 4 | `campaign_save_conflict` | cold_path | shipped | Concurrent edits overwrite flow without warning |
| 5 | `campaign_post_deploy_smoke` | this file | shipped | Misconfig discovered from CH lag, not at publish |
| 6 | `campaign_clone_wizard` | cold_path | shipped | Duplicate funnel without retyping JSON |
| 7 | `campaign_onboarding_wizard_api` | this file | shipped | First campaign without dozens of manual API calls |
| 8 | `migration_streams_flows_import` | competitive | shipped | Rotation weights lost — migration is useless |
| 9 | `migration_live_tracker_pull` | competitive | shipped | Keitaro/Binom pull vs manual JSON export |
| 10 | `automation_rule_presets` | this file | shipped | Automizer setup without raw rule JSON |
| 11 | `automation_eval_interval` | this file | shipped | 15 min default too slow for placement burn |

### P2 — comfort and ops

| Order | Slug | Owner | Status | Why P2 |
| :---: | :--- | :--- | :--- | :--- |
| 12 | `campaign_bulk_mutations` | cold_path | shipped | Mass pause/budget across list |
| 13 | `campaign_onboarding_templates` | this file | shipped | Prefill wizard; not required for first ship |
| 14 | `campaign_placement_block_suggestions` | cold_path | shipped | Report-driven hints; manual block still works |
| 15 | `automation_platform_pause_networks` | this file | shipped | Meta/Google pause shipped; TikTok/Microsoft extra |
| 16 | `ops_stack_health_snapshot` | this file | shipped | Single health JSON; Grafana still works |

### P3 — scale and TCO

| Order | Slug | Owner | Why P3 |
| :---: | :--- | :--- | :--- |
| 17 | `campaign_import_validation_job` | cold_path | Large imports rare; sync validate enough for v1 |
| 18 | `compose_minimal_stack_profile` | this file | Docs/compose profile; does not unblock buyers |

### P4 — admin UI (last)

| Order | Slug | Owner | Requires |
| :---: | :--- | :--- | :--- |
| 19 | `admin_shell` | admin_ui | `session_nav_meta` (shipped) |
| 20 | `admin_directory_pattern` (campaigns) | admin_ui | `campaign_list_server_filters` (shipped) |
| 21 | `admin_detail_pattern` (campaigns) | admin_ui | P0 publish gate + P1 editor APIs |
| 22 | `admin_campaigns_migrate` | admin_ui | P1 migration APIs |
| 23 | `admin_integrations_hub` (automation) | admin_ui | P1 automation presets |

### Deferred

| Slug | Reopen when |
| :--- | :--- |
| `multi_touch_attribution_models` | E-commerce / retargeting SKU with CH identity graph spec |

---

## Summary table

| Slug | Tier | Phase | Owner doc | Status |
| :--- | :--- | :--- | :--- | :--- |
| `campaign_flow_validation` | P0 | 1 | cold_path | shipped |
| `campaign_macro_preview` | P0 | 1 | cold_path | shipped |
| `campaign_publish_gate` | P0 | 1 | this file | shipped |
| `campaign_save_conflict` | P1 | 1 | cold_path | shipped |
| `campaign_post_deploy_smoke` | P1 | 1 | this file | shipped |
| `campaign_clone_wizard` | P1 | 1 | cold_path | shipped |
| `campaign_onboarding_wizard_api` | P1 | 2 | this file | shipped |
| `migration_streams_flows_import` | P1 | 2 | competitive | shipped |
| `migration_live_tracker_pull` | P1 | 2 | competitive | shipped |
| `automation_rule_presets` | P1 | 3 | this file | shipped |
| `automation_eval_interval` | P1 | 3 | this file | shipped |
| `campaign_bulk_mutations` | P2 | 1 | cold_path | shipped |
| `campaign_onboarding_templates` | P2 | 2 | this file | shipped |
| `campaign_placement_block_suggestions` | P2 | 1 | cold_path | shipped |
| `automation_platform_pause_networks` | P2 | 3 | this file | shipped |
| `ops_stack_health_snapshot` | P2 | 4 | this file | shipped |
| `campaign_import_validation_job` | P3 | 1 | cold_path | shipped |
| `compose_minimal_stack_profile` | P3 | 4 | this file | shipped |
| `admin_shell` | P4 | 5 | admin_ui | shipped |
| `admin_directory_pattern` (campaigns) | P4 | 5 | admin_ui | shipped |
| `admin_detail_pattern` (campaigns) | P4 | 5 | admin_ui | shipped |
| `admin_campaigns_migrate` | P4 | 5 | competitive + admin_ui | shipped |
| `admin_integrations_hub` (automation) | P4 | 5 | admin_ui | shipped |
| `multi_touch_attribution_models` | deferred | — | this file | deferred |

**Shipped (do not reopen):** `campaign_publish_gate`, `campaign_editor_shell`, `campaign_patch_dry_run`, `campaign_editor_integration_panel`, `campaign_list_server_filters`, `campaign_flow_validation`, `campaign_macro_preview`, `campaign_save_conflict`, `campaign_clone_wizard`, `campaign_bulk_mutations`, `campaign_placement_block_suggestions`, `campaign_import_validation_job`, `compose_minimal_stack_profile`, automation rules CRUD + dry-run (`docs/INTEGRATIONS.md`), cost sync sub-daily for Meta/Google/TikTok.

---

## Suggested ship order

Follow **priority tier** order (P0 → P4). Phases group related work but do not override tier.

```
P0   campaign_publish_gate                    ← shipped
P1   campaign_post_deploy_smoke
     campaign_onboarding_wizard_api
     migration_streams_flows_import → migration_live_tracker_pull
     automation_rule_presets → automation_eval_interval
P2   campaign_onboarding_templates
     automation_platform_pause_networks → ops_stack_health_snapshot
P3   campaign_import_validation_job → compose_minimal_stack_profile
P4   admin_shell → admin_directory_pattern → admin_detail_pattern
     admin_campaigns_migrate → admin_integrations_hub
```

P4 starts only after P0–P1 APIs have handler tests and OpenAPI entries cited in milestone section 7.

---

## Phase 1 — Campaign safety and editor API

### Slugs owned by [cold_path_admin_api_backlog.md](./cold_path_admin_api_backlog.md)

Implement per `COLD_<SLUG>_MILESTONE.md` from `MILESTONE_TEMPLATE.md`. This file does not duplicate their done gates.

| Slug | Buyer pain addressed |
| :--- | :--- |
| `campaign_flow_validation` | Cyclic redirects, dead offers, weight sum errors before traffic hits prod |
| `campaign_macro_preview` | Broken `{sub1}` / `{click_id}` tokens visible before save |
| `campaign_save_conflict` | Two operators overwriting each other's flow edits |
| `campaign_clone_wizard` | Duplicate campaign + flow in one API call |
| `campaign_bulk_mutations` | Pause or cap budget across placement/campaign list |
| `campaign_placement_block_suggestions` | Server suggests blocks from report rollups |
| `campaign_import_validation_job` | Async validate large import before commit |

**Blocks UI:** `ADMIN_DETAIL_MILESTONE_CAMPAIGNS.md` (`admin_ui_redesign_backlog.md`).

---

### `campaign_publish_gate`

**Priority:** P0

**Gap:** Campaign can go live with invalid flow; traffic falls through to fallback or 404/500. Dry-run `POST /api/v1/campaigns/{id}/validate` exists but is optional — no hard gate on status transition to `active`.

**Surface:** `internal/controlplane/campaign_validate.go`; `PATCH /api/v1/campaigns/{id}` and dedicated `POST /api/v1/campaigns/{id}/publish`; outbox apply only after gate passes.

**Target:**

- Transition to `active` (or equivalent live flag) runs full validation bundle: flow graph, macro tokens, integration schema refs, budget invariants.
- Response `422` with `field_errors[]` and `warning_slugs[]`; no Redis/catalog publish on failure.
- `force=true` query param requires `campaigns:admin` permission and audit row (escape hatch for ops).
- Holdout: invalid flow cannot reach tracker snapshot without `force`.

**Depends on:** `campaign_flow_validation`, `campaign_macro_preview` (or inline same checks).

### Done gates

- [ ] Holdout: patch to `active` with cyclic redirect returns 422; tracker catalog unchanged
- [ ] Handler test: `force=true` writes audit event; buyer role denied
- [ ] OpenAPI documents `422` body and `force` param

---

### `campaign_post_deploy_smoke`

**Priority:** P1

**Gap:** Operator learns about misconfiguration from ClickHouse lag, not at publish time. SaaS trackers synthetic-click the tracking URL after save.

**Surface:** `internal/controlplane/campaign_smoke_job.go`; `POST /api/v1/campaigns/{id}/smoke`; optional auto-enqueue from `campaign_publish_gate` success.

**Target:**

- Job issues internal HTTP `GET /click` (or configured smoke URL) with test `click_id`; follows redirects up to N hops; records terminal status, final host, macro expansion sample.
- Does not debit production budget: `smoke=1` query token or dedicated smoke campaign flag ignored by billing Lua.
- Result DTO: `passed`, `redirect_chain[]`, `failure_reason`, `checked_at`.
- RBAC `campaigns:write`; timeout bounded (`cold-path.mdc`).

### Done gates

- [x] Integration test: broken lander URL returns `passed=false` with reason slug (`TestFollowCampaignSmokeRedirects_brokenLander_returnsNonSuccess`)
- [x] Holdout: smoke click does not increment campaign budget spend (`TestFilterEngine_smokeSkipsUnifiedBudgetDebit_holdout`)
- [x] No hot-path PG/CH from tracker on smoke request (`smoke=1` skips stream XADD via `fcap:ignored`)

---

## Phase 2 — Migration and onboarding

### Slugs owned by [competitive_backlog.md](./competitive_backlog.md)

| Slug | Buyer pain addressed |
| :--- | :--- |
| `migration_live_tracker_pull` | No manual JSON export from Keitaro/Binom |
| `migration_streams_flows_import` | Rotation weights and offer paths lost on import |

---

### `campaign_onboarding_wizard_api`

**Priority:** P1

**Gap:** First campaign requires assembling dozens of fields across campaigns, flows, templates, and integrations. Documented UI route `/campaigns/wizard` has no server step machine while `web/` is removed.

**Surface:** `GET/POST /api/v1/campaigns/wizard/session`; `internal/controlplane/campaign_wizard_handlers.go`.

**Target:**

- Stateful session (PG or Redis cold): steps `traffic_source` → `integration_template` → `flow_skeleton` → `budget` → `review`.
- Each step validates partial DTO; `review` returns aggregated preview + warnings (reuses migrate preview where possible).
- Final `commit` runs import TX + optional `campaign_publish_gate`.
- Session TTL 24h; RBAC `campaigns:write`.

**Depends on:** `campaign_publish_gate`, `migration_streams_flows_import` (optional for step `flow_skeleton`).

### Done gates

- [x] Handler test: incomplete session cannot `commit` (`TestCampaignWizard_commitIncompleteSession_holdout`)
- [x] Integration test: commit creates campaign + flow + applies integration template (`TestCampaignWizard_commitCreatesCampaignFlow_holdout`)
- [x] OpenAPI documents step payloads; no secrets in session GET (`TestCampaignWizardSessionGET_omitsSecrets`)

---

### `campaign_onboarding_templates`

**Priority:** P2

**Gap:** Every buyer reinvents JSON for pop/push/native/Telegram funnels.

**Surface:** `GET /api/v1/campaigns/onboarding-templates`; bundled YAML under `deploy/schemas/onboarding/` or extend `deploy/schemas/traffic_*.v1.yaml` catalog.

**Target:**

- Templates: `key`, `title`, `description`, `traffic_family`, `default_flow`, `integration_schema_refs[]`, `sample_macros`.
- `POST /api/v1/campaigns/wizard/session` accepts `template_key` to prefill step payloads.
- Templates are data-only — no new hot-path code per template.

### Done gates

- [x] At least three templates with fixture round-trip through wizard `commit`
- [x] `openapi_gate.sh` lists template keys matching handler registry

---

## Phase 3 — Automation parity (Automizer-class)

Baseline shipped: `internal/automation`, 15 min worker interval, metrics `roi_pct` / `spend_micro` / `fraud_reject_rate`, actions `pause_campaign`, `blacklist_placement`, `platform_pause`, `notify` (`docs/INTEGRATIONS.md`). Gaps below close **reaction speed** and **setup friction**.

---

### `automation_eval_interval`

**Priority:** P1

**Gap:** Default 15 min evaluation is too slow for placement burn (Voluum-class rules often 5–30 min with faster cost refresh). Buyers sleep; budget drains.

**Surface:** `AUTOMATION_RULES_INTERVAL_MIN` env; per-rule `eval_interval_minutes` floor; worker scheduler in `internal/controlplane/automation_worker.go`.

**Target:**

- Allow rule-level interval `{5, 10, 15, 30, 60}` minutes; global floor configurable, default stays 15 for existing installs.
- Document interaction with cost sync sub-daily (`sync_interval_minutes` 15–60): ROI rules on `spend_micro` include partial-day API spend when credential supports it.
- Rate limit: max N rule evaluations per customer per tick to protect ClickHouse.

### Done gates

- [x] Handler test: interval below floor rejected at create
- [x] Integration test: rule with 5 min window fires within bounded test clock
- [x] `docs/INTEGRATIONS.md` updated with interval matrix (not SLA fiction)

---

### `automation_rule_presets`

**Priority:** P1

**Gap:** Defining a rule via raw JSON is Automizer-level work for a developer. Buyers expect named presets: "ROI below -40% for 30 minutes → blacklist placement".

**Surface:** `GET /api/v1/automation/presets`; `POST /api/v1/automation/rules` accepts `preset_key` + parameter overrides.

**Target:**

- Bundled presets (versioned JSON in repo): `placement_roi_guard`, `fraud_rate_guard`, `spend_cap_guard`, `silent_reject_spike`.
- Each preset: `parameters_schema` (threshold, window, cooldown), default actions, required license features.
- `POST .../dry-run` works with preset-expanded rule body.

### Done gates

- [x] Preset expand matches hand-written rule in dry-run fixture
- [x] Unknown `preset_key` returns 400 with catalog keys
- [x] No preset enables `platform_pause` without Enterprise license gate

---

### `automation_platform_pause_networks`

**Priority:** P2

**Gap:** `platform_pause` today is Meta + Google only (`internal/platformsync`, Enterprise SKU). TikTok, Microsoft, Taboola buyers still manual-pause in network UI.

**Surface:** `internal/platformsync/` adapters; extend `platform_campaign_mutations` outbox handlers.

**Target:**

- Add pause/resume for TikTok and Microsoft Ads where Cost Sync OAuth already exists.
- Dry-run + idempotency keys unchanged; document networks still read-only vs mutable in `docs/INTEGRATIONS.md`.
- Fail closed: mutation error leaves local campaign state unchanged and fires `notify` action if configured.

**Depends on:** Cost Sync credential parity per network.

### Done gates

- [x] httptest fixture per new network pause path
- [x] Holdout: failed platform API does not mark local placement as paused
- [x] SKU gate unchanged for Enterprise-only networks

---

## Phase 4 — Ops and TCO

Self-hosted TCO exceeds SaaS subscription when SRE time is included. These slugs reduce **operational blind spots**, not license price.

---

### `ops_stack_health_snapshot`

**Priority:** P2

**Gap:** Operator discovers CH lag, outbox backlog, or Redis shard drift from Grafana digression or buyer complaints.

**Surface:** `GET /api/v1/ops/health/snapshot`; `internal/controlplane/ops_health_handlers.go`.

**Target:**

- JSON snapshot: `clickhouse_lag_seconds`, `outbox_oldest_pending_seconds`, `redis_shard_reachable`, `cost_sync_last_success`, `automation_worker_last_tick`, `license_state`.
- RBAC `ops:read` or admin role; no per-request hot-path calls.
- `status`: `ok` | `degraded` | `critical` with threshold constants in handler (documented, not marketing SLA).

### Done gates

- [x] Handler test with mocked deps returns `degraded` when outbox > 30s
- [x] Response contains no secrets or connection strings

---

### `compose_minimal_stack_profile`

**Priority:** P3

**Gap:** Full compose stack (eBPF, broker, multi-shard Redis) scares teams comparing to $300/mo SaaS.

**Surface:** `deploy/compose/docker-compose.minimal.yaml`; `docs/DEVELOPMENT.md` section `Minimal buyer stack`.

**Target:**

- Profile runs tracker + control + PG + single Redis + CH with antifraud reduced (document which filters/signals disabled).
- `bash scripts/dev/stack.sh minimal` entry; explicit list of features unavailable in minimal mode.
- Not a separate product — same binaries, different compose env defaults.

### Done gates

- [x] `stack.sh minimal up` documented with env file template
- [x] README or DEVELOPMENT.md lists capability delta vs full stack (honest, no "unlimited")

---

## Phase 5 — Admin UI (P4, last)

No implementation in this backlog. UI slugs consume APIs from Phases 1–4.

| Order | Slug | Spec | Requires API |
| :--- | :--- | :--- | :--- |
| 1 | `admin_shell` | `ADMIN_SHELL_MILESTONE.md` | `session_nav_meta` (shipped) |
| 2 | `admin_detail_pattern` | `ADMIN_DETAIL_MILESTONE_CAMPAIGNS.md` | Phase 1 editor + publish gate |
| 3 | `admin_directory_pattern` | `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS.md` | `campaign_list_server_filters` (shipped) |
| 4 | `admin_campaigns_migrate` | `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE.md` | Phase 2 migration |
| 5 | `admin_integrations_hub` | `ADMIN_INTEGRATIONS_MILESTONE_HUB.md` | automation presets + cost sync |

Full UI index: [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md).

---

## Deferred

### `multi_touch_attribution_models`

**Status:** deferred (not competitive parity for affiliate/pop core)

**Gap:** RedTrack-class First Click / Linear / Data-Driven paths across weeks-long funnels. Product today is S2S last-touch (`click_id`) plus cost sync `token` / `spread` for spend allocation.

**Reopen when:** explicit e-commerce or retargeting SKU with CH identity graph design. Until then, document as non-goal in operator-facing docs — do not imply in `docs/INTEGRATIONS.md`.

---

## Explicit non-goals

| SaaS expectation | Stance |
| :--- | :--- |
| Buyer works with zero API literacy | Requires Phase 5 UI; engine alone is not that product |
| Sub-5-minute Automizer on all networks | Bounded by network API rate limits and cost sync adapters |
| Guaranteed creative upload to Meta/Google | Out of scope (`docs/INTEGRATIONS.md`) |
| Multi-touch attribution in v1 parity | Deferred slug above |
| "Free" self-hosted without SRE | `compose_minimal_stack_profile` reduces scope, not headcount to zero |
| ML per-click ghosting from admin toggle | Per-IP effects via fraud stream/outbox only (`ANTIFRAUD.md`) |

---

## Milestone index (this file only)

Create `MARKETING_<SLUG>_MILESTONE.md` from `MILESTONE_TEMPLATE.md` before implementation.

| Slug | Spec file (create on start) |
| :--- | :--- |
| `campaign_publish_gate` | `MARKETING_CAMPAIGN_PUBLISH_GATE_MILESTONE.md` |
| `campaign_post_deploy_smoke` | `MARKETING_CAMPAIGN_POST_DEPLOY_SMOKE_MILESTONE.md` |
| `campaign_onboarding_wizard_api` | `MARKETING_CAMPAIGN_ONBOARDING_WIZARD_MILESTONE.md` |
| `campaign_onboarding_templates` | `MARKETING_CAMPAIGN_ONBOARDING_TEMPLATES_MILESTONE.md` |
| `automation_eval_interval` | `MARKETING_AUTOMATION_EVAL_INTERVAL_MILESTONE.md` |
| `automation_rule_presets` | `MARKETING_AUTOMATION_RULE_PRESETS_MILESTONE.md` |
| `automation_platform_pause_networks` | `MARKETING_AUTOMATION_PLATFORM_PAUSE_MILESTONE.md` |
| `ops_stack_health_snapshot` | `MARKETING_OPS_STACK_HEALTH_MILESTONE.md` |
| `compose_minimal_stack_profile` | `MARKETING_COMPOSE_MINIMAL_STACK_MILESTONE.md` |

Slugs delegated to `cold_path_admin_api_backlog.md` or `competitive_backlog.md` use `COLD_*` or existing competitive milestone naming when those files add an index row.

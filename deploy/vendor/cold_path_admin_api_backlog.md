# Cold path admin API backlog

Server-side analytics, formatting, RBAC, workflow logic, and **campaign editor contracts** for `cmd/control` / `internal/controlplane`. Browser renders API output only (`ui.mdc`, `boundaries.mdc`).

**Status:** Cold-path admin API backlog P1–P3 shipped (including API key `scopes` PG persistence).

**Canonical:** `cold-path.mdc`, `control-plane.mdc`, `ui.mdc`, `boundaries.mdc`, `deploy/vendor/ANTIFRAUD.md`

**Milestone structure:** `deploy/vendor/MILESTONE_TEMPLATE.md` — implementation specs: `deploy/vendor/COLD_<SLUG>_MILESTONE.md` (UPPERCASE, flat folder)

**Out of scope here:** hot-path ingest (`internal/ingestion`), admin UI components (`web/` per [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md)), ML model training (`cmd/fraud-scorer`), competitive migration parity ([competitive_backlog.md](./competitive_backlog.md)). Buyer-workflow orchestration and ship order: [marketing_tracker_parity_backlog.md](./marketing_tracker_parity_backlog.md).

Cross-reference slugs in PR descriptions. Do not mark a slug closed until done gates pass in the same commit as code.

---

## Placement rules

| Tier | Allowed | Forbidden |
| :--- | :--- | :--- |
| Cold (`internal/controlplane`, workers) | PG/CH queries, RBAC, DTO formatting, async report jobs, outbox mutations | Sync call from `/track`; per-request license HTTP |
| Client (`web/` when rebuilt) | Fetch, parse JSON, render `items` / `*_display` / `status_label` | Filter/sort/aggregate over full datasets; role checks for mutations |
| Hot (`internal/ingestion`) | Signal emit, fraud stream columns | Report SQL, rule corpus export, PII to client |

Existing cold surfaces to extend (not replace): `reports_*`, `dashboards_handlers.go`, `authz/`, `rbac.go`, `meta_handlers.go`, `views_handlers.go`, `report_jobs.go`, `scrubCampaignFields`.

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [x] Every new symbol resolves (`go build -o /dev/null ./cmd/control/`)
- [x] Hot path does not import new controlplane handlers (`boundaries.mdc`)
- [x] OpenAPI fragment + handler registered together (`openapi_gate.sh`)
- [x] RBAC enforced in handler; not duplicated as client `if (role)` logic (`ui.mdc`)
- [x] Verification commands pasted in PR with package path (`quality.mdc`)
- [x] Holdout or handler test when authorization or redaction is non-obvious (`testing.mdc`)
- [x] No microbench cited as admin API SLA (`anti-slop.mdc`)
- [x] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `deploy/vendor/` prose (`naming.mdc`)

---

## Summary table

| Slug | Priority | Area | Status |
| :--- | :--- | :--- | :--- |
| `report_catalog_rbac` | P1 | analytics + nav | shipped |
| `wire_signal_breakdown_report` | P1 | analytics | shipped |
| `universal_list_envelope` | P1 | formatting | shipped |
| `display_field_formatting` | P1 | formatting | shipped |
| `allowed_actions_matrix` | P1 | RBAC | shipped |
| `field_redaction_profiles` | P1 | RBAC | shipped |
| `layer_desync_drilldown` | P2 | analytics | shipped |
| `rtt_split_tunnel_analytics` | P2 | analytics | shipped |
| `signal_effectiveness_report` | P2 | analytics | shipped |
| `campaign_toggle_cohort_diff` | P2 | analytics | shipped |
| `ingest_data_quality_extended` | P2 | analytics | shipped |
| `dashboard_chart_series` | P2 | formatting | shipped |
| `export_redaction_profiles` | P2 | formatting | shipped |
| `campaign_ownership_acl` | P2 | RBAC | shipped |
| `saved_view_spec_validation` | P2 | RBAC | shipped |
| `selfserve_api_key_scopes` | P2 | RBAC | shipped |
| `session_nav_meta` | P2 | nav | shipped |
| `license_feature_gate_helper` | P2 | policy | shipped |
| `budget_approval_rules` | P3 | workflow | shipped |
| `fraud_preset_governance` | P3 | workflow | shipped |
| `fraud_evidence_pack_bulk` | P3 | async | shipped |
| `campaign_import_validation_job` | P3 | async | shipped |
| `ml_shadow_delta_snapshot_worker` | P3 | async | shipped |
| `campaign_editor_shell` | P1 | campaign editor | shipped |
| `campaign_patch_dry_run` | P1 | campaign editor | shipped |
| `campaign_editor_integration_panel` | P1 | campaign editor | shipped |
| `campaign_list_server_filters` | P1 | campaign editor | shipped |
| `campaign_flow_validation` | P2 | campaign editor | shipped |
| `campaign_macro_preview` | P2 | campaign editor | shipped |
| `campaign_save_conflict` | P2 | campaign editor | shipped |
| `campaign_clone_wizard` | P2 | campaign editor | shipped |
| `campaign_diff_compare` | P2 | campaign editor | shipped |
| `campaign_editor_context_links` | P2 | campaign editor | shipped |
| `campaign_schedule_preview` | P2 | campaign editor | shipped |
| `campaign_bulk_mutations` | P2 | campaign editor | shipped |
| `campaign_margin_guard_on_save` | P2 | campaign editor | shipped |
| `campaign_placement_block_suggestions` | P2 | campaign editor | shipped |
| `campaign_editor_audit_sidebar` | P3 | campaign editor | shipped |
| `campaign_geo_summary_text` | P3 | campaign editor | shipped |
| `campaign_fraud_tab_preview` | P3 | campaign editor | shipped |
| `fraud_reason_category_map` | P1 | customer fraud | shipped |
| `customer_fraud_report_rbac` | P1 | customer fraud | shipped |
| `customer_fraud_overview_dashboard` | P1 | customer fraud | shipped |
| `customer_fraud_by_type_report` | P1 | customer fraud | shipped |
| `customer_fraud_by_dimension_report` | P2 | customer fraud | shipped |
| `customer_fraud_dispute_evidence` | P2 | customer fraud | shipped |
| `customer_fraud_export_schedule` | P2 | customer fraud | shipped |
| `customer_fraud_invalid_spend_kpi` | P3 | customer fraud | shipped |

**Suggested ship order:** `report_catalog_rbac` → `universal_list_envelope` + `display_field_formatting` → `allowed_actions_matrix` + `field_redaction_profiles` → `fraud_reason_category_map` + `customer_fraud_report_rbac` → `customer_fraud_overview_dashboard` + `customer_fraud_by_type_report` → `campaign_editor_shell` + `campaign_list_server_filters` → `wire_signal_breakdown_report` → remaining P2/P3 by operator priority.

**Customer fraud analytics block (before admin fraud report UI):** `fraud_reason_category_map` → `customer_fraud_report_rbac` → `customer_fraud_overview_dashboard` + `customer_fraud_by_type_report` → `customer_fraud_by_dimension_report` → `ADMIN_REPORT_MILESTONE_CUSTOMER_FRAUD.md` per [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md).

**Campaign editor block (before `admin_detail_pattern` UI):** `campaign_editor_shell` → `campaign_patch_dry_run` + `campaign_editor_integration_panel` → `campaign_save_conflict` → `ADMIN_DETAIL_MILESTONE_CAMPAIGNS.md` per [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md).

---

## Milestone index

Create `COLD_<SLUG>_MILESTONE.md` from `MILESTONE_TEMPLATE.md` before implementation. Fill sections 1.1–1.4, 4, 5, 6, 7 per slug.

| Slug | Spec file (create on start) |
| :--- | :--- |
| `report_catalog_rbac` | `COLD_REPORT_CATALOG_RBAC_MILESTONE.md` |
| `wire_signal_breakdown_report` | `COLD_WIRE_SIGNAL_BREAKDOWN_MILESTONE.md` |
| `universal_list_envelope` | `COLD_UNIVERSAL_LIST_ENVELOPE_MILESTONE.md` |
| `display_field_formatting` | `COLD_DISPLAY_FIELD_FORMATTING_MILESTONE.md` |
| `allowed_actions_matrix` | `COLD_ALLOWED_ACTIONS_MATRIX_MILESTONE.md` |
| `field_redaction_profiles` | `COLD_FIELD_REDACTION_PROFILES_MILESTONE.md` |
| `layer_desync_drilldown` | `COLD_LAYER_DESYNC_DRILLDOWN_MILESTONE.md` |
| `session_nav_meta` | `COLD_SESSION_NAV_META_MILESTONE.md` |
| `campaign_editor_shell` | `COLD_CAMPAIGN_EDITOR_SHELL_MILESTONE.md` |
| `campaign_patch_dry_run` | `COLD_CAMPAIGN_PATCH_DRY_RUN_MILESTONE.md` |
| `campaign_list_server_filters` | `COLD_CAMPAIGN_LIST_SERVER_FILTERS_MILESTONE.md` |
| `fraud_reason_category_map` | `COLD_FRAUD_REASON_CATEGORY_MAP_MILESTONE.md` |
| `customer_fraud_overview_dashboard` | `COLD_CUSTOMER_FRAUD_OVERVIEW_MILESTONE.md` |
| `customer_fraud_by_type_report` | `COLD_CUSTOMER_FRAUD_BY_TYPE_MILESTONE.md` |

Admin UI (after cold contracts): `ADMIN_DETAIL_MILESTONE_CAMPAIGNS.md`, `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS.md`, `ADMIN_REPORT_MILESTONE_CUSTOMER_FRAUD.md` from `MILESTONE_TEMPLATE.md`.

Other slugs may share a milestone when shipped in one PR (document combined scope in section 5).

---

## `report_catalog_rbac`

**Priority:** P1

**Gap:** Report keys and dashboard links are implicit in OpenAPI and handler registration. Future admin UI would hardcode nav or duplicate permission logic in the browser.

**Surface:** `GET /api/v1/reports/catalog` (new); `internal/controlplane/reports_catalog.go`; `api/openapi/paths/reports_catalog.yaml`.

**Target:**

- Rows: `key`, `title`, `description`, `required_permissions[]`, `parameters_schema` (JSON Schema subset), `default_range`, `export_formats[]`, `license_gated` (bool + `feature_key`).
- Filter rows server-side from `authz.Snapshot` + license snapshot; buyer role never sees ops-only keys.
- Optional `category` enum (`fraud`, `traffic`, `billing`, `rtb`, `telegram`) for nav grouping.
- Cross-link to existing routes in `documented_routes.go`; catalog is discoverability layer, not a second query engine.

### Done gates

- [x] Handler test: masked buyer snapshot omits `fraud-evidence-pack` and ops reports
- [x] `openapi_gate.sh` green; catalog keys match registered report handlers
- [x] No raw CH SQL or rule DSL in response

---

## `wire_signal_breakdown_report`

**Priority:** P1

**Gap:** Hot path emits L7/TLS/H2/behavior/mobile/residential signals (`ANTIFRAUD.md`); operators lack a dedicated CH breakdown report. Generic `fraud-breakdown` may not dimension by individual wire reasons.

**Surface:** `GET /api/v1/reports/wire-signal-breakdown`; `reports_wire_signal_breakdown.go`; CH `fraud_events` (and related MVs).

**Target:**

- Group by `campaign_id`, `fraud_reason` (or parsed reason token), time bucket; counts for impressions, hard blocks, silent rejects.
- Signal families as filter query params: `family=l7|tls|h2|tcp|behavior|mobile|residential`.
- `DataFreshnessDTO` + cursor pagination; `AuthorizeCustomerAccess` / campaign scope same as `reports_auth.go`.
- Row field `signals_degraded` when edge headers absent (CDN fail-open; align with `ANTIFRAUD.md` CDN limits).

### Done gates

- [x] Integration or handler test with fixture rows for `tls_ja4_mismatch`, `layer_desync_count` correlation column optional
- [x] Buyer/masked role does not receive IP or full landing URL columns
- [x] Registered in report catalog slug `report_catalog_rbac`

---

## `universal_list_envelope`

**Priority:** P1

**Gap:** Directory endpoints return heterogeneous shapes. Admin rebuild needs one list contract (`ui.mdc` cold path first).

**Surface:** `pkg/coldpath` or `internal/controlplane/list_envelope.go`; adopt on one pilot handler (e.g. `GET /api/v1/customers`), then roll through directories.

**Target:**

- Response envelope: `items`, `total`, `limit`, `offset`, `freshness` (`DataFreshnessDTO`), `filters_applied` (echo of validated query params), `sort` (`field`, `order`).
- Server validates sort whitelist per resource; reject unknown `sort` with 400.
- Pagination math and active-filter echo stay server-side; client refetches on param change only.

### Done gates

- [x] Pilot handler test asserts envelope keys on happy path and invalid `sort`
- [x] OpenAPI schema `ListEnvelope` component reused in at least two paths
- [x] Documented in `ui.mdc` or milestone section 4 (no duplicate essay in backlog)

---

## `display_field_formatting`

**Priority:** P1

**Gap:** Money and deltas often exposed as raw micros or floats; UI would format client-side (`ui.mdc` forbids business formatting in browser).

**Surface:** shared helpers in `internal/controlplane/format_display.go`; apply to campaign stats, billing lines, dashboard KPIs.

**Target:**

- Pair machine + display fields: `spend_micros` + `spend_display` + `currency`; `delta_pct` + `delta_label` + `delta_tone` (`positive`|`negative`|`neutral`).
- Status chips: `status`, `status_label`, `status_tone` on campaign, invoice, job rows.
- Locale: server default UTC formatting for v1; no client `Intl` requirement for money.

### Done gates

- [x] Unit tests for micros → display edge cases (zero, negative delta)
- [x] At least campaign dashboard + one billing handler return `*_display` fields
- [x] Masked role still receives display fields where underlying value is redacted (placeholder label)

---

## `allowed_actions_matrix`

**Priority:** P1

**Gap:** UI would infer button visibility from `role` string. Mutations must be authorized server-side anyway; visibility should match authorization without leaking rules.

**Surface:** `allowed_actions[]` and optional `denied_reasons` map on `GET /api/v1/campaigns/{id}`, `GET /api/v1/customers/{id}`; helper `computeAllowedActions(ctx, entity)`.

**Target:**

- Actions as tokens: `pause`, `resume`, `clone`, `edit_fraud`, `edit_budget`, `export`, `delete` (per entity type).
- Denied entries: `{ "edit_budget": "requires_team_lead_approval" }` when pending approval exists.
- PATCH/POST still call `RequirePermission`; matrix is UX hint only, not auth bypass.

### Done gates

- [x] Handler test: buyer snapshot has `pause` only, not `edit_fraud`
- [x] Holdout: action token absent → corresponding route still returns 403 if called directly

---

## `field_redaction_profiles`

**Priority:** P1

**Gap:** `scrubCampaignFields` and `BrandCreativeDTO.Scrub` exist; coverage is incomplete and clients cannot tell which fields were hidden.

**Surface:** extend `authz.MaskLevel` usage; `fields_redacted[]` on DTOs; audit log does not store pre-scrub secrets for support role.

**Target:**

| Role | Typical redactions |
| :--- | :--- |
| Buyer | `budget_limit`, payout fields, `offer_url`, blacklist internals |
| Support | masked URLs, no payout |
| Publisher | competitor campaign names, offer URLs |

- Central `redactCampaignDTO`, `redactCustomerDTO` called at handler boundary once.
- OpenAPI documents optional fields as omitted when redacted, not nulled secrets.

### Done gates

- [x] Tests per role: `fields_redacted` contains expected keys; response body lacks raw values
- [x] No new parallel DTO type for same table (`cold-path.mdc`)

---

## `layer_desync_drilldown`

**Priority:** P2

**Gap:** `layer-desync-summary` and `fraud-evidence-pack` shipped; operators cannot drill into top mismatching signal pairs or hourly trend.

**Surface:** `GET /api/v1/reports/layer-desync-drilldown`; extend CH queries on `fraud_events.layer_desync_count`, `fraud_reason`.

**Target:**

- Filters: `layer_desync_count` bucket, `campaign_id`, date range.
- Rows: top `fraud_reason` pairs or multi-reason strings where `layer_desync_count >= 2`.
- Hourly trend series (pre-bucketed) for fraud dashboard widget.
- `signals_degraded` row flag when TCP/TLS headers missing.

### Done gates

- [x] Query test fixtures mirror `reports_layer_desync_summary_test.go` patterns
- [x] Signed evidence pack fields remain consistent with drilldown aggregates

---

## `rtt_split_tunnel_analytics`

**Priority:** P2

**Gap:** `rtt_split_delta_ms` in CH; cold `ivt-detector` rule exists; no operator report.

**Surface:** `GET /api/v1/reports/rtt-split-tunnel`; CH column from migration `00026_rtt_split_tunnel.sql`.

**Target:**

- Distribution by ASN, country, campaign; optional join to `residential_proxy` flag rate.
- KPI: `split_tunnel_share` (server-computed ratio) on fraud dashboard.
- Fail-open: rows with missing RTT omitted from denominator with `coverage_pct` field.

### Done gates

- [x] Handler test with synthetic CH rows
- [x] Document CDN/edge requirement for RTT signal in `ANTIFRAUD.md` cross-link only

---

## `signal_effectiveness_report`

**Priority:** P2

**Gap:** Operators enable wire signals without block-rate or silent-reject impact visibility. Rule weights and corpora must stay server-side.

**Surface:** `GET /api/v1/reports/signal-effectiveness`; optional extension to `POST /api/v1/campaigns/{id}/fraud/preview` response.

**Target:**

- Per enabled signal (campaign-scoped): `block_rate`, `silent_reject_rate`, `event_volume`, optional `labeled_fp_rate` when fraud labels exist.
- Server-only recommendation field: `suggested_weight_tier` (`low`|`medium`|`high`) — not raw registry JSON.
- RBAC: `audit:read` or `campaigns:read`; no export of corpus files.

### Done gates

- [x] No `fraudReasonRegistry` or corpus path in JSON response
- [x] Stale CH banner via `DataFreshnessDTO`

---

## `campaign_toggle_cohort_diff`

**Priority:** P2

**Gap:** Toggling `accept_lang_geo_enabled`, `json_serialization_enabled`, or silent reject lacks before/after analytics.

**Surface:** `GET /api/v1/reports/campaign-toggle-cohort`; PG audit timestamps + CH windows.

**Target:**

- Query params: `campaign_id`, `toggle_field`, `toggle_at` (or auto-detect last change from audit).
- Windows: equal-length before/after; metrics: impressions, rejects, conversions, ROI if available.
- `delta_label` / `delta_tone` on KPI rows (`display_field_formatting` dependency optional but recommended).

### Done gates

- [x] Handler rejects cross-customer `campaign_id`
- [x] Empty before-window returns explicit `insufficient_data` code, not zero fiction

---

## `ingest_data_quality_extended`

**Priority:** P2

**Gap:** `reports/data-quality` exists; missing ingest completeness (behavior telemetry), per-report CH lag, postback recon by fraud signal.

**Surface:** extend `reports_data_quality.go` or new `reports_ingest_quality.go`.

**Target:**

- `telemetry_missing_rate` when campaign expects behavior telemetry.
- `ch_lag_seconds_by_report_key` map for operator ops view.
- Optional slice: conversions from IPs that had hard fraud blocks (recon signal).

### Done gates

- [x] Freshness fields match `control-plane.mdc` staleness rules
- [x] No sync PG/CH call from tracker

---

## `dashboard_chart_series`

**Priority:** P2

**Gap:** Dashboard handlers may return KPI scalars only; chart libraries would aggregate raw events client-side.

**Surface:** `dashboards_handlers.go` — add `series[]` blocks to `fraud`, `buyer`, `campaign` dashboards.

**Target:**

- Pre-bucketed points: `{ "label", "impressions", "blocks", "spend_micros" }` with server-chosen bucket width from range param.
- Max points cap (e.g. 366) enforced server-side.

### Done gates

- [x] Range param validation; 400 on excessive window without aggregation
- [x] Buyer dashboard series omit sensitive dimensions

---

## `export_redaction_profiles`

**Priority:** P2

**Gap:** Report jobs export raw columns; role-based column omission not systematic.

**Surface:** `report_jobs.go`, export handlers; profile keyed by `authz.Snapshot`.

**Target:**

- Profiles: `operator_full`, `buyer_summary`, `support_masked` — column allowlists per report key.
- CSV/PDF header: `exported_by`, `exported_at`, `deployment_id` from meta/branding.
- Async job only; no full-table sync export on request thread.

### Done gates

- [x] Job output test: buyer profile excludes IP column
- [x] `ReadLimitedBody` / job size limits unchanged (`cold-path.mdc`)

---

## `campaign_ownership_acl`

**Priority:** P2

**Gap:** Team members exist (`team_handlers.go`); media buyer may see all customer campaigns instead of assignments only.

**Surface:** `AuthorizeCampaignAccess`, list queries with team assignment join; `authz` scope `team`.

**Target:**

- Media buyer: `campaign_id IN (assignments)` for list, stats, reports.
- Team lead: customer scope unchanged.
- 404 vs 403 policy documented in milestone section 4 (prefer 404 for cross-team ID enumeration).

### Done gates

- [x] Handler test: assigned vs unassigned campaign access
- [x] Report handlers use same authorize helper as `reports_auth.go`

---

## `saved_view_spec_validation`

**Priority:** P2

**Gap:** `views_handlers.go` stores arbitrary `spec` JSON; execution does not validate against catalog or RBAC.

**Surface:** validate on `POST/PUT /api/v1/views` and on report execution path.

**Target:**

- Reject unknown `report_key`; strip disallowed spec keys per role (e.g. `include_ip` for buyer).
- Reject `customer_id` outside actor scope.
- Cap date range by license tier (pilot max 7 days).

### Done gates

- [x] Holdout: buyer cannot save spec with ops-only report key
- [x] Shared view cannot escalate permissions beyond owner snapshot

---

## `selfserve_api_key_scopes`

**Priority:** P2

**Gap:** Self-serve API keys may be all-or-nothing; scopes belong server-side.

**Surface:** `POST /api/v1/selfserve/api-keys`; PG column `scopes text[]`; middleware on selfserve routes.

**Target:**

- Scope tokens mirror permission strings (`campaigns:read`, `campaigns:pause`) — subset only.
- No `ops:write`, `blacklist:write`, fraud evidence on self-serve keys.
- Rate limit per key id in existing limiter.

### Done gates

- [x] Integration test: scoped key denied on `fraud-evidence-pack`
- [x] Key material shown once at create; audit row without secret
- [x] `scopes text[]` persisted on `api_keys`; verify path returns scopes to middleware

---

## `session_nav_meta`

**Priority:** P2

**Gap:** `GET /api/v1/meta` covers license/EULA; no permission-filtered nav for admin rebuild.

**Surface:** extend `meta_handlers.go` or `GET /api/v1/session` (new).

**Target:**

- Fields: `nav_items[]` (`href`, `label`, `icon_key`), `role`, `mask_level`, `default_customer_id`, `timezone`, `stale_banner` (global CH lag / outbox age).
- Nav filtered by catalog + license; mutations still use session auth, not meta hints.

### Done gates

- [x] Buyer nav excludes ops and billing write links
- [x] Meta/session does not embed permission rule map (tokens only)

---

## `license_feature_gate_helper`

**Priority:** P2

**Gap:** Feature gating scattered; handlers return ad-hoc 403 text.

**Surface:** `internal/controlplane/license_gate.go`; `RequireFeature(ctx, "openrtb")` wrapper.

**Target:**

- Consistent 403 body: `feature_required`, `feature_key`, `plan_code`.
- Used by RTB routes, ML reports, eBPF ops surfaces per JWT `sku.yaml`.
- Hot path unchanged (snapshot only).

### Done gates

- [x] Unit test per feature key with mock license snapshot
- [x] OpenAPI documents optional `feature_required` error schema

---

## `budget_approval_rules`

**Priority:** P3

**Gap:** `team/budget-approvals` exists; auto-deny and effective budget not computed server-side.

**Surface:** `team_handlers.go`, approval worker or inline on approve/deny.

**Target:**

- Auto-deny when request exceeds team cap or license host limit.
- `effective_budget_micros` on campaign DTO when pending approval exists.
- Chain: TL → manager optional per customer setting.

### Done gates

- [x] `AssertBudgetInvariant` unaffected; approval does not debit until approved
- [x] Audit row on auto-deny (`AUTO_DENY_BUDGET_APPROVAL`)
- [x] `effective_budget_micros` on campaign DTO when pending approval exists

---

## `fraud_preset_governance`

**Priority:** P3

**Gap:** Global fraud presets and campaign overrides need consistent RBAC and preview parity.

**Surface:** `fraud_handlers.go`, `ops_handlers.go` preset routes.

**Target:**

- `ops:write` for global preset PATCH; `campaigns:write` + license ML flag for campaign override.
- Preview endpoint uses same analyzer inputs as production; no rule JSON in response.

### Done gates

- [x] Support role cannot PATCH global preset
- [x] Preview holdout matches existing `fraud/preview` tests

---

## `fraud_evidence_pack_bulk`

**Priority:** P3

**Gap:** Single evidence pack report; operators need date-range bulk export with signing.

**Surface:** `POST /api/v1/reports/jobs` job type `fraud_evidence_pack_bulk`; reuse `fraud_evidence_pack_sign.go`.

**Target:**

- ZIP of signed packs per campaign/day; async poll + download.
- RBAC: `audit:read`; redaction profile `operator_full` only.

### Done gates

- [x] Job cancel via existing report job DELETE
- [x] Signature verify test on sample archive member

---

## `campaign_import_validation_job`

**Priority:** P3

**Gap:** Import preview is synchronous; large migrations need async validation.

**Surface:** job type on `POST /api/v1/campaigns/import` or dedicated validate endpoint.

**Target:**

- Returns warnings/errors JSON without committing TX until operator confirms separate import call.
- Reuse migration source adapters (`competitive_backlog.md` slugs) without duplicating logic.

### Done gates

- [x] Failed validation leaves PG unchanged (holdout)
- [x] Cold path outbound HTTP timeouts on pull adapters
- [x] Async job `POST /api/v1/campaigns/import/validate/jobs` + poll GET

---

## `ml_shadow_delta_snapshot_worker`

**Priority:** P3

**Gap:** `reports/ml/shadow-delta` may query CH live; heavy ranges need materialized snapshot.

**Surface:** control worker tick + CH MV or PG snapshot table; report reads snapshot only.

**Target:**

- Nightly refresh; `stale=true` when snapshot age > 24h.
- No `internal/fraud` scoring on request thread.

### Done gates

- [x] Worker registered in `cmd/control` only (`serve.go` → `StartMLShadowDeltaSnapshotWorker`)
- [x] Report handler documents snapshot timestamp in `freshness`

---

## Campaign editor and admin UX

Cold-path contracts for the campaigns directory and detail editor. UI milestones (`ADMIN_*_MILESTONE_CAMPAIGNS.md`) consume these endpoints; no business rules in `web/src` (`ui.mdc`, `boundaries.mdc`).

Existing surfaces to extend: `campaigns_handlers.go`, `campaign_dto.go`, `campaign_integration_health.go`, `campaign_import_export.go`, `fraud_handlers.go`, `service_forecast.go`, `GET /api/v1/dashboards/campaign/{id}`, `POST /api/v1/forecast/campaign`, `POST /api/v1/campaigns/{id}/apply-templates`.

---

## `campaign_editor_shell`

**Priority:** P1

**Gap:** `GET /api/v1/campaigns/{id}` returns a flat DTO. A rebuilt editor would hardcode tabs, field order, and license-gated sections in React.

**Surface:** `GET /api/v1/campaigns/{id}/editor-shell` (new) or embedded `editor` block on campaign GET; `campaign_editor_shell.go`.

**Target:**

- `sections[]`: `id`, `title`, `order`, `visible` (RBAC + license), `complete` (bool), `issue_count`, `issue_tone` (`ok`|`warn`|`fail`).
- Section ids align to editor areas: `general`, `budget_pacing`, `targeting`, `flow`, `tracking`, `postbacks`, `fraud`, `integrations`, `schedule`.
- `fields[]` per section: `key`, `label`, `editable`, `required`, `value_source` (`campaign`|`flow`|`fraud`), `help_slug` (doc anchor, not prose essay).
- `completion_pct` server-computed from required visible fields; buyer masked role gets fewer sections.
- Reuse `allowed_actions_matrix` tokens for section-level `actions[]`.

### Done gates

- [x] Handler test: license without OpenRTB hides RTB-linked section
- [x] Section `issue_count` reflects integration health + validation summary without second round-trip
- [x] OpenAPI documents `editor` block; no duplicate parallel `CampaignEditorDTO` tree beyond one response wrapper

---

## `campaign_patch_dry_run`

**Priority:** P1

**Gap:** Operators discover invalid PATCH combinations only after save (budget vs pacing, fraud flags vs license, broken flow refs).

**Surface:** `POST /api/v1/campaigns/{id}/validate` or `PATCH` with `?dry_run=1`; `campaign_validate.go`.

**Target:**

- Accept same body as `PatchCampaignRequest` (+ optional nested flow/fraud fragments).
- Response: `valid` (bool), `field_errors` map (`field` → `{ code, message }`), `warnings[]` (non-blocking), `estimated_outbox_events[]` (slug list only).
- No PG commit; read-only TX or in-memory validation against snapshot row.
- Fraud preview hook optional when body touches fraud flags (`POST .../fraud/preview` parity).

### Done gates

- [x] Holdout: dry_run does not increment `current_spend` or enqueue outbox
- [x] Invalid pacing mode returns 200 with `valid: false` (not 500)
- [x] Masked role cannot dry-run write to redacted fields

---

## `campaign_editor_integration_panel`

**Priority:** P1

**Gap:** `GetCampaignIntegrationHealth` exists but is a separate route; editor would stitch postback test, conversion mappings, and template status client-side.

**Surface:** `GET /api/v1/campaigns/{id}/integration-panel`; compose from `campaign_integration_health.go`, postback config, conversion mappings.

**Target:**

- Single response: `overall_status`, `overall_status_label`, `overall_status_tone`, `rows[]` (extend `IntegrationHealthRow` with `action_id`, `fix_hint_label`).
- Row actions as tokens: `test_postback`, `edit_conversion_mapping`, `apply_traffic_template` — mapped to `allowed_actions`.
- Include `last_postback_test_at`, `last_postback_test_label` when test endpoint has been run.
- Traffic template row links `traffic_template_id` to catalog entry title (server resolve).

### Done gates

- [x] Fixture tests mirror `campaign_integration_health_test.go` rows
- [x] No credential secrets in panel JSON
- [x] 404 when campaign outside authorize scope

---

## `campaign_list_server_filters`

**Priority:** P1

**Gap:** `GET /api/v1/campaigns` supports `customer_id`, `status`, pagination only. Directory UX needs search, sort, and facet filters server-side (`ui.mdc`).

**Surface:** extend `listCampaigns` query params; adopt `universal_list_envelope` when shipped.

**Target:**

- Query params: `q` (name/id search), `sort` (`name`|`spend`|`updated_at`|`margin_breach`), `order`, `owner_id`, `pacing_mode`, `margin_breach_only`, `integration_status` (`warn`|`fail`).
- Response: envelope + `facets` (`status_counts`, `pacing_counts`) optional when `include_facets=1`.
- `AttachCampaignListMarginBreach` retained; add `status_label`, `utilization_display` per row (`display_field_formatting`).
- URL is source of truth; no client filter over full `items[]`.

### Done gates

- [x] Handler test: invalid `sort` → 400; `q` SQL-injection safe (parameterized)
- [x] Buyer list respects `campaign_ownership_acl` when that slug ships
- [x] OpenAPI query params documented

---

## `campaign_flow_validation`

**Priority:** P2

**Gap:** Flow paths, weights, and lander/offer refs validated on import; inline editor lacks live validation before save.

**Surface:** `POST /api/v1/campaigns/{id}/flow/validate`; reuse `exportFlowBundle` / flow parse helpers.

**Target:**

- Checks: path weights sum to 100 (±0.01), dead lander/offer refs, empty path names, circular redirects, hosted lander publish state.
- Response: `valid`, `path_errors[]` (`path_index`, `code`, `message`), `suggested_fix_action` token.
- Optional: `normalized_paths` preview when operator accepts auto-fix (separate confirm PATCH).

### Done gates

- [x] Holdout: weight sum below 100 fails validation
- [x] No flow mutation without explicit save PATCH
- [x] DB lander/offer existence checks via `validateFlowPaths`

---

## `campaign_macro_preview`

**Priority:** P2

**Gap:** Click URL presets and `click_query_params` macros are opaque in the editor; operators paste broken URLs.

**Surface:** `POST /api/v1/campaigns/{id}/macro-preview`; input: sample context (`sub1`, `country`, etc.).

**Target:**

- Return `resolved_click_url`, `resolved_postback_url` (if configured), `unresolved_macros[]`, `warnings[]` (http vs https, missing domain).
- Server runs same macro resolver as ingest (`docs/INTEGRATIONS.md`); no tracker round-trip.
- Masked role: preview uses redacted offer URL placeholder.

### Done gates

- [x] Unit test with fixture macros from campaign clone tests
- [x] Invalid macro syntax returns field-level error, not panic

---

## `campaign_save_conflict`

**Priority:** P2

**Gap:** Concurrent editors overwrite each other; no optimistic concurrency on PATCH.

**Surface:** `version` or `etag` on `CampaignDTO`; `If-Match` header on PATCH; 409 `CONFLICT` body with `current` snapshot subset.

**Target:**

- `updated_at` or monotonic `revision` int on row; PATCH requires match.
- 409 response includes `server_revision`, `conflict_fields[]`, `merge_hint_label` (operator-facing, server-generated).
- Audit log records conflict rejections.

### Done gates

- [x] Holdout: stale `If-Match` does not apply PATCH
- [x] Idempotent retry with same revision succeeds

---

## `campaign_clone_wizard`

**Priority:** P2

**Gap:** `POST .../clone` copies fixed bundle; operators cannot choose what to duplicate (flow only, reset budget, copy fraud flags).

**Surface:** extend `CloneCampaignSpec` request body; `campaign_clone_test.go` patterns.

**Target:**

- Options: `include_flow`, `include_postbacks`, `include_fraud`, `include_placement_blocks`, `reset_spend`, `name_suffix`.
- Response preview step: `POST .../clone/preview` returns `would_create` summary without TX.
- Idempotency key unchanged.

### Done gates

- [x] Preview does not create rows; clone TX still atomic
- [x] `reset_spend` holdout: clone `current_spend` always 0

---

## `campaign_diff_compare`

**Priority:** P2

**Gap:** No way to compare draft vs live or campaign A vs B before merge/import.

**Surface:** `GET /api/v1/campaigns/{id}/diff?against={other_id}` or `POST` with inline PATCH body.

**Target:**

- Rows: `path` (JSON pointer), `label`, `left_display`, `right_display`, `severity` (`change`|`add`|`remove`).
- Exclude secrets; mask URLs per role.
- Cap diff size (top 200 paths); `truncated: true` when over limit.

### Done gates

- [x] Cross-customer diff returns 404
- [x] Diff against self returns empty rows

---

## `campaign_editor_context_links`

**Priority:** P2

**Gap:** Editor isolated from reports; operators hunt nav for campaign stats, fraud breakdown, pacing drift.

**Surface:** `context_links[]` on editor-shell or campaign dashboard DTO.

**Target:**

- Links: `href`, `label`, `report_key` or `dashboard_key`, `default_range`, `required_permissions[]`.
- Server builds query string (`campaign_id`, `customer_id`); client does not construct report params.
- Examples: `campaign-overview`, `pacing-drift`, `filter-rejects`, `wire-signal-breakdown` (when shipped).

### Done gates

- [x] Buyer role omits fraud evidence link
- [x] Links match `report_catalog_rbac` keys when both shipped

---

## `campaign_schedule_preview`

**Priority:** P2

**Gap:** Schedule rules stored as structured fields; editor shows raw cron/daypart without human summary.

**Surface:** `schedule_summary` on editor-shell or dedicated `GET .../schedule-preview`.

**Target:**

- Fields: `summary_label` (`"Active Mon–Fri 09:00–18:00 UTC"`), `next_activation_at`, `next_deactivation_at`, `currently_active` (bool).
- Timezone from customer or platform setting; server computes instants.
- Warn when schedule excludes >50% of week (`schedule_sparse_warning`).

### Done gates

- [x] DST edge case test for at least one timezone
- [x] No client cron parser required

---

## `campaign_bulk_mutations`

**Priority:** P2

**Gap:** Adops pauses or retargets dozens of campaigns; no bulk API.

**Surface:** `POST /api/v1/campaigns/bulk`; async job when `campaign_ids` > threshold.

**Target:**

- Actions: `pause`, `resume`, `set_pacing_mode`, `adjust_budget_pct` (bounded ±%).
- Request: `campaign_ids[]` or filter snapshot (`customer_id` + `status`); RBAC per id.
- Response: `job_id` or sync `results[]` (`id`, `ok`, `error_code`); audit per campaign.

### Done gates

- [x] Partial failure does not leave unbounded outbox storm (batch limit)
- [x] Buyer cannot bulk-edit fraud or budget above cap

---

## `campaign_margin_guard_on_save`

**Priority:** P2

**Gap:** `GetCampaignMargin` and list margin breach exist; PATCH does not surface pre-save advisory.

**Surface:** integrate into `campaign_patch_dry_run` or PATCH response `advisories[]`.

**Target:**

- When margin below floor: `advisory_code` `margin_floor_breach`, `severity` `warn`, `requires_confirm` true.
- Include `projected_margin_pct`, `floor_pct`, `impact_label` (server formatted).
- Does not block save unless customer policy `hard_margin_enforce` (PG flag).

### Done gates

- [x] Advisory present on dry-run; save without confirm still allowed when policy off
- [x] `AssertBudgetInvariant` unchanged

---

## `campaign_placement_block_suggestions`

**Priority:** P2

**Gap:** `POST .../placement-blocks` is manual; operators lack CH-backed suggestions.

**Surface:** `GET /api/v1/campaigns/{id}/placement-block-suggestions`; CH query on placements report.

**Target:**

- Rows: `placement_id`, `impressions`, `ivt_rate`, `reason_label`, `suggested_action` (`block`|`watch`).
- `freshness` block; min impression threshold to avoid noise.
- One-click block still goes through existing POST (client sends id only).

### Done gates

- [x] Stale CH sets `stale: true` on suggestions
- [x] No auto-block without explicit POST

---

## `campaign_editor_audit_sidebar`

**Priority:** P3

**Gap:** `GET .../events` returns raw audit rows; editor timeline needs human labels and grouping.

**Surface:** extend `CampaignEventDTO` or `GET .../events?format=timeline`.

**Target:**

- Fields: `title_label`, `actor_label`, `change_summary`, `section_id` (links to editor section), `occurred_at_display`.
- Group by day server-side: `days[]` with `events[]` nested.
- Cap 90 days; cursor for older.

### Done gates

- [x] Support role sees masked actor email per `field_redaction_profiles`
- [x] No raw JSON diff blobs in response

---

## `campaign_geo_summary_text`

**Priority:** P3

**Gap:** Geo allow/block lists are JSON arrays; editor map UI would parse rules client-side.

**Surface:** `geo_summary` on editor-shell targeting section.

**Target:**

- `included_label` (`"12 countries, 3 regions"`), `excluded_label`, `conflict_warning` when allow+block overlap.
- Expand `GET .../geo-summary?expand=1` returns country names resolved server-side (ISO → label table).
- No GeoIP download to browser.

### Done gates

- [x] Empty geo rules return neutral label, not error
- [x] Expand list capped at 50 rows + `truncated`

---

## `campaign_fraud_tab_preview`

**Priority:** P3

**Gap:** `POST .../fraud/preview` exists; fraud tab needs structured cards, not raw analyzer JSON.

**Surface:** extend fraud preview response or `GET .../fraud/editor-summary`.

**Target:**

- Cards: `tier`, `estimated_block_rate_label`, `silent_reject_impact_label`, `top_signals[]` (`signal_label`, `volume_pct` — no raw reason registry).
- `preview_stale` when sample window empty; link to `signal_effectiveness_report` via `context_links`.
- Toggle preview for draft fraud PATCH body (same as dry-run pattern).

### Done gates

- [x] No corpus paths or rule weights in JSON
- [x] Preview rate limit shared with existing fraud preview guard

---

## Customer fraud analytics (advertiser-facing)

Sanitized fraud quality analytics for clients (`campaigns:read:masked` / buyer role). Operators keep full `audit:read` surfaces (`GET /api/v1/dashboards/fraud`, raw `fraud_reason`, evidence pack, ML ops).

**Gap today:** CH pipeline is live (`fraud_events.fraud_reason`, `silent_reject_event`, `fraud_score`, `layer_desync_count`), but dedicated fraud reports require `campaigns:read` or `audit:read` — not `campaigns:read:masked`. Buyer dashboard exposes `ivt_rate` on sources only.

**RBAC target:**

| Role | Permission | Surfaces |
| :--- | :--- | :--- |
| Operator | `audit:read` | Full breakdown, filter-rejects, evidence pack, ML dashboard, wire-signal operator report |
| Client (buyer) | `campaigns:read:masked` | Category labels, rates, trends, placement/sub/geo slices — no raw IPs, no rule weights |
| Client manager | `campaigns:read` | Same as buyer + optional dispute evidence when license allows |

**Data flow:** hot path writes `fraud_events` → cold handlers scope `campaign_id IN (listCustomerCampaignIDs)` → server maps `fraud_reason` → `fraud_category` + `fraud_category_label` → UI renders (`ui.mdc`).

**Category taxonomy (server-only map; never ship registry JSON to browser):**

| `fraud_category` | Client label | Example internal reasons |
| :--- | :--- | :--- |
| `invalid_device_signals` | Invalid device signals | `tls_ja4_mismatch`, `client_hints_mismatch`, `header_order_mismatch`, `h2_*` |
| `automated_traffic` | Automated traffic | `json_serialization_bot`, `behavior_telemetry_missing` |
| `geo_language_mismatch` | Geo or language mismatch | `accept_lang_geo_mismatch`, geo filter rejects |
| `proxy_datacenter` | Proxy or datacenter traffic | `residential_proxy`, `datacenter_ip`, `tcp_tunnel_mss` |
| `policy_reject` | Campaign policy | schedule, device, budget filter kinds |
| `ivt_tier` | Invalid traffic tier | score tiers pass / suspect / ivt / block |

Existing queries to extend: `reports_fraud.go`, `reports_ivt.go`, `reports_silent_reject_impression.go`, `dashboards_handlers.go` (buyer dashboard).

---

## `fraud_reason_category_map`

**Priority:** P1

**Gap:** Internal `fraud_reason` tokens (`tls_ja4_mismatch`, etc.) are meaningful to operators only. Client UI would hardcode labels or leak registry structure.

**Surface:** `internal/controlplane/fraud_reason_categories.go`; used by all customer fraud handlers.

**Target:**

- Pure function: `reasonToCategory(reason string) (category, label string)`.
- Multi-reason comma strings split and dedupe categories for aggregation.
- Unknown reason → `other` + generic label; metric `ad_fraud_category_unknown_total` for ops.
- Unit tests per row in `ANTIFRAUD.md` signal table; holdout: operator DTO may still include `fraud_reason`, masked DTO must not.

### Done gates

- [x] Corpus test covers all shipped hot-path signals from `residential_proxy_detection_backlog.md`
- [x] No import of `internal/fraud` scoring or rule registry files into response builders
- [x] `check_no_legacy_naming.sh` clean

---

## `customer_fraud_report_rbac`

**Priority:** P1

**Gap:** Buyer role cannot call `fraud-breakdown`, `ivt-by-source`, or `silent-reject-impression-funnel` (`campaigns:read:masked` not in route perms).

**Surface:** `reports_fraud.go`, `reports_ivt.go`, `reports_silent_reject_impression.go`; `scrubFraudReportRow(ctx, row)`.

**Target:**

- Add `campaigns:read:masked` to `permAny` on:
  - `GET /api/v1/reports/fraud-breakdown`
  - `GET /api/v1/reports/ivt-by-source`
  - `GET /api/v1/reports/silent-reject-impression-funnel`
- When `authz.MaskLevel == MaskMasked`:
  - Omit or hash `placement_id` when policy requires
  - Replace `fraud_reason` with `fraud_category` + `fraud_category_label` only
  - Omit `fraud_score` raw value; optional `tier_label` only
- `resolveReportCustomerID` + `listCustomerCampaignIDs` unchanged; holdout cross-customer 403/404.

### Done gates

- [x] Handler test: buyer token 200 on fraud-breakdown; response has no raw `fraud_reason`
- [x] Handler test: buyer token 403 on `fraud-evidence-pack` and `filter-rejects`
- [x] Operator with `audit:read` still receives full `fraud_reason` on same routes

---

## `customer_fraud_overview_dashboard`

**Priority:** P1

**Gap:** `GET /api/v1/dashboards/buyer` has `ivt_rate` on sources but no fraud-volume KPIs or trend series for invalid traffic share.

**Surface:** extend `BuyerDashboardDTO` or `GET /api/v1/dashboards/customer-fraud`; CH aggregates on `fraud_events` + impressions.

**Target:**

- KPIs: `total_events`, `blocked_events`, `silent_reject_events`, `ivt_rate`, `block_rate`, `silent_reject_rate` with `*_display` and `freshness`.
- Optional: `invalid_traffic_share_pct` + `share_label` (depends on `customer_fraud_invalid_spend_kpi`).
- `series[]`: daily buckets (`label`, `blocked_events`, `silent_reject_events`, `ivt_events`) — max 366 points (`dashboard_chart_series`).
- `disclaimer` when edge headers absent (`signals_degraded` from CH coverage probe).
- Query: `customer_id`, `from`, `to`; RBAC `campaigns:read:masked`.

### Done gates

- [x] Buyer dashboard test returns fraud KPI block; masked role only own `customer_id`
- [x] `stale: true` when CH lag > 5 min per `control-plane.mdc`
- [x] No client-side aggregation contract in response

---

## `customer_fraud_by_type_report`

**Priority:** P1

**Gap:** Clients need “types of fraud in my data” — counts and shares by category, not operator reason tokens.

**Surface:** `GET /api/v1/reports/customer-fraud-by-type` (new); or masked mode on extended `fraud-breakdown`.

**Target:**

- Rows: `campaign_id`, `fraud_category`, `fraud_category_label`, `event_count`, `silent_reject_count`, `share_pct`, `share_label`, `silent_reject_ratio`.
- CH query: same base as `fraudBreakdownQuery`; aggregate in Go or SQL by category map.
- Filters: `customer_id`, `campaign_id`, `from`, `to`, optional `fraud_category`.
- Cursor pagination + `DataFreshnessDTO`; register in `report_catalog_rbac` under category `fraud`.

### Done gates

- [x] Fixture rows: two reasons map to one category; share_pct sums to ~100% per campaign slice
- [x] Masked role: no `fraud_reason` key in JSON
- [x] `openapi_gate.sh` green

---

## `customer_fraud_by_dimension_report`

**Priority:** P2

**Gap:** Clients need to see where fraud concentrates (placement, sub1/sub2, country) without operator filter-reject tables.

**Surface:** `GET /api/v1/reports/customer-fraud-by-dimension`; join patterns from `ivt-by-source` and `placements` report.

**Target:**

- Query param `dimension`: `placement` | `sub1` | `sub2` | `country` | `campaign`.
- Rows: dimension value, `impressions`, `clicks`, `ivt_events`, `blocked_events`, `ivt_rate`, `ivt_rate_label`, top `fraud_category` + `fraud_category_label`.
- Compare prior period optional (`compare=1`) with `delta_label` / `delta_tone` (`display_field_formatting`).
- Masked role: placement ids only (no IP); cap high-cardinality dimensions (top 500 + `truncated`).

### Done gates

- [x] Dimension whitelist enforced; unknown → 400
- [x] Cross-customer campaign_id in filter → 403/404
- [x] Stale CH banner on response

---

## `customer_fraud_dispute_evidence`

**Priority:** P2

**Gap:** `fraud-evidence-pack` is operator-only (`audit:read` + full `campaigns:read`). Advertisers disputing CPA need signed bundles without ops access.

**Surface:** `GET /api/v1/reports/fraud-evidence-pack` policy split or `GET /api/v1/reports/customer-fraud-evidence`.

**Target:**

- Gate: `campaigns:read` (not masked) + license feature `fraud_dispute_evidence` (SKU field).
- Response: same signing as `fraud_evidence_pack_sign.go` but redacted payload (no raw IP, category labels instead of full reason list where masked).
- Rate limit per customer; audit row per download.
- Buyer (`campaigns:read:masked`) remains 403.

### Done gates

- [x] Holdout: buyer cannot fetch evidence pack
- [x] Signature verify test on redacted bundle
- [x] `ANTIFRAUD.md` cross-link for dispute workflow only; no guarantee language

---

## `customer_fraud_export_schedule`

**Priority:** P2

**Gap:** Clients cannot schedule or export fraud-by-type CSV; `report-schedules` exists but report keys omit customer fraud surfaces.

**Surface:** `report_keys.go`, `report_job_export.go`, `report_schedule_handlers.go`; `export_redaction_profiles`.

**Target:**

- Job types: `customer-fraud-by-type`, `customer-fraud-by-dimension`.
- Profile `buyer_summary`: category labels, rates, no IP, no `fraud_reason`.
- Schedule CRUD allowed for `campaigns:read:masked`; catalog entry in `report_catalog_rbac`.
- CSV header: `exported_by`, `exported_at`, `deployment_id`, `disclaimer` row when `signals_degraded`.

### Done gates

- [x] Export job output test matches masked DTO shape
- [x] Buyer export preamble includes `signals_degraded` when CH freshness is stale
- [x] Scheduled run respects customer_id scope on stored spec (`saved_view_spec_validation`)

## `customer_fraud_invalid_spend_kpi`

**Priority:** P3

**Gap:** Advertisers ask “how much spend went to invalid traffic?” — not exposed on buyer dashboard.

**Surface:** extend `customer_fraud_overview_dashboard` KPIs; CH join spend on blocked/silent events.

**Target:**

- `invalid_spend_micros`, `invalid_spend_display`, `invalid_spend_share_pct`, `share_label`.
- Definition documented in handler (blocked + silent reject attributable spend; fail-open when spend attribution ambiguous).
- Optional compare prior period delta fields.

### Done gates

- [x] Unit test with fixture economics rows; no double-count when event both blocked and billed
- [x] Disclaimer field when attribution coverage < 90%

---

## Forbidden on cold path (reference)

Do not add to handlers or expose to clients:

- Raw fraud rule DSL, corpus file contents, residential intel feed payloads
- ClickHouse SQL strings or ad-hoc query builders in JSON
- Unmasked PII for buyer/support roles
- Client-side aggregation contracts (full event dumps for charting)
- Per-request license server HTTP calls
- `KEYS` / `FLUSHALL` on Redis (`cold_path_static_gate.sh`)
- Client-facing responses: `fraudReasonRegistry` weights, corpus paths, ML shard ops, unqualified “fraud-free” claims

---

## Related backlogs

| Backlog | Relationship |
| :--- | :--- |
| [admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md) | UI consumes this API; campaign editor slugs before `ADMIN_DETAIL_MILESTONE_CAMPAIGNS.md` |
| [residential_proxy_detection_backlog.md](./residential_proxy_detection_backlog.md) | Hot signals; `fraud_reason_category_map` labels those signals for clients |
| [competitive_backlog.md](./competitive_backlog.md) | Migration jobs; `campaign_import_validation_job` shares adapters |
| `deploy/vendor/ANTIFRAUD.md` | Operator signal reference; client copy uses category map only |

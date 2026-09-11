# Cold path tasks (MB / TL)

Backlog for Control Plane improvements: analytics visibility, exports, team governance.
Audience: media buyer (MB), team lead (TL).

Scope policy: `docs/CONTROL_PLANE_UI_SCOPE.md` — CP stays export-centric; no full in-browser BI tables.

Implementation order: **OpenAPI + Go handlers first**, then `web/src/api/*`, then unstyled TSX (logic and buttons only). Visual styling is a separate track (M9).

---

## Hard constraints

### Documentation in this file

- No wide markdown tables — use headings and bullet lists only.

### Frontend DOM (mandatory)

Heavy directory-style tables break the admin SPA on real campaign volume (~35 columns x 100+ rows). **Do not ship new large interactive tables** for analytics in this backlog.

Allowed UI patterns:

- **KPI blocks** — `<dl>` / stat tiles from API aggregates (dashboard, team metrics).
- **Small capped lists** — server `limit` <= 25 rows; "Export full report" for the rest.
- **Single-row or popover detail** — campaign stats in editor, metrics popover (seed from parent batch).
- **Export Hub + async jobs** — canonical path for tabular BI (CSV/XLSX/Sheets).
- **Unstyled semantic `<table>`** only when row count is bounded by API (e.g. schedule list, alert history page, leaderboard top N).

Forbidden for new cold-path surfaces:

- New `DirectoryTable` / column-resize / reorder matrices for report data.
- Client-held hundreds of rows without server pagination.
- Reviving `web/src/domains/reports/*` directory runners as mounted routes.
- Div-grid pretending to be a data table.

Reference: `.cursor/rules/frontend-slop.mdc` (directory tables, RF-*, VL-17), `ui.mdc` (semantic table vs grid).

---

## Unstyled UI convention (M0)

Add shared primitives under `web/src/shell/unstyled/`:

- `unstyled_section.tsx` — section + heading
- `unstyled_toolbar.tsx` — `role="toolbar"` button row
- `unstyled_form_row.tsx` — label + control
- `unstyled_table.tsx` — minimal semantic table for **small** lists only

Mark up with `data-testid` / `data-role` only. No Tailwind layout matrices until M9.

Every surface: `ErrorBlock` on fetch failure, `StubBanner` on 501/license, coalesced Refresh (R1).

---

## Milestone overview

**M0** — Foundation + ship XLSX / Google Sheets export (in branch)  
**M1** — Lite dashboard (buyer / adops) — KPI blocks, no report table  
**M2** — Export Hub: compare period, re-run job, notifications  
**M3** — Report schedules UI + Sheets on cron  
**M4** — Saved views wired into Export Hub — **shipped**
**M5** — Smart alerts lite (template rules only) — **shipped**
**M6** — Team KPI, leaderboard (capped rows), MB approval status  
**M7** — Campaign pacing/budget signals on list + stats panel in editor  
**M9** — Styling pass (Tailwind, `PageLayout`, gates) — after logic stable  

Dependencies: M0 before M1/M2; M2 before M3/M4; M1 before M5/M6/M7; M9 after M1–M7.

Suggested sprints: S1 = M0+M1, S2 = M2+M4, S3 = M3+M5, S4 = M6+M7, S5 = M9.

---

## M0 — Foundation

**Status:** shipped in branch (backend + Export Hub + Integrations + `web/src/shell/unstyled/`).

**Goal:** Unstyled shell + complete XLSX / Google Sheets one-shot export.

### Backend

- Finish `internal/reports/export/` xlsx + `csv_to_xlsx.go`
- Finish `internal/integrations/googlesheets/` + migrations `00145`, `00146`, identity `00016`
- Wire `google_sheets_bridge.go`, extended job timeout in `reportjob`
- Audit `LiveReportExportKeys()` vs catalog gaps (doc or fix)

### OpenAPI

- `ReportJobSpec`: `format: xlsx`, `destination: google_sheet` (already in `ops_reports.yaml`)
- Refresh `REPORT_EXPORT_XLSX_SHEETS_SPEC.md` status to implemented

### Frontend (logic only)

- `web/src/shell/unstyled/*`
- Export Hub: destination, format, Sheets fields, job lifecycle link
- Integrations Google Sheets connect/disconnect page

### Done when

- Job with `destination=google_sheet` returns `spreadsheet_url` on status
- XLSX download works via job download URL
- No new heavy tables introduced

---

## M1 — Lite dashboard

**Status:** shipped in branch (buyer + adops KPI pages, routes unfrozen).

**Goal:** Morning KPI without Export Hub. **No report table in browser.**

### API (existing)

- `GET /api/v1/dashboards/buyer?customer_id&from&to`
- `GET /api/v1/dashboards/adops?...`
- `GET /api/v1/dashboards/buyer/drilldown?customer_id` — cap breakdown at API (~100 rows); UI shows top slice + export link

RBAC: `campaigns:read` / team scope via backend.

### Backend (minimal)

- Ensure `freshness_label` / stale flags on dashboard DTO when CH degraded
- MB drilldown scoped to own campaigns; TL to team

### Frontend

- `web/src/api/dashboards_api.ts`
- `web/src/domains/dashboards/` — KPI blocks (`dashboard_kpi_blocks.tsx`), optional **small** breakdown list (not campaigns directory)
- `web/src/pages/dashboard_buyer_page.tsx`, `dashboard_adops_page.tsx`
- Buttons: Apply dates, Refresh, "Export true-roi", "Open campaigns"

### Routes

- Unfreeze `/dashboards/buyer`, `/dashboards/adops` (stop redirect to `/exports`)
- Optional: role default home — MB `/dashboards/buyer`, TL `/dashboards/adops` (env `CP_ROLE_HOME=1`)

### Done when

- Live `GET /dashboards/buyer` drives KPI blocks
- CH down shows stale banner, not empty fake zeros
- Full drilldown available via Export Hub pre-fill, not a 40-column table

---

## M2 — Export Hub enhancements (shipped)

**Goal:** Compare periods, re-run jobs, notify on completion.

**Status:** Shipped — compare/notify on `ReportJobSpec`, in-app notifications table + feed, rerun handler, Export Hub UI (compare fields, notify channel, bell, re-run on recent jobs). Placements export writes delta columns when compare range set.

### OpenAPI (new)

Extend `ReportJobSpec`:

- `compare_from`, `compare_to` (date-time)

Add `notify` object: channels `none | email | slack_webhook | in_app`, optional email / webhook URL.

New routes:

- `POST /api/v1/reports/jobs/{id}/rerun` — clone spec to new job (`exports:run`)
- `GET /api/v1/reports/notifications` — in-app feed (`exports:read`)
- `POST /api/v1/reports/notifications/{id}/ack`

### Backend

- Pass compare range into export writers where report supports deltas
- On job terminal state enqueue `internal/notify/` (dedup per job id)
- `rerunJob` handler in `reportjob`

Holdout: failed job creates one notification; idempotent rerun.

### Frontend

- `export_hub_compare_fields.tsx`, `export_hub_notify_fields.tsx`
- Re-run on recent job row; notification bell + ack
- Extend `use_export_hub_page_workspace.ts`

### Done when

- Operator gets in-app "job ready" without manual poll spam
- Re-run duplicates last job parameters

---

## M3 — Report schedules (shipped)

**Goal:** Cron exports for TL; Sheets on schedule.

**Status:** Shipped — extended schedule schema (destination, Sheets owner, notify, last run status), `exports:run` on mutations, `POST .../run`, worker OAuth guard, `/exports/schedules` UI.

### OpenAPI

Extend `ReportSchedule` / create/update in `platform.yaml`:

- `destination`: `download | google_sheet`
- `google_sheet` sub-object (same as job spec)
- `notify` (same as M2)
- `owner_user_id` — OAuth actor for scheduled Sheets jobs
- `last_run_status`, `last_run_error_public`

Permissions: create/update schedules — `exports:run` (not only `campaigns:write`).

### Backend

- `buildReportJobSpecFromSchedule` — destination, Sheets, `ExportedBy`
- PG migration on `report_schedules`
- Validate OAuth for owner before enable

### Frontend

- Route `/exports/schedules` or tab on Export Hub
- **Small** schedules list (unstyled table, one row per schedule)
- Form: cron, report_key, format, destination, notify
- Buttons: Create, Enable/Disable, Delete, Run now, View last job

### Done when

- Daily `pacing-drift` schedule enqueues job; failure visible in UI
- MB without `exports:run` gets 403 on POST

---

## M4 — Saved views

**Status:** shipped in branch (Export Hub presets + `POST /api/v1/views/{id}/export`).

**Goal:** One-click export presets.

### API (existing + new)

- `GET/POST/PUT/DELETE /api/v1/views`, `GET /api/v1/views/{id}`
- `POST /api/v1/views/{id}/export` — enqueue report job from stored spec (`exports:run`)

### Backend

- Extended saved-view spec keys: compare, notify, destination, format, row_limit, google_sheet, import_payload, kind, entry, billing_format, redact_pii
- `ValidateReportViewForActor` alias; hub preset report keys `billing-export`, `audit-export`
- `BuildReportJobSpecFromSavedView` + runner `ExportSavedView`

### Frontend

- `/exports` section: load/save/delete preset; export shortcut for report presets
- `web/src/api/views_api.ts`, `export_hub_saved_views.tsx`, `export_hub_saved_view_spec.ts`

### Done when

- Save restores all Export Hub form fields including compare/notify when present

---

## M5 — Smart alerts lite

**Status:** shipped in branch (template rules, worker evaluators, `/alerts` UI).

**Goal:** Template rules only — no free-form query builder.

### API

- `GET/POST/PATCH/DELETE /api/v1/smart-alerts/rules`, `GET /api/v1/smart-alerts/history`, `POST /api/v1/smart-alerts/events/{id}/ack`
- `SmartAlertRuleTemplate`: `budget_burn_pct`, `roi_below`, `pacing_drift`, `export_job_failed`, `margin_breach`
- Create/update accepts `template`, `threshold`, optional `campaign_id`, `webhook_url` (legacy metric/operator still supported)

### Backend

- `internal/smartalerts/templates.go` maps templates to fixed metric/operator/window
- Worker template evaluators in `worker_templates.go` (PG + CH; no client SQL)
- Webhook payload includes `template` when applicable

### Frontend

- Route `/alerts` (legacy `/smart-alerts/*` redirects)
- `web/src/api/smart_alerts_api.ts`, `web/src/domains/alerts/*`
- Rules list, paginated history, template form (Create / Enable / Disable / Ack)

### Done when

- Template fires → history row + webhook; no custom SQL field in UI

---

## M6 — Team lead workspace

**Status:** shipped

**Goal:** Team aggregate KPI + capped leaderboard + MB approval visibility.

### OpenAPI (new)

- `GET /api/v1/team/metrics?customer_id&from&to` — aggregate + `by_owner[]` (each with `MetricsBlockDTO`)
- `GET /api/v1/team/budget-approvals/mine?customer_id` — MB own requests
- Extend `TeamOverview`: `pending_approvals_count`, `pending_for_me_count`

Permissions: metrics — `team:read`; mine — authenticated team member.

### Backend

- CH query grouped by `owner_user_id` with team scope filter
- Reuse list metrics helpers; hard cap `by_owner` length (e.g. 50)

### Frontend

- Extend `web/src/domains/team/` — KPI panel, **small** leaderboard (not campaigns directory)
- MB panel: my approval status
- Nav badge for TL pending count

### Done when

- TL sees team ROI ranking for date range (bounded rows)
- MB sees own pending/denied/approved without opening TL-only tab

---

## M7 — Campaign operational signals

**Status:** shipped

**Goal:** Pacing/budget on list; stats in editor — **no new list table columns explosion**.

Prefer badges on existing list columns + editor panel over widening the campaigns grid.

### OpenAPI

Extend campaign list metrics / batch DTO:

- `budget_burn_pct`, `pacing_mode`, `pacing_health` (`ok | drift | exhausted`)
- `metrics_stale`, `metrics_as_of`

Ensure `GET /api/v1/campaigns/{id}/stats` documented (hourly/daily buckets).

### Backend

- Derive `pacing_health` from spend velocity vs EVEN mode
- Surface CH freshness on list metrics batch

### Frontend

- List: pacing badge, budget burn text, stale hint (inline cells — not 10 new columns)
- Editor: `campaign_stats_panel.tsx` — time series as **simple list or placeholder chart**, not wide table
- Button: Export `campaign-stats` deep link

### Done when

- Pacing drift visible on list without export job
- Editor stats uses existing stats API; errors surfaced

---

## M9 — Styling pass (separate track)

**Status:** shipped

After M1–M7 logic is stable:

- Migrate unstyled sections to `PageLayout` / `DirectoryPageShell` where appropriate
- Dashboard bento grid; Export Hub Operations nav (Phase C/D in scope doc)
- Cold-path polish: `BentoSection` + `PageSectionStack` on exports, alerts, schedules, integrations Google Sheets; `adminTypography` instead of raw `text-sm` in domains
- `bash scripts/ci/admin/ui_slop.sh`, `npm run typecheck`, E2E L1+ with `waitForResponse` (operator-run before release claim)

Do not add heavy tables during styling — campaigns list keeps existing width-probe table only.

---

## Verification (operator-run)

- `bash scripts/ci/admin/openapi.sh` — every milestone touching API
- Scoped Go tests: `reportjob`, `dashboardadmin`, `platformadmin`, `smartalerts` as touched
- RBAC: curl as seeded MB/TL — forbidden routes return 403 with error envelope
- Frontend: `read_lints` on touched TSX during logic milestones; full `web.sh` before M9 claim

---

## Out of scope (explicit)

- In-browser report tables (`/reports/*` runners)
- Full fraud hub, RTB console, creative studio
- Multi-sheet XLSX per job
- Client-side filter/sort over full datasets
- New campaigns-style directory for analytics

---

## Related docs

- `docs/CONTROL_PLANE_UI_SCOPE.md` — KEEP vs FREEZE
- `REPORT_EXPORT_XLSX_SHEETS_SPEC.md` — export formats (update when M0 lands)
- `deploy/operator/roles.yaml` — MB / TL permissions
- `.cursor/rules/ui.mdc`, `frontend-slop.mdc` — layout and table performance

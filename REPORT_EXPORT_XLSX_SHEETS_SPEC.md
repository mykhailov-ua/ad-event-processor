# Report Export: Excel (XLSX) and Google Sheets — Technical Specification

Status: **implemented** (M0, branch).  
Owner: control plane / reports / admin UI (Export Hub).  
Canonical surfaces: `internal/reportjob/`, `internal/reports/export/`, `internal/integrations/googlesheets/`, `web/src/domains/exports/`, `web/src/domains/integrations/`, `api/openapi/paths/ops_reports.yaml`.

Binding rules (agents and reviewers):

| Rule file | Applies to |
| :--- | :--- |
| `.cursor/rules/cold-path.mdc` | Handlers, workers, OAuth store, mutation validation S1–S7 |
| `.cursor/rules/data-layer.mdc` | Postgres pool budget, ClickHouse read-only analytics, no hot-path coupling |
| `.cursor/rules/control-plane.mdc` | RBAC on new routes, outbox if config side effects |
| `.cursor/rules/ui.mdc` | Export Hub UI, ErrorBlock, cold-path fetch |
| `.cursor/rules/frontend-slop.mdc` | Export Hub only (not legacy `domains/reports/`) |
| `.cursor/rules/anti-slop.mdc` | Verification honesty, no fake "wired", holdout tests |
| `.cursor/rules/agent-self-initiative.mdc` | Scope boundary: no shell/layout/deploy theater |

**Agent mandate:** treat this doc as the shipped contract for XLSX and Google Sheets export. Update it when behavior or routes change.

---

## 1. Purpose

Add operator-facing export destinations for analytics report jobs:

1. **Excel (`.xlsx`)** — async job artifact downloadable from Export Hub (same lifecycle as CSV).
2. **Google Sheets** — async push to a new or existing spreadsheet via Google Sheets API (OAuth per operator).

Non-goals (v1):

- Multi-sheet Excel workbooks per report dimension.
- Live two-way sync with Sheets.
- Billing ledger XLSX (separate pipeline in `internal/billingadmin/export_handlers.go`).
- Hot-path (`/track`, Redis Lua, stream producer) changes (`hot-path.mdc`, `data-layer.mdc`).

---

## 2. Current state (shipped baseline)

### 2.1 Architecture

```
Export Hub (/exports)
  -> POST /api/v1/reports/jobs (ReportJobSpec: format, destination, google_sheet)
  -> ReportJobRunner (PG worker batch=4, or in-memory goroutine)
  -> internal/reports/export (CSV / XLSX / JSON / ZIP writers)
  -> optional: internal/integrations/googlesheets (OAuth + batch writer)
  -> ClickHouse + Postgres paginated reads
  -> REPORT_EXPORT_DIR on disk (download destination) OR Google Sheets API push
  -> GET .../jobs/{id}/download (download destination only)
  -> GET .../jobs/{id} includes spreadsheet_url when destination=google_sheet
```

| Component | Path | Notes |
| :--- | :--- | :--- |
| HTTP | `internal/reportjob/http_handlers.go` | `exports:run` / `exports:read` |
| Runner | `internal/reportjob/report_jobs.go` | Formats: `csv`, `xlsx`, `json`, `zip`; extended timeout for Sheets |
| PG worker | `internal/reportjob/report_jobs_worker.go` | Poll 2s; **batch 4** jobs (`SKIP LOCKED`) |
| Writers | `internal/reports/export/` | CSV + `writeReportXLSX` via `csv_to_xlsx.go` |
| Google Sheets | `internal/integrations/googlesheets/` | OAuth store, batch writer, HTTP handlers |
| Bridge | `internal/controlplane/google_sheets_bridge.go` | Wires handlers when encryption key present |
| UI | `web/src/domains/exports/export_hub.tsx` | Format xlsx, destination download/google_sheet |
| Integrations UI | `web/src/domains/integrations/integrations_google_sheets.tsx` | Connect / disconnect OAuth |
| Unstyled shell | `web/src/shell/unstyled/` | M0 layout primitives (logic-only markup) |
| API client | `web/src/api/reports_api.ts`, `integrations_api.ts` | Jobs + Sheets status |
| Wiring | `internal/controlplane/reports_bridge.go` | `ExportChunkMaxBytes`, license row limits |
| Migrations | `00145`, `00146` (ingest), `00016` (identity) | OAuth tokens + job columns |

### 2.2 OpenAPI drift (remaining)

| Issue | Runtime | OpenAPI |
| :--- | :--- | :--- |
| POST jobs status | **201** | May still document **202** — verify bundle |
| Download content-type | xlsx/zip/json paths | Confirm all MIME types in ops_reports.yaml |

`format=xlsx`, `destination=google_sheet`, and `google_sheet` sub-object are in `ops_reports.yaml`.

### 2.3 Catalog writers

Previously missing CSV writers for seven `LiveReportExportKeys()` entries are implemented in `internal/reports/export/report_job_export.go` (including `wire-signal-breakdown`, `click-log`, cohort/toggle keys). Re-run export job integration tests when adding new catalog keys.

### 2.4 Existing limits

| Knob | Value | Source |
| :--- | :--- | :--- |
| Date range | max 90 days | `report_jobs.go` |
| Row limit default / max | 100k / 5M | `export_row_limit.go` |
| License-gated cap | 1k rows | chunk bytes < 2 MiB |
| RTB keys + license_gated | 1k rows | same |
| CH export page size | 1000 | `report_export_paginate.go` |
| CH query timeout (reports) | 10s | `internal/reports/clickhouse/core.go` |
| CH query gate | sem default **8** | `CH_QUERY_MAX_CONCURRENCY`, `clickhouse_query.go` |
| Job run timeout | **2 min** | `reportJobRunTimeout` |
| Redaction | server RBAC profile | `export_redaction_profiles.go` |

---

## 3. Goals

### 3.1 Functional — Excel (X-01..X-07)

| ID | Requirement |
| :--- | :--- |
| X-01 | `format=xlsx` on `POST /api/v1/reports/jobs` |
| X-02 | Same async lifecycle: enqueue, poll, download `.xlsx` |
| X-03 | One worksheet per job; columns = CSV columns for same `report_key` + redaction profile |
| X-04 | Typed cells where column semantics are numeric/datetime (money from micros or display fields) |
| X-05 | Same row/date/redaction limits as CSV |
| X-06 | Billing export out of v1 scope |
| X-07 | `fraud-evidence-pack-bulk` stays `zip` only |

### 3.2 Functional — Google Sheets (G-01..G-07)

| ID | Requirement |
| :--- | :--- |
| G-01 | `destination: download \| google_sheet` (default `download`) |
| G-02 | On COMPLETED + `google_sheet`: `spreadsheet_url` in job status (no file download) |
| G-03 | `google_sheet.mode`: `create` (new file) or `append` (existing `spreadsheet_id`) |
| G-04 | OAuth scope `https://www.googleapis.com/auth/spreadsheets`; **per-operator** connection (v1) |
| G-05 | Refresh tokens encrypted in Postgres; not in license JWT |
| G-06 | Google API errors -> job `FAILED` + `SanitizeExportJobError` message |
| G-07 | Redaction applied before any row leaves the server |

---

## 4. Connection pool and concurrency budget

Report export is **cold path** (`cold-path.mdc`). It must not starve Postgres settlement, tracker read pools, or ClickHouse ingest. Budget uses existing env knobs (`data-layer.mdc`, `.env.example`).

### 4.1 Postgres (`pgxpool`)

| Consumer | Pool | Default max | Export usage |
| :--- | :--- | :--- | :--- |
| Control plane | Shared `DB_DSN` pool | `DB_TRACKER_MAX_CONNS` **4** | Job CRUD, `ListCustomerCampaignIDs`, hybrid PG reports |
| Settlement | Separate settle pool | `PostgresPoolSettleMaxConns` | **Must not** be used by export workers |
| Budget guard | `PG_MAX_CONNECTIONS` | operator-set | `ValidatePostgresPoolBudget()` |

**Requirements (PG-EXP-*):**

| ID | Rule |
| :--- | :--- |
| PG-EXP-01 | Export worker uses **existing** control-plane pool only; no new pool per format |
| PG-EXP-02 | Max **4** concurrent export jobs (`reportJobWorkerBatchSize`) must not each hold a PG connection for the full 2 min; release after scope resolution / PG-only segments |
| PG-EXP-03 | No N+1 campaign lookups in export loops (`cold-path.mdc`, `cold_path_static_gate.sh`) |
| PG-EXP-04 | OAuth token read/write: single sqlc query per job start; no connection held during CH/Sheets I/O |
| PG-EXP-05 | If `PG_MAX_CONNECTIONS` is set, document export share in operator runbook; do not raise `DB_TRACKER_MAX_CONNS` without `ValidatePostgresPoolBudget()` green |

**Verify:**

```bash
go test ./internal/config/ -run TestValidatePostgresPoolBudget -count=1
bash scripts/ci/static/cold_path_static.sh
```

### 4.2 ClickHouse (read-only analytics)

| Knob | Default | Role |
| :--- | :--- | :--- |
| `CH_MAX_CONNS` | 8 | Native driver pool (`clickhouse_connect.go`) |
| `CH_QUERY_MAX_CONCURRENCY` | 8 | In-process sem on `database.ClickHouseQuery` |
| Per-query timeout (reports) | 10s | `ReportClickHouseQueryTimeout()` |
| Gate timeout | 30s | `ClickHouseQuery` wrapper |
| `max_memory` per query | 1 GiB | SETTINGS via `ClickHouseQuery` |

**Export query pattern:** paginate `LIMIT 1000 OFFSET n` + separate `count()` per page (`report_export_paginate.go`). One job = **O(rows/1000)** sequential CH round-trips.

**Requirements (CH-EXP-*):**

| ID | Rule |
| :--- | :--- |
| CH-EXP-01 | Each CH call uses `ReportClickHouseQueryTimeout()` (10s); job ctx capped at **2 min** total |
| CH-EXP-02 | At most **one** in-flight CH query per export goroutine (no parallel page fetches inside one job) |
| CH-EXP-03 | With worker batch **4**, worst case **4** concurrent CH queries -> must stay within `CH_QUERY_MAX_CONCURRENCY`; if gate returns `ErrClickHouseQueryRejected`, job fails with retryable operator message (not hang) |
| CH-EXP-04 | No new ClickHouse write paths; analytics read-only (`data-layer.mdc`) |
| CH-EXP-05 | XLSX/Sheets writers consume rows from existing paginate iterators — no second full-table CH scan |

**Capacity note:** 4 jobs x 1 query each fits default sem=8. If operator raises batch size or adds parallel page fetch, **must** re-run budget and add metric `ad_report_export_ch_gate_rejected_total`.

**Verify:**

```bash
go test ./internal/database/ -run TestClickHouseQuery -count=1
# integration: make test-integration (CH testcontainers) when claiming CH wiring
```

### 4.3 Redis / hot path

| ID | Rule |
| :--- | :--- |
| REDIS-EXP-01 | **No** Redis commands on export path (`cold-path.mdc`: no KEYS/FLUSH*) |
| REDIS-EXP-02 | No unified-filter Lua, no stream XADD, no budget debit (`hot-path.mdc`, `data-layer.mdc`) |

### 4.4 Google Sheets HTTP client

| ID | Rule |
| :--- | :--- |
| GS-EXP-01 | Dedicated `http.Client` with timeout **30s** per batch request; job ctx **2 min** (may need Phase 3 timeout bump — see open decisions) |
| GS-EXP-02 | Batch writes: max **10k cells** per `batchUpdate` (Google quota); exponential backoff on 429/503 |
| GS-EXP-03 | No connection pool shared with postback/cost-sync OAuth clients; reuse token refresh helper pattern only (`internal/costsync/provider/oauth_google.go`) |
| GS-EXP-04 | OAuth token refresh: at most **1** refresh per job; store new refresh token in PG in same txn as audit row |

### 4.5 Disk and memory (XLSX)

| ID | Rule |
| :--- | :--- |
| DISK-EXP-01 | Artifacts under `REPORT_EXPORT_DIR` (default `./data/report-export`); same TTL/cleanup as CSV |
| DISK-EXP-02 | XLSX writer **streams** rows; forbidden: materialize full `[]Row` for 5M rows in RAM |
| DISK-EXP-03 | Optional byte cap for xlsx: if artifact exceeds `max_export_chunk_bytes` tier, fail job with public error (xlsx ~3–5x CSV size) |

---

## 5. API changes

### 5.1 `ReportJobSpec` (extend)

```yaml
format:
  enum: [csv, json, zip, xlsx]
destination:
  enum: [download, google_sheet]
  default: download
google_sheet:
  type: object
  properties:
    mode: { enum: [create, append] }
    spreadsheet_id: { type: string }
    sheet_title: { type: string }
```

Validation:

- `destination=google_sheet` requires connected OAuth for `exported_by` operator.
- `append` requires non-empty `spreadsheet_id`.
- `format=xlsx` incompatible with `fraud-evidence-pack-bulk` (use zip).

### 5.2 `ReportJobStatus` (extend)

```yaml
spreadsheet_url: { type: string, format: uri }
spreadsheet_id: { type: string }
```

### 5.3 Google OAuth routes (new)

| Method | Path | Permission |
| :--- | :--- | :--- |
| GET | `/api/v1/integrations/google-sheets/connect` | `settings:write` or `integrations:write` |
| GET | `/api/v1/integrations/google-sheets/callback` | callback (state CSRF) |
| GET | `/api/v1/integrations/google-sheets/status` | `settings:read` |
| DELETE | `/api/v1/integrations/google-sheets` | `settings:write` |

Env: `GOOGLE_SHEETS_CLIENT_ID`, `GOOGLE_SHEETS_CLIENT_SECRET`, redirect URI = control plane public URL.

### 5.4 Catalog

`ReportCatalogRow.export_formats` adds `xlsx` where CSV writer exists; optional `supports_google_sheet: true`.

### 5.5 Download

`GET .../download` for `destination=google_sheet` -> **409** with code `USE_SPREADSHEET_URL` (no artifact).

---

## 6. Backend design

### 6.1 Packages

| Package | Change |
| :--- | :--- |
| `internal/reportjob/` | Validate new fields; extend status DTO; optional job timeout env for Sheets |
| `internal/reports/export/` | Refactor row emitter; add `writeReportXLSX`; dispatch in `register.go` |
| `internal/integrations/googlesheets/` (new) | OAuth store, Sheets client, batch writer |
| `internal/controlplane/*_bridge.go` | Wire OAuth routes + env |
| `internal/reports/catalog.go` | `export_formats` |

Dependency: `github.com/xuri/excelize/v2` (or equivalent; license review in PR). **Not** on hot path — no `escape_heap_gate` for ingest files.

### 6.2 Row emitter refactor (required for X-01 and G-07)

Extract shared interface from CSV writer:

```go
type ReportRowWriter interface {
    WriteHeader(cols []string) error
    WriteRow(values []string) error
    Close() error
}
```

CSV and XLSX implement; Google Sheets adapter batches into API calls.

### 6.3 Google Sheets job flow

```
POST job (destination=google_sheet)
  -> load OAuth token (PG, decrypt)
  -> paginate CH/PG same as CSV
  -> batchUpdate / append rows
  -> COMPLETED + spreadsheet_url
```

Audit: `admin_audit_log` event `report.export.google_sheet` with masked `spreadsheet_id`.

### 6.4 Phase 0 (prerequisite)

1. Implement or remove 7 missing CSV writers.
2. Sync OpenAPI (zip, status code, content-types).
3. Align catalog `export_formats` with writers.

---

## 7. Frontend (Export Hub)

Scope: `web/src/domains/exports/` only (`agent-self-initiative.mdc` **SD-PAGE** — no new domain layout constants).

| UI | Change |
| :--- | :--- |
| Format select | Add **Excel (.xlsx)** when catalog allows |
| Destination | **Download file** / **Google Sheet** |
| Google block | If not connected: link to Settings + `StubBanner` |
| Append | Optional `Spreadsheet ID` field |
| Job lifecycle | COMPLETED + `spreadsheet_url` -> primary **Open in Google Sheets** link |
| Download button | Hidden when `destination=google_sheet` |

API: extend `reports_api.ts` + new `integrations_api.ts` after `bash scripts/ci/admin/openapi.sh`.

RBAC: server `exports:run` / `403`; nav hide is not security (`frontend-slop.mdc` **RB-L***).

---

## 8. Security and compliance

| Topic | Requirement |
| :--- | :--- |
| RBAC | `exports:run` POST; `exports:read` GET/download |
| Redaction | Same profiles as CSV before XLSX/Sheets output |
| Secrets | Encrypt OAuth refresh at rest; never log tokens |
| PII | Sheets cells follow redaction; no raw IP/UA (`data-layer.mdc`, `pkg/piihash`) |
| License | Optional JWT features `report_export_xlsx`, `report_export_google_sheets` |
| Self-hosted | Operator registers OAuth redirect on their control plane hostname |

---

## 9. Performance targets (cold path — not tracker SLA)

From `core.mdc`: tracker p95/p99 SLAs do **not** apply. Export is async admin work.

| Scenario | Target | Measurement |
| :--- | :--- | :--- |
| CSV job 100k rows | Complete < 2 min | job `COMPLETED`, default timeout |
| XLSX job 100k rows | Complete < 5 min | may require `REPORT_JOB_RUN_TIMEOUT_SEC` env (Phase 1) |
| 4 concurrent jobs | No `ErrClickHouseQueryRejected` storm at default CH sem=8 | metrics + integration |
| PG pool | No `too many clients` on compose `minimal` profile | stack smoke |
| Sheets 50k rows | Complete or fail with public error within extended timeout | manual T1+ |

Forbidden claims (`anti-slop.mdc`): citing unit microbench as prod SLA; "wired" from mock CH only.

---

## 10. Testing and verification

| Tier | Command / artifact |
| :--- | :--- |
| Unit | `go test ./internal/reportjob/ ./internal/reports/export/ -short -count=1` |
| Holdout | `TestWriteReportXLSX_rowCountMatchesCSV_holdout` (fails if xlsx path removed) |
| Handler | httptest: POST `google_sheet` without OAuth -> 400/403 |
| OpenAPI | `bash scripts/ci/admin/openapi.sh` |
| Cold static | `bash scripts/ci/static/cold_path_static.sh`, `anti_slop.sh` |
| UI | `cd web && npm run typecheck`; `bash scripts/ci/admin/ui_slop.sh` on touched paths |
| Integration | `make test-integration` when CH/PG containers used |
| E2E (optional) | Playwright **L2**: create xlsx job + `waitForResponse` on `/api/v1/reports/jobs` |

Paste command + exit code in PR (`quality.mdc`, `anti-slop.mdc`).

---

## 11. Delivery phases

| Phase | Deliverable | DoD gate |
| :--- | :--- | :--- |
| **0** | Missing CSV writers + OpenAPI sync | Section 12 Phase 0 checklist |
| **1** | `format=xlsx`, writer, Export Hub, download | Section 12 Phase 1 checklist |
| **2** | Google OAuth settings + PG migration | Section 12 Phase 2 checklist |
| **3** | `destination=google_sheet`, push worker, UI link | Section 12 Phase 3 checklist |
| **4** (opt) | Report schedules xlsx/sheets; billing xlsx | Product approval |

---

## 12. Definition of Done (DoD)

### 12.1 Global DoD (all phases that ship code)

- [ ] OpenAPI matches handlers (`openapi.sh` exit 0).
- [ ] RBAC: `exports:run` / `exports:read` on all new/changed routes; httest deny row for role without permission.
- [ ] No `_ = json.Unmarshal` / silent `w.Write` in handlers (`anti_slop.sh`).
- [ ] No Redis/CH writes on export path; no ingest imports (`boundaries.mdc`).
- [ ] `cold_path_static.sh` + `cold_path_json.sh` exit 0 on touched packages.
- [ ] Holdout or negative test for non-obvious behavior (`testing.mdc`).
- [ ] PR lists verification commands with exit codes (`anti-slop.mdc` **Agent checklist**).
- [ ] Agent scope: no unrelated shell/layout/deploy (`agent-self-initiative.mdc` **SD-*** avoided).

### 12.2 Phase 0 DoD

- [ ] All `LiveReportExportKeys()` either have CSV writer or are removed from catalog export list.
- [ ] OpenAPI includes `zip`; POST status documented as **201**; download documents csv/json/zip content-types.
- [ ] `TestWriteReportCSV_supportsAllLiveReportKeys` fails on `unsupported report_key` (not dependency skip).

### 12.3 Phase 1 DoD — Excel

- [ ] `POST /api/v1/reports/jobs` with `format=xlsx` produces downloadable `.xlsx` for at least 3 catalog keys (traffic, fraud, portfolio class).
- [ ] Row count matches CSV for same spec (holdout test on one key).
- [ ] Redaction profiles produce identical column sets vs CSV (spot-check test).
- [ ] Export Hub shows xlsx option; download works; `ErrorBlock` on failed job.
- [ ] **PG-EXP-01..05** and **CH-EXP-01..05** verified (no pool budget regression on default `.env.example`).
- [ ] **DISK-EXP-01..03**: streaming writer; job fails cleanly when row limit exceeded.
- [ ] Manual: open file in Excel/LibreOffice; numeric columns not all strings.

### 12.4 Phase 2 DoD — Google OAuth

- [ ] Settings UI: connect / disconnect / status (`StubBanner` on misconfig).
- [ ] Tokens stored encrypted; disconnect removes row.
- [ ] OAuth callback CSRF state validated.
- [ ] Audit log entry on connect/disconnect.
- [ ] **GS-EXP-04** token refresh at most once per downstream job.

### 12.5 Phase 3 DoD — Google Sheets export

- [ ] `destination=google_sheet`, `mode=create` -> COMPLETED + valid `spreadsheet_url`.
- [ ] `mode=append` with valid `spreadsheet_id` appends rows.
- [ ] Without OAuth: POST fails closed (4xx), UI shows connect prompt (not empty success).
- [ ] Download endpoint returns 409 for sheets jobs.
- [ ] Job failure surfaces sanitized message (`export_error_public.go` pattern).
- [ ] **GS-EXP-01..03** batching and backoff covered by unit test or integration mock.
- [ ] T1+ manual: `curl -sf :8188/health`; complete one sheets job; open URL in browser.

---

## 13. Open decisions (operator sign-off)

| # | Question | Default if silent |
| :--- | :--- | :--- |
| 1 | Google OAuth: per-operator vs per-customer vs service account | Per-operator (spec v1) |
| 2 | License gating for xlsx/sheets | Same as CSV unless SKU adds flags |
| 3 | `REPORT_JOB_RUN_TIMEOUT_SEC` for large xlsx/sheets | 2 min CSV; 10 min xlsx/sheets env override |
| 4 | Sheets append: new tab vs append rows on default sheet | New tab per export |
| 5 | Separate xlsx byte cap multiplier | Fail at same row limit first; byte cap in Phase 1.1 if needed |

---

## 14. Workaround today (no implementation)

Export Hub -> **CSV** -> open in Excel or File -> Import in Google Sheets. Scheduled: `POST /api/v1/report-schedules` with `format: csv`.

---

## 15. Related files (implementation index)

| Area | Path |
| :--- | :--- |
| Job HTTP | `internal/reportjob/http_handlers.go` |
| Job runner | `internal/reportjob/report_jobs.go`, `report_jobs_worker.go` |
| Export writers | `internal/reports/export/report_job_export.go`, `register.go` |
| Row limits | `internal/reportjob/export_row_limit.go` |
| CH timeout | `internal/reports/clickhouse/core.go` |
| CH gate | `internal/database/clickhouse_query.go`, `clickhouse_query_config.go` |
| PG pools | `internal/database/postgres_connect.go`, `internal/config/postgres_pool_budget.go` |
| OpenAPI | `api/openapi/components/schemas/ops_reports.yaml`, `paths/ops_reports.yaml` |
| Export Hub | `web/src/domains/exports/export_hub.tsx`, `use_export_hub_page_workspace.ts` |
| API client | `web/src/api/reports_api.ts` |
| OAuth reference | `internal/costsync/provider/oauth_google.go` |

---

## 16. Agent execution protocol (when implementing)

When operator cites `@agent-self-initiative.mdc` or rejects scope creep:

```text
Operator ask (verbatim): ...
In scope: internal/reportjob, internal/reports/export, web/src/domains/exports, openapi
Out of scope: page_layout, deploy/marketing, hot-path, billing export (unless Phase 5)
Reference: billing_exports.tsx / export_hub.tsx patterns
Pool budget: PG-EXP-* CH-EXP-* checked
I will NOT: new domain layout classes, deploy without ask, fake verification
```

Completion block required (`agent-self-initiative.mdc` **Required completion block**).

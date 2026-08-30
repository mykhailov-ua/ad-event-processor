// Package views provides saved report view CRUD and schedule validation for admin reporting.
//
// Role:
//   - ViewsHTTPHandlers: GET/POST/PUT/DELETE /api/v1/views; registered from controlplane adminapi_wire_domains.go.
//   - ViewsStore persists to Postgres report_saved_views (store_pg.go); in-memory map when pool nil (tests).
//   - ValidateReportScheduleForActor validates reportjob schedule specs (RBAC, license, customer binding).
//   - validate.go enforces report_key against reports.LiveReportExportKeys via SetLiveReportExportKeys (reports/views_bridge.go init).
//
// Topology:
//   - Types re-exported from internal/reports (views_exports.go) for controlplane wiring.
//   - reportjob schedule worker calls ValidateReportScheduleForActor before enqueue.
//
// Invariants:
//   - Saved view spec JSON max 8 KiB; allowed keys: from, to, compare, campaign_id, limit, columns, from_offset_days, to_offset_days.
//   - authz.MaskMasked buyers blocked from ops-only report keys (filter-rejects, fraud-evidence-pack, layer-desync-summary).
//   - Masked buyers capped to 7-day range on saved views and schedules.
//   - Bound customer actors cannot save views or schedules for a different customer_id (ErrForbidden).
//   - Shared views apply effective mask policy so buyers cannot inherit operator-only report keys.
//
// Forbidden:
//   - ClickHouse queries or report math in this package.
//   - Import internal/reportjob (validation only; no runner coupling).
//
// Verify:
// go list -e ./internal/reports/views/
// go test ./internal/reports/views/ -short -count=1
// go test ./internal/reports/views/ -short -run TestValidateSavedViewActorPolicy_buyerBlocksOpsReport_holdout -count=1
// go test ./internal/reports/views/ -short -run TestValidateSharedSavedViewForActor_opsSharedReportBuyerDenied_holdout -count=1
// go test ./internal/reports/views/ -short -run TestValidateSavedViewRangeCap_buyerExceedsSevenDays_holdout -count=1
package views

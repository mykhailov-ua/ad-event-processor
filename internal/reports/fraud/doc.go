// Package fraud registers fraud and IVT report HTTP routes and export helpers on the reports host.
//
// Role:
//   - init sets reports.SetFraudRegistrar(Register) and reports.SetFraudExports; wired via blank import in controlplane/register.go.
//   - HTTP routes (GET /api/v1/reports/*): ivt-by-source, fraud-breakdown, customer-fraud-by-type, wire-signal-breakdown,
//     layer-desync-summary, layer-desync-drilldown, customer-fraud-by-dimension, signal-effectiveness,
//     fraud-evidence-pack, customer-fraud-evidence, silent-reject-impression-funnel (ghost-impression-funnel URL alias only),
//     ml/score-distribution, ml/shadow-delta, ml/feature-spikes.
//   - fraud_report_scrub.go masks reason and PII fields for authz.MaskMasked actors on customer fraud reports.
//   - fraud_evidence_pack_sign.go signs evidence JSON; bulk ZIP export for report jobs.
//   - filter-rejects lives in parent internal/reports (rejects.go), not this subpackage.
//
// Topology:
//   - CH queries via database.ClickHouseQuery; scrub helpers read authz.SnapshotFromContext.
//   - exports.go exposes FraudExportAPI callbacks for parent reports and export subpackage.
//
// Invariants:
//   - fraud-evidence-pack and customer-fraud-evidence denied for authz.MaskMasked (403).
//   - Silent reject analytics use silent_reject_event column semantics (not legacy ghost_event).
//   - Evidence pack requires click_id; missing HMAC secret returns 503.
//   - ML shadow delta snapshot freshness gate: stale after 24h (ml_shadow_delta_snapshot.go).
//
// Forbidden:
//   - ghost_* report keys or ghost_event CH column names in new queries.
//   - balance_ledger as fraud analytics source.
//
// Verify:
// go list -e ./internal/reports/fraud/
// go test ./internal/reports/fraud/ -short -count=1
// go test ./internal/reports/fraud/ -short -run TestSilentRejectImpressionFunnelQuery_joinsFraudEvents_holdout -count=1
// go test ./internal/reports/fraud/ -short -run TestFraudEvidencePack_signAndVerify_holdout -count=1
// go test ./internal/reports/fraud/ -short -run TestCustomerFraudEvidencePerms_buyerMaskedDenied_holdout -count=1
// go test ./internal/reports/fraud/ -short -run TestScrubFraudBreakdownRow_holdoutMasksReason -count=1
package fraud

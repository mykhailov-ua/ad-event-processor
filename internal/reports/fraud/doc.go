// Package fraud registers fraud and IVT report HTTP routes on reports.ReportsHTTPHandlers.
//
// Role:
//   - Routes: filter-rejects, fraud-breakdown, layer-desync, silent-reject-impression-funnel, ML reports, evidence packs.
//   - init registers reports.SetFraudRegistrar and fraud export API for report jobs.
//   - CH queries with campaign/customer scrub helpers from reports package.
//
// Invariants:
//   - fraud-evidence-pack denied for authz.MaskMasked snapshots.
//   - Silent reject analytics use silent_reject_event column semantics (not legacy ghost_event).
//
// Verify:
//
//	go test ./internal/reports/fraud/ -short -count=1
package fraud

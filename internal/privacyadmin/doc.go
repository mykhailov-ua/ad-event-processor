// Package privacyadmin serves GDPR/consent export and erasure request HTTP for compliance operators.
//
// Role:
//   - HTTP under /api/v1/privacy/* for data subject requests, export jobs, and consent audit reads.
//   - Delegates CH/PG scrub to reports and domain consent_store helpers.
//
// Topology:
//   - Wired from controlplane; async export jobs poll like reportjob pattern.
//
// Invariants:
//   - Erasure requests audit before enqueue; duplicate request id rejected.
//   - Export bundles redact third-party tokens (supportbundle redact rules).
//
// Forbidden:
//   - Hard delete without outbox-coordinated CH TTL and PG anonymize steps.
//
// Verify:
//
//	go test ./internal/privacyadmin/... -short -count=1
package privacyadmin

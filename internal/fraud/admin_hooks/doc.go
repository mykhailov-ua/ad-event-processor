// Package admin_hooks is the HTTP client surface ivt-detector and fraud-scorer use to enqueue cold-path fraud side effects.
//
// Role:
//   - ControlplaneClient posts to /api/v1/ops/blacklist and /api/v1/ops/fraud-threat on control :8188.
//   - BlacklistBlocker port wraps batch enqueue for boost, blacklist, and silent_reject actions.
//   - ResolveManagementBlockerFromConfig builds the client from management URL, port, and admin API key.
//
// Topology:
//   - Subpackage of internal/fraud; wired at cmd/ivt-detector and cmd/fraud-scorer bootstrap.
//   - HTTP handlers persist outbox rows (ML_SCORE_BOOST, ML_SILENT_REJECT, blacklist ops); OutboxWorker applies Redis.
//   - Not invoked from fraudadmin UI hooks or campaign fraud PATCH handlers on the request thread.
//
// Invariants:
//   - Enqueue paths are cold only: management HTTP -> Postgres outbox -> async Redis fan-out.
//   - silent_reject enqueue adds blacklist:fraud entry; does not UPDATE campaigns.silent_reject_enabled (ANTIFRAUD.md).
//   - ghost action alias on wire is normalized to silent_reject before outbox insert.
//   - Client calls must not block ivt-detector scan loop on CH queries; rules fetch CH, hooks only POST enqueue.
//
// Forbidden:
//   - Direct Redis SADD or ml:score:boost writes from this package (outbox worker owns side effects).
//   - Import internal/ingest hot handlers or run synchronous enforcement on /track.
//
// Verify:
//
//	go list -e ./internal/fraud/admin_hooks/
//	go test ./internal/fraud/ -short -run TestTrackerDepGraphExcludesFraudScoringRuntime -count=1
//	make test-integration (TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag in internal/outbox)
package admin_hooks

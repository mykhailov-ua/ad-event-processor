// Package privacyadmin owns consent PG state, privacy erasure orchestration, and retention workers.
//
// Role:
//   - Store (store.go): RecordConsent, UpdateCampaignConsentRequirements, CreatePrivacyErasureRequest.
//   - Workers (workers.go): ErasureWorker, ConsentRetentionWorker, EventsRetentionWorker (batch PG deletes).
//   - VerifyConsentHMAC for signed consent webhook bodies.
//   - HTTP lives outside this package: POST /api/v1/consent and GET /api/v1/ops/consent/proofs in opsadmin;
//     controlplane privacy_bridge.go implements Host and delegates store calls.
//
// Topology:
//   - ErasureWorker interval from config ErasureWorkerIntervalMs (serve_workers.go).
//   - Consent and campaign consent mutations enqueue outbox in the same PG transaction (SYNC_USER_CONSENT,
//     UPDATE_CAMPAIGN_CONSENT); PURGE_USER_DATA outbox applies Redis purge via outbox worker.
//   - CH fraud-event delete uses Host.ClickHouseDeleteFraudEventsByUser on the erasure tick.
//
// Invariants:
//   - Erasure pipeline statuses: PENDING -> PGANONYMIZED -> REDISPURGED -> CHPURGED -> COMPLETED (or FAILED).
//   - PG anonymize runs before PURGE_USER_DATA outbox enqueue; subject_user_id cleared only after CH step.
//   - Consent retention uses Host.ConsentRetentionMonths(); events retention batches 10_000 rows per tick.
//   - Empty consent HMAC secret rejects VerifyConsentHMAC with ErrConsentInvalidSignature.
//
// Forbidden:
//   - Tracker or ingest hot-path imports.
//   - Synchronous Redis purge or CH delete on consent webhook accept path (outbox + worker only).
//   - Hard delete of PG events without the erasure state machine and audit row.
//
// Verify:
//
//	go list -e ./internal/privacyadmin/
//	go test ./internal/controlplane/ -short -run 'Consent|Erasure|EventsRetention' -count=1
package privacyadmin

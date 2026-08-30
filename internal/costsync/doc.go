// Package costsync ingests ad network cost reports and attributes spend to campaigns for recon and margin guard.
//
// Role:
//   - Worker (worker_sync.go) polls provider APIs on hourly and 15-minute ticks; RunManual serves POST /api/v1/cost-sync/run.
//   - worker_credential.go decrypts PG credentials, refreshes OAuth tokens, and holds the PG advisory lock (single leader).
//   - worker_reconcile.go writes reconciliation ledger rows after import.
//   - cost_sync_batch.go batches PG campaign_costs inserts and ECB FX conversion to USD micro-units.
//   - clickhouse.go inserts ad_event_processor.cost_snapshots when CH conn is wired.
//   - attribute.go matches network placement tokens to ClickHouse clicks (token or spread modes).
//   - credential_fields.go defines per-network admin credential schemas; ingest_key.go builds dedup ingest keys.
//   - token_mapping_apply.go applies platform-campaign link token overrides before fetch.
//
// Topology:
//   - Started from internal/control/run.go with optional WithClickHouse, WithClickAttributor, WithOAuth WorkerOptions.
//   - Admin HTTP: GET/PUT/DELETE /api/v1/cost-sync/credentials/{network}, GET history/networks, POST run (controlplane handlers).
//   - Vendor HTTP clients live in internal/costsync/provider; this package owns PG/CH persistence and worker orchestration.
//   - platformsync reuses costsync.Credential for remote pause/resume on linked campaigns.
//
// Invariants:
//   - Credential secrets encrypted at rest; credential_fields_test and MaskExtraConfigForResponse hide secrets in API responses.
//   - Attribute apply idempotent per (sync_run_id, campaign_id, placement_id) via cost_sync_attribution_applied ON CONFLICT DO NOTHING.
//   - Duplicate import of the same ingest key is a no-op (TestIdempotency_DuplicateImport, TestFault_DuplicateReportLedgerBalanced).
//   - Only one cost-sync leader per cluster (PG advisory lock; TestWorker_AdvisoryLock).
//   - CH and PG writes batched; no per-row synchronous PG inside provider fetch loops.
//
// Forbidden:
//   - Hot-path tracker imports.
//   - Storing raw API secrets in CH columns or structured logs.
//
// Verify:
//
//	go list -e ./internal/costsync/
//	go test ./internal/costsync/ -short -count=1
//	go test ./internal/costsync/ -short -run TestIdempotency_DuplicateImport -count=1
//	go test ./internal/costsync/ -short -run TestFault_DuplicateReportLedgerBalanced -count=1
//	go test ./internal/costsync/ -short -run TestWorker_AdvisoryLock -count=1
//	go test ./internal/costsync/ -short -run TestMaskExtraConfigForResponse_hidesSecrets -count=1
//	go test ./internal/costsync/ -short -run TestOAuthRefresh_Meta -count=1
//	go test ./internal/costsync/ -short -run TestAttribute_tokenMatch_integration -count=1
package costsync

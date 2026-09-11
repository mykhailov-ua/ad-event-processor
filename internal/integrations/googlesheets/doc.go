// Package googlesheets owns Google Sheets OAuth and export push for report jobs.
//
// Role:
// - Per-operator OAuth refresh tokens in Postgres (encrypted at rest).
// - Cold-path HTTP routes under /api/v1/integrations/google-sheets/*.
// - Sheets API v4 client with batched value writes (no hot-path usage).
//
// Invariants:
// - OAuth scope: https://www.googleapis.com/auth/spreadsheets only.
// - At most one token refresh per export job (GS-EXP-04).
// - Batch writes capped at 10k cells per request with 429/503 backoff.
// - CSV upload streams rows from disk; no full-file [][]string materialization (DISK-EXP-02).
//
// Forbidden:
// - Redis, ClickHouse writes, or tracker imports.
//
// Verify:
// go test ./internal/integrations/googlesheets/ -short -count=1
// go test ./internal/reportjob/ -short -run TestCreateReportJob_googleSheetWithoutOAuth_holdout -count=1
package googlesheets

// Package main delivers outbound postbacks from Postgres (cold path).
//
// Role:
//   - Load config; connect Postgres pool via database.Connect.
//   - Run internal/postback.PostbackWorker: poll pending postback rows, HTTP deliver, retry/backoff.
//   - Metrics on cfg.Postback.MetricsAddr (default 127.0.0.1:9119); POSTBACK_ENCRYPTION_KEY for payload crypto.
//   - Batch size and stale-processing thresholds from config postback section.
//
// Topology:
//   - Cold-path worker daemon; tools compose profile.
//   - No Redis, ClickHouse, or tracker gnet wiring in main.
//
// Invariants:
//   - Shutdown on lifecycle.WaitSignal then context cancel on worker.
//   - Does not write balance_ledger (payment/settlement handlers own ledger).
//
// Forbidden:
//   - Not on /track accept path; no synchronous postback from ingest handlers.
//
// Verify:
//
//	go test ./internal/postback/... -short -count=1
//	go build -o bin/postback-sender ./cmd/postback-sender/
package main

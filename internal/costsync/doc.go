// Package costsync ingests ad network cost reports and attributes spend to campaigns for recon and margin guard.
//
// Role:
//   - cost_sync.go polls provider APIs (Facebook, TikTok, etc.) and writes attributed rows to ClickHouse/PG.
//   - attribute.go maps network campaign ids to internal campaigns; ingest_key.go manages credential rotation fields.
//
// Topology:
//   - Worker tick started from controlplane; provider/ subpackage holds per-network HTTP clients.
//   - Uses internal/costsync/provider for vendor-specific report parsers.
//
// Invariants:
//   - Credential fields never logged; credential_fields_test guards redaction shapes.
//   - Attribute window idempotent per (network, date, campaign) key.
//   - CH writes batched; no per-row synchronous PG in hot poll loop.
//
// Forbidden:
//   - Hot-path tracker imports.
//   - Storing raw API secrets in CH columns.
//
// Verify:
//
//	go test ./internal/costsync/ -short -count=1
//	go test ./internal/costsync/ -short -run TestAttribute -count=1
package costsync

// Package opsadmin serves operator ops HTTP, stack health reads, DLQ/outbox views, and edge metric scrapers.
//
// Role:
//   - HTTP under /api/v1/ops/*, /api/v1/audit/export, and related dashboard/support/RUM/ML ops routes.
//   - ManagementOpsReader aggregates PG, Redis, ClickHouse, and Prometheus reads for incident and health panels.
//   - Background workers: filter reject rollup, metric scraper, stack health probes (started from controlplane).
//
// Topology:
//   - Wired via ops_reader_bridge.go; HTTPHandlers receive Reader, payment, consent, blacklist, and fraud preset deps.
//   - Does not own domain mutation SQL; delegates retry/recon to controlplane Host callbacks where needed.
//
// Invariants:
//   - Ops routes require shards:read or shards:write (or audit:read for export).
//   - DLQ retry endpoints idempotent per message id; inbox retry separate from legacy DLQ list.
//   - Support bundle generation respects coldpath body and timeout limits.
//
// Forbidden:
//   - Tracker hot path imports.
//   - Direct Redis FLUSH or KEYS from ops handlers.
//
// Verify:
//
//	go test ./internal/opsadmin/ -short -count=1
//	go test ./internal/opsadmin/ -short -run TestStackHealth -count=1
package opsadmin
